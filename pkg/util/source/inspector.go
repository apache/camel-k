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

package source

import (
	"regexp"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

var (
	singleQuotedFrom = regexp.MustCompile(`from\s*\(\s*'([a-z0-9-]+:[^']+)'\s*\)`)
	doubleQuotedFrom = regexp.MustCompile(`from\s*\(\s*"([a-z0-9-]+:[^"]+)"\s*\)`)
	singleQuotedTo   = regexp.MustCompile(`\.to\s*\(\s*'([a-z0-9-]+:[^']+)'\s*\)`)
	singleQuotedToD  = regexp.MustCompile(`\.toD\s*\(\s*'([a-z0-9-]+:[^']+)'\s*\)`)
	singleQuotedToF  = regexp.MustCompile(`\.toF\s*\(\s*'([a-z0-9-]+:[^']+)'[^)]*\)`)
	doubleQuotedTo   = regexp.MustCompile(`\.to\s*\(\s*"([a-z0-9-]+:[^"]+)"\s*\)`)
	doubleQuotedToD  = regexp.MustCompile(`\.toD\s*\(\s*"([a-z0-9-]+:[^"]+)"\s*\)`)
	doubleQuotedToF  = regexp.MustCompile(`\.toF\s*\(\s*"([a-z0-9-]+:[^"]+)"[^)]*\)`)
)

// Inspector --
type Inspector interface {
	FromURIs(v1alpha1.SourceSpec) ([]string, error)
	ToURIs(v1alpha1.SourceSpec) ([]string, error)
}

// InspectorForLanguage --
func InspectorForLanguage(language v1alpha1.Language) Inspector {
	switch language {
	case v1alpha1.LanguageJavaSource:
		return &JavaSourceInspector{}
	case v1alpha1.LanguageXML:
		return &XMLInspector{}
	case v1alpha1.LanguageGroovy:
		return &GroovyInspector{}
	case v1alpha1.LanguageJavaScript:
		return &JavaScriptInspector{}
	case v1alpha1.LanguageKotlin:
		return &KotlinInspector{}
	case v1alpha1.LanguageYamlFlow:
		return &YAMLFlowInspector{}
	}
	return &noInspector{}
}

func findAllDistinctStringSubmatch(data string, regexps ...*regexp.Regexp) []string {
	candidates := make([]string, 0)
	alreadyFound := make(map[string]bool)
	for _, reg := range regexps {
		hits := reg.FindAllStringSubmatch(data, -1)
		for _, hit := range hits {
			if len(hit) > 1 {
				for _, match := range hit[1:] {
					if _, ok := alreadyFound[match]; !ok {
						alreadyFound[match] = true
						candidates = append(candidates, match)
					}
				}
			}
		}
	}
	return candidates
}

type noInspector struct {
}

func (i noInspector) FromURIs(source v1alpha1.SourceSpec) ([]string, error) {
	return []string{}, nil
}
func (i noInspector) ToURIs(source v1alpha1.SourceSpec) ([]string, error) {
	return []string{}, nil
}
