/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"context"
	"time"

	instanaV1Beta1 "github.com/instana/instana-agent-operator/api/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	rbacV1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Instana agent controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When user first time install the agent operator", func() {
		It("Should create all needed kubernetes resources to run the agent when apply agent's CRD minimal required configuration to the cluster", func() {
			By("By creating a new namespace for InstanaAgent")
			ctx := context.Background()
			AgentNamespace := &coreV1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   AgentNameSpace,
					Labels: buildLabels(),
				},
			}
			Expect(k8sClient.Create(ctx, AgentNamespace)).Should(Succeed())

			By("By creating a new InstanaAgent CRD instance with minimal required inputs")
			agentCRD := &instanaV1Beta1.InstanaAgent{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "agents.instana.com/v1beta1",
					Kind:       "InstanaAgent",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      AppName,
					Namespace: AgentNameSpace,
				},
				Spec: instanaV1Beta1.InstanaAgentSpec{
					ZoneName: "my-zone",
					Key:      "nqtbV5cEQ5ev0MFzOIwskg",
					Endpoint: &instanaV1Beta1.InstanaAgentEndpoint{
						Host: "ingress-red-saas.instana.io",
						Port: "443",
					},
					ClusterName: "testCluster",
				},
			}
			Expect(k8sClient.Create(ctx, agentCRD)).Should(Succeed())

			agentCRDLookupKey := types.NamespacedName{Name: AppName, Namespace: AgentNameSpace}
			Eventually(func() bool {
				newCrd := &instanaV1Beta1.InstanaAgent{}
				err := k8sClient.Get(ctx, agentCRDLookupKey, newCrd)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			//we expect daemonset, service, serviceaccount, secret, pullSecret, configmap, clusterrole and clusterrolebinding resources to be created
			daemonset := &appV1.DaemonSet{}
			service := &coreV1.Service{}
			configMap := &coreV1.ConfigMap{}
			secret := &coreV1.Secret{}
			imagePullSecret := &coreV1.Secret{}
			clusterRole := &rbacV1.ClusterRole{}
			clusterRoleBinding := &rbacV1.ClusterRoleBinding{}
			serviceAccount := &coreV1.ServiceAccount{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: AgentSecretName, Namespace: AgentNameSpace}, secret)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: AgentImagePullSecretName, Namespace: AgentNameSpace}, imagePullSecret)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: AppName, Namespace: AgentNameSpace}, service)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: AppName, Namespace: AgentNameSpace}, serviceAccount)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: AppName, Namespace: AgentNameSpace}, configMap)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: AppName}, clusterRole)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: AppName}, clusterRoleBinding)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: AppName, Namespace: AgentNameSpace}, daemonset)
				return err == nil
			}, timeout, interval).Should(BeTrue())

		})
	})

})
