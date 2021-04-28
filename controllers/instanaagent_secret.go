/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"context"
	"fmt"

	instanaV1Beta1 "github.com/instana/instana-agent-operator/api/v1beta1"
	"github.com/pkg/errors"
	coreV1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newSecretForCRD(crdInstance *instanaV1Beta1.InstanaAgent) (*coreV1.Secret, error) {
	agentKey := crdInstance.Spec.Key
	agentDownloadKey := crdInstance.Spec.DownloadKey
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

func (r *InstanaAgentReconciler) reconcileSecrets(ctx context.Context, crdInstance *instanaV1Beta1.InstanaAgent) error {
	secret := &coreV1.Secret{}
	err := r.Get(ctx, client.ObjectKey{Name: AgentSecretName, Namespace: AgentNameSpace}, secret)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.Log.Info("No InstanaAgent config secret deployed before, creating new one")
			if secret, err = newSecretForCRD(crdInstance); err != nil {
				return err
			}
			if err := r.Create(ctx, secret); err == nil {
				r.Log.Info(fmt.Sprintf("%s secret created successfully", AgentSecretName))
				return nil
			} else {
				r.Log.Error(err, "failed to create secret")
			}
		}
		return err
	}
	return nil
}
