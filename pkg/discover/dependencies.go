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

package discover

import (
	"regexp"
	"sort"
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/camel"
)

var (
	singleQuotedURI *regexp.Regexp
	doubleQuotedURI *regexp.Regexp
)

func init() {
	singleQuotedURI = regexp.MustCompile("'([a-z0-9-]+):[^']+'")
	doubleQuotedURI = regexp.MustCompile("\"([a-z0-9-]+):[^\"]+\"")
}

// Dependencies returns a list of dependencies required by the given source code
func Dependencies(source v1alpha1.SourceSpec) []string {
	candidateMap := make(map[string]bool)
	regexps := getRegexpsForLanguage(source.Language)
	subMatches := findAllStringSubmatch(source.Content, regexps...)
	for _, uriPrefix := range subMatches {
		candidateComp := decodeComponent(uriPrefix)
		if candidateComp != "" {
			candidateMap[candidateComp] = true
		}
	}
	// Remove duplicates and sort
	candidateComponents := make([]string, 0, len(candidateMap))
	for cmp := range candidateMap {
		candidateComponents = append(candidateComponents, cmp)
	}
	sort.Strings(candidateComponents)
	return candidateComponents
}

func getRegexpsForLanguage(language v1alpha1.Language) []*regexp.Regexp {
	switch language {
	case v1alpha1.LanguageJavaSource:
		return []*regexp.Regexp{doubleQuotedURI}
	case v1alpha1.LanguageXML:
		return []*regexp.Regexp{doubleQuotedURI}
	case v1alpha1.LanguageGroovy:
		return []*regexp.Regexp{singleQuotedURI, doubleQuotedURI}
	case v1alpha1.LanguageJavaScript:
		return []*regexp.Regexp{singleQuotedURI, doubleQuotedURI}
	case v1alpha1.LanguageKotlin:
		return []*regexp.Regexp{doubleQuotedURI}
	}
	return []*regexp.Regexp{}
}

func findAllStringSubmatch(data string, regexps ...*regexp.Regexp) []string {
	candidates := make([]string, 0)
	for _, reg := range regexps {
		hits := reg.FindAllStringSubmatch(data, -1)
		for _, hit := range hits {
			if hit != nil && len(hit) > 1 {
				for _, match := range hit[1:] {
					candidates = append(candidates, match)
				}
			}
		}
	}
	return candidates
}

func decodeComponent(uriStart string) string {
	if component := camel.Runtime.GetArtifactByScheme(uriStart); component != nil {
		artifactID := component.ArtifactID
		if strings.HasPrefix(artifactID, "camel-") {
			return "camel:" + artifactID[6:]
		}
		return "mvn:" + component.GroupID + ":" + artifactID + ":" + component.Version
	}
	return ""
}
