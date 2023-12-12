// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	colfeaturegate "go.opentelemetry.io/collector/featuregate"
	"go.uber.org/zap/zapcore"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	otelv1alpha1 "github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/controllers"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
	"github.com/open-telemetry/opentelemetry-operator/internal/webhook/podmutation"
	"github.com/open-telemetry/opentelemetry-operator/pkg/autodetect"
	"github.com/open-telemetry/opentelemetry-operator/pkg/cmd"
	collectorupgrade "github.com/open-telemetry/opentelemetry-operator/pkg/collector/upgrade"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
	"github.com/open-telemetry/opentelemetry-operator/pkg/instrumentation"
	instrumentationupgrade "github.com/open-telemetry/opentelemetry-operator/pkg/instrumentation/upgrade"
	"github.com/open-telemetry/opentelemetry-operator/pkg/sidecar"
	// +kubebuilder:scaffold:imports
)

var (
	setupLog = ctrl.Log.WithName("setup")
)

// things are starting and working, but it is ignoring parts of config input; so it only works if I hack the tests to pretend this is ok
// it is ignoring configs because mgrOptions (ctrl.Options) is not being wired up with values from flags or config file
// *** when mgrOptions is created in root.go, it should have proper config values supplied
// config needs to be suppliable from flags as well (unclear if this is working)
//
// next steps:
// - finish the implementation
//  - add missing config
// - (does this need to be done) pull flags into config
// - find all the flags we have removed
// - double check and add config for those flags
// - compare everything with the tempo implementation
// - run e2e tests and add config for anything broken (e.g. liveness/readiness port)
// 	- parity on validation
// - cleanup

