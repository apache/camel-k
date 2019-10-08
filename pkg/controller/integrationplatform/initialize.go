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
	"fmt"
	"time"

	"github.com/apache/camel-k/pkg/util/maven"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/openshift"

	platformutil "github.com/apache/camel-k/pkg/platform"
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

func (action *initializeAction) CanHandle(platform *v1alpha1.IntegrationPlatform) bool {
	return platform.Status.Phase == "" || platform.Status.Phase == v1alpha1.IntegrationPlatformPhaseDuplicate
}

func (action *initializeAction) Handle(ctx context.Context, platform *v1alpha1.IntegrationPlatform) (*v1alpha1.IntegrationPlatform, error) {
	duplicate, err := action.isDuplicate(ctx, platform)
	if err != nil {
		return nil, err
	}
	if duplicate {
		// another platform already present in the namespace
		if platform.Status.Phase != v1alpha1.IntegrationPlatformPhaseDuplicate {
			platform := platform.DeepCopy()
			platform.Status.Phase = v1alpha1.IntegrationPlatformPhaseDuplicate

			return platform, nil
		}

		return nil, nil
	}

	// update missing fields in the resource
	if platform.Spec.Cluster == "" {
		// determine the kind of cluster the platform is installed into
		isOpenShift, err := openshift.IsOpenShift(action.client)
		switch {
		case err != nil:
			return nil, err
		case isOpenShift:
			platform.Spec.Cluster = v1alpha1.IntegrationPlatformClusterOpenShift
		default:
			platform.Spec.Cluster = v1alpha1.IntegrationPlatformClusterKubernetes
		}
	}

	if platform.Spec.Build.PublishStrategy == "" {
		if platform.Spec.Cluster == v1alpha1.IntegrationPlatformClusterOpenShift {
			platform.Spec.Build.PublishStrategy = v1alpha1.IntegrationPlatformBuildPublishStrategyS2I
		} else {
			platform.Spec.Build.PublishStrategy = v1alpha1.IntegrationPlatformBuildPublishStrategyKaniko
		}
	}

	if platform.Spec.Build.BuildStrategy == "" {
		// If the operator is global, a global build strategy should be used
		if platformutil.IsCurrentOperatorGlobal() {
			// The only global strategy we have for now
			platform.Spec.Build.BuildStrategy = v1alpha1.IntegrationPlatformBuildStrategyPod
		} else {
			if platform.Spec.Build.PublishStrategy == v1alpha1.IntegrationPlatformBuildPublishStrategyKaniko {
				// The build output has to be shared with Kaniko via a persistent volume
				platform.Spec.Build.BuildStrategy = v1alpha1.IntegrationPlatformBuildStrategyPod
			} else {
				platform.Spec.Build.BuildStrategy = v1alpha1.IntegrationPlatformBuildStrategyRoutine
			}
		}
	}

	err = action.setDefaults(ctx, platform)
	if err != nil {
		return nil, err
	}

	if platform.Spec.Build.PublishStrategy == v1alpha1.IntegrationPlatformBuildPublishStrategyKaniko && platform.Spec.Build.Registry.Address == "" {
		action.L.Info("No registry specified for publishing images")
	}

	if platform.Spec.Build.Maven.Timeout.Duration != 0 {
		action.L.Infof("Maven Timeout set to %s", platform.Spec.Build.Maven.Timeout.Duration)
	}

	err = action.client.Update(ctx, platform)
	if err != nil {
		return nil, err
	}

	if platform.Spec.Build.PublishStrategy == v1alpha1.IntegrationPlatformBuildPublishStrategyKaniko {
		// Create the persistent volume claim used to coordinate build pod output
		// with Kaniko cache and build input
		action.L.Info("Create persistent volume claim")
		err := createPersistentVolumeClaim(ctx, action.client, platform)
		if err != nil {
			return nil, err
		}

		if platform.Spec.Build.IsKanikoCacheEnabled() {
			// Create the Kaniko warmer pod that caches the base image into the Camel K builder volume
			action.L.Info("Create Kaniko cache warmer pod")
			err = createKanikoCacheWarmerPod(ctx, action.client, platform)
			if err != nil {
				return nil, err
			}
			platform.Status.Phase = v1alpha1.IntegrationPlatformPhaseWarming
		} else {
			// Skip the warmer pod creation
			platform.Status.Phase = v1alpha1.IntegrationPlatformPhaseCreating
		}

	} else {
		platform.Status.Phase = v1alpha1.IntegrationPlatformPhaseCreating
	}
	platform.Status.Version = defaults.Version

	return platform, nil
}

