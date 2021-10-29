/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"context"
	"path/filepath"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "k8s.io/api/core/v1"

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
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

const (
	InstanaAgentNamespace = "instana-agent"
)

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var mgrCancel context.CancelFunc

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t,
		"Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
		CRDInstallOptions:     envtest.CRDInstallOptions{CleanUpAfterUse: true},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = instanav1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	// "Live" client to interact directly with the Kubernetes test cluster. The Manager client does caching but for verification
	// we want to read directly what got created.
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Set up the Manager as we'd do in the main.go, but disable some (unneeded) config and use the above cluster configuration
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Namespace: "instana-agent",
		Scheme:    scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	// Create the Reconciler / Controller and register with the Manager (just like in the main.go)
	err = Add(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	var mgrCtx context.Context
	mgrCtx, mgrCancel = context.WithCancel(ctrl.SetupSignalHandler())

	go func() {
		err = k8sManager.Start(mgrCtx)
		Expect(err).ToNot(HaveOccurred())
	}()

}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	if mgrCancel != nil {
		mgrCancel()
	}
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// SetupTest will set up a testing environment.
// This includes:
// * creating a Namespace to be used during the test
// * cleanup the Namespace after the test
// Call this function at the start of each of your tests.
func SetupTest(ctx context.Context) *v1.Namespace {
	ns := &v1.Namespace{}

	BeforeEach(func() {
		*ns = v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: InstanaAgentNamespace},
		}

		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred(), "failed to create \"instana-agent\" test namespace")
	})

	AfterEach(func() {
		err := k8sClient.Delete(ctx, ns)
		Expect(err).NotTo(HaveOccurred(), "failed to delete \"instana-agent\" test namespace")
	})

	return ns
}
