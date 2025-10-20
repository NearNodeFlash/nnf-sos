/*
 * Copyright 2021-2025 Hewlett Packard Enterprise Development LP
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
	"os"
	"path/filepath"
	"strings"
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
	"sigs.k8s.io/controller-runtime/pkg/event"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
	zapcr "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/ghodss/yaml"

	dwsv1alpha6 "github.com/DataWorkflowServices/dws/api/v1alpha6"
	lusv1beta1 "github.com/NearNodeFlash/lustre-fs-operator/api/v1beta1"
	nnf "github.com/NearNodeFlash/nnf-ec/pkg"

	nnfv1alpha7 "github.com/NearNodeFlash/nnf-sos/api/v1alpha7"
	nnfv1alpha8 "github.com/NearNodeFlash/nnf-sos/api/v1alpha8"
	nnfv1alpha9 "github.com/NearNodeFlash/nnf-sos/api/v1alpha9"

	_ "github.com/DataWorkflowServices/dws/config/crd/bases"
	_ "github.com/DataWorkflowServices/dws/config/webhook"
	_ "github.com/NearNodeFlash/lustre-fs-operator/config/crd/bases"
	_ "github.com/NearNodeFlash/lustre-fs-operator/config/webhook"

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

func loadNNFDWDirectiveRuleset(filename string) (dwsv1alpha6.DWDirectiveRule, error) {
	ruleset := dwsv1alpha6.DWDirectiveRule{}

	bytes, err := os.ReadFile(filename)
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
	var err error

	// See https://github.com/kubernetes-sigs/controller-runtime/issues/1882
	// about getting the conversion webhook to register properly.
	// Begin by relocating the code that builds the scheme, so it happens
	// before calling envtest.Start().
	// Then add the scheme to envtest.CRDInstallOptions.

	err = dwsv1alpha6.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = lusv1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = nnfv1alpha7.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = nnfv1alpha8.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = nnfv1alpha9.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	webhookPaths := []string{}
	webhookPathsAlt := []string{
		filepath.Join("..", "..", "config", "webhook"),
		filepath.Join("..", "..", "vendor", "github.com", "DataWorkflowServices", "dws", "config", "webhook"),
		filepath.Join("..", "..", "vendor", "github.com", "NearNodeFlash", "lustre-fs-operator", "config", "webhook"),
	}
	// Envtest doesn't run kustomize, so the basic configs don't have the
	// `namePrefix` and they will all collide on the same name. The WEBHOOK_DIRS
	// variable points to pre-processed webhook configs that have had the
	// namePrefix added to them.
	if env, found := os.LookupEnv("WEBHOOK_DIRS"); found {
		webhookPaths = append(webhookPaths, strings.Split(env, ":")...)
	} else {
		webhookPaths = append(webhookPaths, webhookPathsAlt...)
	}

	testEnv = &envtest.Environment{
		WebhookInstallOptions: envtest.WebhookInstallOptions{Paths: webhookPaths},
		ErrorIfCRDPathMissing: true,
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
			filepath.Join("..", "..", "vendor", "github.com", "DataWorkflowServices", "dws", "config", "crd", "bases"),
			filepath.Join("..", "..", "vendor", "github.com", "NearNodeFlash", "lustre-fs-operator", "config", "crd", "bases"),
		},
		CRDInstallOptions: envtest.CRDInstallOptions{
			// This adds the conversion webhook configuration to
			// the CRDs.
			Scheme: scheme.Scheme,
		},
		//AttachControlPlaneOutput: true,
	}

	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

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
		Start webhooks
	*/

	err = (&dwsv1alpha6.Workflow{}).SetupWebhookWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&lusv1beta1.LustreFileSystem{}).SetupWebhookWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&nnfv1alpha9.NnfStorageProfile{}).SetupWebhookWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&nnfv1alpha9.NnfContainerProfile{}).SetupWebhookWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&nnfv1alpha9.NnfDataMovementProfile{}).SetupWebhookWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&nnfv1alpha9.NnfAccess{}).SetupWebhookWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&nnfv1alpha9.NnfDataMovement{}).SetupWebhookWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&nnfv1alpha9.NnfDataMovementManager{}).SetupWebhookWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&nnfv1alpha9.NnfLustreMGT{}).SetupWebhookWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&nnfv1alpha9.NnfNode{}).SetupWebhookWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&nnfv1alpha9.NnfNodeBlockStorage{}).SetupWebhookWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&nnfv1alpha9.NnfNodeECData{}).SetupWebhookWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&nnfv1alpha9.NnfNodeStorage{}).SetupWebhookWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&nnfv1alpha9.NnfPortManager{}).SetupWebhookWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&nnfv1alpha9.NnfStorage{}).SetupWebhookWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&nnfv1alpha9.NnfSystemStorage{}).SetupWebhookWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// +crdbumper:scaffold:builder

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

	err = (&dwsctrls.SystemConfigurationReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("SystemConfiguration"),
		Scheme: testEnv.Scheme,
	}).SetupWithManager(k8sManager)
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

	err = (&NnfSystemStorageReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("NnfSystemStorage"),
		Scheme: testEnv.Scheme,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	/*
		Start NNF-SOS NLC pieces
	*/

	err = (&NnfLustreMGTReconciler{
		Client:         k8sManager.GetClient(),
		Log:            ctrl.Log.WithName("controllers").WithName("NnfLustreMgt"),
		Scheme:         testEnv.Scheme,
		ControllerType: ControllerTest,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// Coordinate the startup of the NLC controllers that use EC.

	semNnfNodeDone := make(chan struct{})
	err = (&NnfNodeReconciler{
		Client:           k8sManager.GetClient(),
		Log:              ctrl.Log.WithName("controllers").WithName("NnfNode"),
		Scheme:           testEnv.Scheme,
		Events:           make(chan event.GenericEvent),
		SemaphoreForDone: semNnfNodeDone,
		Options:          &nnf.Options{},
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
		Events:            make(chan event.GenericEvent),
		SemaphoreForStart: semNnfNodeECDone,
		SemaphoreForDone:  semNnfNodeBlockStorageDone,
		Options:           &nnf.Options{},
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
