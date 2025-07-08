package e2e

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	v1 "github.com/instana/instana-agent-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	log "k8s.io/klog/v2"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	e2etypes "sigs.k8s.io/e2e-framework/pkg/types"
)

var (
	AgentRemoteDeploymentName = "instana-agent-r-"
)

func TestInstanaAgentRemoteIdealFlow(t *testing.T) {
	agent := NewAgentCr()
	agentRemote := NewAgentRemoteCr("1")
	agentRemote2 := NewAgentRemoteCr("2")
	initialInstallFeature := features.New("Instana Agent Remote ideal flow. Install Instana Agent, than remote agent(s)").
		Setup(SetupOperatorDevBuild()).
		Setup(DeployAgentCr(&agent)).
		Setup(DeployAgentRemoteCr(&agentRemote)).
		Setup(DeployAgentRemoteCr(&agentRemote2)).
		Assess("wait for instana-agent-controller-manager deployment to become ready", WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Assess("wait for agent daemonset to become ready", WaitForAgentDaemonSetToBecomeReady()).
		Assess("wait for agent remote deployment to become ready", WaitForDeploymentToBecomeReady(AgentRemoteDeploymentName+"1")).
		Assess("wait for agent remote deployment to become ready", WaitForDeploymentToBecomeReady(AgentRemoteDeploymentName+"2")).
		Assess("remove first instana agent remote CR", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			if _, err := DeleteAgentRemoteCRIfPresent(AgentRemoteDeploymentName+"1")(ctx, cfg); err != nil {
				log.Info("Agent Remote CR cleanup err: ", err)
			}
			return ctx
		}).
		Assess("wait for agent remote deployment 1 to be deleted", WaitForDeploymentToBeDeleted(AgentRemoteDeploymentName+"1")).
		Assess("remove second instana agent remote CR", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			if _, err := DeleteAgentRemoteCRIfPresent(AgentRemoteDeploymentName+"2")(ctx, cfg); err != nil {
				log.Info("Agent Remote CR cleanup err: ", err)
			}
			return ctx
		}).
		Assess("wait for agent remote deployment 2 to be deleted", WaitForDeploymentToBeDeleted(AgentRemoteDeploymentName+"2")).
		Assess("remove instana agent CR", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			if _, err := DeleteAgentCRIfPresent()(ctx, cfg); err != nil {
				log.Info("Agent CR cleanup err: ", err)
			}
			return ctx
		}).
		Feature()

	// test feature
	testEnv.Test(t, initialInstallFeature)

}

func TestInstanaAgentRemoteFlowRemoteFirst(t *testing.T) {
	agent := NewAgentCr()
	agentRemote := NewAgentRemoteCr("1")
	initialInstallFeature := features.New("Instana Agent Agent Remote First flow. Install Agent Remote, than Instana Agent").
		Setup(SetupOperatorDevBuild()).
		Setup(DeployAgentRemoteCr(&agentRemote)).
		Assess("wait for instana-agent-controller-manager deployment to become ready", WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Assess("ensure no remote-agent deployment exists", EnsureDeploymentDoesNotExist(AgentRemoteDeploymentName+"1")).
		Assess("deploy agent CR", DeployAgentCr(&agent)).
		Assess("wait for agent daemonset to become ready", WaitForAgentDaemonSetToBecomeReady()).
		Assess("wait for agent remote deployment to become ready", WaitForDeploymentToBecomeReady(AgentRemoteDeploymentName+"1")).
		Feature()

	// test feature
	testEnv.Test(t, initialInstallFeature)
}

func TestInstanaAgentRemoteFlowCascadeDelete(t *testing.T) {
	agent := NewAgentCr()
	agentRemote := NewAgentRemoteCr("1")
	initialInstallFeature := features.New("Instana Agent Remote ideal flow. Install Instana Agent, than remote agent(s)").
		Setup(SetupOperatorDevBuild()).
		Setup(DeployAgentCr(&agent)).
		Setup(DeployAgentRemoteCr(&agentRemote)).
		Assess("wait for instana-agent-controller-manager deployment to become ready", WaitForDeploymentToBecomeReady(InstanaOperatorDeploymentName)).
		Assess("wait for agent daemonset to become ready", WaitForAgentDaemonSetToBecomeReady()).
		Assess("wait for agent remote deployment to become ready", WaitForDeploymentToBecomeReady(AgentRemoteDeploymentName+"1")).
		Assess("remove instana agent CR", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			if _, err := DeleteAgentCRIfPresent()(ctx, cfg); err != nil {
				log.Info("Agent CR cleanup err: ", err)
			}
			return ctx
		}).
		Assess("ensure agent remote has been deleted", EnsureDeploymentDoesNotExist(AgentRemoteDeploymentName+"1")).
		Feature()

	// test feature
	testEnv.Test(t, initialInstallFeature)
}

