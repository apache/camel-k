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
	"github.com/apache/camel-k/pkg/platform/images"
)

type imagesTrait struct {
	BaseTrait `property:",squash"`
}

func newImagesTrait() *imagesTrait {
	return &imagesTrait{
		BaseTrait: newBaseTrait("images"),
	}
}

func (t *imagesTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled == nil || !*t.Enabled {
		// Disabled by default
		return false, nil
	}

	if e.IntegrationContextInPhase("") {
		return true, nil
	}

	return false, nil
}

func (t *imagesTrait) Apply(e *Environment) error {
	// Try to lookup a image from predefined images
	image := images.LookupPredefinedImage(e.CamelCatalog, e.Context.Spec.Dependencies)
	if image == "" {
		return nil
	}

	// Change the context type to external
	if e.Context.Labels == nil {
		e.Context.Labels = make(map[string]string)
	}
	e.Context.Labels["camel.apache.org/context.type"] = v1alpha1.IntegrationContextTypeExternal

	e.Context.Spec.Image = image
	return nil
}
