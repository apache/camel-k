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

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	yaml "gopkg.in/yaml.v2"
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
	xmlTagFrom       = regexp.MustCompile(`<\s*from\s+[^>]*uri\s*=\s*"([a-z0-9-]+:[^"]+)"[^>]*>`)
	xmlTagTo         = regexp.MustCompile(`<\s*to\s+[^>]*uri\s*=\s*"([a-z0-9-]+:[^"]+)"[^>]*>`)
	xmlTagToD        = regexp.MustCompile(`<\s*toD\s+[^>]*uri\s*=\s*"([a-z0-9-]+:[^"]+)"[^>]*>`)
)

// LanguageInspector --
type LanguageInspector interface {
	FromURIs(v1alpha1.SourceSpec) ([]string, error)
	ToURIs(v1alpha1.SourceSpec) ([]string, error)
}

type languageInspector struct {
	from func(v1alpha1.SourceSpec) ([]string, error)
	to   func(v1alpha1.SourceSpec) ([]string, error)
}

func (i languageInspector) FromURIs(source v1alpha1.SourceSpec) ([]string, error) {
	return i.from(source)
}
func (i languageInspector) ToURIs(source v1alpha1.SourceSpec) ([]string, error) {
	return i.to(source)
}

// GetInspectorForLanguage --
func GetInspectorForLanguage(language v1alpha1.Language) LanguageInspector {
	switch language {
	case v1alpha1.LanguageJavaSource:
		return &languageInspector{
			from: func(source v1alpha1.SourceSpec) ([]string, error) {
				answer := findAllDistinctStringSubmatch(
					source.Content,
					doubleQuotedFrom,
				)

				return answer, nil
			},
			to: func(source v1alpha1.SourceSpec) ([]string, error) {
				answer := findAllDistinctStringSubmatch(
					source.Content,
					doubleQuotedTo,
					doubleQuotedToD,
					doubleQuotedToF,
				)

				return answer, nil
			},
		}
	case v1alpha1.LanguageXML:
		return &languageInspector{
			from: func(source v1alpha1.SourceSpec) ([]string, error) {
				answer := findAllDistinctStringSubmatch(
					source.Content,
					xmlTagFrom,
				)

				return answer, nil
			},
			to: func(source v1alpha1.SourceSpec) ([]string, error) {
				answer := findAllDistinctStringSubmatch(
					source.Content,
					xmlTagTo,
					xmlTagToD,
				)

				return answer, nil
			},
		}
	case v1alpha1.LanguageGroovy:
		return &languageInspector{
			from: func(source v1alpha1.SourceSpec) ([]string, error) {
				answer := findAllDistinctStringSubmatch(
					source.Content,
					singleQuotedFrom,
					doubleQuotedFrom,
				)

				return answer, nil
			},
			to: func(source v1alpha1.SourceSpec) ([]string, error) {
				answer := findAllDistinctStringSubmatch(
					source.Content,
					singleQuotedTo,
					doubleQuotedTo,
					singleQuotedToD,
					doubleQuotedToD,
					singleQuotedToF,
					doubleQuotedToF,
				)

				return answer, nil
			},
		}
	case v1alpha1.LanguageJavaScript:
		return &languageInspector{
			from: func(source v1alpha1.SourceSpec) ([]string, error) {
				answer := findAllDistinctStringSubmatch(
					source.Content,
					singleQuotedFrom,
					doubleQuotedFrom,
				)

				return answer, nil
			},
			to: func(source v1alpha1.SourceSpec) ([]string, error) {
				answer := findAllDistinctStringSubmatch(
					source.Content,
					singleQuotedTo,
					doubleQuotedTo,
					singleQuotedToD,
					doubleQuotedToD,
					singleQuotedToF,
					doubleQuotedToF,
				)

				return answer, nil
			},
		}
	case v1alpha1.LanguageKotlin:
		return &languageInspector{
			from: func(source v1alpha1.SourceSpec) ([]string, error) {
				answer := findAllDistinctStringSubmatch(
					source.Content,
					doubleQuotedFrom,
				)

				return answer, nil
			},
			to: func(source v1alpha1.SourceSpec) ([]string, error) {
				answer := findAllDistinctStringSubmatch(
					source.Content,
					doubleQuotedTo,
					doubleQuotedToD,
					doubleQuotedToF,
				)

				return answer, nil
			},
		}
	case v1alpha1.LanguageYamlFlow:
		var flows []v1alpha1.Flow

		return &languageInspector{
			from: func(source v1alpha1.SourceSpec) ([]string, error) {
				if err := yaml.Unmarshal([]byte(source.Content), &flows); err != nil {
					return []string{}, nil
				}

				uris := make([]string, 0)

				for _, flow := range flows {
					if flow.Steps[0].URI != "" {
						uris = append(uris, flow.Steps[0].URI)
					}

				}
				return uris, nil
			},
			to: func(source v1alpha1.SourceSpec) ([]string, error) {
				if err := yaml.Unmarshal([]byte(source.Content), &flows); err != nil {
					return []string{}, nil
				}

				uris := make([]string, 0)

				for _, flow := range flows {
					for i := 1; i < len(flow.Steps); i++ {
						if flow.Steps[i].URI != "" {
							uris = append(uris, flow.Steps[i].URI)
						}
					}
				}

				return uris, nil
			},
		}
	}
	return &languageInspector{
		from: func(source v1alpha1.SourceSpec) ([]string, error) {
			return []string{}, nil
		},
		to: func(source v1alpha1.SourceSpec) ([]string, error) {
			return []string{}, nil
		},
	}
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
