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
	corev1 "k8s.io/api/core/v1" // for core Kubernetes objects

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1" // for OpenTelemetryCollector API
	"github.com/open-telemetry/opentelemetry-operator/internal/config" // for operator's internal configuration
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming" // for naming utilities in operator
)

// Volumes function builds the volumes for the given instance, including the config map volume.
func Volumes(cfg config.Config, otelcol v1alpha1.OpenTelemetryCollector) []corev1.Volume {
	// The function sets up a ConfigMap volume for the target allocator's configuration map.
	volumes := []corev1.Volume{{
		Name: naming.TAConfigMapVolume(),
		VolumeSource: corev1.VolumeSource{
			// ConfigMap is used as the source for the volume
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: naming.TAConfigMap(otelcol)},
				// Specifies the items to populate the volume
				Items: []corev1.KeyToPath{
					{
						// Key refers to the key containing the data that will be used
						Key:  cfg.TargetAllocatorConfigMapEntry(),
						// Path is the relative path to be mounted inside the volume
						Path: cfg.TargetAllocatorConfigMapEntry(),
					}},
			},
		},
	}}

	// It then returns the list of volumes.
	return volumes
}
