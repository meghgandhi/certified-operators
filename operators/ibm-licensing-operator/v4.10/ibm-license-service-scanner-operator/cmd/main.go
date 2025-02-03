/*
Copyright 2024 IBM Corporation

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

package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/go-logr/zapr"
	uzap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	odlm "github.com/IBM/operand-deployment-lifecycle-manager/api/v1alpha1"
	operatorv1 "github.ibm.com/cloud-license-reporter/ibm-license-service-scanner-operator/api/v1"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-scanner-operator/internal/controller"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-scanner-operator/version"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	//+kubebuilder:scaffold:imports
)

const (
	operandRequestCRDName = "operandrequests.operator.ibm.com"
)

var (
	scheme   = pkgruntime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(odlm.AddToScheme(scheme))
	utilruntime.Must(operatorv1.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func getLogger() *uzap.Logger {
	cfg := uzap.Config{
		Development:      true,
		Encoding:         "console",
		Level:            uzap.NewAtomicLevelAt(zapcore.DebugLevel),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey: "message",

			LevelKey:    "level",
			EncodeLevel: zapcore.CapitalLevelEncoder,

			TimeKey:    "time",
			EncodeTime: zapcore.ISO8601TimeEncoder,
		},
	}

	logger, err := cfg.Build()
	if err != nil {
		panic("Logger config error")
	}

	return logger
}

/*
getWatchNamespace returns the Namespace the operator should be watching for changes.

An empty env var value means the operator is running with cluster scope.
*/
func getOperatorNamespace() (string, error) {
	const operatorNamespaceEnvVar = "OPERATOR_NAMESPACE"

	ns, found := os.LookupEnv(operatorNamespaceEnvVar)
	if !found || ns == "" {
		return "", fmt.Errorf("%s env var must be set and not empty", operatorNamespaceEnvVar)
	}

	return ns, nil
}

func main() {
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zapr.NewLogger(getLogger()))

	logBuildInfo()

	// Get operator namespace
	namespace, err := getOperatorNamespace()
	if err != nil {
		setupLog.Error(err, "Couldn't retrieve operator namespace")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "45a871b6.ibm.com",
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				namespace: {},
			},
		},
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Define a variable to pass data to the main reconciliation loop and attempt auto-connection with License Service
	enableAutoConnect := true

	// Try to get the operand request CRD to see if the automatic connection to License Service should be enabled
	clientSet, err := clientset.NewForConfig(mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "Failed to check if the operand request CRD exists")
		enableAutoConnect = false
	} else {
		if _, err = clientSet.ApiextensionsV1().CustomResourceDefinitions().Get(
			context.Background(),
			operandRequestCRDName,
			metav1.GetOptions{},
		); err != nil {
			if client.IgnoreNotFound(err) != nil {
				setupLog.Error(err, "Failed to check if the operand request CRD exists")
			}
			setupLog.Info("Operand request CRD not found")
			enableAutoConnect = false
		}
	}

	if err = (&controller.ScannerReconciler{
		Client:            mgr.GetClient(),
		Scheme:            mgr.GetScheme(),
		Recorder:          mgr.GetEventRecorderFor("IBMLicenseServiceScanner"),
		EnableAutoConnect: enableAutoConnect,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "IBMLicenseServiceScanner")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info(fmt.Sprintf("Starting manager in namespace: %s", namespace))
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

/*
Finds and prints the following:
  - operator version
  - go version
  - operating system and architecture
  - build date (date on which used image was built)
  - build commit (git commit id of source code used to build image)

Image build date and release files are expected to be in /opt/scanner.
If they don't exist, logger informs about files missing an proceeds to the next instruction.
*/
func logBuildInfo() {
	const (
		BuildDateFile   = "/opt/scanner/IMAGE_BUILDDATE"
		BuildCommitFile = "/opt/scanner/IMAGE_RELEASE"
	)

	setupLog.Info(fmt.Sprintf("Version: %s", version.Version))

	buildDate, err := os.ReadFile(BuildDateFile)
	if err == nil {
		setupLog.Info(fmt.Sprintf("Build date: %s", string(buildDate)))
	} else if os.IsNotExist(err) {
		setupLog.Info("Build date file not found")
	} else {
		setupLog.Error(err, "Failed to read build date file")
	}

	buildCommit, err := os.ReadFile(BuildCommitFile)
	if err == nil {
		setupLog.Info(fmt.Sprintf("Build commit: %s", string(buildCommit)))
	} else if os.IsNotExist(err) {
		setupLog.Info("Build commit file not found")
	} else {
		setupLog.Error(err, "Failed to read build commit file")
	}

	setupLog.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	setupLog.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}
