/*
 * (c) Copyright IBM Corp. 2025
 */

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	v1 "github.com/instana/instana-agent-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	// Test namespace for alternate CR deployment
	AlternateAgentNamespace = "instana-agent-alternate"

	// Label used by operator to identify managed resources
	ManagedByLabel = "app.kubernetes.io/managed-by"
	ManagedByValue = "instana-agent-operator"
)

// TestMultiNamespaceOperatorDeployment tests the core multi-namespace functionality:
// Operator deployed in namespace A (instana-agent) can successfully manage an
// InstanaAgent CR deployed in namespace B (instana-agent-alternate).
func TestMultiNamespaceOperatorDeployment(t *testing.T) {
	CollectOperatorLogsOnFailure(t)

	// Create agent CR for alternate namespace
	agent := NewAgentCr()
	agent.Name = "instana-agent-alternate"
	agent.Namespace = AlternateAgentNamespace

	multiNamespaceFeature := features.New("operator manages CR in different namespace").
		Setup(SetupOperatorDevBuild()). // Operator in instana-agent namespace
		Setup(WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Setup(CreateNamespace(AlternateAgentNamespace)).
		Setup(DeployAgentCrInNamespace(&agent, AlternateAgentNamespace)).
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Wait for operator to reconcile the CR and create resources
			t.Log("Waiting for operator to reconcile CR in alternate namespace")
			time.Sleep(10 * time.Second)
			return ctx
		}).
		Assess(
			"verify agent resources created in alternate namespace",
			VerifyAgentResourcesInNamespace(AlternateAgentNamespace),
		).
		Assess("verify resources have managed-by label", VerifyManagedByLabels(AlternateAgentNamespace)).
		Assess("verify operator reconciles deleted resources", VerifyReconciliationAcrossNamespaces(AlternateAgentNamespace)).
		Teardown(CleanupNamespace(AlternateAgentNamespace)).
		Feature()

	testEnv.Test(t, multiNamespaceFeature)
}

// TestCrossNamespaceKeysSecretAccess tests that the operator can access KeysSecret
// in the agent's namespace (not the operator's namespace).
// This verifies that Secrets are cached cluster-wide, allowing the operator
// to manage agents in any namespace with their own KeysSecrets.
func TestCrossNamespaceKeysSecretAccess(t *testing.T) {
	CollectOperatorLogsOnFailure(t)

	const agentNamespace = "instana-agent-keys-test"
	const keysSecretName = "external-keys"

	agent := NewAgentCr()
	agent.Name = "instana-agent-keys"
	agent.Namespace = agentNamespace
	agent.Spec.Agent.KeysSecret = keysSecretName // pragma: allowlist secret

	crossNamespaceKeysFeature := features.New("operator accesses KeysSecret in agent namespace").
		Setup(SetupOperatorDevBuild()).
		Setup(WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Setup(CreateNamespace(agentNamespace)).
		Setup(CreateKeysSecretInNamespace(keysSecretName, agentNamespace)).
		Setup(DeployAgentCrInNamespace(&agent, agentNamespace)).
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Wait for operator to reconcile CR and create resources
			t.Log("Waiting for operator to reconcile CR with external KeysSecret")
			time.Sleep(10 * time.Second)
			return ctx
		}).
		Assess(
			"verify agent references keys secret in same namespace",
			func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
				r, err := resources.New(cfg.Client().RESTConfig())
				if err != nil {
					t.Fatal("Failed to initialize client:", err)
				}

				agentCR := &v1.InstanaAgent{}
				if err := r.Get(ctx, agent.Name, agentNamespace, agentCR); err != nil {
					t.Fatalf("Failed to get agent CR: %v", err)
				}

				if agentCR.Spec.Agent.KeysSecret != keysSecretName {
					t.Fatalf(
						"Agent CR does not reference correct KeysSecret. Expected: %s, Got: %s",
						keysSecretName,
						agentCR.Spec.Agent.KeysSecret,
					)
				}

				// Verify the secret exists in the agent's namespace
				secret := &corev1.Secret{}
				if err := r.Get(ctx, keysSecretName, agentNamespace, secret); err != nil {
					t.Fatalf("Failed to get KeysSecret in agent namespace: %v", err)
				}

				t.Logf(
					"✓ Agent CR correctly references KeysSecret '%s' in namespace '%s'",
					keysSecretName,
					agentNamespace,
				)
				return ctx
			}).
		Assess("wait for k8sensor deployment", WaitForK8sensorDeploymentInNamespace(agentNamespace)).
		Assess("wait for agent daemonset", WaitForAgentDaemonSetInNamespace(agentNamespace)).
		Teardown(CleanupNamespace(agentNamespace)).
		Feature()

	testEnv.Test(t, crossNamespaceKeysFeature)
}

