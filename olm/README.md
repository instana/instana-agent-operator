## Automation for creating OLM Artifacts

These files generate artifacts for the OLM community operator.

To create the templates, run

   ./createCSV.sh

This requires a couple of prerequisites (`jsonnet`, `python3`, `pyyaml`, and `operator-courier`), which are packaged in the Dockerfile for running in CI.

### TODO

- [ ] Automatically pull latest release as previous version
- [ ] Create descriptions from other documentation in this repository
