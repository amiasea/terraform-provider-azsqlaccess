// Copyright (c) Mews Systems
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"
	"testing"

	pf_provider "github.com/hashicorp/terraform-plugin-framework/provider"
	pf_schema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
)

func TestProviderSchema_OIDCAttributes(t *testing.T) {
	p := &AzsqlaccessProvider{version: "test"}
	var resp pf_provider.SchemaResponse
	p.Schema(context.Background(), pf_provider.SchemaRequest{}, &resp)

	attrs := resp.Schema.Attributes

	// oidc_token should be a string attribute and marked sensitive
	attr, ok := attrs["oidc_token"]
	if !ok {
		t.Fatalf("schema missing oidc_token attribute")
	}
	strAttr, ok := attr.(pf_schema.StringAttribute)
	if !ok {
		t.Fatalf("oidc_token attribute has wrong type: %T", attr)
	}
	if !strAttr.Sensitive {
		t.Error("oidc_token attribute must be Sensitive = true")
	}

	// use_oidc should be a boolean attribute and optional
	attr2, ok := attrs["use_oidc"]
	if !ok {
		t.Fatalf("schema missing use_oidc attribute")
	}
	boolAttr, ok := attr2.(pf_schema.BoolAttribute)
	if !ok {
		t.Fatalf("use_oidc attribute has wrong type: %T", attr2)
	}
	if !boolAttr.Optional {
		t.Error("use_oidc attribute must be Optional = true")
	}
}

func TestParseARMUseOIDC_EnvFallback(t *testing.T) {
	cases := map[string]bool{
		"":      false,
		"0":     false,
		"false": false,
		"no":    false,
		"1":     true,
		"true":  true,
		"TRUE":  true,
		" yes ": true,
		"on":    true,
	}
	for v, want := range cases {
		t.Run(v, func(t *testing.T) {
			t.Setenv("ARM_USE_OIDC", v)
			if got := parseARMUseOIDC(); got != want {
				t.Errorf("parseARMUseOIDC() with ARM_USE_OIDC=%q = %v, want %v", v, got, want)
			}
		})
	}

	// ensure we clean up any env we set
	_ = os.Unsetenv("ARM_USE_OIDC")
}