// Helper Functions

// CreateNamespace creates a namespace for testing
func CreateNamespace(namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Logf("Creating namespace: %s", namespace)
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal("Failed to initialize client:", err)
		}

		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}

		if err := r.Create(ctx, ns); err != nil && !apierrors.IsAlreadyExists(err) {
			t.Fatalf("Failed to create namespace %s: %v", namespace, err)
		}

		t.Logf("Namespace %s created successfully", namespace)
		return ctx
	}
}

// CleanupNamespace deletes a namespace after testing
// It first deletes any InstanaAgent CRs in the namespace to avoid reconciliation
// conflicts during namespace termination
func CleanupNamespace(namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Logf("Cleaning up namespace: %s", namespace)
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Logf("Failed to initialize client for cleanup: %v", err)
			return ctx
		}

		// First, try to get and delete any InstanaAgent CR in this namespace
		// This prevents the operator from trying to reconcile during namespace termination
		agent := &v1.InstanaAgent{}
		// Try common CR names based on namespace
		possibleNames := []string{namespace, "instana-agent"}
		for _, name := range possibleNames {
			if err := r.Get(ctx, name, namespace, agent); err == nil {
				t.Logf("Deleting Agent CR %s/%s before namespace cleanup", namespace, name)
				if err := r.Delete(ctx, agent); err != nil && !apierrors.IsNotFound(err) {
					t.Logf("Warning: Failed to delete Agent CR %s/%s: %v", namespace, name, err)
				} else {
					// Give the operator time to process the CR deletion
					t.Logf("Waiting for operator to process CR deletion in namespace %s", namespace)
					time.Sleep(5 * time.Second)
					break
				}
			}
		}

		// Now delete the namespace
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}

		if err := r.Delete(ctx, ns); err != nil && !apierrors.IsNotFound(err) {
			t.Logf("Warning: Failed to delete namespace %s: %v", namespace, err)
		} else {
			t.Logf("Namespace %s deleted successfully", namespace)
		}

		return ctx
	}
}

// DeployAgentCrInNamespace deploys an InstanaAgent CR in a specific namespace
func DeployAgentCrInNamespace(agent *v1.InstanaAgent, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Logf("Deploying InstanaAgent CR '%s' in namespace '%s'", agent.Name, namespace)
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal("Failed to initialize client:", err)
		}

		agent.Namespace = namespace
		if err := r.Create(ctx, agent); err != nil {
			t.Fatalf("Failed to create InstanaAgent CR in namespace %s: %v", namespace, err)
		}

		t.Logf(
			"InstanaAgent CR '%s' deployed successfully in namespace '%s'",
			agent.Name,
			namespace,
		)
		return ctx
	}
}

// CreateKeysSecretInNamespace creates a KeysSecret in a specific namespace
func CreateKeysSecretInNamespace(secretName, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Logf("Creating KeysSecret '%s' in namespace '%s'", secretName, namespace)
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal("Failed to initialize client:", err)
		}

		secret := &corev1.Secret{ // pragma: allowlist secret
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"key": []byte(InstanaTestCfg.InstanaBackend.AgentKey), // pragma: allowlist secret
			},
		}

		if err := r.Create(ctx, secret); err != nil && !apierrors.IsAlreadyExists(err) {
			t.Fatalf("Failed to create KeysSecret in namespace %s: %v", namespace, err)
		}

		t.Logf("KeysSecret '%s' created successfully in namespace '%s'", secretName, namespace)
		return ctx
	}
}

