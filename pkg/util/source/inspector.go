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
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/camel"
)

var (
	singleQuotedFrom = regexp.MustCompile(`from\s*\(\s*'([a-zA-Z0-9-]+:[^']+)'`)
	doubleQuotedFrom = regexp.MustCompile(`from\s*\(\s*"([a-zA-Z0-9-]+:[^"]+)"`)
	singleQuotedTo   = regexp.MustCompile(`\.to\s*\(\s*'([a-zA-Z0-9-]+:[^']+)'`)
	singleQuotedToD  = regexp.MustCompile(`\.toD\s*\(\s*'([a-zA-Z0-9-]+:[^']+)'`)
	singleQuotedToF  = regexp.MustCompile(`\.toF\s*\(\s*'([a-zA-Z0-9-]+:[^']+)'`)
	doubleQuotedTo   = regexp.MustCompile(`\.to\s*\(\s*"([a-zA-Z0-9-]+:[^"]+)"`)
	doubleQuotedToD  = regexp.MustCompile(`\.toD\s*\(\s*"([a-zA-Z0-9-]+:[^"]+)"`)
	doubleQuotedToF  = regexp.MustCompile(`\.toF\s*\(\s*"([a-zA-Z0-9-]+:[^"]+)"`)
	languageRegexp   = regexp.MustCompile(`language\s*\(\s*["|']([a-zA-Z0-9-]+[^"|']+)["|']\s*,.*\)`)

	sourceDependencies = struct {
		main    map[string]string
		quarkus map[string]string
	}{
		main: map[string]string{
			`.*JsonLibrary\.Jackson.*`:                         "camel:jackson",
			`.*\.hystrix().*`:                                  "camel:hystrix",
			`.*restConfiguration().*`:                          "camel:rest",
			`.*rest(("[a-zA-Z0-9-/]+")*).*`:                    "camel:rest",
			`^\s*rest\s*{.*`:                                   "camel:rest",
			`.*\.groovy\s*\(.*\).*`:                            "camel:groovy",
			`.*\.?(jsonpath|jsonpathWriteAsString)\s*\(.*\).*`: "camel:jsonpath",
			`.*\.ognl\s*\(.*\).*`:                              "camel:ognl",
			`.*\.mvel\s*\(.*\).*`:                              "camel:mvel",
			`.*\.?simple\s*\(.*\).*`:                           "camel:bean",
			`.*\.xquery\s*\(.*\).*`:                            "camel:saxon",
			`.*\.?xpath\s*\(.*\).*`:                            "camel:xpath",
			`.*\.xtokenize\s*\(.*\).*`:                         "camel:jaxp",
		},
		quarkus: map[string]string{
			`.*restConfiguration().*`:       "camel-quarkus:rest",
			`.*rest(("[a-zA-Z0-9-/]+")*).*`: "camel-quarkus:rest",
			`^\s*rest\s*{.*`:                "camel-quarkus:rest",
			`.*\.?simple\s*\(.*\).*`:        "camel-quarkus:bean",
		},
	}
)

// Inspector --
type Inspector interface {
	Extract(v1alpha1.SourceSpec, *Metadata) error
}

// InspectorForLanguage --
func InspectorForLanguage(catalog *camel.RuntimeCatalog, language v1alpha1.Language) Inspector {
	switch language {
	case v1alpha1.LanguageJavaSource:
		return &JavaSourceInspector{
			baseInspector: baseInspector{
				catalog: catalog,
			},
		}
	case v1alpha1.LanguageXML:
		return &XMLInspector{
			baseInspector: baseInspector{
				catalog: catalog,
			},
		}
	case v1alpha1.LanguageGroovy:
		return &GroovyInspector{
			baseInspector: baseInspector{
				catalog: catalog,
			},
		}
	case v1alpha1.LanguageJavaScript:
		return &JavaScriptInspector{
			baseInspector: baseInspector{
				catalog: catalog,
			},
		}
	case v1alpha1.LanguageKotlin:
		return &KotlinInspector{
			baseInspector: baseInspector{
				catalog: catalog,
			},
		}
	case v1alpha1.LanguageYaml:
		return &YAMLInspector{
			baseInspector: baseInspector{
				catalog: catalog,
			},
		}
	}
	return &baseInspector{}
}

type baseInspector struct {
	catalog *camel.RuntimeCatalog
}

func (i baseInspector) Extract(v1alpha1.SourceSpec, *Metadata) error {
	return nil
}

// discoverDependencies returns a list of dependencies required by the given source code
func (i *baseInspector) discoverDependencies(source v1alpha1.SourceSpec, meta *Metadata) {
	uris := util.StringSliceJoin(meta.FromURIs, meta.ToURIs)

	for _, uri := range uris {
		candidateComp := i.decodeComponent(uri)
		if candidateComp != "" {
			meta.Dependencies.Add(candidateComp)
		}
	}

	var additionalDependencies map[string]string
	if i.catalog.RuntimeProvider != nil && i.catalog.RuntimeProvider.Quarkus != nil {
		additionalDependencies = sourceDependencies.quarkus
	} else {
		additionalDependencies = sourceDependencies.main
	}
	for pattern, dep := range additionalDependencies {
		pat := regexp.MustCompile(pattern)
		if pat.MatchString(source.Content) {
			meta.Dependencies.Add(dep)
		}
	}

	for _, match := range languageRegexp.FindAllStringSubmatch(source.Content, -1) {
		if len(match) > 1 {
			if dependency, ok := i.catalog.GetLanguageDependency(match[1]); ok {
				meta.Dependencies.Add(dependency)
			}
		}
	}
}

func (i *baseInspector) decodeComponent(uri string) string {
	uriSplit := strings.SplitN(uri, ":", 2)
	if len(uriSplit) < 2 {
		return ""
	}
	uriStart := uriSplit[0]
	if component := i.catalog.GetArtifactByScheme(uriStart); component != nil {
		artifactID := component.ArtifactID
		if component.GroupID == "org.apache.camel" && strings.HasPrefix(artifactID, "camel-") {
			return "camel:" + artifactID[6:]
		}
		if component.GroupID == "org.apache.camel.quarkus" && strings.HasPrefix(artifactID, "camel-quarkus-") {
			return "camel-quarkus:" + artifactID[14:]
		}
		return "mvn:" + component.GroupID + ":" + artifactID + ":" + component.Version
	}
	return ""
}
