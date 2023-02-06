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

package integrationplatform

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func NewWarmAction(reader ctrl.Reader) Action {
	return &warmAction{
		reader: reader,
	}
}

type warmAction struct {
	baseAction
	reader ctrl.Reader
}

func (action *warmAction) Name() string {
	return "warm"
}

func (action *warmAction) CanHandle(platform *v1.IntegrationPlatform) bool {
	return platform.Status.Phase == v1.IntegrationPlatformPhaseWarming
}

func (action *warmAction) Handle(ctx context.Context, platform *v1.IntegrationPlatform) (*v1.IntegrationPlatform, error) {
	// Check Kaniko warmer pod status
	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: platform.Namespace,
			Name:      platform.Name + "-cache",
		},
	}

	err := action.reader.Get(ctx, types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}, &pod)
	if err != nil {
		return nil, err
	}

	switch pod.Status.Phase {
	case corev1.PodSucceeded:
		action.L.Info("Kaniko cache successfully warmed up")
		platform.Status.Phase = v1.IntegrationPlatformPhaseCreating
		return platform, nil
	case corev1.PodFailed:
		return nil, errors.New("failed to warm up Kaniko cache")
	default:
		action.L.Info("Waiting for Kaniko cache to warm up...")
		// Requeue
		return nil, nil
	}
}
