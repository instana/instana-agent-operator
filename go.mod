module github.com/instana/instana-agent-operator

go 1.18

require (
	github.com/blang/semver v3.5.1+incompatible
	github.com/containerd/containerd v1.5.8 // indirect
	github.com/go-logr/logr v0.4.0
	github.com/google/go-cmp v0.5.5
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/opencontainers/image-spec v1.0.2 // indirect
	github.com/pkg/errors v0.9.1
	github.com/procyon-projects/chrono v1.0.0
	helm.sh/helm/v3 v3.7.0
	k8s.io/api v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/cli-runtime v0.22.1
	k8s.io/client-go v0.22.1
	sigs.k8s.io/controller-runtime v0.9.2
	sigs.k8s.io/yaml v1.2.0

)
