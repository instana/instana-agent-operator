/*
(c) Copyright IBM Corp. 2024
(c) Copyright Instana Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/collections/list"
	instanaclient "github.com/instana/instana-agent-operator/pkg/k8s/client"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/pointer"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var agentNamespace = types.NamespacedName{
	Name:      "instana-agent",
	Namespace: "default",
}

// The agent schema that will be used throughout the tests
var agent = &instanav1.InstanaAgent{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "instana.io/v1",
		Kind:       "InstanaAgent",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:       agentNamespace.Name,
		Namespace:  agentNamespace.Namespace,
		Finalizers: []string{"test"},
	},
	Spec: instanav1.InstanaAgentSpec{
		Zone:    instanav1.Name{Name: "test"},
		Cluster: instanav1.Name{Name: "test"},
		Agent: instanav1.BaseAgentSpec{
			Key:          "test",
			EndpointHost: "ingress-red-saas.instana.io",
			EndpointPort: "443",
		},
		K8sSensor: instanav1.K8sSpec{
			PodDisruptionBudget: instanav1.Enabled{Enabled: pointer.To(true)},
			DeploymentSpec: instanav1.KubernetesDeploymentSpec{
				Enabled:  instanav1.Enabled{Enabled: pointer.To(true)},
				Replicas: 1,
			},
		},
	},
}

type object struct {
	gvk schema.GroupVersionKind
	key types.NamespacedName
}

// number of agent resources used for diffing whether the controller functions properly
var (
	agentDaemonset = object{
		gvk: schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "DaemonSet",
		},
		key: agentNamespace,
	}
	agentHeadlessService = object{
		gvk: schema.GroupVersionKind{
			Version: "v1",
			Kind:    "Service",
		},
		key: client.ObjectKey{
			Name:      agentNamespace.Name + "-headless",
			Namespace: agentNamespace.Namespace,
		},
	}
	agentSecretConfig = object{
		gvk: schema.GroupVersionKind{
			Version: "v1",
			Kind:    "Secret",
		},
		key: client.ObjectKey{
			Name:      agentNamespace.Name + "-config",
			Namespace: agentNamespace.Namespace,
		},
	}
	agentService = object{
		gvk: schema.GroupVersionKind{
			Version: "v1",
			Kind:    "Service",
		},
		key: agentNamespace,
	}
	agentKeysSecret = object{
		gvk: schema.GroupVersionKind{
			Version: "v1",
			Kind:    "Secret",
		},
		key: agentNamespace,
	}
	agentContainerSecret = object{
		gvk: schema.GroupVersionKind{
			Version: "v1",
			Kind:    "Secret",
		},
		key: client.ObjectKey{
			Name:      agentNamespace.Name + "-containers-instana-io",
			Namespace: agentNamespace.Namespace,
		},
	}
	agentServiceAccount = object{
		gvk: schema.GroupVersionKind{
			Version: "v1",
			Kind:    "ServiceAccount",
		},
		key: agentNamespace,
	}
)

// number of k8sensor resources used for diffing whether the controller functions properly
var (
	k8SensorConfigMap = object{
		gvk: schema.GroupVersionKind{
			Version: "v1",
			Kind:    "ConfigMap",
		},
		key: client.ObjectKey{
			Name:      agentNamespace.Name + "-k8sensor",
			Namespace: agentNamespace.Namespace,
		},
	}
	k8SensorDeployment = object{
		gvk: schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "Deployment",
		},
		key: client.ObjectKey{
			Name:      agentNamespace.Name + "-k8sensor",
			Namespace: agentNamespace.Namespace,
		},
	}
	k8SensorPdb = object{
		gvk: schema.GroupVersionKind{
			Group:   "policy",
			Version: "v1",
			Kind:    "PodDisruptionBudget",
		},
		key: client.ObjectKey{
			Name:      agentNamespace.Name + "-k8sensor",
			Namespace: agentNamespace.Namespace,
		},
	}
	k8SensorClusterRole = object{
		gvk: schema.GroupVersionKind{
			Group:   "rbac.authorization.k8s.io",
			Version: "v1",
			Kind:    "ClusterRole",
		},
		key: client.ObjectKey{
			Name:      agentNamespace.Name + "-k8sensor",
			Namespace: agentNamespace.Namespace,
		},
	}
	k8SensorClusterRoleBinding = object{
		gvk: schema.GroupVersionKind{
			Group:   "rbac.authorization.k8s.io",
			Version: "v1",
			Kind:    "ClusterRoleBinding",
		},
		key: client.ObjectKey{
			Name:      agentNamespace.Name + "-k8sensor",
			Namespace: agentNamespace.Namespace,
		},
	}
	k8SensorServiceAccount = object{
		gvk: schema.GroupVersionKind{
			Version: "v1",
			Kind:    "ServiceAccount",
		},
		key: client.ObjectKey{
			Name:      agentNamespace.Name + "-k8sensor",
			Namespace: agentNamespace.Namespace,
		},
	}
)

// TestInstanaAgentControllerTestSuite is the method that is called to run InstanaAgentControllerTestSuite
func TestInstanaAgentControllerTestSuite(t *testing.T) {
	// Set up the logger for controller-runtime before running the test suite
	logf.SetLogger(zap.New(zap.WriteTo(os.Stdout), zap.UseDevMode(true)))

	// Check if KUBEBUILDER_ASSETS environment variable is set
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		// Check if we're running in an environment where we can't access the kubebuilder assets
		if _, err := os.Stat("/root/.local/share/kubebuilder-envtest/k8s/1.32.0-linux-amd64/etcd"); err != nil {
			t.Skip("Skipping test due to environment setup issues: kubebuilder assets not found")
			return
		}
	}

	suite.Run(t, new(InstanaAgentControllerTestSuite))
}

type InstanaAgentControllerTestSuite struct {
	suite.Suite
	testEnv            *envtest.Environment
	k8sClient          client.Client
	instanaAgentClient instanaclient.InstanaAgentClient
	scheme             *runtime.Scheme
	ctx                context.Context
	cancel             context.CancelFunc
}

// SetupSuite prepares the controller package for testing i.e. BeforeSuite
func (suite *InstanaAgentControllerTestSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())

	// Prepare scheme with instana scheme
	suite.scheme = runtime.NewScheme()
	err := scheme.AddToScheme(suite.scheme)
	require.NoError(suite.T(), err)

	// Add instana agent types to scheme
	err = instanav1.AddToScheme(suite.scheme)
	require.NoError(suite.T(), err)

	// Prepare environment
	suite.testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
		CRDInstallOptions:     envtest.CRDInstallOptions{CleanUpAfterUse: true},
		Scheme:                scheme.Scheme,
	}

	cfg, err := suite.testEnv.Start()
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), cfg)

	// Prepare clients and most importantly Instana Agent Client
	suite.k8sClient, err = client.New(cfg, client.Options{Scheme: suite.scheme})
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), suite.k8sClient)
	suite.instanaAgentClient = instanaclient.NewInstanaAgentClient(suite.k8sClient)

	// Start the manager and controller
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: suite.scheme,
	})
	require.NoError(suite.T(), err)
	err = Add(mgr)
	require.NoError(suite.T(), err)

	// Start the controller in a goroutine
	go func() {
		suite.T().Log("Starting controller manager")
		if err := mgr.Start(suite.ctx); err != nil {
			suite.T().Logf("Error starting manager: %v", err)
		}
	}()

	// Give the manager a moment to start
	time.Sleep(5 * time.Second)
	suite.T().Log("Controller manager should be running now")
}

// TearDownSuite i.e. AfterSuite
func (suite *InstanaAgentControllerTestSuite) TearDownSuite() {
	suite.cancel()

	// Add a longer delay to ensure all operations complete before stopping the environment
	time.Sleep(5 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stopCh := make(chan struct{})
	go func() {
		err := suite.testEnv.Stop()
		if err != nil {
			// Log the error but don't fail the test
			suite.T().Logf("Error stopping test environment: %v", err)
		}
		close(stopCh)
	}()

	select {
	case <-stopCh:
		suite.T().Log("Test environment stopped successfully")
	case <-ctx.Done():
		suite.T().Log("Timed out waiting for test environment to stop")
	}
}

// all is a utility method to iterate through objects and use the user defined validation function to verify validity
func (suite *InstanaAgentControllerTestSuite) all(validatorFunc func(object) bool, o ...object) func() bool {
	return func() bool {
		return list.NewConditions(o).All(validatorFunc)
	}
}

// exist is a utility method to unwrap the result struct of InstanaAgentClient and return whether the obj existed
func (suite *InstanaAgentControllerTestSuite) exist(obj object) bool {
	exists, err := suite.instanaAgentClient.Exists(suite.ctx, obj.gvk, obj.key).Get()
	if err != nil {
		// Log the error but continue with the test
		suite.T().Logf("Error checking if object exists - GVK: %v, Key: %v, Error: %v",
			obj.gvk, obj.key, err)
	}
	if !exists {
		suite.T().Logf("Object does not exist yet - GVK: %v, Key: %v", obj.gvk, obj.key)
	}
	return exists
}

// notExist is a utility method to unwrap the result struct of InstanaAgentClient and return whether the obj didn't exist
func (suite *InstanaAgentControllerTestSuite) notExist(obj object) bool {
	exists, err := suite.instanaAgentClient.Exists(suite.ctx, obj.gvk, obj.key).Get()
	if err != nil {
		// Log the error but continue with the test
		suite.T().Logf("Error checking if object doesn't exist - GVK: %v, Key: %v, Error: %v",
			obj.gvk, obj.key, err)
	}
	return !exists
}

// TestInstanaAgentCR is the test method to verify the whole lifecycle of the Instana Agent custom resource from start to deletion against the EnvTest
func (suite *InstanaAgentControllerTestSuite) TestInstanaAgentCR() {
	_, err := suite.instanaAgentClient.Apply(suite.ctx, agent).Get()
	require.NoError(suite.T(), err, "Should not throw an error when applying the InstanaAgent schema")

	// Verify the agent was created correctly
	createdAgent := &instanav1.InstanaAgent{}
	err = suite.k8sClient.Get(suite.ctx, client.ObjectKey{Name: agent.Name, Namespace: agent.Namespace}, createdAgent)
	require.NoError(suite.T(), err, "Should be able to get the created InstanaAgent")

	require.Eventually(suite.T(),
		suite.all(
			suite.exist,
			agentDaemonset,
			agentHeadlessService,
			agentSecretConfig,
			agentService,
			agentServiceAccount,
			agentKeysSecret,
			k8SensorConfigMap,
			k8SensorDeployment,
			k8SensorServiceAccount,
			k8SensorClusterRole,
			k8SensorClusterRoleBinding,
			k8SensorPdb,
		),
		120*time.Second,
		time.Second,
		"Should contain all objects in the schema",
	)

	agentNew := agent.DeepCopy()
	agentNew.Spec.K8sSensor.PodDisruptionBudget.Enabled = pointer.To(false)
	agentNew.Spec.Agent.KeysSecret = "test"
	agentNew.Spec.Agent.ImageSpec.Name = helpers.ContainersInstanaIORegistry + "/instana-agent"
	err = suite.instanaAgentClient.Patch(
		suite.ctx,
		agentNew,
		client.MergeFrom(agent),
	)
	require.NoError(suite.T(), err, "Should not throw an error when patching the InstanaAgent schema with a new version")

	require.Eventually(suite.T(),
		suite.all(
			suite.exist,
			agentDaemonset,
			agentHeadlessService,
			agentSecretConfig,
			agentService,
			agentServiceAccount,
			agentContainerSecret,
			k8SensorConfigMap,
			k8SensorDeployment,
			k8SensorServiceAccount,
			k8SensorClusterRole,
			k8SensorClusterRoleBinding,
		),
		60*time.Second,
		time.Second,
		"Should contain listed objects in the patched schema",
	)
	require.Eventually(suite.T(),
		suite.all(
			suite.notExist,
			agentKeysSecret,
			k8SensorPdb,
		),
		60*time.Second,
		time.Second,
		"Should not contain listed objects after the patched schema",
	)

	err = suite.k8sClient.Delete(suite.ctx, agent)
	require.NoError(suite.T(), err, "Should not return an error while deleting the agent")
	require.Eventually(suite.T(),
		suite.all(
			suite.notExist,
			agentDaemonset,
			agentHeadlessService,
			agentSecretConfig,
			agentService,
			agentServiceAccount,
			agentKeysSecret,
			agentContainerSecret,
			k8SensorConfigMap,
			k8SensorDeployment,
			k8SensorServiceAccount,
			k8SensorClusterRole,
			k8SensorClusterRoleBinding,
			k8SensorPdb,
		),
		30*time.Second,
		time.Second,
		"Should delete all objects from the schema",
	)
}
