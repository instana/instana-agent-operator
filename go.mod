module github.com/instana/instana-agent-operator

go 1.15

require (
	github.com/go-logr/logr v0.4.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/pkg/errors v0.9.1
	github.com/spf13/pflag v1.0.5 // indirect
	go.uber.org/zap v1.15.0
	helm.sh/helm/v3 v3.5.4
	honnef.co/go/tools v0.1.4 // indirect
	k8s.io/api v0.20.4
	k8s.io/apimachinery v0.20.4
	k8s.io/cli-runtime v0.20.4 // indirect
	k8s.io/client-go v0.20.4
	k8s.io/helm v2.17.0+incompatible
	sigs.k8s.io/controller-runtime v0.7.2
	sigs.k8s.io/yaml v1.2.0 // indirect

)

replace (
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
)
