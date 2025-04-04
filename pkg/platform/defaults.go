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

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/kamelet/repository"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/log"
	"github.com/apache/camel-k/v2/pkg/util/openshift"
)

const (
	BuilderServiceAccount = "camel-k-builder"

	defaultBuildTimeout = 5 * time.Minute
)

// ConfigureDefaults fills with default values all missing details about the integration platform.
// Defaults are set in the status fields, not in the spec.
func ConfigureDefaults(ctx context.Context, c client.Client, p *v1.IntegrationPlatform, verbose bool) error {
	// Reset the state to initial values
	p.ResyncStatusFullConfig()
	// Set the operator version controlling this resource
	p.Status.Version = defaults.Version
	// Apply settings from global integration platform that is bound to this operator
	if err := applyGlobalPlatformDefaults(ctx, c, p); err != nil {
		return err
	}

	if p.Status.Build.RuntimeProvider == "" {
		p.Status.Build.RuntimeProvider = v1.RuntimeProviderQuarkus
		log.Debugf("Integration Platform %s [%s]: setting default runtime provider %s", p.Name, p.Namespace, v1.RuntimeProviderQuarkus)
	}
	if p.Status.Build.RuntimeVersion == "" {
		p.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
		log.Debugf("Integration Platform %s [%s]: setting default runtime version %s", p.Name, p.Namespace, p.Status.Build.PublishStrategy)
	}

	// update missing fields in the resource
	if p.Status.Cluster == "" {
		log.Debugf("Integration Platform %s [%s]: setting cluster status", p.Name, p.Namespace)
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
		p.Status.Build.PublishStrategy = v1.IntegrationPlatformBuildPublishStrategyJib
		log.Debugf("Integration Platform %s [%s]: setting publishing strategy %s", p.Name, p.Namespace, p.Status.Build.PublishStrategy)
	}

	if p.Status.Build.BuildConfiguration.Strategy == "" {
		defaultStrategy := v1.BuildStrategyRoutine
		p.Status.Build.BuildConfiguration.Strategy = defaultStrategy
		log.Debugf("Integration Platform %s [%s]: setting build strategy %s", p.Name, p.Namespace, p.Status.Build.BuildConfiguration.Strategy)
	}

	if p.Status.Build.BuildConfiguration.OrderStrategy == "" {
		p.Status.Build.BuildConfiguration.OrderStrategy = v1.BuildOrderStrategyDependencies
		log.Debugf("Integration Platform %s [%s]: setting build order strategy %s", p.Name, p.Namespace, p.Status.Build.BuildConfiguration.OrderStrategy)
	}

	err := setPlatformDefaults(p, verbose)
	if err != nil {
		return err
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
		p.Status.Build.PublishStrategy == v1.IntegrationPlatformBuildPublishStrategyS2I &&
		p.Status.Build.Registry.Address == "" {

		err := configureForOpenShiftS2i(ctx, c, p)
		if err != nil {
			return err
		}
	}

	log.Debugf("Final Registry Address: %s", p.Status.Build.Registry.Address)
	return nil
}

func configureForOpenShiftS2i(ctx context.Context, c client.Client, p *v1.IntegrationPlatform) error {
	log.Debugf("Integration Platform %s [%s]: setting registry address", p.Name, p.Namespace)
	// Default to using OpenShift internal container images registry when using a strategy other than S2I
	p.Status.Build.Registry.Address = defaults.OpenShiftRegistryAddress

	// OpenShift automatically injects the service CA certificate into the service-ca.crt key on the ConfigMap
	cm, err := createServiceCaBundleConfigMap(ctx, c, p)
	if err != nil {
		return err
	}

	log.Debugf("Integration Platform %s [%s]: setting registry certificate authority", p.Name, p.Namespace)
	p.Status.Build.Registry.CA = cm.Name

	// Default to using the registry secret that's configured for the builder service account
	if p.Status.Build.Registry.Secret == "" {
		log.Debugf("Integration Platform %s [%s]: setting registry secret", p.Name, p.Namespace)
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

	return nil
}

func applyGlobalPlatformDefaults(ctx context.Context, c client.Client, p *v1.IntegrationPlatform) error {
	operatorNamespace := GetOperatorNamespace()
	if operatorNamespace != "" {
		operatorID := defaults.OperatorID()
		if operatorNamespace != p.Namespace || (operatorID != "" && p.Name != operatorID) {
			if globalPlatform, err := findLocal(ctx, c, operatorNamespace); err != nil && !k8serrors.IsNotFound(err) {
				return err
			} else if globalPlatform != nil {
				applyPlatformSpec(globalPlatform, p)
			}
		}
	}

	return nil
}

func applyPlatformSpec(source *v1.IntegrationPlatform, target *v1.IntegrationPlatform) {
	if target.Status.Cluster == "" {
		target.Status.Cluster = source.Status.Cluster
	}
	if target.Status.Profile == "" {
		log.Debugf("Integration Platform %s [%s]: setting profile", target.Name, target.Namespace)
		target.Status.Profile = source.Status.Profile
	}

	if target.Status.Build.PublishStrategy == "" {
		target.Status.Build.PublishStrategy = source.Status.Build.PublishStrategy
	}

	if target.Status.Build.BuildConfiguration.Strategy == "" {
		target.Status.Build.BuildConfiguration.Strategy = source.Status.Build.BuildConfiguration.Strategy
	}

	if target.Status.Build.BuildConfiguration.OrderStrategy == "" {
		target.Status.Build.BuildConfiguration.OrderStrategy = source.Status.Build.BuildConfiguration.OrderStrategy
	}

	if target.Status.Build.RuntimeVersion == "" {
		log.Debugf("Integration Platform %s [%s]: setting runtime version", target.Name, target.Namespace)
		target.Status.Build.RuntimeVersion = source.Status.Build.RuntimeVersion
	}
	if target.Status.Build.BaseImage == "" {
		log.Debugf("Integration Platform %s [%s]: setting base image", target.Name, target.Namespace)
		target.Status.Build.BaseImage = source.Status.Build.BaseImage
	}

	if target.Status.Build.Maven.LocalRepository == "" {
		log.Debugf("Integration Platform %s [%s]: setting local repository", target.Name, target.Namespace)
		target.Status.Build.Maven.LocalRepository = source.Status.Build.Maven.LocalRepository
	}

	if len(source.Status.Build.Maven.CLIOptions) > 0 && len(target.Status.Build.Maven.CLIOptions) == 0 {
		log.Debugf("Integration Platform %s [%s]: setting CLI options", target.Name, target.Namespace)
		target.Status.Build.Maven.CLIOptions = make([]string, len(source.Status.Build.Maven.CLIOptions))
		copy(target.Status.Build.Maven.CLIOptions, source.Status.Build.Maven.CLIOptions)
	}

	if len(source.Status.Build.Maven.Properties) > 0 {
		log.Debugf("Integration Platform %s [%s]: setting Maven properties", target.Name, target.Namespace)
		if len(target.Status.Build.Maven.Properties) == 0 {
			target.Status.Build.Maven.Properties = make(map[string]string, len(source.Status.Build.Maven.Properties))
		}

		for key, val := range source.Status.Build.Maven.Properties {
			// only set unknown properties on target
			if _, ok := target.Status.Build.Maven.Properties[key]; !ok {
				target.Status.Build.Maven.Properties[key] = val
			}
		}
	}

	if len(source.Status.Build.Maven.Extension) > 0 && len(target.Status.Build.Maven.Extension) == 0 {
		log.Debugf("Integration Platform %s [%s]: setting Maven extensions", target.Name, target.Namespace)
		target.Status.Build.Maven.Extension = make([]v1.MavenArtifact, len(source.Status.Build.Maven.Extension))
		copy(target.Status.Build.Maven.Extension, source.Status.Build.Maven.Extension)
	}

	if target.Status.Build.Registry.Address == "" && source.Status.Build.Registry.Address != "" {
		log.Debugf("Integration Platform %s [%s]: setting registry", target.Name, target.Namespace)
		source.Status.Build.Registry.DeepCopyInto(&target.Status.Build.Registry)
	}

	if err := target.Status.Traits.Merge(source.Status.Traits); err != nil {
		log.Errorf(err, "Integration Platform %s [%s]: failed to merge traits", target.Name, target.Namespace)
	} else if err := target.Status.Traits.Merge(target.Spec.Traits); err != nil {
		log.Errorf(err, "Integration Platform %s [%s]: failed to merge traits", target.Name, target.Namespace)
	}

	// Build timeout
	if target.Status.Build.Timeout == nil {
		log.Debugf("Integration Platform %s [%s]: setting build timeout", target.Name, target.Namespace)
		target.Status.Build.Timeout = source.Status.Build.Timeout
	}

	if target.Status.Build.MaxRunningBuilds <= 0 {
		log.Debugf("Integration Platform %s [%s]: setting max running builds", target.Name, target.Namespace)
		target.Status.Build.MaxRunningBuilds = source.Status.Build.MaxRunningBuilds
	}

	if len(target.Status.Kamelet.Repositories) == 0 {
		log.Debugf("Integration Platform %s [%s]: setting kamelet repositories", target.Name, target.Namespace)
		target.Status.Kamelet.Repositories = source.Status.Kamelet.Repositories
	}
}

func setPlatformDefaults(p *v1.IntegrationPlatform, verbose bool) error {
	if p.Status.Build.RuntimeVersion == "" {
		log.Debugf("Integration Platform %s [%s]: setting runtime version", p.Name, p.Namespace)
		p.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	}
	if p.Status.Build.BaseImage == "" {
		log.Debugf("Integration Platform %s [%s]: setting base image", p.Name, p.Namespace)
		p.Status.Build.BaseImage = defaults.BaseImage()
	}
	if p.Status.Build.Maven.LocalRepository == "" {
		log.Debugf("Integration Platform %s [%s]: setting local repository", p.Name, p.Namespace)
		p.Status.Build.Maven.LocalRepository = defaults.LocalRepository
	}
	if len(p.Status.Build.Maven.CLIOptions) == 0 {
		log.Debugf("Integration Platform %s [%s]: setting CLI options", p.Name, p.Namespace)
		p.Status.Build.Maven.CLIOptions = []string{
			"-V",
			"--no-transfer-progress",
			"-Dstyle.color=never",
		}
	}

	// Build timeout
	if p.Status.Build.GetTimeout().Duration == 0 {
		p.Status.Build.Timeout = &metav1.Duration{
			Duration: defaultBuildTimeout,
		}
	} else {
		d := p.Status.Build.GetTimeout().Duration.Truncate(time.Second)

		if verbose && p.Status.Build.GetTimeout().Duration != d {
			log.Log.Infof("Build timeout minimum unit is sec (configured: %s, truncated: %s)", p.Status.Build.GetTimeout().Duration, d)
		}

		log.Debugf("Integration Platform %s [%s]: setting build timeout", p.Name, p.Namespace)
		p.Status.Build.Timeout = &metav1.Duration{
			Duration: d,
		}
	}

	if p.Status.Build.MaxRunningBuilds <= 0 {
		log.Debugf("Integration Platform %s [%s]: setting max running builds", p.Name, p.Namespace)
		if p.Status.Build.BuildConfiguration.Strategy == v1.BuildStrategyRoutine {
			p.Status.Build.MaxRunningBuilds = 3
		} else if p.Status.Build.BuildConfiguration.Strategy == v1.BuildStrategyPod {
			p.Status.Build.MaxRunningBuilds = 10
		}
	}

	if len(p.Status.Kamelet.Repositories) == 0 {
		log.Debugf("Integration Platform %s [%s]: setting kamelet repositories", p.Name, p.Namespace)
		p.Status.Kamelet.Repositories = append(p.Status.Kamelet.Repositories, v1.KameletRepositorySpec{
			URI: repository.DefaultRemoteRepository,
		})
	}
	setStatusAdditionalInfo(p)

	buildConfig := &p.Status.Build.BuildConfiguration
	if buildConfig.ImagePlatforms == nil {
		if runtime.GOARCH == "arm64" {
			buildConfig.ImagePlatforms = []string{"linux/arm64"}
		}
	}

	if verbose {
		log.Log.Infof("RuntimeVersion set to %s", p.Status.Build.RuntimeVersion)
		log.Log.Infof("BaseImage set to %s", p.Status.Build.BaseImage)
		log.Log.Infof("ImagePlatforms set to %s", buildConfig.ImagePlatforms)
		log.Log.Infof("LocalRepository set to %s", p.Status.Build.Maven.LocalRepository)
		log.Log.Infof("Timeout set to %s", p.Status.Build.GetTimeout())
	}

	return nil
}

func setStatusAdditionalInfo(platform *v1.IntegrationPlatform) {
	platform.Status.Info = make(map[string]string)
	log.Debugf("Integration Platform %s [%s]: setting status info", platform.Name, platform.Namespace)
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
