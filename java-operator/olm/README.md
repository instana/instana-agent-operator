## Automation for creating OLM Artifacts

These files generate artifacts for the OLM community operator and for the operator on the RedHat registry.

To create the templates, run

   ./create-artifacts.sh

This requires a couple of prerequisites (`jsonnet`, `python3`, `pyyaml`, `semver` and `requests`), which are packaged in the Dockerfile for running in CI.

### TODO

- [ ] Create descriptions from other documentation in this repository
