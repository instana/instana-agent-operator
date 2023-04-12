/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	agentv1 "github.com/instana/instana-agent-operator/api/v1"
)

var _ = Describe(
	"Instana-Agent controller", func() {

		// Define utility constants for object names and testing timeouts/durations and intervals.
		const (
			InstanaAgentName = "instana-agent"

			timeout  = time.Second * 5
			duration = time.Second * 10
			interval = time.Millisecond * 250
		)

		ns := SetupTest(context.TODO())

		Context(
			"When creating an InstanaAgent CustomResource", func() {
				It(
					"Should not create any Agent resources", func() {
						By("By validating and exiting early")
						ctx := context.Background()
						instanaAgent := &agentv1.InstanaAgent{
							TypeMeta: metav1.TypeMeta{
								APIVersion: "instana.io/v1",
								Kind:       "InstanaAgent",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name:      InstanaAgentName,
								Namespace: ns.Name,
							},
							Spec: agentv1.InstanaAgentSpec{
								Zone: agentv1.Name{
									Name: "Maaskantje",
								},
								Cluster: agentv1.Name{
									Name: "Maaskantje",
								},
								Agent: agentv1.BaseAgentSpec{
									EndpointHost: "instana.rocks",
									EndpointPort: "443",
								},
							},
						}
						Expect(k8sClient.Create(ctx, instanaAgent)).Should(Succeed())

						agentOperatorLookupKey := types.NamespacedName{Name: InstanaAgentName, Namespace: ns.Name}
						createdAgentOperator := &agentv1.InstanaAgent{}

						// We'll need to retry getting this newly created InstanaAgent, given that creation may not immediately happen.
						Eventually(
							func() bool {
								err := k8sClient.Get(ctx, agentOperatorLookupKey, createdAgentOperator)
								return err == nil
							}, timeout, interval,
						).Should(BeTrue())
						// Let's make sure our Schedule string value was properly converted/handled.
						Expect(createdAgentOperator.Spec.Cluster.Name).Should(Equal("Maaskantje"))

						By("By checking the Status is not set because the CustomResource is invalid")
						Consistently(
							func() (bool, error) {
								err := k8sClient.Get(ctx, agentOperatorLookupKey, createdAgentOperator)
								if err != nil {
									return false, err
								}
								return createdAgentOperator.Status.OldVersionsUpdated, nil
							}, duration, interval,
						).Should(Equal(false))
					},
				)
			},
		)

		// TODO These tests need to be fixed, at least two known issues:
		// - Operator always expects a fixed namespace (instana-agent). But cluster status cannot be cleaned between tests and
		//   deleting namespaces is not possible. So either everything needs to run in a single test or make the Operator work with
		//   variable namespaces
		// - Helm errors with "Kubernetes cluster unreachable"
		//
		// Context("When creating InstanaAgent CustomResource that is valid", func() {
		//	It("Should create Agent resources", func() {
		//		By("By valid CRD creation")
		//		ctx := context.Background()
		//		instanaAgent := &agentv1.InstanaAgent{
		//			TypeMeta: metav1.TypeMeta{
		//				APIVersion: "instana.io/v1",
		//				Kind:       "InstanaAgent",
		//			},
		//			ObjectMeta: metav1.ObjectMeta{
		//				Name:      InstanaAgentName,
		//				Namespace: ns.Name,
		//			},
		//			Spec: agentv1.InstanaAgentSpec{
		//				Zone: agentv1.Name{
		//					Name: "Maaskantje",
		//				},
		//				Cluster: agentv1.Name{
		//					Name: "Maaskantje",
		//				},
		//				Agent: agentv1.BaseAgentSpec{
		//					Key:          "foobar",
		//					EndpointHost: "instana.rocks",
		//					EndpointPort: "443",
		//				},
		//			},
		//		}
		//		Expect(k8sClient.Create(ctx, instanaAgent)).Should(Succeed())
		//
		//		agentOperatorLookupKey := types.NamespacedName{Name: InstanaAgentName, Namespace: ns.Name}
		//		createdAgentOperator := &agentv1.InstanaAgent{}
		//
		//		// We'll need to retry getting this newly created InstanaAgent, given that creation may not immediately happen.
		//		Eventually(func() bool {
		//			err := k8sClient.Get(ctx, agentOperatorLookupKey, createdAgentOperator)
		//			return err == nil
		//		}, timeout, interval).Should(BeTrue())
		//		// Let's make sure our Schedule string value was properly converted/handled.
		//		Expect(createdAgentOperator.Spec.Agent.Key).Should(Equal("foobar"))
		//
		//		By("By checking the Status updated")
		//		Eventually(func() (bool, error) {
		//			err := k8sClient.Get(ctx, agentOperatorLookupKey, createdAgentOperator)
		//			if err != nil {
		//				return false, err
		//			}
		//			return createdAgentOperator.Status.OldVersionsUpdated, nil
		//		}, timeout, interval).Should(BeTrue())
		//
		//		By("By checking a DaemonSet was created")
		//		agentDaemonSetKey := types.NamespacedName{Name: "instana-agent", Namespace: ns.Name}
		//		agentDaemonSet := &appsv1.DaemonSet{}
		//		Eventually(func() bool {
		//			err := k8sClient.Get(ctx, agentDaemonSetKey, agentDaemonSet)
		//			return err == nil
		//		}, timeout, interval).Should(BeTrue())
		//		// Let's make sure our Schedule string value was properly converted/handled.
		//		Expect(len(agentDaemonSet.Spec.Template.Spec.Containers)).Should(Equal(1))
		//		Expect(agentDaemonSet.Spec.Template.Spec.Containers[0].Env).Should(ContainElement(v1.EnvVar{
		//			Name:  "INSTANA_KUBERNETES_CLUSTER_NAME",
		//			Value: "Maaskantje",
		//		}))
		//
		//		Consistently(func() (string, error) {
		//			err := k8sClient.Get(ctx, agentOperatorLookupKey, createdAgentOperator)
		//			if err != nil {
		//				return "", err
		//			}
		//			return createdAgentOperator.Status.DaemonSet.UID, nil
		//		}, duration, interval).Should(MatchRegexp(".+"))
		//
		//	})
		// })

	},
)
