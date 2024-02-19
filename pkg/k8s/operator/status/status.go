package status

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AgentStatusManager interface {
}

type agentStatusManager struct {
	k8sClient client.Client

	agentDaemonsets     []client.ObjectKey
	k8sSensorDeployment client.ObjectKey
}
