/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"context"

	instanaV1Beta1 "github.com/instana/instana-agent-operator/api/v1beta1"
	"github.com/pkg/errors"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func newSecretForCRD(crdInstance *instanaV1Beta1.InstanaAgent) (*coreV1.Secret, error) {
	agentKey := crdInstance.Spec.Agent.Key
	agentDownloadKey := crdInstance.Spec.Agent.DownloadKey
	if len(agentKey) == 0 {
		return nil, errors.New("Failed to create agent secrets, please provide an agent key")
	}
	return &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      AgentSecretName,
			Namespace: AgentNameSpace,
			Labels:    buildLabels(),
		},
		Type: coreV1.SecretTypeOpaque,
		Data: map[string][]byte{
			AgentKey:         []byte(agentKey),
			AgentDownloadKey: []byte(agentDownloadKey),
		},
	}, nil
}

func (r *InstanaAgentReconciler) setSecretsReference(ctx context.Context, crdInstance *instanaV1Beta1.InstanaAgent) error {
	secret := &coreV1.Secret{}
	err := r.Get(ctx, client.ObjectKey{Name: AgentSecretName, Namespace: AgentNameSpace}, secret)
	if err == nil {
		if err = controllerutil.SetControllerReference(crdInstance, secret, r.Scheme); err != nil {
			return err
		}
		if err = r.Update(ctx, secret); err != nil {
			r.Log.Error(err, "Failed to set controller reference for secret")
		}
		r.Log.Info("Set controller reference for secret was successfull")
	}
	return err
}
