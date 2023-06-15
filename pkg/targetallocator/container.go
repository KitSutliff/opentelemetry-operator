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

package targetallocator

import (
	"github.com/go-logr/logr" // for logging interface
	corev1 "k8s.io/api/core/v1" // for core Kubernetes objects

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1" // for OpenTelemetryCollector API
	"github.com/open-telemetry/opentelemetry-operator/internal/config" // for operator's internal configuration
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming" // for naming utilities in operator
)

// Container function builds a container for the given TargetAllocator.
func Container(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) corev1.Container {
	// The function gets the image for the target allocator from the instance's spec.
	// If the image is not specified, it uses the default image from the configuration.
	image := otelcol.Spec.TargetAllocator.Image
	if len(image) == 0 {
		image = cfg.TargetAllocatorImage()
	}

	// The function sets up a volume mount for the target allocator's configuration map.
	volumeMounts := []corev1.VolumeMount{{
		Name:      naming.TAConfigMapVolume(),
		MountPath: "/conf",
	}}

	// It initializes an empty list of environment variables.
	envVars := []corev1.EnvVar{}

	// It then adds an environment variable to the list that gets the namespace of the instance from the field reference.
	envVars = append(envVars, corev1.EnvVar{
		Name: "OTELCOL_NAMESPACE",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.namespace",
			},
		},
	})
	//##########################################################################################
	//loop over otelcol.Spec.TargetAllocator.EnvVars and add them to envVars

	// It initializes an empty list of arguments.
	var args []string
	// If the PrometheusCR feature is enabled in the instance's spec, it adds an argument to enable the Prometheus CR watcher.
	if otelcol.Spec.TargetAllocator.PrometheusCR.Enabled {
		args = append(args, "--enable-prometheus-cr-watcher")
	}

	// It then constructs and returns a container object with the obtained image, environment variables, volume mounts, resources, and arguments.
	return corev1.Container{
		Name:         naming.TAContainer(),
		Image:        image,
		Env:          envVars,
		VolumeMounts: volumeMounts,
		Resources:    otelcol.Spec.TargetAllocator.Resources,
		Args:         args,
	}
}