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

package reconcile

// import necessary packages
import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector"
	"github.com/open-telemetry/opentelemetry-operator/pkg/targetallocator"
)

// Below line provides RBAC rules that indicate that the reconciler needs permissions
// to get, list, watch, create, update, patch, and delete Deployments
// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Deployments function reconciles the deployment(s) required for the instance in the current context.
func Deployments(ctx context.Context, params Params) error {
	// desired deployments initialized as an empty slice
	desired := []appsv1.Deployment{}
	// If the mode of the instance is 'deployment', create a deployment for the collector
	if params.Instance.Spec.Mode == "deployment" {
		desired = append(desired, collector.Deployment(params.Config, params.Log, params.Instance))
	}
	// If target allocator is enabled, create a deployment for the target allocator
	if params.Instance.Spec.TargetAllocator.Enabled {
		desired = append(desired, targetallocator.Deployment(params.Config, params.Log, params.Instance))
	}

	// Reconcile the desired deployments
	if err := expectedDeployments(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the expected deployments: %w", err)
	}

	// Delete the deployments that are no longer needed
	if err := deleteDeployments(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the deployments to be deleted: %w", err)
	}

	return nil
}

// expectedDeployments function reconciles expected deployments with current state
func expectedDeployments(ctx context.Context, params Params, expected []appsv1.Deployment) error {
	for _, obj := range expected {
		// Copy desired deployment to another variable to avoid modifying the range variable
		desired := obj
		// Set controller reference for garbage collection
		if err := controllerutil.SetControllerReference(&params.Instance, &desired, params.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference: %w", err)
		}
		// Check if deployment already exists
		existing := &appsv1.Deployment{}
		nns := types.NamespacedName{Namespace: desired.Namespace, Name: desired.Name}
		err := params.Client.Get(ctx, nns, existing)
		// If not found, create the deployment
		if err != nil && k8serrors.IsNotFound(err) {
			if clientErr := params.Client.Create(ctx, &desired); clientErr != nil {
				return fmt.Errorf("failed to create: %w", clientErr)
			}
			params.Log.V(2).Info("created", "deployment.name", desired.Name, "deployment.namespace", desired.Namespace)
			continue
		} else if err != nil {
			return fmt.Errorf("failed to get: %w", err)
		}

		// If Selector is changed, delete and re-create the deployment as Selector field is immutable
		if !apiequality.Semantic.DeepEqual(desired.Spec.Selector, existing.Spec.Selector) {
			params.Log.V(2).Info("Spec.Selector change detected, trying to delete, the new collector deployment will be created in the next reconcile cycle ", "deployment.name", existing.Name, "deployment.namespace", existing.Namespace)

			if err := params.Client.Delete(ctx, existing); err != nil {
				return fmt.Errorf("failed to delete deployment: %w", err)
			}
			continue
		}

		// If the deployment already exists, merge the existing and desired ones
		// First, create a deep copy of the existing deployment to avoid modifying the original one
		updated := existing.DeepCopy()
		if updated.Annotations == nil {
			updated.Annotations = map[string]string{}
		}
		if updated.Labels == nil {
			updated.Labels = map[string]string{}
		}

		// Update the spec and owner references from the desired deployment
		updated.Spec = desired.Spec
		updated.ObjectMeta.OwnerReferences = desired.ObjectMeta.OwnerReferences

		// Merge annotations and labels from the desired deployment
		for k, v := range desired.ObjectMeta.Annotations {
			updated.ObjectMeta.Annotations[k] = v
		}
		for k, v := range desired.ObjectMeta.Labels {
			updated.ObjectMeta.Labels[k] = v
		}

		// Create a patch from the existing deployment and apply it to the updated one
		patch := client.MergeFrom(existing)

		if err := params.Client.Patch(ctx, updated, patch); err != nil {
			return fmt.Errorf("failed to apply changes: %w", err)
		}

		// Log the update information
		params.Log.V(2).Info("applied", "deployment.name", desired.Name, "deployment.namespace", desired.Namespace)
	}

	return nil
}

// deleteDeployments function deletes deployments that are not expected to exist
func deleteDeployments(ctx context.Context, params Params, expected []appsv1.Deployment) error {
	// List options to find deployments to delete
	opts := []client.ListOption{
		client.InNamespace(params.Instance.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.Instance.Namespace, params.Instance.Name),
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}),
	}
	list := &appsv1.DeploymentList{}
	if err := params.Client.List(ctx, list, opts...); err != nil {
		return fmt.Errorf("failed to list: %w", err)
	}

	// Iterate through the list and delete unwanted deployments
	for i := range list.Items {
		existing := list.Items[i]
		del := true
		for _, keep := range expected {
			if keep.Name == existing.Name && keep.Namespace == existing.Namespace {
				del = false
				break
			}
		}

		if del {
			if err := params.Client.Delete(ctx, &existing); err != nil {
				return fmt.Errorf("failed to delete: %w", err)
			}
			params.Log.V(2).Info("deleted", "deployment.name", existing.Name, "deployment.namespace", existing.Namespace)
		}
	}

	return nil
}

// currentReplicasWithHPA calculates deployment replicas if Horizontal Pod Autoscaler (HPA) is enabled.
// This function ensures the current replica count is within the range specified by the HPA configuration.
// The function takes in the spec containing HPA configuration and the current replica count as input.
// It returns an updated replica count that falls within the HPA range.
func currentReplicasWithHPA(spec v1alpha1.OpenTelemetryCollectorSpec, curr int32) int32 {
	// If the current replica count is less than the minimum specified in the HPA spec,
	// set the replica count to the minimum.
	if curr < *spec.Replicas {
		return *spec.Replicas
	}

	// If the current replica count is greater than the maximum specified in the HPA spec,
	// set the replica count to the maximum.
	if curr > *spec.MaxReplicas {
		return *spec.MaxReplicas
	}

	// If the current replica count is within the HPA range, leave it as is.
	return curr
}