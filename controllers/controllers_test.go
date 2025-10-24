/*
(c) Copyright IBM Corp. 2024, 2025
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
	"fmt"
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

	// Clear any Kubernetes-related environment variables to prevent interference
	// with the test environment when running in a Kubernetes pod
	fmt.Println("Clearing Kubernetes environment variables to isolate test environment")
	if err := os.Unsetenv("KUBERNETES_SERVICE_HOST"); err != nil {
		fmt.Printf("Warning: Failed to unset KUBERNETES_SERVICE_HOST: %v\n", err)
	}
	if err := os.Unsetenv("KUBERNETES_SERVICE_PORT"); err != nil {
		fmt.Printf("Warning: Failed to unset KUBERNETES_SERVICE_PORT: %v\n", err)
	}
	if err := os.Unsetenv("KUBECONFIG"); err != nil {
		fmt.Printf("Warning: Failed to unset KUBECONFIG: %v\n", err)
	}

	// Check if we're running inside a Kubernetes pod by looking for the service account token
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
		fmt.Println(
			"WARNING: Test is running inside a Kubernetes pod. This might affect test behavior.",
		)
		fmt.Println("Setting empty environment variables to ensure we use the test API server")
		if err := os.Setenv("KUBERNETES_SERVICE_HOST", ""); err != nil {
			fmt.Printf("Warning: Failed to set KUBERNETES_SERVICE_HOST: %v\n", err)
		}
		if err := os.Setenv("KUBERNETES_SERVICE_PORT", ""); err != nil {
			fmt.Printf("Warning: Failed to set KUBERNETES_SERVICE_PORT: %v\n", err)
		}
	}

	// Set up the logger for controller-runtime
	logf.SetLogger(zap.New(zap.UseDevMode(true)))

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
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: suite.scheme})
	require.NoError(suite.T(), err)
	err = Add(mgr)
	require.NoError(suite.T(), err)

	go func() {
		err = mgr.Start(suite.ctx)
		require.NoError(suite.T(), err)
	}()
}

// logResourceStatus logs the existence status of each resource for debugging
func (suite *InstanaAgentControllerTestSuite) logResourceStatus(phase string, objects ...object) {
	fmt.Printf("\n--- Resource Status (%s) ---\n", phase)
	for _, obj := range objects {
		exists, _ := suite.instanaAgentClient.Exists(suite.ctx, obj.gvk, obj.key).Get()
		fmt.Printf(
			"Resource %s/%s (%s): %v\n",
			obj.key.Namespace,
			obj.key.Name,
			obj.gvk.Kind,
			exists,
		)
	}
	fmt.Println("----------------------------")
}

// waitForResourceDeletion waits for resources to be deleted with a timeout
func (suite *InstanaAgentControllerTestSuite) waitForResourceDeletion(
	timeout time.Duration,
	objects ...object,
) {
	ctx, cancel := context.WithTimeout(suite.ctx, timeout)
	defer cancel()

	for _, obj := range objects {
		for {
			exists, _ := suite.instanaAgentClient.Exists(ctx, obj.gvk, obj.key).Get()
			if !exists || ctx.Err() != nil {
				break
			}
			time.Sleep(time.Second)
		}
	}
}

// TearDownSuite i.e. AfterSuite
func (suite *InstanaAgentControllerTestSuite) TearDownSuite() {
	// Try to clean up any leftover resources
	agentList := &instanav1.InstanaAgentList{}
	if err := suite.k8sClient.List(suite.ctx, agentList); err == nil {
		for i := range agentList.Items {
			agent := &agentList.Items[i]
			fmt.Printf("Cleaning up agent %s/%s\n", agent.Namespace, agent.Name)
			if err := suite.k8sClient.Delete(suite.ctx, agent); err == nil {
				// Wait for resources to be deleted
				suite.waitForResourceDeletion(30*time.Second,
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
				)
			}
		}
	}

	suite.cancel()
	err := suite.testEnv.Stop()
	require.NoError(suite.T(), err)
}

// all is a utility method to iterate through objects and use the user defined validation function to verify validity
func (suite *InstanaAgentControllerTestSuite) all(
	validatorFunc func(object) bool,
	o ...object,
) func() bool {
	return func() bool {
		return list.NewConditions(o).All(validatorFunc)
	}
}

// exist is a utility method to unwrap the result struct of InstanaAgentClient and return whether the obj existed
func (suite *InstanaAgentControllerTestSuite) exist(obj object) bool {
	exists, _ := suite.instanaAgentClient.Exists(suite.ctx, obj.gvk, obj.key).Get()
	return exists
}

// notExist is a utility method to unwrap the result struct of InstanaAgentClient and return whether the obj didn't exist
func (suite *InstanaAgentControllerTestSuite) notExist(obj object) bool {
	exists, _ := suite.instanaAgentClient.Exists(suite.ctx, obj.gvk, obj.key).Get()
	return !exists
}

// TestInstanaAgentCR is the test method to verify the whole lifecycle of the Instana Agent custom resource from start to deletion against the EnvTest
func (suite *InstanaAgentControllerTestSuite) TestInstanaAgentCR() {
	_, err := suite.instanaAgentClient.Apply(suite.ctx, agent).Get()
	require.NoError(
		suite.T(),
		err,
		"Should not throw an error when applying the InstanaAgent schema",
	)

	// Log resource status before checking
	suite.logResourceStatus("Initial creation",
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
	)

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
		60*time.Second, // Increased timeout from 10s to 60s for CI environments
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
	require.NoError(
		suite.T(),
		err,
		"Should not throw an error when patching the InstanaAgent schema with a new version",
	)

	// Log resource status before checking patched resources
	suite.logResourceStatus("After patch",
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
	)

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
		60*time.Second, // Increased timeout from 10s to 60s for CI environments
		time.Second,
		"Should contain listed objects in the patched schema",
	)
	// Log resource status for objects that should not exist
	suite.logResourceStatus("After patch (should not exist)",
		agentKeysSecret,
		k8SensorPdb,
	)

	require.Eventually(suite.T(),
		suite.all(
			suite.notExist,
			agentKeysSecret,
			k8SensorPdb,
		),
		60*time.Second, // Increased timeout from 10s to 60s for CI environments
		time.Second,
		"Should not contain listed objects after the patched schema",
	)

	err = suite.k8sClient.Delete(suite.ctx, agent)
	require.NoError(suite.T(), err, "Should not return an error while deleting the agent")

	// Log resource status during deletion
	suite.logResourceStatus("After deletion request",
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
	)

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
		60*time.Second, // Increased timeout from 10s to 60s for CI environments
		time.Second,
		"Should delete all objects from the schema",
	)
}
