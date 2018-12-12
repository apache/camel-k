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

package install

import (
	"errors"
	"strconv"
	"time"

	"github.com/apache/camel-k/deploy"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/knative"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/minishift"
	"github.com/apache/camel-k/pkg/util/openshift"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
)

// Operator --
func Operator(namespace string) error {
	isOpenshift, err := openshift.IsOpenShift()
	if err != nil {
		return err
	}
	if isOpenshift {
		if err := installOpenshift(namespace); err != nil {
			return err
		}
	} else {
		if err := installKubernetes(namespace); err != nil {
			return err
		}
	}
	// Additionally, install Knative resources (roles and bindings)
	isKnative, err := knative.IsInstalled()
	if err != nil {
		return err
	}
	if isKnative {
		return installKnative(namespace)
	}
	return nil
}

func installOpenshift(namespace string) error {
	return Resources(namespace,
		"operator-service-account.yaml",
		"operator-role-openshift.yaml",
		"operator-role-binding.yaml",
		"operator-deployment-openshift.yaml",
		"operator-service.yaml",
	)
}

func installKubernetes(namespace string) error {
	return Resources(namespace,
		"operator-service-account.yaml",
		"operator-role-kubernetes.yaml",
		"operator-role-binding.yaml",
		"builder-pvc.yaml",
		"operator-deployment-kubernetes.yaml",
		"operator-service.yaml",
	)
}

func installKnative(namespace string) error {
	return Resources(namespace,
		"operator-role-knative.yaml",
		"operator-role-binding-knative.yaml",
	)
}

// Platform installs the platform custom resource
func Platform(namespace string, registry string, organization string, pushSecret string) error {
	if err := waitForPlatformCRDAvailable(namespace, 25*time.Second); err != nil {
		return err
	}
	isOpenshift, err := openshift.IsOpenShift()
	if err != nil {
		return err
	}
	platformObject, err := kubernetes.LoadResourceFromYaml(deploy.Resources["platform-cr.yaml"])
	if err != nil {
		return err
	}
	pl := platformObject.(*v1alpha1.IntegrationPlatform)

	if !isOpenshift {
		// Kubernetes only (Minikube)
		if registry == "" {
			// This operation should be done here in the installer
			// because the operator is not allowed to look into the "kube-system" namespace
			minishiftRegistry, err := minishift.FindRegistry()
			if err != nil {
				return err
			}
			if minishiftRegistry == nil {
				return errors.New("cannot find automatically a registry where to push images")
			}
			registry = *minishiftRegistry
		}
		pl.Spec.Build.Registry = registry
		pl.Spec.Build.Organization = organization
		pl.Spec.Build.PushSecret = pushSecret
	}

	var knativeInstalled bool
	if knativeInstalled, err = knative.IsInstalled(); err != nil {
		return err
	}
	if knativeInstalled {
		pl.Spec.Profile = v1alpha1.TraitProfileKnative
	}

	return RuntimeObject(namespace, pl)
}

func waitForPlatformCRDAvailable(namespace string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		pla := v1alpha1.NewIntegrationPlatformList()
		if err := sdk.List(namespace, &pla); err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New("cannot list integration platforms after " + strconv.FormatInt(timeout.Nanoseconds()/1000000000, 10) + " seconds")
		}
		time.Sleep(2 * time.Second)
	}
}

// Example --
func Example(namespace string) error {
	return Resources(namespace,
		"cr-example.yaml",
	)
}
