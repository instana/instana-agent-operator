/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"context"
	"fmt"

	"io/ioutil"

	"github.com/go-logr/logr"
	instanaV1Beta1 "github.com/instana/instana-agent-operator/api/v1beta1"
	coreV1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newConfigMapForCRD(crdInstance *instanaV1Beta1.InstanaAgent, Log logr.Logger) *coreV1.ConfigMap {
	data := map[string]string{
		"configuration.yaml": "",
	}
	if len(crdInstance.Spec.ClusterName) > 0 {
		data["cluster_name"] = crdInstance.Spec.ClusterName
	}
	if len(crdInstance.Spec.ConfigFiles) > 0 {
		confFiles := crdInstance.Spec.ConfigFiles
		if val, ok := confFiles["configuration.yaml"]; ok {
			data["configuration.yaml"] = val
		}
	}
	return &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      AppName,
			Namespace: AgentNameSpace,
			Labels:    buildLabels(),
		},
		Data: data,
	}
}

func readFile(filename string, Log logr.Logger) string {
	content, err := ioutil.ReadFile("config/" + filename)
	if err != nil {
		Log.Error(err, "failed to read configuration.yaml")
	}
	return string(content)
}

func (r *InstanaAgentReconciler) reconcileConfigMap(ctx context.Context, crdInstance *instanaV1Beta1.InstanaAgent) error {
	configMap := &coreV1.ConfigMap{}
	err := r.Get(ctx, client.ObjectKey{Name: AppName, Namespace: AgentNameSpace}, configMap)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.Log.Info("No InstanaAgent configMap deployed before, creating new one")
			configMap := newConfigMapForCRD(crdInstance, r.Log)
			if err = r.Create(ctx, configMap); err != nil {
				r.Log.Error(err, "Failed to create configMap")
			} else {
				r.Log.Info(fmt.Sprintf("%s configMap created successfully", AppName))
			}
		}
	}
	return err
}
