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
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/install"
	"github.com/apache/camel-k/pkg/kamelet/repository"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/log"
	"github.com/apache/camel-k/pkg/util/openshift"
)

// BuilderServiceAccount --.
const BuilderServiceAccount = "camel-k-builder"

// ConfigureDefaults fills with default values all missing details about the integration platform.
// Defaults are set in the status fields, not in the spec.
func ConfigureDefaults(ctx context.Context, c client.Client, p *v1.IntegrationPlatform, verbose bool) error {
	// Reset the state to initial values
	p.ResyncStatusFullConfig()

	// update missing fields in the resource
	if p.Status.Cluster == "" {
		// determine the kind of cluster the platform is installed into
		isOpenShift, err := openshift.IsOpenShift(c)
		switch {
		case err != nil:
			return err
		case isOpenShift:
			p.Status.Cluster = v1.IntegrationPlatformClusterOpenShift
		default:
			p.Status.Cluster = v1.IntegrationPlatformClusterKubernetes
		}
	}

	if p.Status.Build.PublishStrategy == "" {
		if p.Status.Cluster == v1.IntegrationPlatformClusterOpenShift {
			p.Status.Build.PublishStrategy = v1.IntegrationPlatformBuildPublishStrategyS2I
		} else {
			p.Status.Build.PublishStrategy = v1.IntegrationPlatformBuildPublishStrategySpectrum
		}
	}

	if p.Status.Build.BuildStrategy == "" {
		// Use the fastest strategy that they support (routine when possible)
		if p.Status.Build.PublishStrategy == v1.IntegrationPlatformBuildPublishStrategyS2I ||
			p.Status.Build.PublishStrategy == v1.IntegrationPlatformBuildPublishStrategySpectrum {
			p.Status.Build.BuildStrategy = v1.BuildStrategyRoutine
		} else {
			// The build output has to be shared via a volume
			p.Status.Build.BuildStrategy = v1.BuildStrategyPod
		}
	}

	err := setPlatformDefaults(p, verbose)
	if err != nil {
		return err
	}

	if p.Status.Build.BuildStrategy == v1.BuildStrategyPod {
		if err := createBuilderServiceAccount(ctx, c, p); err != nil {
			return errors.Wrap(err, "cannot ensure service account is present")
		}
	}

	err = configureRegistry(ctx, c, p)
	if err != nil {
		return err
	}

	if verbose && p.Status.Build.PublishStrategy != v1.IntegrationPlatformBuildPublishStrategyS2I && p.Status.Build.Registry.Address == "" {
		log.Log.Info("No registry specified for publishing images")
	}

	if verbose && p.Status.Build.GetTimeout().Duration != 0 {
		log.Log.Infof("Maven Timeout set to %s", p.Status.Build.GetTimeout().Duration)
	}

	return nil
}

func configureRegistry(ctx context.Context, c client.Client, p *v1.IntegrationPlatform) error {
	if p.Status.Cluster == v1.IntegrationPlatformClusterOpenShift &&
		p.Status.Build.PublishStrategy != v1.IntegrationPlatformBuildPublishStrategyS2I &&
		p.Status.Build.Registry.Address == "" {
		// Default to using OpenShift internal container images registry when using a strategy other than S2I
		p.Status.Build.Registry.Address = "image-registry.openshift-image-registry.svc:5000"

		// OpenShift automatically injects the service CA certificate into the service-ca.crt key on the ConfigMap
		cm, err := createServiceCaBundleConfigMap(ctx, c, p)
		if err != nil {
			return err
		}
		p.Status.Build.Registry.CA = cm.Name

		// Default to using the registry secret that's configured for the builder service account
		if p.Status.Build.Registry.Secret == "" {
			// Bind the required role to push images to the registry
			err := createBuilderRegistryRoleBinding(ctx, c, p)
			if err != nil {
				return err
			}

			sa := corev1.ServiceAccount{}
			err = c.Get(ctx, types.NamespacedName{Namespace: p.Namespace, Name: BuilderServiceAccount}, &sa)
			if err != nil {
				return err
			}
			// We may want to read the secret keys instead of relying on the secret name scheme
			for _, secret := range sa.Secrets {
				if strings.Contains(secret.Name, "camel-k-builder-dockercfg") {
					p.Status.Build.Registry.Secret = secret.Name

					break
				}
			}
		}
	}

	return nil
}

