/*
 * (c) Copyright IBM Corp. 2024
 * (c) Copyright Instana Inc. 2024
 */
package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"

	v1 "github.com/instana/instana-agent-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/support/utils"
)

var (
	testEnv   env.Environment
	namespace string
)

const latestOperatorYaml string = "https://github.com/instana/instana-agent-operator/releases/latest/download/instana-agent-operator.yaml"

func TestMain(m *testing.M) {
	path := conf.ResolveKubeConfigFile()
	cfg := envconf.NewWithKubeConfig(path)
	testEnv = env.NewWithConfig(cfg)
	namespace = "instana-agent"
	testEnv.Setup(
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			// delete agent cr if present
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				return ctx, fmt.Errorf("Cleanup: Error initializing client to delete agent CR: %w", err)
			}
			r.WithNamespace(namespace)
			v1.AddToScheme(r.GetScheme())

			// If the agent cr is available, but the operator is already gone, the finalizer will never be removed
			// This will lead to a terminating namespace which never disappears, to avoid that, patch the agent CR
			// to remove the finalizer. Afterwards, it can be deleted just fine.
			agent := &v1.InstanaAgent{}
			err = r.Get(ctx, "instana-agent", "instana-agent", agent)
			if err != nil {
				fmt.Println(fmt.Errorf("Cleanup: Fetch agent CR failed, might not be present (ignoring): %w", err))
			}

			agentCrToDelete := &v1.InstanaAgent{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "instana.io/v1",
					Kind:       "InstanaAgent",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "instana-agent",
					Namespace: namespace,
				},
			}

			// kubectl patch agent instana-agent -p '{"metadata":{"finalizers":[]}}' --type=merge
			err = r.Patch(ctx, agent, k8s.Patch{
				PatchType: types.MergePatchType,
				Data:      []byte(`{"metadata":{"finalizers":[]}}`),
			})
			if err != nil {
				fmt.Println(fmt.Errorf("Cleanup: Patch agent CR failed, might not be present (ignoring): %w", err))
			}

			// delete explicitly, namespace deletion would delete the agent CR as well if the finalizer is not present
			err = r.Delete(ctx, agentCrToDelete)

			if err != nil {
				fmt.Println(fmt.Errorf("Cleanup: Delete agent CR failed, might not be present (ignoring): %w", err))
			}

			agentCrList := &v1.InstanaAgentList{
				Items: []v1.InstanaAgent{
					{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "instana.io/v1",
							Kind:       "InstanaAgent",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "instana-agent",
							Namespace: namespace,
						},
					},
				},
			}

			// ensure to wait for the agent CR to disappear before continuing
			err = wait.For(conditions.New(r).ResourcesDeleted(agentCrList))
			if err != nil {
				fmt.Println(fmt.Errorf("Cleanup: Waiting for agent CR deletion failed, might not be present (ignoring): %w", err))
			}
			return ctx, nil
		},
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			// wrap delete call to not fail if not present
			client, err := cfg.NewClient()
			if err != nil {
				return ctx, fmt.Errorf("Cleanup: Error initializing client to delete namespace in setup: %w", err)
			}

			nsToDelete := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}

			err = client.Resources().Delete(ctx, nsToDelete)

			if err != nil {
				fmt.Println(fmt.Errorf("Cleanup: Delete namespace failed, might not be present (ignoring): %w", err))
			}

			namespaceList := &corev1.NamespaceList{
				Items: []corev1.Namespace{
					{ObjectMeta: metav1.ObjectMeta{Name: namespace}},
				},
			}

			err = wait.For(conditions.New(client.Resources()).ResourcesDeleted(namespaceList))
			if err != nil {
				fmt.Println(fmt.Errorf("Cleanup: waiting for namespace deletion failed: %w", err))
			}
			return ctx, nil
		},
		envfuncs.CreateNamespace(namespace),
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			// TODO: consider using golang native approach
			if p := utils.RunCommand(
				fmt.Sprintf("kubectl apply -f %s --server-side", latestOperatorYaml),
			); p.Err() != nil {
				return ctx, p.Err()
			}
			return ctx, nil
		},
	)
	// testEnv.Finish(
	// 	envfuncs.DeleteNamespace(namespace),
	// )
	os.Exit(testEnv.Run(m))
}