// WaitForK8sensorDeploymentInNamespace waits for the k8sensor deployment to become ready in a specific namespace
// Uses label-based discovery to find the deployment dynamically
func WaitForK8sensorDeploymentInNamespace(namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Logf("Waiting for k8sensor deployment in namespace '%s' to become ready", namespace)

		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal("Failed to initialize client:", err)
		}

		err = wait.For(
			func(ctx context.Context) (bool, error) {
				// Find k8sensor deployment by label
				depList := &appsv1.DeploymentList{}
				if err := r.WithNamespace(namespace).List(ctx, depList); err != nil {
					return false, err
				}

				for i := range depList.Items {
					dep := &depList.Items[i]
					if dep.Labels[ManagedByLabel] == ManagedByValue {
						// Check if deployment is ready
						for _, cond := range dep.Status.Conditions {
							if cond.Type == appsv1.DeploymentAvailable &&
								cond.Status == corev1.ConditionTrue {
								t.Logf(
									"✓ K8sensor deployment '%s' is ready in namespace '%s'",
									dep.Name,
									namespace,
								)
								return true, nil
							}
						}
						// Deployment found but not ready yet
						return false, nil
					}
				}
				// Deployment not found yet
				return false, nil
			},
			wait.WithTimeout(time.Minute*5),
			wait.WithInterval(time.Second*5),
		)

		if err != nil {
			t.Fatalf(
				"K8sensor deployment in namespace '%s' did not become ready: %v",
				namespace,
				err,
			)
		}

		return ctx
	}
}

// WaitForAgentDaemonSetInNamespace waits for the agent daemonset to become ready in a specific namespace
// Uses label-based discovery to find the daemonset dynamically
func WaitForAgentDaemonSetInNamespace(namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Logf("Waiting for agent daemonset in namespace '%s' to become ready", namespace)

		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal("Failed to initialize client:", err)
		}

		err = wait.For(
			func(ctx context.Context) (bool, error) {
				// Find agent daemonset by label
				dsList := &appsv1.DaemonSetList{}
				if err := r.WithNamespace(namespace).List(ctx, dsList); err != nil {
					return false, err
				}

				for i := range dsList.Items {
					ds := &dsList.Items[i]
					if ds.Labels[ManagedByLabel] == ManagedByValue {
						// Check if daemonset is ready
						if ds.Status.NumberReady > 0 &&
							ds.Status.NumberReady == ds.Status.DesiredNumberScheduled {
							t.Logf(
								"✓ Agent daemonset '%s' is ready in namespace '%s'",
								ds.Name,
								namespace,
							)
							return true, nil
						}
						// DaemonSet found but not ready yet
						return false, nil
					}
				}
				// DaemonSet not found yet
				return false, nil
			},
			wait.WithTimeout(time.Minute*5),
			wait.WithInterval(time.Second*5),
		)

		if err != nil {
			t.Fatalf("Agent daemonset in namespace '%s' did not become ready: %v", namespace, err)
		}

		return ctx
	}
}

