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

package metadata

import (
	"regexp"
	"strings"

	"github.com/apache/camel-k/pkg/util/camel"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

var restIndicator = regexp.MustCompile(`.*rest\s*\([^)]*\).*`)
var xmlRestIndicator = regexp.MustCompile(`.*<\s*rest\s+[^>]*>.*`)

// requiresHTTPService returns true if the integration needs to expose itself through HTTP
func requiresHTTPService(catalog *camel.RuntimeCatalog, source v1alpha1.SourceSpec, fromURIs []string) bool {
	if hasRestIndicator(source) {
		return true
	}
	return containsHTTPURIs(catalog, fromURIs)
}

// hasOnlyPassiveEndpoints returns true if the integration has no endpoint that needs to remain always active
func hasOnlyPassiveEndpoints(catalog *camel.RuntimeCatalog, _ v1alpha1.SourceSpec, fromURIs []string) bool {
	passivePlusHTTP := make(map[string]bool)
	catalog.VisitSchemes(func(id string, scheme camel.Scheme) bool {
		if scheme.HTTP || scheme.Passive {
			passivePlusHTTP[id] = true
		}

		return true
	})

	return containsOnlyURIsIn(fromURIs, passivePlusHTTP)
}

func containsHTTPURIs(catalog *camel.RuntimeCatalog, fromURI []string) bool {
	for _, uri := range fromURI {
		prefix := getURIPrefix(uri)
		scheme, ok := catalog.GetScheme(prefix)

		if !ok {
			// scheme dees not exists
			continue
		}

		if scheme.HTTP {
			return true
		}
	}

	return false
}

func containsOnlyURIsIn(fromURI []string, allowed map[string]bool) bool {
	for _, uri := range fromURI {
		prefix := getURIPrefix(uri)
		if enabled, ok := allowed[prefix]; !ok || !enabled {
			return false
		}
	}
	return true
}

func getURIPrefix(uri string) string {
	parts := strings.SplitN(uri, ":", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func hasRestIndicator(source v1alpha1.SourceSpec) bool {
	pat := getRestIndicatorRegexpsForLanguage(source.InferLanguage())
	return pat.MatchString(source.Content)
}

func getRestIndicatorRegexpsForLanguage(language v1alpha1.Language) *regexp.Regexp {
	switch language {
	case v1alpha1.LanguageXML:
		return xmlRestIndicator
	default:
		return restIndicator
	}
}
