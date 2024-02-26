/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package build

import (
	"context"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/openshift"
)

const (
	builderDir    = "/builder"
	builderVolume = "camel-k-builder"
)

func newBuildPod(ctx context.Context, client client.Client, build *v1.Build) *corev1.Pod {
	var ugfid int64 = 1001
	podSecurityContext := &corev1.PodSecurityContext{
		RunAsUser:  &ugfid,
		RunAsGroup: &ugfid,
		FSGroup:    &ugfid,
	}
	for _, task := range build.Spec.Tasks {
		// get pod security context from security context constraint configuration in namespace
		if task.S2i != nil {
			podSecurityContextConstrained, _ := openshift.GetOpenshiftPodSecurityContextRestricted(ctx, client, build.BuilderPodNamespace())
			if podSecurityContextConstrained != nil {
				podSecurityContext = podSecurityContextConstrained
			}
		}
	}
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: build.BuilderPodNamespace(),
			Name:      buildPodName(build),
			Labels: map[string]string{
				"camel.apache.org/build":     build.Name,
				"camel.apache.org/component": "builder",
			},
			Annotations: build.BuilderConfiguration().Annotations,
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: platform.BuilderServiceAccount,
			RestartPolicy:      corev1.RestartPolicyNever,
			SecurityContext:    podSecurityContext,
			NodeSelector:       build.BuilderConfiguration().NodeSelector,
		},
	}

	pod.Labels = kubernetes.MergeCamelCreatorLabels(build.Labels, pod.Labels)

	for _, task := range build.Spec.Tasks {
		switch {
		// Builder task
		case task.Builder != nil:
			addBuildTaskToPod(ctx, client, build, task.Builder.Name, pod)
		// Custom task
		case task.Custom != nil:
			addCustomTaskToPod(build, task.Custom, pod)
		// Package task
		// It's a type of builder task, we can reuse the same type
		case task.Package != nil:
			addBuildTaskToPod(ctx, client, build, task.Package.Name, pod)
		case task.S2i != nil:
			addBuildTaskToPod(ctx, client, build, task.S2i.Name, pod)
		case task.Spectrum != nil:
			addBuildTaskToPod(ctx, client, build, task.Spectrum.Name, pod)
		case task.Jib != nil:
			addBuildTaskToPod(ctx, client, build, task.Jib.Name, pod)
		}
	}

	// Make sure there is one container defined
	pod.Spec.Containers = pod.Spec.InitContainers[len(pod.Spec.InitContainers)-1 : len(pod.Spec.InitContainers)]
	pod.Spec.InitContainers = pod.Spec.InitContainers[:len(pod.Spec.InitContainers)-1]

	return pod
}

func configureResources(taskName string, build *v1.Build, container *corev1.Container) {
	conf := build.TaskConfiguration(taskName)
	requestsList := container.Resources.Requests
	limitsList := container.Resources.Limits
	var err error
	if requestsList == nil {
		requestsList = make(corev1.ResourceList)
	}
	if limitsList == nil {
		limitsList = make(corev1.ResourceList)
	}

	requestsList, err = kubernetes.ConfigureResource(conf.RequestCPU, requestsList, corev1.ResourceCPU)
	if err != nil {
		Log.WithValues("request-namespace", build.Namespace, "request-name", build.Name).
			Errorf(err, "Could not configure builder resource cpu, leaving default value")
	}
	requestsList, err = kubernetes.ConfigureResource(conf.RequestMemory, requestsList, corev1.ResourceMemory)
	if err != nil {
		Log.WithValues("request-namespace", build.Namespace, "request-name", build.Name).
			Errorf(err, "Could not configure builder resource memory, leaving default value")
	}
	limitsList, err = kubernetes.ConfigureResource(conf.LimitCPU, limitsList, corev1.ResourceCPU)
	if err != nil {
		Log.WithValues("request-namespace", build.Namespace, "request-name", build.Name).
			Errorf(err, "Could not configure builder limit cpu, leaving default value")
	}
	limitsList, err = kubernetes.ConfigureResource(conf.LimitMemory, limitsList, corev1.ResourceMemory)
	if err != nil {
		Log.WithValues("request-namespace", build.Namespace, "request-name", build.Name).
			Errorf(err, "Could not configure builder limit memory, leaving default value")
	}

	container.Resources.Requests = requestsList
	container.Resources.Limits = limitsList
}

