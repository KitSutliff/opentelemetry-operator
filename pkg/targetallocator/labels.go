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
	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
)

// Labels function returns a map with labels common to all TargetAllocator objects that are part of a managed OpenTelemetryCollector.
func Labels(instance v1alpha1.OpenTelemetryCollector, name string) map[string]string {

    // A new map is created to avoid modifying the instance's labels.
	base := map[string]string{}
	
	// If the instance has any labels, they are copied to the new map.
	if nil != instance.Labels {
		for k, v := range instance.Labels {
			base[k] = v
		}
	}

	// The following labels are added to the map:

    // This label indicates that the object is managed by the OpenTelemetry operator.
	base["app.kubernetes.io/managed-by"] = "opentelemetry-operator"
	
    // This label is a unique identifier for the object, created by combining the instance's namespace and name.
	base["app.kubernetes.io/instance"] = naming.Truncate("%s.%s", 63, instance.Namespace, instance.Name)
	
    // This label indicates that the object is part of the OpenTelemetry ecosystem.
	base["app.kubernetes.io/part-of"] = "opentelemetry"
	
    // This label indicates that the object is an OpenTelemetry target allocator.
	base["app.kubernetes.io/component"] = "opentelemetry-targetallocator"

	// If the "app.kubernetes.io/name" label is not set, the name argument is used as its value.
	if _, ok := base["app.kubernetes.io/name"]; !ok {
		base["app.kubernetes.io/name"] = name
	}

	// The function returns the map with the labels.
	return base
}