# BDR-033: Run From a Container With External Configuration

## Status

Proposed

## Behavior

The application starts from its container image, with the server settings, persistence location and auth secrets supplied from outside the image.

## Context

Deployments must be repeatable and must not bake secrets into images.

Traces to: SPEC.md NFR-008, Definition of Done 10.

## Acceptance Criteria

- AC-1: Running the published image with documented environment variables starts a server that answers `GET /api/v1/games`.
- AC-2: Changing the configured listen port or persistence location through configuration changes the running server's behavior with no image rebuild.
- AC-3: The image contains no authentication secrets. Without configured secrets, authenticated operations are rejected.

## Verification

### Scenario 1: Start with documented configuration

- **Given** the application container image and the documented environment variables
- **When** an operator starts the container with a mounted data volume and a configured auth secret
- **Then** `GET /api/v1/games` answers successfully
- **And** games created before a container restart are present after it ([BDR-018](018-survive-server-restart.md))