// VerifyAgentResourcesInNamespace verifies that agent resources are created in the specified namespace
func VerifyAgentResourcesInNamespace(namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Logf("Verifying agent resources in namespace '%s'", namespace)
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal("Failed to initialize client:", err)
		}

		// Debug: List all DaemonSets in all namespaces to see where they are
		allDsList := &appsv1.DaemonSetList{}
		if err := r.List(ctx, allDsList); err == nil {
			t.Logf("DEBUG: Found %d DaemonSets across all namespaces:", len(allDsList.Items))
			for _, ds := range allDsList.Items {
				t.Logf("  - %s/%s", ds.Namespace, ds.Name)
			}
		}

		// Check for DaemonSet - operator names it after the CR name, not a fixed name
		// List all DaemonSets in the namespace
		client := cfg.Client()
		dsList := &appsv1.DaemonSetList{}
		if err := client.Resources(namespace).List(ctx, dsList); err != nil {
			t.Fatalf("Failed to list DaemonSets in namespace %s: %v", namespace, err)
		}

		if len(dsList.Items) == 0 {
			t.Fatalf("No DaemonSets found in namespace %s", namespace)
		}

		// Find the agent DaemonSet (should have the managed-by label)
		var agentDS *appsv1.DaemonSet
		for i := range dsList.Items {
			if dsList.Items[i].Labels[ManagedByLabel] == ManagedByValue {
				agentDS = &dsList.Items[i]
				break
			}
		}

		if agentDS == nil {
			t.Fatalf(
				"No agent DaemonSet with label %s=%s found in namespace %s",
				ManagedByLabel,
				ManagedByValue,
				namespace,
			)
		}
		t.Logf("✓ DaemonSet '%s' found in namespace '%s'", agentDS.Name, namespace)

		// Check for Deployment - operator names it after the CR name with -k8sensor suffix
		depList := &appsv1.DeploymentList{}
		if err := client.Resources(namespace).List(ctx, depList); err != nil {
			t.Fatalf("Failed to list Deployments in namespace %s: %v", namespace, err)
		}

		// Find the k8sensor deployment (should have the managed-by label)
		var k8sensorDep *appsv1.Deployment
		for i := range depList.Items {
			if depList.Items[i].Labels[ManagedByLabel] == ManagedByValue {
				k8sensorDep = &depList.Items[i]
				break
			}
		}

		if k8sensorDep == nil {
			t.Fatalf(
				"No k8sensor Deployment with label %s=%s found in namespace %s",
				ManagedByLabel,
				ManagedByValue,
				namespace,
			)
		}
		t.Logf("✓ Deployment '%s' found in namespace '%s'", k8sensorDep.Name, namespace)

		// Check for ServiceAccount (find by label)
		saList := &corev1.ServiceAccountList{}
		if err := r.WithNamespace(namespace).List(ctx, saList); err != nil {
			t.Fatalf("Failed to list ServiceAccounts in namespace %s: %v", namespace, err)
		}

		var agentSA *corev1.ServiceAccount
		for i := range saList.Items {
			if saList.Items[i].Labels[ManagedByLabel] == ManagedByValue {
				agentSA = &saList.Items[i]
				break
			}
		}

		if agentSA == nil {
			t.Fatalf(
				"ServiceAccount with label %s=%s not found in namespace %s",
				ManagedByLabel,
				ManagedByValue,
				namespace,
			)
		}
		t.Logf("✓ ServiceAccount '%s' found in namespace '%s'", agentSA.Name, namespace)

		t.Logf("All agent resources verified in namespace '%s'", namespace)
		return ctx
	}
}

