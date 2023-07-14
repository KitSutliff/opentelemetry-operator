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

package instrumentation

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
	envJavaToolsOptions = "JAVA_TOOL_OPTIONS"
	javaJVMArgument     = " -javaagent:/otel-auto-instrumentation/javaagent.jar"
)

//add the following to the operator

// 1) When the operator receives a pod we will check for config maps that reference the given pod
// 2) If a config map is present operator will look for JAVA_TOOL_OPTS env var
// 3) I there is a JAVA_TOOL_OPTS env var in the ConfigMap, that will be used to populate the JAVA_TOOL_OPTS env var in the container

func injectJavaagent(javaSpec v1alpha1.Java, pod corev1.Pod, index int) (corev1.Pod, error) {
	// caller checks if there is at least one container.
	container := &pod.Spec.Containers[index]

	err := validateContainerEnv(container.Env, envJavaToolsOptions)
	if err != nil {
		return pod, err
	}

	// inject Java instrumentation spec env vars.
	for _, env := range javaSpec.Env {
		idx := getIndexOfEnv(container.Env, env.Name)
		if idx == -1 {
			container.Env = append(container.Env, env)
		}
	}
	//##############################################################################################################
	idx := getIndexOfEnvFrom(container.EnvFrom, envJavaToolsOptions)
	if idx != -1 {
		name := container.EnvFrom[idx].ConfigMapRef.Marshal()
		config, err := rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
		// creates the clientset
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}

		// Check for config maps that reference the given pod
		configMap, err := clientset.CoreV1().ConfigMaps(pod.Namespace).Get(context.TODO(), name, v1.GetOptions{})
		if err != nil {
			return pod, err
		}

		container.Env[idx].Value = configMap.Data[] +javaJVMArgument
	} else {

	}

	idx := getIndexOfEnv(container.Env, envJavaToolsOptions)
	if idx == -1 {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  envJavaToolsOptions,
			Value: javaJVMArgument,
		})
	} else {
		container.Env[idx].Value = container.Env[idx].Value + javaJVMArgument
	}

	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      volumeName,
		MountPath: "/otel-auto-instrumentation",
	})

	// We just inject Volumes and init containers for the first processed container.
	if isInitContainerMissing(pod) {
		pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			}})

		pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{
			Name:      initContainerName,
			Image:     javaSpec.Image,
			Command:   []string{"cp", "/javaagent.jar", "/otel-auto-instrumentation/javaagent.jar"},
			Resources: javaSpec.Resources,
			VolumeMounts: []corev1.VolumeMount{{
				Name:      volumeName,
				MountPath: "/otel-auto-instrumentation",
			}},
		})
	}
	return pod, err
}
