## Automation for creating OLM Artifacts

These files generate artifacts for the OLM community operator and for the operator on the RedHat registry.

To create the templates, run

   ./create-artifacts.sh

This requires a couple of prerequisites (`jsonnet`, `python3`, `pyyaml`, `semver` and `requests`), which are packaged in the Dockerfile for running in CI.

# Creation of the bundle dockerfile

Create the artifacts

   ./create-artifacts.sh $VERSION redhat

Then from the root directory of this project, run the docker build

   docker build -t scan.connect.redhat.com/ospid-5fc350a1-9257-4291-9f2a-df9257b9e791/instana-agent-operator-bundle:$VERSION -f olm/bundle.Dockerfile target/redhat

And push that image to initiate a bundle scan and publishing of metadata for the operator.

### TODO

- [ ] Create descriptions from other documentation in this repository
- [ ] Automate creation of bundle docker file
