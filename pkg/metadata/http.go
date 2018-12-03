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
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"regexp"
	"strings"
)

var httpURIs = map[string]bool{
	"ahc":                  true,
	"ahc-ws":               true,
	"atmosphere-websocket": true,
	"cxf":         true,
	"cxfrs":       true,
	"grpc":        true,
	"jetty":       true,
	"netty-http":  true,
	"netty4-http": true,
	"rest":        true,
	"restlet":     true,
	"servlet":     true,
	"spark-rest":  true,
	"spring-ws":   true,
	"undertow":    true,
	"websocket":   true,
	"knative":     true,
}

var passiveURIs = map[string]bool{
	"bean":       true,
	"binding":    true,
	"browse":     true,
	"class":      true,
	"controlbus": true,
	"dataformat": true,
	"dataset":    true,
	"direct":     true,
	"direct-vm":  true,
	"language":   true,
	"log":        true,
	"mock":       true,
	"properties": true,
	"ref":        true,
	"seda":       true,
	"stub":       true,
	"test":       true,
	"validator":  true,
	"vm":         true,
}

var restIndicator = regexp.MustCompile(".*rest\\s*\\([^)]*\\).*")
var xmlRestIndicator = regexp.MustCompile(".*<\\s*rest\\s+[^>]*>.*")

// requiresHTTPService returns true if the integration needs to expose itself through HTTP
func requiresHTTPService(source v1alpha1.SourceSpec, fromURIs []string) bool {
	if hasRestIndicator(source) {
		return true
	}
	return containsHTTPURIs(fromURIs)
}

// hasOnlyPassiveEndpoints returns true if the integration has no endpoint that needs to remain always active
func hasOnlyPassiveEndpoints(source v1alpha1.SourceSpec, fromURIs []string) bool {
	passivePlusHTTP := make(map[string]bool)
	for k, v := range passiveURIs {
		passivePlusHTTP[k] = v
	}
	for k, v := range httpURIs {
		passivePlusHTTP[k] = v
	}
	return containsOnlyURIsIn(fromURIs, passivePlusHTTP)
}

func containsHTTPURIs(fromURI []string) bool {
	for _, uri := range fromURI {
		prefix := getURIPrefix(uri)
		if enabled, ok := httpURIs[prefix]; ok && enabled {
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
	pat := getRestIndicatorRegexpsForLanguage(source.Language)
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
