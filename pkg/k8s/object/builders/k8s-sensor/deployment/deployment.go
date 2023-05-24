package deployment

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

type deploymentBuilder struct {
	*instanav1.InstanaAgent
	transformations.PodSelectorLabelGenerator
}

func (d *deploymentBuilder) IsNamepaced() bool {
	return true
}

func (d *deploymentBuilder) ComponentName() string {
	return constants.ComponentK8Sensor
}

func (d *deploymentBuilder) Build() optional.Optional[client.Object] {
	// TODO
	panic("implement me")
}