func DeleteAgentRemoteCRIfPresent(name string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		log.Info("Ensure agent remote CR is not present")
		// Create a client to interact with the Kube API
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			return ctx, fmt.Errorf("cleanup: Error initializing client to delete agent remote CR: %v", err)
		}

		// Assume an existing namespace at this point, check if an agent remote CR is present (requires to adjust schema of current client)
		r.WithNamespace(InstanaNamespace)
		err = v1.AddToScheme(r.GetScheme())
		if err != nil {
			// If this fails, the cleanup will not work properly -> failing
			return ctx, fmt.Errorf("cleanup: Error could not add agent remote types to current scheme: %v", err)
		}

		// If the agent remote cr is available, but the operator is already gone, the finalizer will never be removed
		// This will lead to a delayed namespace termination which never completes. To avoid that, patch the agent CR
		// to remove the finalizer. Afterwards, it can be successfully deleted.
		agent := &v1.InstanaAgentRemote{}
		err = r.Get(ctx, name, InstanaNamespace, agent)
		if errors.IsNotFound(err) {
			// No agent cr found, skip this cleanup step
			log.Info("No agent remote CR present, skipping deletion")
			return ctx, nil
		}

		// The agent CR could not be fetched due to a different reason, failing
		if err != nil {
			return ctx, fmt.Errorf("cleanup: Fetch agent remote CR failed: %v", err)
		}

		// Removing the finalizer from the existing Agent CR to make it deletable
		// kubectl patch agent instana-agent -p '{"metadata":{"finalizers":[]}}' --type=merge
		log.Info("Patching agent remote cr to remove finalizers")
		err = r.Patch(ctx, agent, k8s.Patch{
			PatchType: types.MergePatchType,
			Data:      []byte(`{"metadata":{"finalizers":[]}}`),
		})
		if err != nil {
			return ctx, fmt.Errorf("cleanup: Patch agent remote CR failed: %v", err)
		}

		log.Info("Deleting CR")
		// delete explicitly, namespace deletion would delete the agent CR as well if the finalizer is not present
		err = r.Delete(ctx, agent)

		if err != nil {
			// The deletion failed for some reason, failing the cleanup
			return ctx, fmt.Errorf("cleanup: Delete agent remote CR failed: %v", err)
		}

		agentCrList := &v1.InstanaAgentRemoteList{
			Items: []v1.InstanaAgentRemote{*agent},
		}

		// Ensure to wait for the agent CR to disappear before continuing
		err = wait.For(conditions.New(r).ResourcesDeleted(agentCrList))
		if err != nil {
			return ctx, fmt.Errorf("cleanup: Waiting for agent remote CR deletion failed: %v", err)
		}
		log.Info("Agent remote CR is gone")
		return ctx, nil
	}
}

func NewAgentRemoteCr(name string) v1.InstanaAgentRemote {

	return v1.InstanaAgentRemote{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: InstanaNamespace,
		},
		Spec: v1.InstanaAgentRemoteSpec{
			Zone: v1.Name{
				Name: "e2e",
			},
			// ensure to not overlap between concurrent test runs on different clusters, randomize cluster name, but have consistent zone
			Cluster: v1.Name{Name: envconf.RandomName("e2e", 9)},
			Agent: v1.BaseAgentSpec{
				Key:          InstanaTestCfg.InstanaBackend.AgentKey,
				EndpointHost: InstanaTestCfg.InstanaBackend.EndpointHost,
				EndpointPort: strconv.Itoa(InstanaTestCfg.InstanaBackend.EndpointPort),
			},
		},
	}
}

func DeployAgentRemoteCr(agent *v1.InstanaAgentRemote) e2etypes.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		t.Log("Creating a new Agent Remote CR")

		// Create Agent CR
		r := client.Resources(cfg.Namespace())
		err = v1.AddToScheme(r.GetScheme())
		if err != nil {
			t.Fatal("Could not add Agent CR to client scheme", err)
		}

		err = r.Create(ctx, agent)
		if err != nil {
			t.Fatal("Could not create Agent Remote CR", err)
		}

		return ctx
	}
}

func WaitForDeploymentToBeDeleted(name string) e2etypes.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Logf("Waiting for deployment %s to be deleted", name)

		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		r := client.Resources().WithNamespace(cfg.Namespace())

		err = wait.For(func(ctx context.Context) (bool, error) {
			dep := &appsv1.Deployment{}
			getErr := r.Get(ctx, name, cfg.Namespace(), dep)
			if errors.IsNotFound(getErr) {
				return true, nil // successfully deleted
			}
			if getErr != nil {
				return false, getErr // real error
			}
			return false, nil // still exists
		}, wait.WithTimeout(2*time.Minute), wait.WithInterval(2*time.Second))

		if err != nil {
			t.Fatalf("Deployment %s was not deleted in time: %v", name, err)
		}

		t.Logf("Deployment %s has been successfully deleted", name)
		return ctx
	}
}

func EnsureDeploymentDoesNotExist(name string) e2etypes.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Logf("Ensuring no deployment named %s exists", name)

		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		r := client.Resources().WithNamespace(cfg.Namespace())

		dep := &appsv1.Deployment{}
		err = r.Get(ctx, name, cfg.Namespace(), dep)
		if err == nil {
			t.Fatalf("Expected no Deployment named %s, but one was found", name)
		}
		if !errors.IsNotFound(err) {
			t.Fatalf("Error while checking for Deployment %s: %v", name, err)
		}

		t.Logf("Confirmed: no deployment named %s exists", name)
		return ctx
	}
}
