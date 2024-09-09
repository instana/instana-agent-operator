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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/conf"
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
			// wrap delete call to not fail if not present
			// remove namespace api object
			client, err := cfg.NewClient()
			if err != nil {
				return ctx, fmt.Errorf("Error initializing client to delete namespace in setup: %w", err)
			}

			nsToDelete := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}

			err = client.Resources().Delete(ctx, nsToDelete)

			if err != nil {
				fmt.Println(fmt.Errorf("delete namespace func: %w", err))
			}

			namespaceList := &corev1.NamespaceList{
				Items: []corev1.Namespace{
					{ObjectMeta: metav1.ObjectMeta{Name: namespace}},
				},
			}

			err = wait.For(conditions.New(client.Resources()).ResourcesDeleted(namespaceList))
			if err != nil {
				fmt.Println(fmt.Errorf("delete namespace func: %w", err))
			}
			return ctx, nil
		},
		envfuncs.CreateNamespace(namespace),
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			if p := utils.RunCommand(
				fmt.Sprintf("kubectl apply -f %s --server-side", latestOperatorYaml),
			); p.Err() != nil {
				return ctx, p.Err()
			}
			return ctx, nil
		},
	)
	testEnv.Finish(
	//envfuncs.DeleteNamespace(namespace),
	)
	os.Exit(testEnv.Run(m))
}
