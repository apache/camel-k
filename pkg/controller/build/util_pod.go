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
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

// deleteBuilderPod --
func deleteBuilderPod(ctx context.Context, client k8sclient.Writer, build *v1alpha1.Build) error {
	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: build.Namespace,
			Name:      buildPodName(build.Spec.Meta),
			Labels: map[string]string{
				"camel.apache.org/build": build.Name,
			},
		},
	}

	err := client.Delete(ctx, &pod)
	if err != nil && k8serrors.IsNotFound(err) {
		return nil
	}

	return err
}

// getBuilderPod --
func getBuilderPod(ctx context.Context, client k8sclient.Reader, build *v1alpha1.Build) (*corev1.Pod, error) {
	key := k8sclient.ObjectKey{Namespace: build.Namespace, Name: buildPodName(build.ObjectMeta)}
	pod := corev1.Pod{}

	err := client.Get(ctx, key, &pod)
	if err != nil && k8serrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &pod, nil
}

func buildPodName(object metav1.ObjectMeta) string {
	return "camel-k-" + object.Name + "-builder"
}
