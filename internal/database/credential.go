// Copyright (c) Mews Systems
// SPDX-License-Identifier: MPL-2.0

package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// oidcExchangeAudience is the audience Entra expects for federated client assertions.
const oidcExchangeAudience = "api://AzureADTokenExchange"

// oidcFetchTimeout bounds a single OIDC ID-token fetch.
const oidcFetchTimeout = 30 * time.Second

// BuildEntraCredential constructs the Entra credential the provider uses for
// Azure SQL and Postgres Flexible Server, with precedence matching the common
// Azure provider contract:
//
//  1. Explicit service principal: tenant_id + client_id + client_secret.
//  2. OIDC auth when enabled via ARM_USE_OIDC or use_oidc=true.
//  3. Ambient fallback chain: Azure CLI → workload identity → managed identity.
//
// The optional arguments are kept for compatibility with the provider's explicit
// OIDC configuration path and allow callers to override the environment-driven
// defaults without mutating process state.
func BuildEntraCredential(tenantID, clientID, clientSecret string, optional ...any) (azcore.TokenCredential, error) {
	useOIDC := oidcEnabled()
	oidcToken := os.Getenv("ARM_OIDC_TOKEN")
	oidcTokenFilePath := os.Getenv("ARM_OIDC_TOKEN_FILE_PATH")
	actionsIDTokenRequestURL := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL")
	actionsIDTokenRequestToken := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")

	for i, opt := range optional {
		switch i {
		case 0:
			if v, ok := opt.(bool); ok {
				useOIDC = v
			}
		case 1:
			if v, ok := opt.(string); ok {
				if v != "" {
					oidcToken = v
				}
			}
		case 2:
			if v, ok := opt.(string); ok {
				if v != "" {
					oidcTokenFilePath = v
				}
			}
		case 3:
			if v, ok := opt.(string); ok {
				if v != "" {
					actionsIDTokenRequestURL = v
				}
			}
		case 4:
			if v, ok := opt.(string); ok {
				if v != "" {
					actionsIDTokenRequestToken = v
				}
			}
		}
	}

	if tenantID != "" && clientID != "" && clientSecret != "" {
		cred, err := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
		if err != nil {
			return nil, fmt.Errorf("client secret credential: %w", err)
		}
		return cred, nil
	}

	if useOIDC {
		tID := firstNonEmpty(tenantID, os.Getenv("ARM_TENANT_ID"), os.Getenv("AZURE_TENANT_ID"))
		cID := firstNonEmpty(clientID, os.Getenv("ARM_CLIENT_ID"), os.Getenv("AZURE_CLIENT_ID"))
		if tID == "" || cID == "" {
			return nil, errors.New("ARM_USE_OIDC=true but tenant_id/client_id are not set " +
				"(supply via provider block or ARM_TENANT_ID/ARM_CLIENT_ID env vars)")
		}
		cred, err := newOIDCCredential(tID, cID, oidcToken, oidcTokenFilePath, actionsIDTokenRequestURL, actionsIDTokenRequestToken)
		if err != nil {
			return nil, fmt.Errorf("OIDC credential: %w", err)
		}
		return cred, nil
	}

	return newAmbientCredential()
}

// newAmbientCredential builds the non-interactive fallback chain. CLI is
// tried first so local dev with `az login` succeeds in <100ms; on Azure
// compute it fast-fails (no `az` binary) and the chain falls through to
// workload/managed identity. Putting ManagedIdentity first instead caused
// 30s hangs on laptops where the IMDS endpoint isn't reachable.
func newAmbientCredential() (azcore.TokenCredential, error) {
	var sources []azcore.TokenCredential

	if cli, err := azidentity.NewAzureCLICredential(nil); err == nil {
		sources = append(sources, cli)
	}
	if wic, err := azidentity.NewWorkloadIdentityCredential(nil); err == nil {
		sources = append(sources, wic)
	}
	if mic, err := azidentity.NewManagedIdentityCredential(nil); err == nil {
		sources = append(sources, mic)
	}

	if len(sources) == 0 {
		return nil, errors.New("no Entra credential source available: " +
			"set tenant_id/client_id/client_secret, enable ARM_USE_OIDC, " +
			"or sign in with `az login`")
	}

	cred, err := azidentity.NewChainedTokenCredential(sources, nil)
	if err != nil {
		return nil, fmt.Errorf("chained token credential: %w", err)
	}
	return cred, nil
}

// oidcEnabled mirrors azurerm's parsing of ARM_USE_OIDC so the env-var UX is identical.
func oidcEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("ARM_USE_OIDC"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// newOIDCCredential picks the assertion source based on the resolved OIDC inputs.
func newOIDCCredential(tenantID, clientID, oidcToken, oidcTokenFilePath, reqURL, reqToken string) (azcore.TokenCredential, error) {
	switch {
	case oidcToken != "":
		return azidentity.NewClientAssertionCredential(tenantID, clientID,
			func(_ context.Context) (string, error) { return oidcToken, nil }, nil)

	case oidcTokenFilePath != "":
		return azidentity.NewClientAssertionCredential(tenantID, clientID,
			func(_ context.Context) (string, error) { return readTokenFile(oidcTokenFilePath) }, nil)

	case reqURL != "" && reqToken != "":
		return azidentity.NewClientAssertionCredential(tenantID, clientID,
			func(ctx context.Context) (string, error) {
				return fetchGitHubOIDCToken(ctx, http.DefaultClient, reqURL, reqToken)
			}, nil)

	default:
		return nil, errors.New("ARM_USE_OIDC=true but no OIDC token source is configured " +
			"(set ARM_OIDC_TOKEN, ARM_OIDC_TOKEN_FILE_PATH, or run inside GitHub Actions)")
	}
}

// fetchGitHubOIDCToken calls the runner-injected OIDC endpoint with audience
// api://AzureADTokenExchange. HTTPS-only — refusing other schemes prevents an
// attacker-controlled URL from harvesting the request bearer.
func fetchGitHubOIDCToken(ctx context.Context, client *http.Client, reqURL, reqToken string) (string, error) {
	u, err := url.Parse(reqURL)
	if err != nil {
		return "", fmt.Errorf("parsing OIDC request URL: %w", err)
	}
	if u.Scheme != "https" {
		return "", fmt.Errorf("OIDC request URL must be https, got scheme %q", u.Scheme)
	}
	q := u.Query()
	q.Set("audience", oidcExchangeAudience)
	u.RawQuery = q.Encode()

	ctx, cancel := context.WithTimeout(ctx, oidcFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("building OIDC request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+reqToken)
	req.Header.Set("Accept", "application/json; api-version=2.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling OIDC endpoint: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("reading OIDC response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OIDC endpoint returned HTTP %d: %s",
			resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("decoding OIDC response: %w", err)
	}
	if payload.Value == "" {
		return "", errors.New("OIDC response missing token value")
	}
	return payload.Value, nil
}

func readTokenFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading OIDC token file: %w", err)
	}
	token := strings.TrimSpace(string(b))
	if token == "" {
		return "", fmt.Errorf("OIDC token file %s is empty", path)
	}
	return token, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
