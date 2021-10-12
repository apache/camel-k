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

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	platformutil "github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/defaults"
)

// NewInitializeAction returns a action that initializes the platform configuration when not provided by the user
func NewInitializeAction() Action {
	return &initializeAction{}
}

type initializeAction struct {
	baseAction
}

func (action *initializeAction) Name() string {
	return "initialize"
}

func (action *initializeAction) CanHandle(platform *v1.IntegrationPlatform) bool {
	return platform.Status.Phase == "" || platform.Status.Phase == v1.IntegrationPlatformPhaseDuplicate
}

func (action *initializeAction) Handle(ctx context.Context, platform *v1.IntegrationPlatform) (*v1.IntegrationPlatform, error) {
	duplicate, err := action.isPrimaryDuplicate(ctx, platform)
	if err != nil {
		return nil, err
	}
	if duplicate {
		// another platform already present in the namespace
		if platform.Status.Phase != v1.IntegrationPlatformPhaseDuplicate {
			platform := platform.DeepCopy()
			platform.Status.Phase = v1.IntegrationPlatformPhaseDuplicate

			return platform, nil
		}

		return nil, nil
	}

	if err = platformutil.ConfigureDefaults(ctx, action.client, platform, true); err != nil {
		return nil, err
	}

	if platform.Status.Build.PublishStrategy == v1.IntegrationPlatformBuildPublishStrategyKaniko {
		if platform.Status.Build.IsKanikoCacheEnabled() {
			// Create the persistent volume claim used by the Kaniko cache
			action.L.Info("Create persistent volume claim")
			err := createPersistentVolumeClaim(ctx, action.client, platform)
			if err != nil {
				return nil, err
			}
			// Create the Kaniko warmer pod that caches the base image into the Camel K builder volume
			action.L.Info("Create Kaniko cache warmer pod")
			err = createKanikoCacheWarmerPod(ctx, action.client, platform)
			if err != nil {
				return nil, err
			}
			platform.Status.Phase = v1.IntegrationPlatformPhaseWarming
		} else {
			// Skip the warmer pod creation
			platform.Status.Phase = v1.IntegrationPlatformPhaseCreating
		}
	} else {
		platform.Status.Phase = v1.IntegrationPlatformPhaseCreating
	}
	platform.Status.Version = defaults.Version

	return platform, nil
}

func (action *initializeAction) isPrimaryDuplicate(ctx context.Context, thisPlatform *v1.IntegrationPlatform) (bool, error) {
	if platformutil.IsSecondary(thisPlatform) {
		// Always reconcile secondary platforms
		return false, nil
	}
	platforms, err := platformutil.ListPrimaryPlatforms(ctx, action.client, thisPlatform.Namespace)
	if err != nil {
		return false, err
	}
	for _, p := range platforms.Items {
		p := p // pin
		if p.Name != thisPlatform.Name && platformutil.IsActive(&p) {
			return true, nil
		}
	}

	return false, nil
}

func createPersistentVolumeClaim(ctx context.Context, client client.Client, platform *v1.IntegrationPlatform) error {
	volumeSize, err := resource.ParseQuantity("1Gi")
	if err != nil {
		return err
	}

	pvc := &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "PersistentVolumeClaim",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: platform.Namespace,
			Name:      platform.Status.Build.PersistentVolumeClaim,
			Labels: map[string]string{
				"app": "camel-k",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: volumeSize,
				},
			},
		},
	}

	err = client.Create(ctx, pvc)
	// Skip the error in case the PVC already exists
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}
