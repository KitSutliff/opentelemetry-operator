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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1" // for metadata of Kubernetes objects

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1" // for OpenTelemetryCollector API
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming" // for naming utilities in operator
)

// ServiceAccountName returns the name of the existing or self-provisioned service account to use for the given instance.
func ServiceAccountName(instance v1alpha1.OpenTelemetryCollector) string {
	// If the service account is not specified in the target allocator's specification, 
	// the function returns a service account name generated by the naming utility.
	if len(instance.Spec.TargetAllocator.ServiceAccount) == 0 {
		return naming.ServiceAccount(instance)
	}

	// If a service account is specified in the target allocator's specification, 
	// the function returns that service account.
	return instance.Spec.TargetAllocator.ServiceAccount
}

// ServiceAccount function returns the service account object for the given instance.
func ServiceAccount(otelcol v1alpha1.OpenTelemetryCollector) corev1.ServiceAccount {
	// The function gets the service account name for the target allocator.
	name := naming.TargetAllocatorServiceAccount(otelcol)
	// It also gets the labels for the target allocator.
	labels := Labels(otelcol, name)

	// It then constructs and returns a service account object with the obtained name, labels, and other details from the instance.
	return corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   otelcol.Namespace,
			Labels:      labels,
			Annotations: otelcol.Annotations,
		},
	}
}
