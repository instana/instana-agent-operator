/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/collections/list"
	instanaclient "github.com/instana/instana-agent-operator/pkg/k8s/client"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/pointer"
	// +kubebuilder:scaffold:imports
)

const (
	instanaAgentName      = "instana-agent"
	instanaAgentNamespace = "default"
)

var agent = &instanav1.InstanaAgent{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "instana.io/v1",
		Kind:       "InstanaAgent",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:       instanaAgentName,
		Namespace:  instanaAgentNamespace,
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
	key client.ObjectKey
}

// agent resources
var (
	agentConfigMap = object{
		gvk: schema.GroupVersionKind{
			Version: "v1",
			Kind:    "ConfigMap",
		},
		key: client.ObjectKey{
			Name:      instanaAgentName,
			Namespace: instanaAgentNamespace,
		},
	}
	agentDaemonset = object{
		gvk: schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "DaemonSet",
		},
		key: client.ObjectKey{
			Name:      instanaAgentName,
			Namespace: instanaAgentNamespace,
		},
	}
	agentHeadlessService = object{
		gvk: schema.GroupVersionKind{
			Version: "v1",
			Kind:    "Service",
		},
		key: client.ObjectKey{
			Name:      instanaAgentName + "-headless",
			Namespace: instanaAgentNamespace,
		},
	}
	agentService = object{
		gvk: schema.GroupVersionKind{
			Version: "v1",
			Kind:    "Service",
		},
		key: client.ObjectKey{
			Name:      instanaAgentName,
			Namespace: instanaAgentNamespace,
		},
	}
	agentKeysSecret = object{
		gvk: schema.GroupVersionKind{
			Version: "v1",
			Kind:    "Secret",
		},
		key: client.ObjectKey{
			Name:      instanaAgentName,
			Namespace: instanaAgentNamespace,
		},
	}
	agentContainerSecret = object{
		gvk: schema.GroupVersionKind{
			Version: "v1",
			Kind:    "Secret",
		},
		key: client.ObjectKey{
			Name:      instanaAgentName + "-containers-instana-io",
			Namespace: instanaAgentNamespace,
		},
	}
	agentServiceAccount = object{
		gvk: schema.GroupVersionKind{
			Version: "v1",
			Kind:    "ServiceAccount",
		},
		key: client.ObjectKey{
			Name:      instanaAgentName,
			Namespace: instanaAgentNamespace,
		},
	}
)

// k8sensor resources
var (
	k8SensorConfigMap = object{
		gvk: schema.GroupVersionKind{
			Version: "v1",
			Kind:    "ConfigMap",
		},
		key: client.ObjectKey{
			Name:      instanaAgentName + "-k8sensor",
			Namespace: instanaAgentNamespace,
		},
	}
	k8SensorDeployment = object{
		gvk: schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "Deployment",
		},
		key: client.ObjectKey{
			Name:      instanaAgentName + "-k8sensor",
			Namespace: instanaAgentNamespace,
		},
	}
	k8SensorPdb = object{
		gvk: schema.GroupVersionKind{
			Group:   "policy",
			Version: "v1",
			Kind:    "PodDisruptionBudget",
		},
		key: client.ObjectKey{
			Name:      instanaAgentName + "-k8sensor",
			Namespace: instanaAgentNamespace,
		},
	}
	k8SensorClusterRole = object{
		gvk: schema.GroupVersionKind{
			Group:   "rbac.authorization.k8s.io",
			Version: "v1",
			Kind:    "ClusterRole",
		},
		key: client.ObjectKey{
			Name:      instanaAgentName + "-k8sensor",
			Namespace: instanaAgentNamespace,
		},
	}
	k8SensorClusterRoleBinding = object{
		gvk: schema.GroupVersionKind{
			Group:   "rbac.authorization.k8s.io",
			Version: "v1",
			Kind:    "ClusterRoleBinding",
		},
		key: client.ObjectKey{
			Name:      instanaAgentName + "-k8sensor",
			Namespace: instanaAgentNamespace,
		},
	}
	k8SensorServiceAccount = object{
		gvk: schema.GroupVersionKind{
			Version: "v1",
			Kind:    "ServiceAccount",
		},
		key: client.ObjectKey{
			Name:      instanaAgentName + "-k8sensor",
			Namespace: instanaAgentNamespace,
		},
	}
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var instanaClient instanaclient.InstanaAgentClient
var testEnv *envtest.Environment
var mgrCancel context.CancelFunc
var ctx context.Context
var cancel context.CancelFunc

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(
		t,
		"Controller Suite",
	)
}

