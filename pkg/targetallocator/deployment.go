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
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
)

// Deployment function generates a Kubernetes deployment specification for the TargetAllocator 
// based on the provided configuration, logger, and OpenTelemetryCollector instance.
func Deployment(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) appsv1.Deployment {

    // The name of the target allocator is determined based on the OpenTelemetryCollector instance.
	name := naming.TargetAllocator(otelcol)

    // Labels are generated for the deployment.
	labels := Labels(otelcol, name)

    // The function returns a Deployment object.
	return appsv1.Deployment{

        // ObjectMeta sets the metadata for the deployment like its name, namespace and labels.
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: otelcol.Namespace,
			Labels:    labels,
		},

        // The spec of the deployment is defined.
		Spec: appsv1.DeploymentSpec{

            // The number of replicas for the deployment is determined by the OpenTelemetryCollector instance spec.
			Replicas: otelcol.Spec.TargetAllocator.Replicas,

            // A label selector is created to select pods with the generated labels.
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},

            // The template for the pods to be created as part of the deployment is defined.
			Template: corev1.PodTemplateSpec{

                // The labels and annotations for the pod are set from the OpenTelemetryCollector instance spec.
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: otelcol.Spec.PodAnnotations,
				},

                // The pod spec is defined.
				Spec: corev1.PodSpec{

                    // The service account for the pod is set from the OpenTelemetryCollector instance.
					ServiceAccountName: ServiceAccountName(otelcol),

                    // The containers for the pod are created. Here it's calling another function `Container` 
                    // which presumably builds and returns the specification for the container.
					Containers:         []corev1.Container{Container(cfg, logger, otelcol)},

                    // The volumes for the pod are created based on the provided configuration and the OpenTelemetryCollector instance.
					Volumes:            Volumes(cfg, otelcol),
				},
			},
		},
	}
}