func setPlatformDefaults(p *v1.IntegrationPlatform, verbose bool) error {
	if p.Status.Build.RuntimeVersion == "" {
		p.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	}
	if p.Status.Build.BaseImage == "" {
		p.Status.Build.BaseImage = defaults.BaseImage()
	}
	if p.Status.Build.Maven.LocalRepository == "" {
		p.Status.Build.Maven.LocalRepository = defaults.LocalRepository
	}
	if p.Status.Build.PersistentVolumeClaim == "" {
		p.Status.Build.PersistentVolumeClaim = p.Name
	}

	if p.Status.Build.GetTimeout().Duration != 0 {
		d := p.Status.Build.GetTimeout().Duration.Truncate(time.Second)

		if verbose && p.Status.Build.GetTimeout().Duration != d {
			log.Log.Infof("Build timeout minimum unit is sec (configured: %s, truncated: %s)", p.Status.Build.GetTimeout().Duration, d)
		}

		p.Status.Build.Timeout = &metav1.Duration{
			Duration: d,
		}
	}
	if p.Status.Build.GetTimeout().Duration == 0 {
		p.Status.Build.Timeout = &metav1.Duration{
			Duration: 5 * time.Minute,
		}
	}

	if p.Status.Build.PublishStrategy == v1.IntegrationPlatformBuildPublishStrategyKaniko && p.Status.Build.KanikoBuildCache == nil {
		// Default to disabling Kaniko cache warmer
		// Using the cache warmer pod seems unreliable with the current Kaniko version
		// and requires relying on a persistent volume.
		defaultKanikoBuildCache := false
		p.Status.Build.KanikoBuildCache = &defaultKanikoBuildCache
		if verbose {
			log.Log.Infof("Kaniko cache set to %t", *p.Status.Build.KanikoBuildCache)
		}
	}

	if len(p.Status.Kamelet.Repositories) == 0 {
		p.Status.Kamelet.Repositories = append(p.Status.Kamelet.Repositories, v1.IntegrationPlatformKameletRepositorySpec{
			URI: repository.DefaultRemoteRepository,
		})
	}
	setStatusAdditionalInfo(p)

	if verbose {
		log.Log.Infof("RuntimeVersion set to %s", p.Status.Build.RuntimeVersion)
		log.Log.Infof("BaseImage set to %s", p.Status.Build.BaseImage)
		log.Log.Infof("LocalRepository set to %s", p.Status.Build.Maven.LocalRepository)
		log.Log.Infof("Timeout set to %s", p.Status.Build.GetTimeout())
	}

	return nil
}

func setStatusAdditionalInfo(platform *v1.IntegrationPlatform) {
	platform.Status.Info = make(map[string]string)
	if platform.Spec.Build.PublishStrategy == v1.IntegrationPlatformBuildPublishStrategyBuildah {
		platform.Status.Info["buildahVersion"] = defaults.BuildahVersion
	} else if platform.Spec.Build.PublishStrategy == v1.IntegrationPlatformBuildPublishStrategyKaniko {
		platform.Status.Info["kanikoVersion"] = defaults.KanikoVersion
	}
	platform.Status.Info["goVersion"] = runtime.Version()
	platform.Status.Info["goOS"] = runtime.GOOS
	platform.Status.Info["gitCommit"] = defaults.GitCommit
}

func createServiceCaBundleConfigMap(ctx context.Context, client client.Client, p *v1.IntegrationPlatform) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      BuilderServiceAccount + "-ca",
			Namespace: p.Namespace,
			Annotations: map[string]string{
				"service.beta.openshift.io/inject-cabundle": "true",
			},
		},
	}

	err := client.Create(ctx, cm)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return nil, err
	}

	return cm, nil
}

func createBuilderServiceAccount(ctx context.Context, client client.Client, p *v1.IntegrationPlatform) error {
	sa := corev1.ServiceAccount{}
	key := ctrl.ObjectKey{
		Name:      BuilderServiceAccount,
		Namespace: p.Namespace,
	}

	err := client.Get(ctx, key, &sa)
	if err != nil && k8serrors.IsNotFound(err) {
		return install.BuilderServiceAccountRoles(ctx, client, p.Namespace, p.Status.Cluster)
	}

	return err
}

func createBuilderRegistryRoleBinding(ctx context.Context, client client.Client, p *v1.IntegrationPlatform) error {
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      BuilderServiceAccount + "-registry",
			Namespace: p.Namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: BuilderServiceAccount,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			APIGroup: "rbac.authorization.k8s.io",
			Name:     "system:image-builder",
		},
	}

	err := client.Create(ctx, rb)
	if err != nil {
		if k8serrors.IsForbidden(err) {
			log.Log.Infof("Cannot grant permission to push images to the registry. "+
				"Run 'oc policy add-role-to-user system:image-builder system:serviceaccount:%s:%s' as a system admin.", p.Namespace, BuilderServiceAccount)
		} else if !k8serrors.IsAlreadyExists(err) {
			return err
		}
	}

	return nil
}