func main() {
	// registers any flags that underlying libraries might us
	rootCmd := cmd.NewRootCommand()
	rootCmd.SetArgs(flag.Args())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}

	rootCmdConfig := rootCmd.Context().Value(cmd.RootConfigKey{}).(cmd.RootConfig)
	ctrlConfig, mgrOptions := rootCmdConfig.CtrlConfig, rootCmdConfig.Options
	flagset := featuregate.Flags(colfeaturegate.GlobalRegistry())
	v := version.Get()

	level, err := zapcore.ParseLevel(ctrlConfig.LogLevel)
	if err != nil {
		os.Exit(1)
	}
	option := zap.Options{
		Encoder: zapcore.NewConsoleEncoder(zapcore.EncoderConfig{}),
		Level:   level,
	}

	logger := zap.New(zap.UseFlagOptions(&option))
	ctrl.SetLogger(logger)

	logger.Info("Starting the OpenTelemetry Operator",
		"opentelemetry-operator", v.Operator,
		"opentelemetry-collector", ctrlConfig.CollectorImage,
		"opentelemetry-targetallocator", ctrlConfig.TargetAllocatorImage,
		"operator-opamp-bridge", ctrlConfig.OperatorOpAMPBridgeImage,
		"auto-instrumentation-java", ctrlConfig.AutoInstrumentationJava,
		"auto-instrumentation-nodejs", ctrlConfig.AutoInstrumentationNodeJS,
		"auto-instrumentation-python", ctrlConfig.AutoInstrumentationPython,
		"auto-instrumentation-dotnet", ctrlConfig.AutoInstrumentationDotNet,
		"auto-instrumentation-go", ctrlConfig.AutoInstrumentationGo,
		"auto-instrumentation-apache-httpd", ctrlConfig.AutoInstrumentationApacheHttpd,
		"feature-gates", flagset.Lookup(featuregate.FeatureGatesFlag).Value.String(),
		"build-date", v.BuildDate,
		"go-version", v.Go,
		"go-arch", runtime.GOARCH,
		"go-os", runtime.GOOS,
		"labels-filter", ctrlConfig.LabelsFilter,
	)

	restConfig := ctrl.GetConfigOrDie()

	// builds the operator's configuration
	ad, err := autodetect.New(restConfig)
	if err != nil {
		setupLog.Error(err, "failed to setup auto-detect routine")
		os.Exit(1)
	}

	cfg := config.New(
		config.WithLogger(ctrl.Log.WithName("config")),
		config.WithVersion(v),
		config.WithCollectorImage(ctrlConfig.CollectorImage),
		config.WithTargetAllocatorImage(ctrlConfig.TargetAllocatorImage),
		config.WithOperatorOpAMPBridgeImage(ctrlConfig.OperatorOpAMPBridgeImage),
		config.WithAutoInstrumentationJavaImage(ctrlConfig.AutoInstrumentationJava),
		config.WithAutoInstrumentationNodeJSImage(ctrlConfig.AutoInstrumentationNodeJS),
		config.WithAutoInstrumentationPythonImage(ctrlConfig.AutoInstrumentationPython),
		config.WithAutoInstrumentationDotNetImage(ctrlConfig.AutoInstrumentationDotNet),
		config.WithAutoInstrumentationGoImage(ctrlConfig.AutoInstrumentationGo),
		config.WithAutoInstrumentationApacheHttpdImage(ctrlConfig.AutoInstrumentationApacheHttpd),
		config.WithAutoDetect(ad),
		config.WithLabelFilters(ctrlConfig.LabelsFilter),
	)
	err = cfg.AutoDetect()
	if err != nil {
		setupLog.Error(err, "failed to autodetect config variables")
	}

	watchNamespace, found := os.LookupEnv("WATCH_NAMESPACE")
	if found {
		setupLog.Info("watching namespace(s)", "namespaces", watchNamespace)
	} else {
		setupLog.Info("the env var WATCH_NAMESPACE isn't set, watching all namespaces")
	}

	var namespaces map[string]cache.Config
	if strings.Contains(watchNamespace, ",") {
		namespaces = map[string]cache.Config{}
		for _, ns := range strings.Split(watchNamespace, ",") {
			namespaces[ns] = cache.Config{}
		}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOptions)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	ctx := ctrl.SetupSignalHandler()
	err = addDependencies(ctx, mgr, cfg, v)
	if err != nil {
		setupLog.Error(err, "failed to add/run bootstrap dependencies to the controller manager")
		os.Exit(1)
	}

	if err = controllers.NewReconciler(controllers.Params{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("OpenTelemetryCollector"),
		Scheme:   mgr.GetScheme(),
		Config:   cfg,
		Recorder: mgr.GetEventRecorderFor("opentelemetry-operator"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OpenTelemetryCollector")
		os.Exit(1)
	}

	if err = controllers.NewOpAMPBridgeReconciler(controllers.OpAMPBridgeReconcilerParams{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("OpAMPBridge"),
		Scheme:   mgr.GetScheme(),
		Config:   cfg,
		Recorder: mgr.GetEventRecorderFor("opamp-bridge"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OpAMPBridge")
		os.Exit(1)
	}

	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = otelv1alpha1.SetupCollectorWebhook(mgr, cfg); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "OpenTelemetryCollector")
			os.Exit(1)
		}
		if err = (&otelv1alpha1.Instrumentation{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					otelv1alpha1.AnnotationDefaultAutoInstrumentationJava:        ctrlConfig.AutoInstrumentationJava,
					otelv1alpha1.AnnotationDefaultAutoInstrumentationNodeJS:      ctrlConfig.AutoInstrumentationNodeJS,
					otelv1alpha1.AnnotationDefaultAutoInstrumentationPython:      ctrlConfig.AutoInstrumentationPython,
					otelv1alpha1.AnnotationDefaultAutoInstrumentationDotNet:      ctrlConfig.AutoInstrumentationDotNet,
					otelv1alpha1.AnnotationDefaultAutoInstrumentationGo:          ctrlConfig.AutoInstrumentationGo,
					otelv1alpha1.AnnotationDefaultAutoInstrumentationApacheHttpd: ctrlConfig.AutoInstrumentationApacheHttpd,
				},
			},
		}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Instrumentation")
			os.Exit(1)
		}
		decoder := admission.NewDecoder(mgr.GetScheme())
		mgr.GetWebhookServer().Register("/mutate-v1-pod", &webhook.Admission{
			Handler: podmutation.NewWebhookHandler(cfg, ctrl.Log.WithName("pod-webhook"), decoder, mgr.GetClient(),
				[]podmutation.PodMutator{
					sidecar.NewMutator(logger, cfg, mgr.GetClient()),
					instrumentation.NewMutator(logger, mgr.GetClient(), mgr.GetEventRecorderFor("opentelemetry-operator")),
				}),
		})

		if err = otelv1alpha1.SetupOpAMPBridgeWebhook(mgr, cfg); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "OpAMPBridge")
			os.Exit(1)
		}
	} else {
		ctrl.Log.Info("Webhooks are disabled, operator is running an unsupported mode", "ENABLE_WEBHOOKS", "false")
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func addDependencies(_ context.Context, mgr ctrl.Manager, cfg config.Config, v version.Version) error {
	// adds the upgrade mechanism to be executed once the manager is ready
	err := mgr.Add(manager.RunnableFunc(func(c context.Context) error {
		up := &collectorupgrade.VersionUpgrade{
			Log:      ctrl.Log.WithName("collector-upgrade"),
			Version:  v,
			Client:   mgr.GetClient(),
			Recorder: record.NewFakeRecorder(collectorupgrade.RecordBufferSize),
		}
		return up.ManagedInstances(c)
	}))
	if err != nil {
		return fmt.Errorf("failed to upgrade OpenTelemetryCollector instances: %w", err)
	}

	// adds the upgrade mechanism to be executed once the manager is ready
	err = mgr.Add(manager.RunnableFunc(func(c context.Context) error {
		u := &instrumentationupgrade.InstrumentationUpgrade{
			Logger:                     ctrl.Log.WithName("instrumentation-upgrade"),
			DefaultAutoInstJava:        cfg.AutoInstrumentationJavaImage(),
			DefaultAutoInstNodeJS:      cfg.AutoInstrumentationNodeJSImage(),
			DefaultAutoInstPython:      cfg.AutoInstrumentationPythonImage(),
			DefaultAutoInstDotNet:      cfg.AutoInstrumentationDotNetImage(),
			DefaultAutoInstGo:          cfg.AutoInstrumentationDotNetImage(),
			DefaultAutoInstApacheHttpd: cfg.AutoInstrumentationApacheHttpdImage(),
			DefaultAutoInstNginx:       cfg.AutoInstrumentationNginxImage(),
			Client:                     mgr.GetClient(),
			Recorder:                   mgr.GetEventRecorderFor("opentelemetry-operator"),
		}
		return u.ManagedInstances(c)
	}))
	if err != nil {
		return fmt.Errorf("failed to upgrade Instrumentation instances: %w", err)
	}
	return nil
}
