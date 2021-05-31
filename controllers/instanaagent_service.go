/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"context"

	instanaV1Beta1 "github.com/instana/instana-agent-operator/api/v1beta1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func newServiceForCRD() *coreV1.Service {
	return &coreV1.Service{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      AppName,
			Namespace: AgentNameSpace,
			Labels:    buildLabels(),
		},
		Spec: coreV1.ServiceSpec{
			Selector: buildLabels(),
			Ports: []coreV1.ServicePort{
				{
					Name:       "opentelemetry",
					Protocol:   coreV1.ProtocolTCP,
					Port:       OpenTelemetryPort,
					TargetPort: intstr.FromInt(OpenTelemetryPort),
				},
				{
					Name:       "agent-apis",
					Protocol:   coreV1.ProtocolTCP,
					Port:       AgentPort,
					TargetPort: intstr.FromInt(AgentPort),
				},
			},
			TopologyKeys: []string{"kubernetes.io/hostname"},
		},
	}
}
func (r *InstanaAgentReconciler) setServicesReference(ctx context.Context, crdInstance *instanaV1Beta1.InstanaAgent) error {
	service := &coreV1.Service{}
	err := r.Get(ctx, client.ObjectKey{Name: AppName, Namespace: AgentNameSpace}, service)
	if err == nil {
		if err = controllerutil.SetControllerReference(crdInstance, service, r.Scheme); err != nil {
			return err
		}
		if err = r.Update(ctx, service); err != nil {
			r.Log.Error(err, "Failed to set controller reference for service")
		}
		r.Log.Info("Set controller reference for service was successfull")
	}
	return nil
}
