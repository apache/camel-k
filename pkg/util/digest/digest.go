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

package digest

import (
	"crypto/sha256"
	"encoding/base64"
	"math/rand"
	"strconv"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/version"
)

// ComputeForIntegration a digest of the fields that are relevant for the deployment
// Produces a digest that can be used as docker image tag
func ComputeForIntegration(integration *v1alpha1.Integration) string {
	hash := sha256.New()
	// Operator version is relevant
	hash.Write([]byte(version.Version))
	// Integration Context is relevant
	hash.Write([]byte(integration.Spec.Context))

	// Integration code
	if integration.Spec.Source.Content != "" {
		hash.Write([]byte(integration.Spec.Source.Content))
	}
	// Integration dependencies
	for _, item := range integration.Spec.Dependencies {
		hash.Write([]byte(item))
	}
	// Integration configuration
	for _, item := range integration.Spec.Configuration {
		hash.Write([]byte(item.String()))
	}

	// Add a letter at the beginning and use URL safe encoding
	return "v" + base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
}

// ComputeForIntegrationContext a digest of the fields that are relevant for the deployment
// Produces a digest that can be used as docker image tag
func ComputeForIntegrationContext(context *v1alpha1.IntegrationContext) string {
	hash := sha256.New()
	// Operator version is relevant
	hash.Write([]byte(version.Version))

	for _, item := range context.Spec.Dependencies {
		hash.Write([]byte(item))
	}
	for _, item := range context.Spec.Configuration {
		hash.Write([]byte(item.String()))
	}

	// Add a letter at the beginning and use URL safe encoding
	return "v" + base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
}

// Random --
func Random() string {
	return "v" + strconv.FormatInt(rand.Int63(), 10)
}
