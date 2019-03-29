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

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/pkg/errors"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

// NewScheduleAction creates a new schedule action
func NewScheduleAction() Action {
	return &scheduleAction{}
}

type scheduleAction struct {
	baseAction
}

// Name returns a common name of the action
func (action *scheduleAction) Name() string {
	return "schedule"
}

// CanHandle tells whether this action can handle the build
func (action *scheduleAction) CanHandle(build *v1alpha1.Build) bool {
	return build.Status.Phase == v1alpha1.BuildPhaseScheduling
}

// Handle handles the builds
func (action *scheduleAction) Handle(ctx context.Context, build *v1alpha1.Build) error {
	pod := newBuildPod(build)

	// Set the Build instance as the owner and controller
	if err := controllerutil.SetControllerReference(build, pod, action.client.GetScheme()); err != nil {
		return err
	}

	err := action.client.Delete(ctx, pod)
	if err != nil && !k8serrors.IsNotFound(err) {
		return errors.Wrap(err, "cannot delete build pod")
	}

	err = action.client.Create(ctx, pod)
	if err != nil {
		return errors.Wrap(err, "cannot create build pod")
	}

	target := build.DeepCopy()
	target.Status.Phase = v1alpha1.BuildPhasePending
	action.L.Info("Build state transition", "phase", target.Status.Phase)

	return action.client.Status().Update(ctx, target)
}

func newBuildPod(build *v1alpha1.Build) *corev1.Pod {
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: build.Namespace,
			Name:      buildPodName(build.Spec.Meta),
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "camel-k-operator",
			Containers: []corev1.Container{
				{
					Name:  "builder",
					Image: "docker.io/apache/camel-k:0.3.2-SNAPSHOT",
					Args: []string{
						"camel-k-builder",
						build.Namespace,
						build.Name,
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	return pod
}

func buildPodName(object metav1.ObjectMeta) string {
	return "camel-k-" + object.Name + "-builder"
}
