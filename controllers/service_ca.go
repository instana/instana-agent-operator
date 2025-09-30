/*
(c) Copyright IBM Corp. 2025

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/client"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/pointer"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateServiceCAConfigMap creates a ConfigMap with "service.beta.openshift.io/inject-cabundle"
// annotation for OpenShift
// See: https://docs.redhat.com/en/documentation/openshift_container_platform/4.9/html/security_and_compliance/
// configuring-certificates#add-service-certificate-configmap_service-serving-certificate
func CreateServiceCAConfigMap(
	ctx context.Context,
	c client.InstanaAgentClient,
	agent *instanav1.InstanaAgent,
	logger logr.Logger,
) error {
	// Create a ConfigMap with "service.beta.openshift.io/inject-cabundle" annotation
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.ServiceCAConfigMapName,
			Namespace: agent.Namespace,
			Annotations: map[string]string{
				constants.OpenShiftInjectCABundleAnnotation: "true",
			},
			// Add owner reference so it's garbage-collected with the CR
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: agent.APIVersion,
					Kind:       agent.Kind,
					Name:       agent.Name,
					UID:        agent.UID,
					Controller: pointer.To(true),
				},
			},
		},
		Data: map[string]string{},
	}

	// Use the Apply method with the client
	_, err := c.Apply(ctx, configMap).Get()
	if err != nil {
		logger.Error(err, "Failed to apply service-CA ConfigMap")
		return err
	}

	logger.Info("Service-CA ConfigMap created/updated successfully")
	return nil
}
