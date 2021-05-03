/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"context"
	"fmt"

	instanaV1Beta1 "github.com/instana/instana-agent-operator/api/v1beta1"
	coreV1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
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
func (r *InstanaAgentReconciler) reconcileServices(ctx context.Context, crdInstance *instanaV1Beta1.InstanaAgent) error {
	service := &coreV1.Service{}
	err := r.Get(ctx, client.ObjectKey{Name: AppName, Namespace: AgentNameSpace}, service)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.Log.Info("No InstanaAgent service deployed before, creating new one")
			service = newServiceForCRD()
			if err = controllerutil.SetControllerReference(crdInstance, service, r.Scheme); err != nil {
				return err
			}
			if err = r.Create(ctx, service); err == nil {
				r.Log.Info(fmt.Sprintf("%s service created successfully", AppName))
				return nil
			} else {
				r.Log.Error(err, "Failed to create service")
			}
		}
		return err
	}
	return nil
}
