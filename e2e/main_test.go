/*
 * (c) Copyright IBM Corp. 2024
 * (c) Copyright Instana Inc. 2024
 */

package e2e

import (
	"context"
	"os"
	"testing"

	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
)

var testEnv env.Environment

func TestMain(m *testing.M) {
	path := conf.ResolveKubeConfigFile()
	cfg := envconf.NewWithKubeConfig(path)
	cfg.WithNamespace(InstanaNamespace)
	testEnv = env.NewWithConfig(cfg)
	// cluster level setup
	testEnv.Setup(
		AdjustOcpPermissionsIfNecessary(),
	)
	// ensure a new clean namespace before every test
	// EnvFuncs are only allowed in testEnv.Setup, testEnv.BeforeEachTest requires TestEnvFuncs, therefore converting below
	testEnv.BeforeEachTest(
		func(ctx context.Context, cfg *envconf.Config, t *testing.T) (context.Context, error) {
			return EnsureAgentNamespaceDeletion()(ctx, cfg)
		},
		func(ctx context.Context, cfg *envconf.Config, t *testing.T) (context.Context, error) {
			return envfuncs.CreateNamespace(cfg.Namespace())(ctx, cfg)
		},
	)
	// Consider leave artifacts in cluster for easier debugging,
	// as a new run needs to cleanup anyways. Cleanup for now to ensure
	// that the existing test suite is not facing issues.
	testEnv.Finish(EnsureAgentNamespaceDeletion())
	os.Exit(testEnv.Run(m))
}
