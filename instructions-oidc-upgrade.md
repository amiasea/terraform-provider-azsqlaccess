# Implementation Specification: AzureRM-Compatible OIDC Authentication for Amiasea `terraform-provider-azsqlaccess`

## Goal

Implement OIDC provider configuration in the Amiasea fork of `MewsSystems/terraform-provider-azsqlaccess`, closely following the authentication configuration contract and behavior of HashiCorp's AzureRM provider.

The target use case is HCP Terraform Stacks passing its generated identity token directly to the provider:

```hcl
provider "azsqlaccess" {
  tenant_id = var.azure_tenant_id
  client_id = var.azure_client_id

  oidc_token = var.azure_identity_token
  use_oidc   = true
}
```

This is an implementation task, not a redesign of authentication. Preserve the upstream provider's existing functionality and introduce the smallest change that achieves the target behavior.

## 1. Use AzureRM as the reference implementation

Before editing, inspect the actual source code and tests in both repositories.

AzureRM repository: https://github.com/hashicorp/terraform-provider-azurerm

Inspect these specific areas:

* `internal/provider/provider.go`: schema definitions and `providerConfigure`.
* The implementation of `getOidcToken` and its tests. Locate the source rather than assuming its file path.
* The AzureRM authentication configuration type and the code that consumes its OIDC fields.
* AzureRM's OIDC documentation and tests covering token selection and authentication enablement.

Also inspect the current Amiasea checkout:

* `internal/provider/provider.go`
* `internal/provider/provider_test.go`
* `internal/database/credential.go`
* `internal/database/credential_test.go`
* `internal/database/mssql/connector.go`
* `go.mod`

Use the current source checked out or fetched from the repositories. Do not treat documentation snippets or this specification as a substitute for inspecting the implementation.

Before changing code, summarize the relevant AzureRM behavior and map each behavior to the corresponding location in the Amiasea codebase.

## 2. Mirror AzureRM's configuration schema

Implement the following provider configuration fields using the Terraform Plugin Framework conventions already used by Amiasea.

### `oidc_token`

Mirror AzureRM's field semantics:

* Optional string.
* Environment fallback: `ARM_OIDC_TOKEN`.
* An explicitly configured provider value takes precedence over the environment fallback.
* Represents an OIDC ID token used as an authentication assertion, not an Azure access token.

Mark the value sensitive in the Amiasea provider schema so it is not exposed as ordinary configuration data or diagnostics.

Do not log or print the token.

### `use_oidc`

Mirror AzureRM's field semantics:

* Optional boolean.
* Default `false`.
* Environment fallback: `ARM_USE_OIDC`.
* Controls whether the provider enables the OIDC authentication mechanisms.

Do not automatically reinterpret a configured token as equivalent to `use_oidc = true`. Keep token resolution and authentication enablement as distinct operations, as AzureRM does.

### Related OIDC fields

Inspect AzureRM's additional OIDC fields:

* `oidc_token_file_path`
* `oidc_request_url`
* `oidc_request_token`
* `ado_pipeline_service_connection_id`

Determine which of these are relevant to preserving existing Amiasea behavior or implementing the same OIDC contract.

Do not automatically add every AzureRM field merely to imitate its schema. Add fields that are required for the requested compatibility or for preserving existing upstream functionality. Document any deliberately unsupported AzureRM mechanisms.

## 3. Reproduce AzureRM's token-resolution behavior

Locate AzureRM's `getOidcToken` implementation and tests. Reproduce its relevant selection rules rather than inventing a new precedence order.

Specifically, determine and implement:

1. How an explicit `oidc_token` value is resolved.
2. How `ARM_OIDC_TOKEN` supplies the fallback value.
3. How `oidc_token_file_path` is resolved, if supported.
4. How the file-based token is read and how read failures are reported.
5. Whether and how token-file contents are cached or read only once.
6. How an empty or missing token is handled.
7. How token acquisition through `oidc_request_url` and `oidc_request_token` works, if included.
8. How these inputs interact with `use_oidc`.

Use AzureRM's actual tests to establish the expected behavior wherever possible.

Keep token resolution separate from credential construction. A helper that resolves a token should not itself silently enable OIDC or mutate process-wide environment variables.

If the existing Amiasea provider already implements part of this behavior, preserve and extend that implementation rather than creating a parallel token-resolution subsystem.

## 4. Reproduce AzureRM's authentication-configuration pattern

AzureRM's provider configuration performs these distinct operations:

1. Resolve the OIDC token.
2. Resolve the other authentication inputs, including client ID and tenant ID.
3. Resolve the OIDC enablement setting.
4. Populate a shared authentication configuration with the resolved values.
5. Pass that configuration to the downstream client/credential construction path.

Follow the same separation of responsibilities in Amiasea.

Adapt the architecture to the existing `BuildEntraCredential` implementation. Do not copy AzureRM's internal authentication types or introduce a dependency on AzureRM.

The resolved OIDC token and enablement flag must be passed explicitly through the existing provider-to-credential path. Do not use environment-variable mutation as a substitute for passing configuration.

