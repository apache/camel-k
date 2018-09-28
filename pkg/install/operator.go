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
	"github.com/apache/camel-k/deploy"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/minishift"
	"github.com/apache/camel-k/pkg/util/openshift"
)

// Operator --
func Operator(namespace string) error {
	isOpenshift, err := openshift.IsOpenShift()
	if err != nil {
		return err
	}
	var operatorRole string
	if isOpenshift {
		operatorRole = "operator-role-openshift.yaml"
	} else {
		operatorRole = "operator-role-kubernetes.yaml"
	}
	return Resources(namespace,
		"operator-service-account.yaml",
		operatorRole,
		"operator-role-binding.yaml",
		"builder-pvc.yaml",
		"operator-deployment.yaml",
		"operator-service.yaml",
	)
}

// Platform installs the platform custom resource
func Platform(namespace string, registry string) error {
	isOpenshift, err := openshift.IsOpenShift()
	if err != nil {
		return err
	}
	if isOpenshift {
		return Resource(namespace, "platform-cr.yaml")
	}
	platform, err := kubernetes.LoadResourceFromYaml(deploy.Resources["platform-cr.yaml"])
	if err != nil {
		return err
	}
	if pl, ok := platform.(*v1alpha1.IntegrationPlatform); !ok {
		panic("cannot find integration platform template")
	} else {
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
		return RuntimeObject(namespace, pl)
	}
}

// Example --
func Example(namespace string) error {
	return Resources(namespace,
		"cr-example.yaml",
	)
}