var _ = BeforeSuite(
	func() {
		ctx, cancel = context.WithCancel(context.Background())

		logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

		err := instanav1.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())

		By("bootstrapping test environment")
		testEnv = &envtest.Environment{
			CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
			ErrorIfCRDPathMissing: true,
			CRDInstallOptions:     envtest.CRDInstallOptions{CleanUpAfterUse: true},
			Scheme:                scheme.Scheme,
		}

		cfg, err = testEnv.Start()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg).NotTo(BeNil())

		// +kubebuilder:scaffold:scheme

		// Set up the Manager as we'd do in the main.go, but disable some (unneeded) config and use the above cluster configuration
		k8sManager, err := ctrl.NewManager(
			cfg, ctrl.Options{
				Scheme: scheme.Scheme,
			},
		)
		Expect(err).ToNot(HaveOccurred())

		k8sClient = k8sManager.GetClient()
		instanaClient = instanaclient.NewClient(k8sClient)

		// Create the Reconciler / Controller and register with the Manager (just like in the main.go)
		err = Add(k8sManager)
		Expect(err).ToNot(HaveOccurred())

		var mgrCtx context.Context
		mgrCtx, mgrCancel = context.WithCancel(ctrl.SetupSignalHandler())

		err = Add(k8sManager)
		Expect(err).NotTo(HaveOccurred())

		go func() {
			err = k8sManager.Start(mgrCtx)
			Expect(err).ToNot(HaveOccurred())
		}()

	}, 60,
)

var _ = AfterSuite(
	func() {
		By("tearing down the test environment")
		if mgrCancel != nil {
			mgrCancel()
		}
		err := testEnv.Stop()
		Expect(err).NotTo(HaveOccurred())

		cancel()
	},
)

func exist(obj object) bool {
	res, err := instanaClient.Exists(ctx, obj.gvk, obj.key).Get()
	return res && err == nil
}

func allExist(o ...object) func() bool {
	return func() bool {
		objects := list.NewConditions(o)

		return objects.All(exist)
	}
}

func doNotExist(obj object) bool {
	res, err := instanaClient.Exists(ctx, obj.gvk, obj.key).Get()
	return !res && err == nil
}

func noneExist(o ...object) func() bool {
	return func() bool {
		objects := list.NewConditions(o)

		return objects.All(doNotExist)
	}
}

func failTest(err error) {
	Expect(err).NotTo(HaveOccurred())
}

var _ = Describe(
	"An InstanaAgent CR", func() {
		When(
			"the CR is created", func() {
				Specify(
					"using the k8s client", func() {
						instanaClient.Apply(ctx, agent).OnFailure(failTest)
					},
				)
				Specify(
					"the controller should create all of the expected resources", func() {
						Eventually(
							allExist(
								agentConfigMap,
								agentDaemonset,
								agentHeadlessService,
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
						).
							Within(10 * time.Second).
							ProbeEvery(time.Second).
							Should(BeTrue())
					},
				)
			},
		)
		When(
			"the CR is updated", func() {
				Specify(
					"using the k8s client", func() {
						agentNew := agent.DeepCopy()

						agentNew.Spec.K8sSensor.PodDisruptionBudget.Enabled = pointer.To(false)
						agentNew.Spec.Agent.KeysSecret = "test"
						agentNew.Spec.Agent.ImageSpec.Name = helpers.ContainersInstanaIORegistry + "/instana-agent"

						err := k8sClient.Patch(ctx, agentNew, client.MergeFrom(agent))
						Expect(err).NotTo(HaveOccurred())
					},
				)
				Specify(
					"the controller should update the resources", func() {
						Eventually(
							allExist(
								agentConfigMap,
								agentDaemonset,
								agentHeadlessService,
								agentService,
								agentServiceAccount,
								agentContainerSecret,
								k8SensorConfigMap,
								k8SensorDeployment,
								k8SensorServiceAccount,
								k8SensorClusterRole,
								k8SensorClusterRoleBinding,
							),
						).
							Within(10 * time.Second).
							ProbeEvery(time.Second).
							Should(BeTrue())
						Eventually(
							noneExist(
								agentKeysSecret,
								k8SensorPdb,
							),
						).
							Within(10 * time.Second).
							ProbeEvery(time.Second).
							Should(BeTrue())
					},
				)
			},
		)
		When(
			"the CR is deleted", func() {
				Specify(
					"using the k8s client", func() {
						err := k8sClient.Delete(ctx, agent)
						Expect(err).NotTo(HaveOccurred())
					},
				)
				Specify(
					"the controller should delete all of the expected resources", func() {
						Eventually(
							noneExist(
								agentConfigMap,
								agentDaemonset,
								agentHeadlessService,
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
						).
							Within(10 * time.Second).
							ProbeEvery(time.Second).
							Should(BeTrue())
					},
				)
			},
		)
	},
)