func (action *initializeAction) isDuplicate(ctx context.Context, thisPlatform *v1alpha1.IntegrationPlatform) (bool, error) {
	platforms, err := platformutil.ListPlatforms(ctx, action.client, thisPlatform.Namespace)
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

func (action *initializeAction) setDefaults(ctx context.Context, platform *v1alpha1.IntegrationPlatform) error {
	if platform.Spec.Profile == "" {
		platform.Spec.Profile = platformutil.DetermineBestProfile(ctx, action.client, platform)
	}
	if platform.Spec.Build.CamelVersion == "" {
		platform.Spec.Build.CamelVersion = defaults.CamelVersionConstraint
	}
	if platform.Spec.Build.RuntimeVersion == "" {
		platform.Spec.Build.RuntimeVersion = defaults.RuntimeVersion
	}
	if platform.Spec.Build.BaseImage == "" {
		platform.Spec.Build.BaseImage = defaults.BaseImage
	}
	if platform.Spec.Build.LocalRepository == "" {
		platform.Spec.Build.LocalRepository = defaults.LocalRepository
	}
	if platform.Spec.Build.PersistentVolumeClaim == "" {
		platform.Spec.Build.PersistentVolumeClaim = platform.Name
	}

	if platform.Spec.Build.Timeout.Duration != 0 {
		d := platform.Spec.Build.Timeout.Duration.Truncate(time.Second)

		if platform.Spec.Build.Timeout.Duration != d {
			action.L.Infof("Build timeout minimum unit is sec (configured: %s, truncated: %s)", platform.Spec.Build.Timeout.Duration, d)
		}

		platform.Spec.Build.Timeout.Duration = d
	}
	if platform.Spec.Build.Timeout.Duration == 0 {
		platform.Spec.Build.Timeout.Duration = 5 * time.Minute
	}

	if platform.Spec.Build.Maven.Timeout.Duration != 0 {
		d := platform.Spec.Build.Maven.Timeout.Duration.Truncate(time.Second)

		if platform.Spec.Build.Maven.Timeout.Duration != d {
			action.L.Infof("Maven timeout minimum unit is sec (configured: %s, truncated: %s)", platform.Spec.Build.Maven.Timeout.Duration, d)
		}

		platform.Spec.Build.Maven.Timeout.Duration = d
	}
	if platform.Spec.Build.Maven.Timeout.Duration == 0 {
		n := platform.Spec.Build.Timeout.Duration.Seconds() * 0.75
		platform.Spec.Build.Maven.Timeout.Duration = (time.Duration(n) * time.Second).Truncate(time.Second)
	}

	if platform.Spec.Build.Maven.Settings.ConfigMapKeyRef == nil && platform.Spec.Build.Maven.Settings.SecretKeyRef == nil {
		var repositories []maven.Repository
		for i, c := range platform.Spec.Configuration {
			if c.Type == "repository" {
				repository := maven.NewRepository(c.Value)
				if repository.ID == "" {
					repository.ID = fmt.Sprintf("repository-%03d", i)
				}
				repositories = append(repositories, repository)
			}
		}

		settings := maven.NewDefaultSettings(repositories)

		err := createMavenSettingsConfigMap(ctx, action.client, platform, settings)
		if err != nil {
			return err
		}

		platform.Spec.Build.Maven.Settings.ConfigMapKeyRef = &corev1.ConfigMapKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: platform.Name + "-maven-settings",
			},
			Key: "settings.xml",
		}
	}

	if platform.Spec.Build.PublishStrategy == v1alpha1.IntegrationPlatformBuildPublishStrategyKaniko && platform.Spec.Build.KanikoBuildCache == nil {
		// Default to using Kaniko cache warmer
		defaultKanikoBuildCache := true
		platform.Spec.Build.KanikoBuildCache = &defaultKanikoBuildCache
		action.L.Infof("Kaniko cache set to %t", *platform.Spec.Build.KanikoBuildCache)
	}

	action.L.Infof("CamelVersion set to %s", platform.Spec.Build.CamelVersion)
	action.L.Infof("RuntimeVersion set to %s", platform.Spec.Build.RuntimeVersion)
	action.L.Infof("BaseImage set to %s", platform.Spec.Build.BaseImage)
	action.L.Infof("LocalRepository set to %s", platform.Spec.Build.LocalRepository)
	action.L.Infof("Timeout set to %s", platform.Spec.Build.Timeout)
	action.L.Infof("Maven Timeout set to %s", platform.Spec.Build.Maven.Timeout.Duration)

	return nil
}

func createPersistentVolumeClaim(ctx context.Context, client client.Client, platform *v1alpha1.IntegrationPlatform) error {
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
			Name:      platform.Spec.Build.PersistentVolumeClaim,
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

func createMavenSettingsConfigMap(ctx context.Context, client client.Client, platform *v1alpha1.IntegrationPlatform, settings maven.Settings) error {
	cm, err := maven.CreateSettingsConfigMap(platform.Namespace, platform.Name, settings)
	if err != nil {
		return err
	}

	err = client.Create(ctx, cm)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}