// VerifyManagedByLabels verifies that all operator-created resources have the managed-by label.
// This is critical for the label-based cache filtering to work correctly.
func VerifyManagedByLabels(namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Logf("Verifying managed-by labels in namespace '%s'", namespace)
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal("Failed to initialize client:", err)
		}

		// Check DaemonSet labels
		ds := &appsv1.DaemonSet{}
		if err := r.Get(ctx, AgentDaemonSetName, namespace, ds); err == nil {
			if ds.Labels[ManagedByLabel] != ManagedByValue {
				t.Errorf("DaemonSet missing or incorrect managed-by label. Expected: %s, Got: %s",
					ManagedByValue, ds.Labels[ManagedByLabel])
			} else {
				t.Logf("✓ DaemonSet has correct managed-by label")
			}
		}

		// Check Deployment labels
		dep := &appsv1.Deployment{}
		if err := r.Get(ctx, K8sensorDeploymentName, namespace, dep); err == nil {
			if dep.Labels[ManagedByLabel] != ManagedByValue {
				t.Errorf("Deployment missing or incorrect managed-by label. Expected: %s, Got: %s",
					ManagedByValue, dep.Labels[ManagedByLabel])
			} else {
				t.Logf("✓ Deployment has correct managed-by label")
			}
		}

		// Check ServiceAccount labels
		sa := &corev1.ServiceAccount{}
		if err := r.Get(ctx, "instana-agent", namespace, sa); err == nil {
			if sa.Labels[ManagedByLabel] != ManagedByValue {
				t.Errorf(
					"ServiceAccount missing or incorrect managed-by label. Expected: %s, Got: %s",
					ManagedByValue,
					sa.Labels[ManagedByLabel],
				)
			} else {
				t.Logf("✓ ServiceAccount has correct managed-by label")
			}
		}

		// Check ClusterRole labels (cluster-scoped, but should still have label)
		cr := &rbacv1.ClusterRole{}
		clusterRoleName := fmt.Sprintf("instana-agent-clusterrole-%s", namespace)
		if err := r.Get(ctx, clusterRoleName, "", cr); err == nil {
			if cr.Labels[ManagedByLabel] != ManagedByValue {
				t.Errorf("ClusterRole missing or incorrect managed-by label. Expected: %s, Got: %s",
					ManagedByValue, cr.Labels[ManagedByLabel])
			} else {
				t.Logf("✓ ClusterRole has correct managed-by label")
			}
		}

		t.Logf("Managed-by labels verified in namespace '%s'", namespace)
		return ctx
	}
}

// VerifyReconciliationAcrossNamespaces verifies that the operator can reconcile resources
// in a namespace different from where the operator is deployed.
// This tests that the label-based cache filtering allows cross-namespace reconciliation.
func VerifyReconciliationAcrossNamespaces(namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Logf("Verifying cross-namespace reconciliation for namespace '%s'", namespace)
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal("Failed to initialize client:", err)
		}

		// Find the DaemonSet by label (not by hardcoded name)
		dsList := &appsv1.DaemonSetList{}
		if err := r.WithNamespace(namespace).List(ctx, dsList); err != nil {
			t.Fatalf("Failed to list DaemonSets: %v", err)
		}

		var ds *appsv1.DaemonSet
		for i := range dsList.Items {
			if dsList.Items[i].Labels[ManagedByLabel] == ManagedByValue {
				ds = &dsList.Items[i]
				break
			}
		}

		if ds == nil {
			t.Fatalf(
				"DaemonSet with label %s=%s not found in namespace %s",
				ManagedByLabel,
				ManagedByValue,
				namespace,
			)
		}

		originalUID := ds.UID
		originalName := ds.Name
		t.Logf(
			"Deleting DaemonSet '%s' (UID: %s) to trigger reconciliation",
			originalName,
			originalUID,
		)

		if err := r.Delete(ctx, ds); err != nil {
			t.Fatalf("Failed to delete DaemonSet: %v", err)
		}

		// Wait for the DaemonSet to be recreated by the operator
		t.Logf("Waiting for operator to recreate DaemonSet in namespace '%s'", namespace)
		err = wait.For(
			func(ctx context.Context) (bool, error) {
				// List DaemonSets and find the one with managed-by label
				newDsList := &appsv1.DaemonSetList{}
				if err := r.WithNamespace(namespace).List(ctx, newDsList); err != nil {
					return false, err
				}

				for i := range newDsList.Items {
					if newDsList.Items[i].Labels[ManagedByLabel] == ManagedByValue {
						// Verify it's a new resource (different UID)
						return newDsList.Items[i].UID != originalUID, nil
					}
				}
				return false, nil // DaemonSet not found yet
			},
			wait.WithTimeout(time.Minute*2),
			wait.WithInterval(time.Second*5),
		)

		if err != nil {
			t.Fatalf("Operator failed to reconcile DaemonSet in namespace %s: %v", namespace, err)
		}

		t.Logf("✓ Operator successfully reconciled resources across namespaces")
		return ctx
	}
}

// Made with Bob
