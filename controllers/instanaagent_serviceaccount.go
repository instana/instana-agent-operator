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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func newServiceAccountForCRD() *coreV1.ServiceAccount {
	return &coreV1.ServiceAccount{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      AppName,
			Namespace: AgentNameSpace,
			Labels:    buildLabels(),
		}}
}

func (r *InstanaAgentReconciler) reconcileServiceAccounts(ctx context.Context, crdInstance *instanaV1Beta1.InstanaAgent) error {
	serviceAcc := &coreV1.ServiceAccount{}
	err := r.Get(ctx, client.ObjectKey{Name: AgentServiceAccountName, Namespace: AgentNameSpace}, serviceAcc)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.Log.Info("No InstanaAgent service account deployed before, creating new one")
			serviceAcc = newServiceAccountForCRD()
			if err = controllerutil.SetControllerReference(crdInstance, serviceAcc, r.Scheme); err != nil {
				return err
			}
			if err = r.Create(ctx, serviceAcc); err == nil {
				r.Log.Info(fmt.Sprintf("%s service account created successfully", AgentServiceAccountName))
				return nil
			} else {
				r.Log.Error(err, "Failed to create service account")
			}
		}
		return err
	}
	return nil
}
