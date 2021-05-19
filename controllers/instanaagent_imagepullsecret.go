/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	instanaV1Beta1 "github.com/instana/instana-agent-operator/api/v1beta1"
	"github.com/pkg/errors"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func newImagePullSecretForCRD(crdInstance *instanaV1Beta1.InstanaAgent, Log logr.Logger) *coreV1.Secret {
	return &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      AgentImagePullSecretName,
			Namespace: AgentNameSpace,
			Labels:    buildLabels(),
		},
		Type: coreV1.SecretTypeDockerConfigJson,
		Data: generatePullSecretData(crdInstance, Log),
	}
}

func generatePullSecretData(crdInstance *instanaV1Beta1.InstanaAgent, Log logr.Logger) map[string][]byte {
	type auths struct {
		Username string `json:"username,omitempty"`
		Password string `json:"password,omitempty"`
		Auth     string `json:"auth,omitempty"`
	}

	type dockerConfig struct {
		Auths map[string]auths `json:"auths,omitempty"`
	}
	passwordKey := crdInstance.Spec.Agent.Key
	if len(passwordKey) == 0 {
		passwordKey = crdInstance.Spec.Agent.DownloadKey
	}
	a := fmt.Sprintf("%s:%s", "_", passwordKey)
	a = b64.StdEncoding.EncodeToString([]byte(a))

	auth := auths{
		Username: "_",
		Password: passwordKey,
		Auth:     a,
	}

	d := dockerConfig{
		Auths: map[string]auths{
			DockerRegistry: auth,
		},
	}
	j, err := json.Marshal(d)
	if err != nil {
		Log.Error(errors.WithStack(err), "Failed to convert jsonkey")
	}

	return map[string][]byte{".dockerconfigjson": j}
}

func (r *InstanaAgentReconciler) setImagePullSecretsReference(ctx context.Context, crdInstance *instanaV1Beta1.InstanaAgent) error {
	pullSecret := &coreV1.Secret{}
	err := r.Get(ctx, client.ObjectKey{Name: AgentImagePullSecretName, Namespace: AgentNameSpace}, pullSecret)
	if err == nil {
		if err = controllerutil.SetControllerReference(crdInstance, pullSecret, r.Scheme); err != nil {
			return err
		}
		if err = r.Update(ctx, pullSecret); err != nil {
			r.Log.Error(err, "Failed to set controller reference for pullSecret")
		}
		r.Log.Info("Set controller reference for pullSecret was successfull")
	}
	return nil
}
