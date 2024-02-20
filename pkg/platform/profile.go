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

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/log"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ApplyIntegrationProfile resolves integration profile from given object and applies the profile settings to the given integration platform.
func ApplyIntegrationProfile(ctx context.Context, c k8sclient.Reader, ip *v1.IntegrationPlatform, o k8sclient.Object) (*v1.IntegrationProfile, error) {
	profile, err := findIntegrationProfile(ctx, c, o)
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	}

	if ip == nil || profile == nil {
		return nil, nil
	}

	if profile.Status.Build.RuntimeVersion != "" && profile.Status.Build.RuntimeVersion != ip.Status.Build.RuntimeVersion {
		log.Debugf("Integration Platform %s [%s]: setting runtime version", ip.Name, ip.Namespace)
		ip.Status.Build.RuntimeVersion = profile.Status.Build.RuntimeVersion
	}

	if profile.Status.Build.RuntimeProvider != "" && profile.Status.Build.RuntimeProvider != ip.Status.Build.RuntimeProvider {
		log.Debugf("Integration Platform %s [%s]: setting runtime provider", ip.Name, ip.Namespace)
		ip.Status.Build.RuntimeProvider = profile.Status.Build.RuntimeProvider
	}

	if profile.Status.Build.BaseImage != "" && profile.Status.Build.BaseImage != ip.Status.Build.BaseImage {
		log.Debugf("Integration Platform %s [%s]: setting base image", ip.Name, ip.Namespace)
		ip.Status.Build.BaseImage = profile.Status.Build.BaseImage
	}

	if profile.Status.Build.Maven.LocalRepository != "" &&
		profile.Status.Build.Maven.LocalRepository != ip.Status.Build.Maven.LocalRepository {
		log.Debugf("Integration Platform %s [%s]: setting local repository", ip.Name, ip.Namespace)
		ip.Status.Build.Maven.LocalRepository = profile.Status.Build.Maven.LocalRepository
	}

	if len(profile.Status.Build.Maven.CLIOptions) > 0 {
		log.Debugf("Integration Platform %s [%s]: setting CLI options", ip.Name, ip.Namespace)
		if len(ip.Status.Build.Maven.CLIOptions) == 0 {
			ip.Status.Build.Maven.CLIOptions = make([]string, len(profile.Status.Build.Maven.CLIOptions))
			copy(ip.Status.Build.Maven.CLIOptions, profile.Status.Build.Maven.CLIOptions)
		} else {
			util.StringSliceUniqueConcat(&ip.Status.Build.Maven.CLIOptions, profile.Status.Build.Maven.CLIOptions)
		}
	}

	if len(profile.Status.Build.Maven.Properties) > 0 {
		log.Debugf("Integration Platform %s [%s]: setting Maven properties", ip.Name, ip.Namespace)
		if len(ip.Status.Build.Maven.Properties) == 0 {
			ip.Status.Build.Maven.Properties = make(map[string]string, len(profile.Status.Build.Maven.Properties))
		}

		for key, val := range profile.Status.Build.Maven.Properties {
			// only set unknown properties on target
			if _, ok := ip.Status.Build.Maven.Properties[key]; !ok {
				ip.Status.Build.Maven.Properties[key] = val
			}
		}
	}

	if len(profile.Status.Build.Maven.Extension) > 0 && len(ip.Status.Build.Maven.Extension) == 0 {
		log.Debugf("Integration Platform %s [%s]: setting Maven extensions", ip.Name, ip.Namespace)
		ip.Status.Build.Maven.Extension = make([]v1.MavenArtifact, len(profile.Status.Build.Maven.Extension))
		copy(ip.Status.Build.Maven.Extension, profile.Status.Build.Maven.Extension)
	}

	if profile.Status.Build.Registry.Address != "" && profile.Status.Build.Registry.Address != ip.Status.Build.Registry.Address {
		log.Debugf("Integration Platform %s [%s]: setting registry", ip.Name, ip.Namespace)
		profile.Status.Build.Registry.DeepCopyInto(&ip.Status.Build.Registry)
	}

	if err := ip.Status.Traits.Merge(profile.Status.Traits); err != nil {
		log.Errorf(err, "Integration Platform %s [%s]: failed to merge traits", ip.Name, ip.Namespace)
	} else if err := ip.Status.Traits.Merge(ip.Spec.Traits); err != nil {
		log.Errorf(err, "Integration Platform %s [%s]: failed to merge traits", ip.Name, ip.Namespace)
	}

	// Build timeout
	if profile.Status.Build.Timeout != nil {
		log.Debugf("Integration Platform %s [%s]: setting build timeout", ip.Name, ip.Namespace)
		ip.Status.Build.Timeout = profile.Status.Build.Timeout
	}

	if len(profile.Status.Kamelet.Repositories) > 0 {
		log.Debugf("Integration Platform %s [%s]: setting kamelet repositories", ip.Name, ip.Namespace)
		ip.Status.Kamelet.Repositories = append(ip.Status.Kamelet.Repositories, profile.Status.Kamelet.Repositories...)
	}

	return profile, nil
}

// findIntegrationProfile finds profile from given resource annotations and resolves the profile in given resource namespace or operator namespace as a fallback option.
func findIntegrationProfile(ctx context.Context, c k8sclient.Reader, o k8sclient.Object) (*v1.IntegrationProfile, error) {
	if profileName := v1.GetIntegrationProfileAnnotation(o); profileName != "" {
		namespace := v1.GetIntegrationProfileNamespaceAnnotation(o)
		if namespace == "" {
			namespace = o.GetNamespace()
		}

		profile, err := kubernetes.GetIntegrationProfile(ctx, c, profileName, namespace)
		if err != nil && k8serrors.IsNotFound(err) {
			operatorNamespace := GetOperatorNamespace()
			if operatorNamespace != "" && operatorNamespace != namespace {
				profile, err = kubernetes.GetIntegrationProfile(ctx, c, profileName, operatorNamespace)
			}
		}
		return profile, err
	}

	return nil, nil
}