Preserve the existing provider's credential construction and error-handling conventions.

## 5. Implement the explicit-token authentication path correctly

https://github.com/hashicorp/terraform-provider-azurerm

Inspect the existing Azure Identity SDK version in `go.mod` and determine which credential type accepts a directly supplied federated client assertion.

Implement the explicit-token path using the appropriate SDK mechanism, with the configured tenant ID and client ID.

The implementation must:

* Use the supplied OIDC assertion to obtain an Azure access token for the SQL authentication flow.
* Avoid treating the supplied assertion as an access token.
* Avoid falling back silently to an unrelated developer credential or managed identity when explicit OIDC authentication is enabled but cannot be configured.
* Preserve the provider's existing handling of SQL connections and Azure SQL token acquisition.
* Keep authentication errors actionable without exposing credentials.

Follow AzureRM's enablement semantics. If `use_oidc` is enabled, configure the OIDC authentication path; do not require the token itself to be the enablement switch.

When OIDC is enabled but no usable assertion or supported OIDC token-acquisition mechanism is available, return a clear configuration error consistent with the behavior established by the upstream reference and Amiasea's framework.

Do not invent new validation rules unless necessary for correctness or required by the existing SDK.

## 6. Preserve existing authentication mechanisms

The upstream provider already contains credential logic and tests. Treat that functionality as a compatibility requirement.

Before and after the change, establish which existing mechanisms are supported, including any of:

* Environment-based OIDC tokens.
* OIDC token files.
* Existing service principal or other Azure Identity credential paths.

Preserve all currently supported mechanisms unless a concrete incompatibility makes that impossible.

If two OIDC mechanisms compete, follow AzureRM's verified behavior where applicable. Do not assume that all mechanisms should be combined into one credential chain.

Avoid modifying SQL query behavior, database permissions, migrations, or unrelated provider functionality.

## 7. Implement tests based on AzureRM's behavior

Add or update tests in the existing provider and credential test files.

Tests must establish behavior, not merely successful object construction.

### Provider schema and configuration

Test that:

* `oidc_token` is optional and sensitive.
* `use_oidc` is optional and defaults to `false`.
* `ARM_OIDC_TOKEN` and `ARM_USE_OIDC` are respected as fallbacks.
* Explicit provider configuration takes precedence over corresponding environment defaults.
* OIDC token resolution does not automatically enable OIDC.
* The enablement setting is propagated to the credential construction path.

### Token resolution

Mirror the applicable AzureRM tests for:

* Explicit token.
* Environment fallback.
* Token-file fallback, if supported.
* Missing token.
* Empty token.
* Token-file read failures.
* Request-based OIDC, if supported.
* Conflicting configured inputs and the expected precedence.

### Credential construction

Test that:

* An explicitly supplied assertion reaches the intended Azure Identity credential path.
* The configured tenant ID and client ID are used.
* OIDC enablement is honored.
* Missing required authentication inputs produce actionable errors.
* Existing non-OIDC and environment-based authentication paths remain functional.
* No test needs to contact a live Azure tenant.

### Secret handling

Verify that:

* The token is not logged.
* Error diagnostics do not include the token.
* Failure messages identify the configuration problem without exposing secret values.

Use the existing test conventions and narrow test doubles where necessary. Do not weaken tests merely to make them pass.

## 8. Dependencies and scope

Use the existing Azure Identity SDK dependency if it supports the required credential type.

Only change dependencies if source inspection proves the current version cannot implement the behavior correctly. Explain the technical reason before introducing an unnecessary dependency or broad version upgrade.

Do not modify the following as part of this task:

* The provider's Go module path or registry address.
* Release workflows or GoReleaser configuration.
* SQL resources or role assignments.
* Unrelated provider resources.
* Repository structure or naming.

Those changes are separate from the authentication feature.

## 9. Validation

After implementation:

1. Run `gofmt` on all modified Go files.
2. Run focused tests for provider configuration and credential construction.
3. Run `go test ./...`.
4. Run existing lint and validation commands when practical.
5. Review the full Git diff.
6. Verify that existing authentication behavior is preserved and no token values can leak through diagnostics or logs.

Report the exact commands run and their actual results. Distinguish tests that passed from tests that could not be run.

## 10. Final report

Return:

1. The AzureRM source files and tests inspected.
2. A concise mapping between AzureRM's implementation and Amiasea's implementation.
3. The files changed and the reason for each change.
4. The resulting token-resolution precedence and OIDC enablement behavior.
5. The tests run and their results.
6. Any AzureRM behavior not implemented, with a specific reason.

Do not commit, push, publish a release, or change the upstream repository.

## Acceptance criteria

The feature is complete when Amiasea's provider follows AzureRM's verified separation between token resolution and OIDC enablement, accepts the explicit HCP Terraform Stacks token through `oidc_token`, uses the assertion correctly through Azure Identity, preserves existing authentication paths, protects the token from disclosure, and passes the relevant automated tests.
