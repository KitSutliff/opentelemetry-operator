package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1alpha1 "sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
)

type TlsConfig struct {
	MinVersion   string
	CipherSuites []string
}

// ProjectConfig is the Schema for the projectconfigs API.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type ProjectConfig struct {
	metav1.TypeMeta `json:",inline"`

	// The address the metric endpoint binds to
	MetricsAddr string `json:"metricsAddr,omitempty"`

	// The address the probe endpoint binds to
	ProbeAddr string `json:"healthProbeAddr,omitempty"`

	// The address to expose the pprof server. Default is empty string which disables the pprof server
	PprofAddr string `json:"pprofAddr,omitempty"`

	// Enable leader election for controller manager
	// Enabling this will ensure there is only one active controller manager
	EnableLeaderElection bool `json:"enableLeaderElection,omitempty"`

	// The default OpenTelemetry collector image. This image is used when no image is specified in the CustomResource
	CollectorImage string `json:"collectorImage,omitempty"`

	// The default OpenTelemetry target allocator image. This image is used when no image is specified in the CustomResource
	TargetAllocatorImage string `json:"targetAllocatorImage,omitempty"`

	// The default OpenTelemetry Operator OpAMP Bridge image. This image is used when no image is specified in the CustomResource
	OperatorOpAMPBridgeImage string `json:"operatorOpAMPBridgeImage,omitempty"`

	// The default OpenTelemetry Java instrumentation image. This image is used when no image is specified in the CustomResource
	AutoInstrumentationJava string `json:"autoInstrumentationJavaImage,omitempty"`

	// The default OpenTelemetry NodeJS instrumentation image. This image is used when no image is specified in the CustomResource
	AutoInstrumentationNodeJS string `json:"autoInstrumentationNodeJSImage,omitempty"`

	// The default OpenTelemetry Python instrumentation image. This image is used when no image is specified in the CustomResource
	AutoInstrumentationPython string `json:"autoInstrumentationPythonImage,omitempty"`

	// The default OpenTelemetry DotNet instrumentation image. This image is used when no image is specified in the CustomResource
	AutoInstrumentationDotNet string `json:"autoInstrumentationDotNetImage,omitempty"`

	// The default OpenTelemetry Go instrumentation image. This image is used when no image is specified in the CustomResource
	AutoInstrumentationGo string `json:"autoInstrumentationGoImage,omitempty"`

	// The default OpenTelemetry ApacheHttpd instrumentation image. This image is used when no image is specified in the CustomResource
	AutoInstrumentationApacheHttpd string `json:"autoInstrumentationApacheHttpdImage,omitempty"`

	// Labels to filter away from propagating onto deploys
	LabelsFilter []string `json:"labels,omitempty"`

	// The port the webhook endpoint binds to
	WebhookPort int `json:"webhookPort,omitempty"`

	// Minimum TLS version supported. Value must match version names from https://golang.org/pkg/crypto/tls/#pkg-constants
	TlsOpt TlsConfig `json:"tlsOpt,omitempty"`

	// The log level for the operator
	LogLevel string `json:"logLevel,omitempty"`
}

// Validate validates the controller configuration (ProjectConfig).
func (c *ProjectConfig) Validate() error {
	return nil
}

func (p *ProjectConfig) Complete() (v1alpha1.ControllerManagerConfigurationSpec, error) {
	return v1alpha1.ControllerManagerConfigurationSpec{}, nil
}

func init() {
	cfg := &ProjectConfig{}
	SchemeBuilder.Register(cfg)
}
