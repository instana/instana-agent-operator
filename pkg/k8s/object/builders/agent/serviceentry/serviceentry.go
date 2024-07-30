/*
 * (c) Copyright IBM Corp. 2024
 * (c) Copyright Instana Inc. 2024
 */

package serviceentry

import (
	"fmt"
	"strings"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/optional"
	networkingv1alpha3api "istio.io/api/networking/v1alpha3"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	componentName = constants.ComponentInstanaAgent
	agentPort     = constants.AgentPort
)

type serviceEntryListBuilder struct {
	*instanav1.InstanaAgent
	helpers.Helpers
	nodeIP string
}

func (s *serviceEntryListBuilder) Build() builder.OptionalObject {
	return optional.Of[client.Object](
		&networkingv1alpha3.ServiceEntry{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "networking.istio.io/v1alpha3",
				Kind:       "ServiceEntry",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-worker-%s", componentName, s.nodeIP),
				Namespace: s.Namespace,
			},
			Spec: networkingv1alpha3api.ServiceEntry{
				Hosts: []string{fmt.Sprintf("%s.%s.%s.svc", s.nodeIP, s.HeadlessServiceName(), s.Namespace)},
				Ports: []*networkingv1alpha3api.ServicePort{
					{
						Number:   agentPort,
						Protocol: "TCP",
						Name:     "agent",
					},
				},
				Resolution: networkingv1alpha3api.ServiceEntry_DNS,
				Location:   networkingv1alpha3api.ServiceEntry_MESH_EXTERNAL,
			},
		},
	)
}

func (s *serviceEntryListBuilder) ComponentName() string {
	return componentName
}

func (s *serviceEntryListBuilder) IsNamespaced() bool {
	return true
}

func NewServiceEntriesBuilder(agent *instanav1.InstanaAgent, nodeIP string) builder.ObjectBuilder {
	return &serviceEntryListBuilder{
		InstanaAgent: agent,
		Helpers:      helpers.NewHelpers(agent),
		nodeIP:       strings.ReplaceAll(nodeIP, ".", "-"),
	}
}
