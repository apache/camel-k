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

package trait

import (
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

// A Environment provides the context where the trait is executed
type Environment struct {
	Platform            *v1alpha1.IntegrationPlatform
	Context             *v1alpha1.IntegrationContext
	Integration         *v1alpha1.Integration
	ExecutedCustomizers []string
}

// A DeploymentCustomizer performs customization of the deployed objects
type DeploymentCustomizer interface {
	// The Name of the customizer
	Name() string
	// Customize executes the trait customization on the resources and return true if the resources have been changed
	Customize(environment Environment, resources *kubernetes.Collection) (bool, error)
}
