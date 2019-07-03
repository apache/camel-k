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
	"sort"
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/scylladb/go-set/strset"
)

var (
	singleQuotedFrom = regexp.MustCompile(`from\s*\(\s*'([a-z0-9-]+:[^']+)'`)
	doubleQuotedFrom = regexp.MustCompile(`from\s*\(\s*"([a-z0-9-]+:[^"]+)"`)
	singleQuotedTo   = regexp.MustCompile(`\.to\s*\(\s*'([a-z0-9-]+:[^']+)'`)
	singleQuotedToD  = regexp.MustCompile(`\.toD\s*\(\s*'([a-z0-9-]+:[^']+)'`)
	singleQuotedToF  = regexp.MustCompile(`\.toF\s*\(\s*'([a-z0-9-]+:[^']+)'`)
	doubleQuotedTo   = regexp.MustCompile(`\.to\s*\(\s*"([a-z0-9-]+:[^"]+)"`)
	doubleQuotedToD  = regexp.MustCompile(`\.toD\s*\(\s*"([a-z0-9-]+:[^"]+)"`)
	doubleQuotedToF  = regexp.MustCompile(`\.toF\s*\(\s*"([a-z0-9-]+:[^"]+)"`)

	additionalDependencies = map[string]string{
		".*JsonLibrary\\.Jackson.*": "camel:jackson",
		".*\\.hystrix().*":          "camel:hystrix",
		".*<hystrix>.*":             "camel:hystrix",
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
func (i *baseInspector) discoverDependencies(source v1alpha1.SourceSpec, meta *Metadata) []string {
	uris := util.StringSliceJoin(meta.FromURIs, meta.ToURIs)
	candidates := strset.New()
	candidates.Add(meta.Dependencies...)

	for _, uri := range uris {
		candidateComp := i.decodeComponent(uri)
		if candidateComp != "" {
			candidates.Add(candidateComp)
		}
	}

	for pattern, dep := range additionalDependencies {
		pat := regexp.MustCompile(pattern)
		if pat.MatchString(source.Content) {
			candidates.Add(dep)
		}
	}

	components := candidates.List()

	sort.Strings(components)

	return components
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
		if component.GroupID == "org.apache.camel.k" && strings.HasPrefix(artifactID, "camel-") {
			return "camel-k:" + artifactID[6:]
		}
		return "mvn:" + component.GroupID + ":" + artifactID + ":" + component.Version
	}
	return ""
}
