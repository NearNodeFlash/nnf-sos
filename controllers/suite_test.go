/*
Copyright 2021 Hewlett Packard Enterprise Development LP
*/

package controllers

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/ghodss/yaml"

	dwsv1alpha1 "github.hpe.com/hpe/hpc-dpm-dws-operator/api/v1alpha1"
	lusv1alpha1 "github.hpe.com/hpe/hpc-rabsw-lustre-fs-operator/api/v1alpha1"
	nnf "github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg"
	"github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/ec"
	nnfv1alpha1 "github.hpe.com/hpe/hpc-rabsw-nnf-sos/api/v1alpha1"

	_ "github.hpe.com/hpe/hpc-dpm-dws-operator/config/crd/bases"
	_ "github.hpe.com/hpe/hpc-dpm-dws-operator/config/webhook"
	_ "github.hpe.com/hpe/hpc-rabsw-lustre-fs-operator/config/crd/bases"

	dwsctrls "github.hpe.com/hpe/hpc-dpm-dws-operator/controllers"
	dwsctrls_daemon "github.hpe.com/hpe/hpc-dpm-dws-operator/mount-daemon/controllers"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

type envSetting struct {
	envVar string
	value  string
}

var envVars = []envSetting{
	{"POD_NAMESPACE", "default"},
	{"NNF_POD_IP", "172.0.0.1"},
	{"NNF_NODE_NAME", "nnf-test-node"},
	{"ACK_GINKGO_DEPRECATIONS", "1.16.4"},
	{"DWS_DRIVER_ID", "nnf"},
	{"RABBIT_NODE", "0"},
	{"RABBIT_TEST_ENV_BYPASS_SERVER_STORAGE_CHECK", "true"},

	// Enable certain quirks necessary for test
	{"NNF_TEST_ENVIRONMENT", "true"},
}

func loadNNFDWDirectiveRuleset(filename string) (dwsv1alpha1.DWDirectiveRule, error) {
	ruleset := dwsv1alpha1.DWDirectiveRule{}

	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return ruleset, err
	}

	err = yaml.Unmarshal(bytes, &ruleset)
	return ruleset, err
}

// This is the single entry point for Go test, and ensures all Ginkgo style tests are run
func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	// Setup environment variables for the test
	for _, v := range envVars {
		os.Setenv(v.envVar, v.value)
	}

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")

	testEnv = &envtest.Environment{
		WebhookInstallOptions: envtest.WebhookInstallOptions{Paths: []string{
			filepath.Join("..", "vendor", "github.hpe.com", "hpe", "hpc-dpm-dws-operator", "config", "webhook"),
			filepath.Join("..", "config", "dws")}},
		ErrorIfCRDPathMissing: true,
		CRDDirectoryPaths: []string{
			filepath.Join("..", "config", "crd", "bases"),
			filepath.Join("..", "vendor", "github.hpe.com", "hpe", "hpc-dpm-dws-operator", "config", "crd", "bases"),
			filepath.Join("..", "vendor", "github.hpe.com", "hpe", "hpc-rabsw-lustre-fs-operator", "config", "crd", "bases"),
		},
		AttachControlPlaneOutput: true,
	}

	// Start and initialize the NNF Controller
	persistence := false
	controller := nnf.NewController(nnf.NewMockOptions(persistence))
	Expect(controller.Init(ec.NewDefaultTestOptions())).NotTo(HaveOccurred())

	go controller.Run()

	var err error

	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = nnfv1alpha1.AddToScheme(testEnv.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = dwsv1alpha1.AddToScheme(testEnv.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = lusv1alpha1.AddToScheme(testEnv.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: testEnv.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// start webhook server using Manager
	webhookInstallOptions := &testEnv.WebhookInstallOptions
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             testEnv.Scheme,
		Host:               webhookInstallOptions.LocalServingHost,
		Port:               webhookInstallOptions.LocalServingPort,
		CertDir:            webhookInstallOptions.LocalServingCertDir,
		LeaderElection:     false,
		MetricsBindAddress: "0",
	})
	Expect(err).NotTo(HaveOccurred())

	/*
		Start Everything
	*/
	err = (&dwsv1alpha1.Workflow{}).SetupWebhookWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&dwsctrls.WorkflowReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Workflow"),
		Scheme: testEnv.Scheme,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&NnfWorkflowReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("NnfWorkflow"),
		Scheme: testEnv.Scheme,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&DirectiveBreakdownReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("DirectiveBreakdown"),
		Scheme: testEnv.Scheme,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&NnfAccessReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("NnfAccess"),
		Scheme: testEnv.Scheme,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&NnfNodeReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("NnfNode"),
		Scheme: testEnv.Scheme,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&NnfNodeStorageReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("NnfNodeStorage"),
		Scheme: testEnv.Scheme,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&NnfStorageReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("NnfStorage"),
		Scheme: testEnv.Scheme,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&NnfNodeSLCReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("NnfNodeSLC"),
		Scheme: testEnv.Scheme,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&DWSServersReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Servers"),
		Scheme: testEnv.Scheme,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&NnfClientMountReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("NnfClientMount"),
		Scheme: testEnv.Scheme,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&dwsctrls_daemon.ClientMountReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ClientMount"),
		Scheme: testEnv.Scheme,
		Mock:   true,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// err = (&)
	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
		gexec.KillAndWait(4 * time.Second)

		// Teardown the test environment once controller is fnished.
		// Otherwise from Kubernetes 1.21+, teardon timeouts waiting on
		// kube-apiserver to return
		err := testEnv.Stop()
		Expect(err).ToNot(HaveOccurred())

		// defer GinkgoRecover()
		// err = k8sManager.Start(ctrl.SetupSignalHandler())
		// Expect(err).NotTo(HaveOccurred())
	}()

	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).ToNot(BeNil())

	// Load the NNF ruleset to enable the webhook to parse #DW directives
	ruleset, err := loadNNFDWDirectiveRuleset(filepath.Join("..", "config", "dws", "nnf-ruleset.yaml"))
	Expect(err).ToNot(HaveOccurred())

	ruleset.Namespace = "default"
	Expect(k8sClient.Create(context.Background(), &ruleset)).Should(Succeed())

})

var _ = AfterSuite(func() {
	By("tearing down the test environment")

	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
