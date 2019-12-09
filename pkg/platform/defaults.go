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

package platform

import (
	"context"
	"fmt"
	"time"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/log"
	"github.com/apache/camel-k/pkg/util/maven"
	"github.com/apache/camel-k/pkg/util/openshift"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

// ConfigureDefaults fills with default values all missing details about the integration platform.
// Defaults are set in the status->appliedConfiguration fields, not in the spec.
func ConfigureDefaults(ctx context.Context, c client.Client, p *v1alpha1.IntegrationPlatform, verbose bool) error {
	// Reset the state to initial values
	p.ResyncStatusFullConfig()

	// update missing fields in the resource
	if p.Status.FullConfig.Cluster == "" {
		// determine the kind of cluster the platform is installed into
		isOpenShift, err := openshift.IsOpenShift(c)
		switch {
		case err != nil:
			return err
		case isOpenShift:
			p.Status.FullConfig.Cluster = v1alpha1.IntegrationPlatformClusterOpenShift
		default:
			p.Status.FullConfig.Cluster = v1alpha1.IntegrationPlatformClusterKubernetes
		}
	}

	if p.Status.FullConfig.Build.PublishStrategy == "" {
		if p.Status.FullConfig.Cluster == v1alpha1.IntegrationPlatformClusterOpenShift {
			p.Status.FullConfig.Build.PublishStrategy = v1alpha1.IntegrationPlatformBuildPublishStrategyS2I
		} else {
			p.Status.FullConfig.Build.PublishStrategy = v1alpha1.IntegrationPlatformBuildPublishStrategyKaniko
		}
	}

	if p.Status.FullConfig.Build.BuildStrategy == "" {
		// If the operator is global, a global build strategy should be used
		if IsCurrentOperatorGlobal() {
			// The only global strategy we have for now
			p.Status.FullConfig.Build.BuildStrategy = v1alpha1.IntegrationPlatformBuildStrategyPod
		} else {
			if p.Status.FullConfig.Build.PublishStrategy == v1alpha1.IntegrationPlatformBuildPublishStrategyKaniko {
				// The build output has to be shared with Kaniko via a persistent volume
				p.Status.FullConfig.Build.BuildStrategy = v1alpha1.IntegrationPlatformBuildStrategyPod
			} else {
				p.Status.FullConfig.Build.BuildStrategy = v1alpha1.IntegrationPlatformBuildStrategyRoutine
			}
		}
	}

	err := setPlatformDefaults(ctx, c, p, verbose)
	if err != nil {
		return err
	}

	if verbose && p.Status.FullConfig.Build.PublishStrategy == v1alpha1.IntegrationPlatformBuildPublishStrategyKaniko && p.Status.FullConfig.Build.Registry.Address == "" {
		log.Log.Info("No registry specified for publishing images")
	}

	if verbose && p.Status.FullConfig.Build.Maven.Timeout.Duration != 0 {
		log.Log.Infof("Maven Timeout set to %s", p.Status.FullConfig.Build.Maven.Timeout.Duration)
	}

	return nil
}

func setPlatformDefaults(ctx context.Context, c client.Client, p *v1alpha1.IntegrationPlatform, verbose bool) error {
	if p.Status.FullConfig.Profile == "" {
		p.Status.FullConfig.Profile = DetermineBestProfile(ctx, c, p)
	}
	if p.Status.FullConfig.Build.CamelVersion == "" {
		p.Status.FullConfig.Build.CamelVersion = defaults.CamelVersionConstraint
	}
	if p.Status.FullConfig.Build.RuntimeVersion == "" {
		p.Status.FullConfig.Build.RuntimeVersion = defaults.RuntimeVersionConstraint
	}
	if p.Status.FullConfig.Build.BaseImage == "" {
		p.Status.FullConfig.Build.BaseImage = defaults.BaseImage
	}
	if p.Status.FullConfig.Build.Maven.LocalRepository == "" {
		p.Status.FullConfig.Build.Maven.LocalRepository = defaults.LocalRepository
	}
	if p.Status.FullConfig.Build.PersistentVolumeClaim == "" {
		p.Status.FullConfig.Build.PersistentVolumeClaim = p.Name
	}

	if p.Status.FullConfig.Build.Timeout.Duration != 0 {
		d := p.Status.FullConfig.Build.Timeout.Duration.Truncate(time.Second)

		if verbose && p.Status.FullConfig.Build.Timeout.Duration != d {
			log.Log.Infof("Build timeout minimum unit is sec (configured: %s, truncated: %s)", p.Status.FullConfig.Build.Timeout.Duration, d)
		}

		p.Status.FullConfig.Build.Timeout.Duration = d
	}
	if p.Status.FullConfig.Build.Timeout.Duration == 0 {
		p.Status.FullConfig.Build.Timeout.Duration = 5 * time.Minute
	}

	if p.Status.FullConfig.Build.Maven.Timeout.Duration != 0 {
		d := p.Status.FullConfig.Build.Maven.Timeout.Duration.Truncate(time.Second)

		if verbose && p.Status.FullConfig.Build.Maven.Timeout.Duration != d {
			log.Log.Infof("Maven timeout minimum unit is sec (configured: %s, truncated: %s)", p.Status.FullConfig.Build.Maven.Timeout.Duration, d)
		}

		p.Status.FullConfig.Build.Maven.Timeout.Duration = d
	}
	if p.Status.FullConfig.Build.Maven.Timeout.Duration == 0 {
		n := p.Status.FullConfig.Build.Timeout.Duration.Seconds() * 0.75
		p.Status.FullConfig.Build.Maven.Timeout.Duration = (time.Duration(n) * time.Second).Truncate(time.Second)
	}

	if p.Status.FullConfig.Build.Maven.Settings.ConfigMapKeyRef == nil && p.Status.FullConfig.Build.Maven.Settings.SecretKeyRef == nil {
		var repositories []maven.Repository
		for i, c := range p.Status.FullConfig.Configuration {
			if c.Type == "repository" {
				repository := maven.NewRepository(c.Value)
				if repository.ID == "" {
					repository.ID = fmt.Sprintf("repository-%03d", i)
				}
				repositories = append(repositories, repository)
			}
		}

		settings := maven.NewDefaultSettings(repositories)

		err := createDefaultMavenSettingsConfigMap(ctx, c, p, settings)
		if err != nil {
			return err
		}

		p.Status.FullConfig.Build.Maven.Settings.ConfigMapKeyRef = &corev1.ConfigMapKeySelector {
			LocalObjectReference: corev1.LocalObjectReference{
				Name: p.Name + "-maven-settings",
			},
			Key: "settings.xml",
		}
	}

	if p.Status.FullConfig.Build.PublishStrategy == v1alpha1.IntegrationPlatformBuildPublishStrategyKaniko && p.Status.FullConfig.Build.KanikoBuildCache == nil {
		// Default to using Kaniko cache warmer
		defaultKanikoBuildCache := true
		p.Status.FullConfig.Build.KanikoBuildCache = &defaultKanikoBuildCache
		if verbose {
			log.Log.Infof("Kaniko cache set to %t", *p.Status.FullConfig.Build.KanikoBuildCache)
		}
	}

	if verbose {
		log.Log.Infof("CamelVersion set to %s", p.Status.FullConfig.Build.CamelVersion)
		log.Log.Infof("RuntimeVersion set to %s", p.Status.FullConfig.Build.RuntimeVersion)
		log.Log.Infof("BaseImage set to %s", p.Status.FullConfig.Build.BaseImage)
		log.Log.Infof("LocalRepository set to %s", p.Status.FullConfig.Build.Maven.LocalRepository)
		log.Log.Infof("Timeout set to %s", p.Status.FullConfig.Build.Timeout)
		log.Log.Infof("Maven Timeout set to %s", p.Status.FullConfig.Build.Maven.Timeout.Duration)
	}

	return nil
}

func createDefaultMavenSettingsConfigMap(ctx context.Context, client client.Client, p *v1alpha1.IntegrationPlatform, settings maven.Settings) error {
	cm, err := maven.CreateSettingsConfigMap(p.Namespace, p.Name, settings)
	if err != nil {
		return err
	}

	err = client.Create(ctx, cm)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}
