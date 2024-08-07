/*
 * Copyright 2021-2024 Hewlett Packard Enterprise Development LP
 * Other additional copyright holders may be indicated within.
 *
 * The entirety of this work is licensed under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 *
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package controller

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	ctrl "sigs.k8s.io/controller-runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
	zapcr "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/ghodss/yaml"

	dwsv1alpha2 "github.com/DataWorkflowServices/dws/api/v1alpha2"
	lusv1beta1 "github.com/NearNodeFlash/lustre-fs-operator/api/v1beta1"
	nnf "github.com/NearNodeFlash/nnf-ec/pkg"

	nnfv1alpha1 "github.com/NearNodeFlash/nnf-sos/api/v1alpha1"

	_ "github.com/DataWorkflowServices/dws/config/crd/bases"
	_ "github.com/DataWorkflowServices/dws/config/webhook"
	_ "github.com/NearNodeFlash/lustre-fs-operator/config/crd/bases"

	dwsctrls "github.com/DataWorkflowServices/dws/controllers"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc

type envSetting struct {
	envVar string
	value  string
}

var envVars = []envSetting{
	{"POD_NAMESPACE", "default"},
	{"NNF_STORAGE_PROFILE_NAMESPACE", "default"},
	{"NNF_CONTAINER_PROFILE_NAMESPACE", "default"},
	{"NNF_DM_PROFILE_NAMESPACE", "default"},
	{"NNF_PORT_MANAGER_NAME", "nnf-port-manager"},
	{"NNF_PORT_MANAGER_NAMESPACE", "default"},
	{"NNF_POD_IP", "172.0.0.1"},
	{"NNF_NODE_NAME", "nnf-test-node"},
	{"ACK_GINKGO_DEPRECATIONS", "1.16.4"},
	{"DWS_DRIVER_ID", "nnf"},
	{"RABBIT_NODE", "0"},
	{"RABBIT_TEST_ENV_BYPASS_SERVER_STORAGE_CHECK", "true"},

	// Enable certain quirks necessary for test
	{"NNF_TEST_ENVIRONMENT", "true"},
}

func loadNNFDWDirectiveRuleset(filename string) (dwsv1alpha2.DWDirectiveRule, error) {
	ruleset := dwsv1alpha2.DWDirectiveRule{}

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

	encoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	zaplogger := zapcr.New(zapcr.WriteTo(GinkgoWriter), zapcr.Encoder(encoder), zapcr.UseDevMode(true))
	logf.SetLogger(zaplogger)

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")

	webhookPaths := []string{
		filepath.Join("..", "..", "vendor", "github.com", "DataWorkflowServices", "dws", "config", "webhook"),
		filepath.Join("..", "..", "config", "dws"),
	}
	if env, found := os.LookupEnv("WEBHOOK_DIR"); found {
		webhookPaths = append(webhookPaths, env)
	}

	testEnv = &envtest.Environment{
		WebhookInstallOptions: envtest.WebhookInstallOptions{Paths: webhookPaths},
		ErrorIfCRDPathMissing: true,
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
			filepath.Join("..", "..", "vendor", "github.com", "DataWorkflowServices", "dws", "config", "crd", "bases"),
			filepath.Join("..", "..", "vendor", "github.com", "NearNodeFlash", "lustre-fs-operator", "config", "crd", "bases"),
		},
		//AttachControlPlaneOutput: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = nnfv1alpha1.AddToScheme(testEnv.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = dwsv1alpha2.AddToScheme(testEnv.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = lusv1beta1.AddToScheme(testEnv.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = nnfv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// start webhook server using Manager
	webhookInstallOptions := &testEnv.WebhookInstallOptions
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:    webhookInstallOptions.LocalServingHost,
			Port:    webhookInstallOptions.LocalServingPort,
			CertDir: webhookInstallOptions.LocalServingCertDir,
		}),
	})
	Expect(err).NotTo(HaveOccurred())

	/*
		Start DWS pieces
	*/

	err = (&dwsctrls.WorkflowReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Workflow"),
		Scheme: testEnv.Scheme,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&dwsctrls.ClientMountReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ClientMount"),
		Scheme: testEnv.Scheme,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&dwsv1alpha2.Workflow{}).SetupWebhookWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	/*
		Start NNF-SOS SLC pieces
	*/

	err = (&NnfSystemConfigurationReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("NnfSystemConfiguration"),
		Scheme: testEnv.Scheme,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&NnfPortManagerReconciler{
		Client: k8sManager.GetClient(),
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

	err = (&PersistentStorageReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("PersistentStorage"),
		Scheme: testEnv.Scheme,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&DWSStorageReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Storage"),
		Scheme: testEnv.Scheme,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&DWSServersReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Servers"),
		Scheme: testEnv.Scheme,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&NnfAccessReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("NnfAccess"),
		Scheme: testEnv.Scheme,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&NnfStorageReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("NnfStorage"),
		Scheme: testEnv.Scheme,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&nnfv1alpha1.NnfStorageProfile{}).SetupWebhookWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&nnfv1alpha1.NnfContainerProfile{}).SetupWebhookWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&nnfv1alpha1.NnfDataMovementProfile{}).SetupWebhookWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	/*
		Start NNF-SOS NLC pieces
	*/

	// Coordinate the startup of the NLC controllers that use EC.

	semNnfNodeDone := make(chan struct{})
	err = (&NnfNodeReconciler{
		Client:           k8sManager.GetClient(),
		Log:              ctrl.Log.WithName("controllers").WithName("NnfNode"),
		Scheme:           testEnv.Scheme,
		SemaphoreForDone: semNnfNodeDone,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	semNnfNodeECDone := make(chan struct{})
	err = (&NnfNodeECDataReconciler{
		Client:            k8sManager.GetClient(),
		Scheme:            testEnv.Scheme,
		Options:           nnf.NewMockOptions(false),
		SemaphoreForStart: semNnfNodeDone,
		SemaphoreForDone:  semNnfNodeECDone,
		RawLog:            ctrl.Log,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	semNnfNodeBlockStorageDone := make(chan struct{})
	err = (&NnfNodeBlockStorageReconciler{
		Client:            k8sManager.GetClient(),
		Log:               ctrl.Log.WithName("controllers").WithName("NnfNodeBlockStorage"),
		Scheme:            testEnv.Scheme,
		SemaphoreForStart: semNnfNodeECDone,
		SemaphoreForDone:  semNnfNodeBlockStorageDone,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	semNnfNodeStorageDone := make(chan struct{})
	err = (&NnfNodeStorageReconciler{
		Client:            k8sManager.GetClient(),
		Log:               ctrl.Log.WithName("controllers").WithName("NnfNodeStorage"),
		Scheme:            testEnv.Scheme,
		SemaphoreForStart: semNnfNodeBlockStorageDone,
		SemaphoreForDone:  semNnfNodeStorageDone,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// The NLC controllers relying on the readiness of EC.

	err = (&NnfClientMountReconciler{
		Client:            k8sManager.GetClient(),
		Log:               ctrl.Log.WithName("controllers").WithName("NnfClientMount"),
		Scheme:            testEnv.Scheme,
		SemaphoreForStart: semNnfNodeStorageDone,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err := k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

	// Load the NNF ruleset to enable the webhook to parse #DW directives
	ruleset, err := loadNNFDWDirectiveRuleset(filepath.Join("..", "..", "config", "dws", "nnf-ruleset.yaml"))
	Expect(err).ToNot(HaveOccurred())

	ruleset.Namespace = "default"
	Expect(k8sClient.Create(context.Background(), &ruleset)).Should(Succeed())

})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
