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
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Instana agent controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		InstanaAgentName      = "test-instana-agent"
		InstanaAgentNamespace = "test-instana-agent-namespace"

		timeout = time.Second * 10
		// duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When user first time install the agent operator", func() {
		It("Should create all needed kubernetes resources to run the agent when apply agent's CRD to the cluster", func() {
			By("By creating a new namespace for InstanaAgent")
			ctx := context.Background()
			AgentNamespace := &coreV1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   InstanaAgentNamespace,
					Labels: buildLabels(),
				},
			}
			Expect(k8sClient.Create(ctx, AgentNamespace)).Should(Succeed())

			By("By creating a new InstanaAgent CRD instance")
			agentCRD := &instanaV1Beta1.InstanaAgent{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "agents.instana.com/v1beta1",
					Kind:       "InstanaAgent",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      InstanaAgentName,
					Namespace: InstanaAgentNamespace,
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

			agentCRDLookupKey := types.NamespacedName{Name: InstanaAgentName, Namespace: InstanaAgentNamespace}
			createdAgentCRDInstance := &instanaV1Beta1.InstanaAgent{}

			// We'll need to retry getting this newly created InstanaAgent, given that creation may not immediately happen.
			Eventually(func() bool {
				// fetchCrdInstance()
				err := k8sClient.Get(ctx, agentCRDLookupKey, createdAgentCRDInstance)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			// we can add checking for validated fields here
			// Expect(createdAgentCRDInstance.Spec.Env).Should(Equal(""))

		})
	})

})
