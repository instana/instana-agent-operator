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
	securityv1 "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
)

var testEnv env.Environment

const namespace string = "instana-agent"

// DeleteAgentNamespace ensures a proper cleanup of existing instana agent installations.
// The namespace cannot be just deleted in all scenarios, as finalizers on the agent CR might block the namespace termination
func DeleteAgentNamespaceIfPresent() env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		// Create a client to interact with the Kube API
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			return ctx, fmt.Errorf("Cleanup: Error initializing client to delete agent CR: %v", err)
		}

		// Check if namespace exist, otherwise just skip over it
		agentNamespace := &corev1.Namespace{}
		err = r.Get(ctx, namespace, namespace, agentNamespace)
		if errors.IsNotFound(err) {
			return ctx, nil
		}
		// Something on the API request failed, this should fail the cleanup
		if err != nil {
			return ctx, fmt.Errorf("Cleanup: Getting namespace failed: %v", err)
		}

		// Cleanup a potentially existing Agent CR first
		if _, err = DeleteAgentCRIfPresent()(ctx, cfg); err != nil {
			return ctx, err
		}

		// Delete the Namespace
		if err = r.Delete(ctx, agentNamespace); err != nil {
			return ctx, fmt.Errorf("Cleanup: Delete namespace failed: %v", err)
		}

		// Wait for the termination of the namespace
		namespaceList := &corev1.NamespaceList{
			Items: []corev1.Namespace{
				*agentNamespace,
			},
		}

		err = wait.For(conditions.New(r).ResourcesDeleted(namespaceList))
		if err != nil {
			return ctx, fmt.Errorf("Cleanup: waiting for namespace deletion failed: %v", err)
		}
		return ctx, nil
	}
}

func DeleteAgentCRIfPresent() env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		// Create a client to interact with the Kube API
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			return ctx, fmt.Errorf("Cleanup: Error initializing client to delete agent CR: %v", err)
		}

		// Assume an existing namespace at this point, check if an agent CR is present (requires to adjust schema of current client)
		r.WithNamespace(namespace)
		err = v1.AddToScheme(r.GetScheme())
		if err != nil {
			// If this fails, the cleanup will not work properly -> failing
			return ctx, fmt.Errorf("Cleanup: Error could not add agent types to current scheme: %v", err)
		}

		// If the agent cr is available, but the operator is already gone, the finalizer will never be removed
		// This will lead to a delayed namespace termination which never completes. To avoid that, patch the agent CR
		// to remove the finalizer. Afterwards, it can be successfully deleted.
		agent := &v1.InstanaAgent{}
		err = r.Get(ctx, "instana-agent", "instana-agent", agent)
		if errors.IsNotFound(err) {
			// No agent cr found, skip this cleanup step
			return ctx, nil
		}

		// The agent CR could not be fetched due to a different reason, failing
		if err != nil {
			return ctx, fmt.Errorf("Cleanup: Fetch agent CR failed: %v", err)
		}

		// Removing the finalizer from the existing Agent CR to make it deletable
		// kubectl patch agent instana-agent -p '{"metadata":{"finalizers":[]}}' --type=merge
		err = r.Patch(ctx, agent, k8s.Patch{
			PatchType: types.MergePatchType,
			Data:      []byte(`{"metadata":{"finalizers":[]}}`),
		})
		if err != nil {
			return ctx, fmt.Errorf("Cleanup: Patch agent CR failed: %v", err)
		}

		// delete explicitly, namespace deletion would delete the agent CR as well if the finalizer is not present
		err = r.Delete(ctx, agent)

		if err != nil {
			// The deletion failed for some reason, failing the cleanup
			return ctx, fmt.Errorf("Cleanup: Delete agent CR failed: %v", err)
		}

		agentCrList := &v1.InstanaAgentList{
			Items: []v1.InstanaAgent{*agent},
		}

		// Ensure to wait for the agent CR to disappear before continuing
		err = wait.For(conditions.New(r).ResourcesDeleted(agentCrList))
		if err != nil {
			return ctx, fmt.Errorf("Cleanup: Waiting for agent CR deletion failed: %v", err)
		}
		return ctx, nil
	}
}

// On OpenShift we need to ensure the instana-agent service account gets permission to the privilged security context
func AdjustOcpPermissionsIfNecessary() env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		// Create a client to interact with the Kube API
		clientSet, err := kubernetes.NewForConfig(cfg.Client().RESTConfig())
		if err != nil {
			return ctx, fmt.Errorf("Error creating a clientset: %v", err)
		}

		discoveryClient := discovery.NewDiscoveryClient(clientSet.RESTClient())
		apiGroups, err := discoveryClient.ServerGroups()
		if err != nil {
			return ctx, fmt.Errorf("Failed to fetch apiGroups: %v", err)
		}

		isOpenShift := false
		for _, group := range apiGroups.Groups {
			if group.Name == "apps.openshift.io" {
				isOpenShift = true
				break
			}
		}

		if isOpenShift {
			command := "oc adm policy add-scc-to-user privileged -z instana-agent -n instana-agent"
			fmt.Printf("OpenShift detected, adding instana-agent service account to SecurityContextConstraints: %s\n", command)

			// replaced command execution with SDK call to not require `oc` cli
			securityClient, err := securityv1.NewForConfig(cfg.Client().RESTConfig())
			if err != nil {
				return ctx, fmt.Errorf("Could not initialize securityClient: %v", err)
			}

			// security context
			scc, err := securityClient.SecurityContextConstraints().Get(ctx, "privileged", metav1.GetOptions{})
			if err != nil {
				return ctx, fmt.Errorf("Failed to get SecurityContextContraints: %v", err)
			}

			serviceAccountId := fmt.Sprintf("system:serviceaccount:%s:%s", namespace, "instana-agent")
			userFound := false

			for _, user := range scc.Users {
				if user == serviceAccountId {
					userFound = true
					break
				}
			}

			if userFound {
				fmt.Printf("Security Context Constraint \"privileged\" already lists service account user: %v\n", serviceAccountId)
				return ctx, nil
			}

			// updating Security Context Constraints
			scc.Users = append(scc.Users, serviceAccountId)

			_, err = securityClient.SecurityContextConstraints().Update(ctx, scc, metav1.UpdateOptions{})
			if err != nil {
				return ctx, fmt.Errorf("Could not update Security Context Constraints on OCP cluster: %v", err)
			}

			// p := utils.RunCommand(command)
			// return ctx, p.Err()
			return ctx, nil
		} else {
			fmt.Println("Vanilla Kubernetes detected")
		}
		return ctx, nil
	}
}

func TestMain(m *testing.M) {
	path := conf.ResolveKubeConfigFile()
	cfg := envconf.NewWithKubeConfig(path)
	testEnv = env.NewWithConfig(cfg)
	testEnv.Setup(
		DeleteAgentNamespaceIfPresent(),
		envfuncs.CreateNamespace(namespace),
		AdjustOcpPermissionsIfNecessary(),
	)
	// Consider leave artifacts in cluster for easier debugging,
	// as a new run needs to cleanup anyways. Cleanup for now to ensure
	// that the existing test suite is not facing issues.
	testEnv.Finish(DeleteAgentNamespaceIfPresent())
	os.Exit(testEnv.Run(m))
}
