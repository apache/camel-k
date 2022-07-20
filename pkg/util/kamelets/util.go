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

package kamelets

import (
	"context"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/source"
)

// ExtractKameletFromSources provide a list of Kamelets referred into the Integration sources.
func ExtractKameletFromSources(context context.Context, c client.Client, catalog *camel.RuntimeCatalog,
	resources *kubernetes.Collection, it *v1.Integration) ([]string, error) {
	var kamelets []string

	sources, err := kubernetes.ResolveIntegrationSources(context, c, it, resources)
	if err != nil {
		return nil, err
	}
	metadata.Each(catalog, sources, func(_ int, meta metadata.IntegrationMetadata) bool {
		util.StringSliceUniqueConcat(&kamelets, meta.Kamelets)
		return true
	})

	// Check if a Kamelet is configured as default error handler URI
	defaultErrorHandlerURI := it.Spec.GetConfigurationProperty(v1alpha1.ErrorHandlerAppPropertiesPrefix + ".deadLetterUri")
	if defaultErrorHandlerURI != "" {
		if strings.HasPrefix(defaultErrorHandlerURI, "kamelet:") {
			kamelets = append(kamelets, source.ExtractKamelet(defaultErrorHandlerURI))
		}
	}

	return kamelets, nil
}
