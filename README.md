Instana Agent Operator
======================

This is an experimental alpha version of the upcoming Kubernetes Operator for the [Instana APM Agent](https://www.instana.com).

There are two ways to install the operator:

* [docs/install-via-olm.md](docs/install-via-olm.md) describes how to install the operator using the Operator Lifecycle Manager (OLM).
* [docs/install-manually.md](docs/install-manually.md) describes how to install the operator by creating the required resources manually.

Additional documentation for developers:

* [docs/build.md](docs/build.md) describes how to build the Docker image from source code.
* [docs/testing-with-kind.md](docs/testing-with-kind.md) shows how to test the operator in a local Kind cluster.
* [docs/run-operator-registry-locally.md](docs/run-operator-registry-locally.md) describes how to set up a local Operator Lifecycle Manager and Registry to test the OLM deployment locally.
