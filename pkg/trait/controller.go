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
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/metadata"
	"golang.org/x/net/context"
)

// ControllerStrategy determines the kind of controller that needs to be created for the integration
type ControllerStrategy string

// List of controller strategies
const (
	ControllerStrategyDeployment     = "deployment"
	ControllerStrategyKnativeService = "knative-service"
)

// DetermineControllerStrategy determines the type of controller that should be used for the integration
func DetermineControllerStrategy(ctx context.Context, c client.Client, e *Environment) (ControllerStrategy, error) {
	if e.DetermineProfile() != v1alpha1.TraitProfileKnative {
		return ControllerStrategyDeployment, nil
	}

	sources := e.Integration.Sources()
	var err error
	if sources, err = GetEnrichedSources(ctx, c, e, sources); err != nil {
		return "", err
	}

	// In Knative profile: use knative service only if needed
	meta := metadata.ExtractAll(sources)
	if !meta.RequiresHTTPService {
		return ControllerStrategyDeployment, nil
	}

	return ControllerStrategyKnativeService, nil
}
