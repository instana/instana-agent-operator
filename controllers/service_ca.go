/*
(c) Copyright IBM Corp. 2024, 2025

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

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// createServiceCAConfigMap creates a ConfigMap with service-ca.crt annotation for OpenShift
func (r *InstanaAgentReconciler) createServiceCAConfigMap(ctx context.Context, agent *instanav1.InstanaAgent) error {
	log := r.loggerFor(ctx, agent)

	// Create a ConfigMap with service-ca.crt annotation
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "etcd-ca",
			Namespace: agent.Namespace,
			Annotations: map[string]string{
				"service.beta.openshift.io/inject-cabundle": "true",
			},
		},
		Data: map[string]string{},
	}

	// Apply the ConfigMap using server-side apply
	_, err := r.client.Apply(ctx, configMap).Get()
	if err != nil {
		log.Error(err, "Failed to apply service-CA ConfigMap")
		return err
	}

	log.Info("Service-CA ConfigMap created/updated successfully")
	return nil
}
