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

package builder

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/log"
)

// ********************************
//
// Pod builder
//
// ********************************

type podBuilder struct {
	log       log.Logger
	client    client.Client
	running   int32
	namespace string
}

// NewPodBuilder --
func NewPodBuilder(c client.Client, namespace string) Builder {
	m := podBuilder{
		log:       log.WithName("pod builder"),
		client:    c,
		namespace: namespace,
	}

	return &m
}

func (b *podBuilder) IsBuilding(object metav1.ObjectMeta) bool {
	pod := &corev1.Pod{}
	err := b.client.Get(context.TODO(), types.NamespacedName{Namespace: object.Namespace, Name: b.buildPodName(object)}, pod)
	if err == nil && pod.Status.Phase != corev1.PodSucceeded && pod.Status.Phase != corev1.PodFailed {
		return true
	}
	return false
}

// Submit --
func (b *podBuilder) Submit(request Request, handler func(*Result)) {
	b.process(request, handler)
}

func (b *podBuilder) Close() {

}

// ********************************
//
// Helpers
//
// ********************************

func (b *podBuilder) buildPodName(object metav1.ObjectMeta) string {
	return "camel-k-" + object.Name + "-builder"
}

func (b *podBuilder) process(request Request, handler func(*Result)) {
	result := Result{
		Builder: b,
		Request: request,
		Task: Task{
			StartedAt: time.Now(),
		},
	}

	steps := make([]string, len(request.Steps))
	for i, s := range request.Steps {
		steps[i] = s.ID()
	}

	build := &v1alpha1.Build{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "camel.apache.org/v1alpha1",
			Kind:       "Build",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: b.namespace,
			Name:      request.Meta.Name,
		},
		Spec: v1alpha1.BuildSpec{
			Meta:           request.Meta,
			CamelVersion:   request.Catalog.Version,
			RuntimeVersion: request.RuntimeVersion,
			Image:          request.Image,
			Platform:       request.Platform,
			Dependencies:   request.Dependencies,
			Repositories:   request.Repositories,
			Steps:          steps,
			//Resources:    request.Resources,
		},
	}

	err := b.client.Create(request.C, build)
	if err != nil {
		result.Error = err
		result.Status = v1alpha1.BuildError
		handler(&result)
		return
	}

	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: b.namespace,
			Name:      b.buildPodName(request.Meta),
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "camel-k-operator",
			Containers: []corev1.Container{
				{
					Name:  "builder",
					Image: "docker.io/apache/camel-k:0.3.2-SNAPSHOT",
					Args: []string{
						"camel-k-builder",
						request.Meta.Namespace,
						request.Meta.Name,
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	err = b.client.Delete(request.C, &pod)
	if err != nil && !apierrors.IsNotFound(err) {
		result.Error = err
		result.Status = v1alpha1.BuildError
		handler(&result)
		return
	}

	err = b.client.Create(request.C, &pod)
	if err != nil {
		result.Error = err
		result.Status = v1alpha1.BuildError
		handler(&result)
		return
	}

	err = kubernetes.WaitCondition(request.C, b.client, &pod, func(obj interface{}) (bool, error) {
		if val, ok := obj.(*corev1.Pod); ok {
			switch phase := val.Status.Phase; phase {
			case corev1.PodPending:
				result.Status = v1alpha1.BuildScheduled
				handler(&result)
			case corev1.PodRunning:
				result.Status = v1alpha1.BuildStarted
				handler(&result)
			case corev1.PodSucceeded:
				return true, nil
			case corev1.PodFailed:
				return false, fmt.Errorf("builder failed: %s", val.Status.Message)
			}
		}
		return false, nil
	}, 10*time.Minute)

	if err != nil {
		result.Error = err
		result.Status = v1alpha1.BuildError
		handler(&result)
		return
	}

	result.Task.CompletedAt = time.Now()

	if result.Status != v1alpha1.BuildInterrupted {
		err = b.client.Get(request.C, types.NamespacedName{Namespace: build.GetNamespace(), Name: build.GetName()}, build)
		if err != nil {
			result.Error = err
			result.Status = v1alpha1.BuildError
			handler(&result)
			return
		}

		result.Status = build.Status.Phase
		result.BaseImage = build.Status.BaseImage
		result.Image = build.Status.Image
		result.PublicImage = build.Status.PublicImage
		result.Error = fmt.Errorf(build.Status.Error)
		result.Artifacts = make([]v1alpha1.Artifact, 0, len(build.Status.Artifacts))
		result.Artifacts = append(result.Artifacts, build.Status.Artifacts...)

		//b.log.Infof("builder request %s executed in %f seconds", request.Meta.Name, result.Task.Elapsed().Seconds())
		b.log.Infof("dependencies: %s", request.Dependencies)
		b.log.Infof("artifacts: %s", ArtifactIDs(result.Artifacts))
		//b.log.Infof("artifacts selected: %s", ArtifactIDs(c.SelectedArtifacts))
		b.log.Infof("requested image: %s", request.Image)
		b.log.Infof("base image: %s", result.BaseImage)
		b.log.Infof("resolved image: %s", result.Image)
		b.log.Infof("resolved public image: %s", result.PublicImage)
	} else {
		b.log.Infof("builder request %s interrupted after %f seconds", request.Meta.Name, result.Task.Elapsed().Seconds())
	}

	handler(&result)
}
