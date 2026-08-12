## Purpose

Define a minimal, secure first-run configuration experience that makes model access and an OCI execution provider ready before an authorized hunt starts.

## ADDED Requirements

### Requirement: Interactive initialization
The system SHALL provide `cyberpilot init` as an interactive first-run command that configures exactly one default model profile and one default local OCI runner profile.

#### Scenario: Successful first run
- **WHEN** the user runs `cyberpilot init`, supplies valid model settings and credentials, and selects an available Docker or Podman provider
- **THEN** the system validates both profiles, stores the non-secret configuration, protects the credential through the operating-system credential facility, and reports that CyberPilot is ready

#### Scenario: Existing configuration
- **WHEN** the user runs `cyberpilot init` after initialization has already completed
- **THEN** the system displays the configured profile summaries and requires explicit confirmation before replacing either profile

### Requirement: Model configuration validation
The system SHALL validate the configured endpoint, model availability, authentication, tool calling, and structured output before accepting a model profile.

#### Scenario: Unsupported model capability
- **WHEN** the endpoint is reachable but the selected model cannot complete the required structured tool-call probe
- **THEN** initialization fails with an actionable capability error and does not mark the profile ready

#### Scenario: Credential failure
- **WHEN** authentication to the model provider fails
- **THEN** the system reports an authentication error without printing or persisting the credential in logs or task events

### Requirement: OCI provider discovery
The system SHALL discover local Docker and rootless Podman endpoints, show the endpoint and privilege mode, and require an explicit selection when more than one viable provider exists.

#### Scenario: One viable provider
- **WHEN** exactly one supported local Docker or Podman endpoint responds successfully
- **THEN** the system proposes that provider as the default and verifies image, create, start, exec, stop, and remove operations with a disposable probe sandbox

#### Scenario: No viable provider
- **WHEN** neither Docker nor Podman is available and usable
- **THEN** initialization ends without a ready runner and explains how to install or expose a supported provider

### Requirement: Configuration inspection and modification
The system SHALL provide `cyberpilot config` to display redacted active profiles and interactively replace individual model or runner values.

#### Scenario: Display configuration
- **WHEN** the user opens `cyberpilot config`
- **THEN** the system shows provider, model, endpoint, runner provider, runner endpoint, and credential source while masking secret values

#### Scenario: Modify connection-critical value
- **WHEN** the user changes a model, endpoint, credential, or runner endpoint
- **THEN** the system validates the updated profile before making it active and retains the last valid profile when validation fails

