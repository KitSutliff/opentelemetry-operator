package cmd

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	otelv1alpha1 "github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
	routev1 "github.com/openshift/api/route/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/spf13/cobra"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	scheme = k8sruntime.NewScheme()
)

type RootConfig struct {
	Options    ctrl.Options
	CtrlConfig v1alpha1.ProjectConfig
}

// stays empty
type RootConfigKey struct {
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(otelv1alpha1.AddToScheme(scheme))
	utilruntime.Must(routev1.AddToScheme(scheme))
	utilruntime.Must(monitoringv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func readConfig(cmd *cobra.Command, configFile string, v version.Version) error {
	// default controller configuration
	ctrlConfig := v1alpha1.ProjectConfig{
		MetricsAddr:                    ":8080",
		ProbeAddr:                      ":8081",
		PprofAddr:                      "",
		EnableLeaderElection:           false,
		CollectorImage:                 fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector:%s", v.OpenTelemetryCollector),
		TargetAllocatorImage:           fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/target-allocator:%s", v.TargetAllocator),
		OperatorOpAMPBridgeImage:       fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/operator-opamp-bridge:%s", v.OperatorOpAMPBridge),
		AutoInstrumentationJava:        fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:%s", v.AutoInstrumentationJava),
		AutoInstrumentationNodeJS:      fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-nodejs:%s", v.AutoInstrumentationNodeJS),
		AutoInstrumentationPython:      fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:%s", v.AutoInstrumentationPython),
		AutoInstrumentationDotNet:      fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-dotnet:%s", v.AutoInstrumentationDotNet),
		AutoInstrumentationGo:          fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-go-instrumentation/autoinstrumentation-go:%s", v.AutoInstrumentationGo),
		AutoInstrumentationApacheHttpd: fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-apache-httpd:%s", v.AutoInstrumentationApacheHttpd),
		LabelsFilter:                   []string{},
		WebhookPort:                    9443,
		TlsOpt: v1alpha1.TlsConfig{
			MinVersion:   "VersionTLS12",
			CipherSuites: nil,
		},
	}

	var err error

	options := ctrl.Options{Scheme: scheme}
	if configFile != "" {
		options, err = options.AndFrom(ctrl.ConfigFile().AtPath(configFile).OfKind(&ctrlConfig))
		if err != nil {
			return fmt.Errorf("unable to load the config file: %w", err)
		}
	}

	// this is not printing
	ctx := context.WithValue(cmd.Context(), RootConfigKey{}, RootConfig{options, ctrlConfig})
	fmt.Printf("ctx: %v", ctx)
	cmd.SetContext(ctx)
	return nil
}

// NewRootCommand creates a new cobra root command.
func NewRootCommand() *cobra.Command {
	var configFile string

	rootCmd := &cobra.Command{
		Use:          "opentelemetry-operator",
		SilenceUsage: true,
	}
	rootCmd.SetContext(context.Background())
	readConfig(rootCmd, configFile, version.Get())
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "",
		"The controller will load its initial configuration from this file. "+
			"Omit this flag to use the default configuration values. "+
			"Command-line flags override configuration from this file.")

	return rootCmd
}
