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
	"sort"
	"strconv"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/defaults"
)

// ComputeForIntegration a digest of the fields that are relevant for the deployment
// Produces a digest that can be used as docker image tag
func ComputeForIntegration(integration *v1alpha1.Integration) (string, error) {
	hash := sha256.New()
	// Operator version is relevant
	if _, err := hash.Write([]byte(defaults.Version)); err != nil {
		return "", err
	}
	// Integration Kit is relevant
	if _, err := hash.Write([]byte(integration.Spec.Kit)); err != nil {
		return "", err
	}

	// Integration code
	for _, s := range integration.Spec.Sources {
		if s.Content != "" {
			if _, err := hash.Write([]byte(s.Content)); err != nil {
				return "", err
			}
		}
	}

	// Integration resources
	for _, item := range integration.Spec.Resources {
		if _, err := hash.Write([]byte(item.Content)); err != nil {
			return "", err
		}
	}

	// Integration dependencies
	for _, item := range integration.Spec.Dependencies {
		if _, err := hash.Write([]byte(item)); err != nil {
			return "", err
		}
	}

	// Integration configuration
	for _, item := range integration.Spec.Configuration {
		if _, err := hash.Write([]byte(item.String())); err != nil {
			return "", err
		}
	}

	// Integration traits
	for _, name := range sortedTraitSpecMapKeys(integration.Spec.Traits) {
		if _, err := hash.Write([]byte(name + "[")); err != nil {
			return "", err
		}
		spec := integration.Spec.Traits[name]
		for _, prop := range sortedStringMapKeys(spec.Configuration) {
			val := spec.Configuration[prop]
			if _, err := hash.Write([]byte(prop + "=" + val + ",")); err != nil {
				return "", err
			}
		}
		if _, err := hash.Write([]byte("]")); err != nil {
			return "", err
		}
	}

	// Add a letter at the beginning and use URL safe encoding
	digest := "v" + base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
	return digest, nil
}

// ComputeForIntegrationKit a digest of the fields that are relevant for the deployment
// Produces a digest that can be used as docker image tag
func ComputeForIntegrationKit(kit *v1alpha1.IntegrationKit) (string, error) {
	hash := sha256.New()
	// Operator version is relevant
	if _, err := hash.Write([]byte(defaults.Version)); err != nil {
		return "", err
	}

	for _, item := range kit.Spec.Dependencies {
		if _, err := hash.Write([]byte(item)); err != nil {
			return "", err
		}
	}
	for _, item := range kit.Spec.Configuration {
		if _, err := hash.Write([]byte(item.String())); err != nil {
			return "", err
		}
	}

	// Add a letter at the beginning and use URL safe encoding
	digest := "v" + base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
	return digest, nil
}

// Random --
func Random() string {
	return "v" + strconv.FormatInt(rand.Int63(), 10)
}

func sortedStringMapKeys(m map[string]string) []string {
	res := make([]string, 0, len(m))
	for k := range m {
		res = append(res, k)
	}
	sort.Strings(res)
	return res
}

func sortedTraitSpecMapKeys(m map[string]v1alpha1.TraitSpec) []string {
	res := make([]string, 0, len(m))
	for k := range m {
		res = append(res, k)
	}
	sort.Strings(res)
	return res
}