func deleteBuilderPod(ctx context.Context, c ctrl.Writer, build *v1.Build) error {
	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: build.BuilderPodNamespace(),
			Name:      buildPodName(build),
		},
	}

	err := c.Delete(ctx, &pod)
	if err != nil && k8serrors.IsNotFound(err) {
		return nil
	}

	return err
}

func getBuilderPod(ctx context.Context, c ctrl.Reader, build *v1.Build) (*corev1.Pod, error) {
	pod := corev1.Pod{}
	err := c.Get(ctx, ctrl.ObjectKey{Namespace: build.BuilderPodNamespace(), Name: buildPodName(build)}, &pod)
	if err != nil && k8serrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &pod, nil
}

func buildPodName(build *v1.Build) string {
	return "camel-k-" + build.Name + "-builder"
}

func addBuildTaskToPod(ctx context.Context, client client.Client, build *v1.Build, taskName string, pod *corev1.Pod) {
	if !hasVolume(pod, builderVolume) {
		pod.Spec.Volumes = append(pod.Spec.Volumes,
			// EmptyDir volume used to share the build state across tasks
			corev1.Volume{
				Name: builderVolume,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		)
	}

	var envVars = proxyFromEnvironment()
	envVars = append(envVars,
		corev1.EnvVar{
			Name:  "HOME",
			Value: filepath.Join(builderDir, build.Name),
		},
	)

	container := corev1.Container{
		Name:            taskName,
		Image:           build.BuilderConfiguration().ToolImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command: []string{
			"kamel",
			"builder",
			"--namespace",
			build.Namespace,
			"--build-name",
			build.Name,
			"--task-name",
			taskName,
		},
		WorkingDir: filepath.Join(builderDir, build.Name),
		Env:        envVars,
	}

	// get security context from security context constraint configuration in namespace
	if taskName == "s2i" {
		securityContextConstrained, _ := openshift.GetOpenshiftSecurityContextRestricted(ctx, client, build.BuilderPodNamespace())
		if securityContextConstrained != nil {
			container.SecurityContext = securityContextConstrained
		}
	}

	configureResources(taskName, build, &container)
	addContainerToPod(build, container, pod)
}

func addCustomTaskToPod(build *v1.Build, task *v1.UserTask, pod *corev1.Pod) {
	container := corev1.Container{
		Name:            task.Name,
		Image:           task.ContainerImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         task.ContainerCommands,
		WorkingDir:      filepath.Join(builderDir, build.Name),
		Env:             proxyFromEnvironment(),
	}
	container.Env = append(container.Env, corev1.EnvVar{Name: "INTEGRATION_KIT_IMAGE", Value: task.PublishingImage})
	if task.ContainerUserID != nil {
		container.SecurityContext = &corev1.SecurityContext{
			RunAsUser:  task.ContainerUserID,
			RunAsGroup: task.ContainerUserID,
		}
	}

	configureResources(task.Name, build, &container)
	addContainerToPod(build, container, pod)
}

func addContainerToPod(build *v1.Build, container corev1.Container, pod *corev1.Pod) {
	if hasVolume(pod, builderVolume) {
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      builderVolume,
			MountPath: filepath.Join(builderDir, build.Name),
		})
	}

	pod.Spec.InitContainers = append(pod.Spec.InitContainers, container)
}

func hasVolume(pod *corev1.Pod, name string) bool {
	for _, volume := range pod.Spec.Volumes {
		if volume.Name == name {
			return true
		}
	}
	return false
}

func proxyFromEnvironment() []corev1.EnvVar {
	var envVars []corev1.EnvVar

	if httpProxy, ok := os.LookupEnv("HTTP_PROXY"); ok {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "HTTP_PROXY",
			Value: httpProxy,
		})
	}

	if httpsProxy, ok := os.LookupEnv("HTTPS_PROXY"); ok {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "HTTPS_PROXY",
			Value: httpsProxy,
		})
	}

	if noProxy, ok := os.LookupEnv("NO_PROXY"); ok {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "NO_PROXY",
			Value: noProxy,
		})
	}

	return envVars
}
