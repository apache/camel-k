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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/camel"
)

var (
	singleQuotedFrom        = regexp.MustCompile(`from\s*\(\s*'([a-zA-Z0-9-]+:[^']+)'`)
	doubleQuotedFrom        = regexp.MustCompile(`from\s*\(\s*"([a-zA-Z0-9-]+:[^"]+)"`)
	singleQuotedTo          = regexp.MustCompile(`\.to\s*\(\s*'([a-zA-Z0-9-]+:[^']+)'`)
	singleQuotedToD         = regexp.MustCompile(`\.toD\s*\(\s*'([a-zA-Z0-9-]+:[^']+)'`)
	singleQuotedToF         = regexp.MustCompile(`\.toF\s*\(\s*'([a-zA-Z0-9-]+:[^']+)'`)
	doubleQuotedTo          = regexp.MustCompile(`\.to\s*\(\s*"([a-zA-Z0-9-]+:[^"]+)"`)
	doubleQuotedToD         = regexp.MustCompile(`\.toD\s*\(\s*"([a-zA-Z0-9-]+:[^"]+)"`)
	doubleQuotedToF         = regexp.MustCompile(`\.toF\s*\(\s*"([a-zA-Z0-9-]+:[^"]+)"`)
	languageRegexp          = regexp.MustCompile(`language\s*\(\s*["|']([a-zA-Z0-9-]+[^"|']+)["|']\s*,.*\)`)
	camelTypeRegexp         = regexp.MustCompile(`.*(org.apache.camel.*Component|DataFormat|Language)`)
	jsonLibraryRegexp       = regexp.MustCompile(`.*JsonLibrary\.Jackson.*`)
	jsonLanguageRegexp      = regexp.MustCompile(`.*\.json\(\).*`)
	circuitBreakerRegexp    = regexp.MustCompile(`.*\.circuitBreaker\(\).*`)
	restConfigurationRegexp = regexp.MustCompile(`.*restConfiguration\(\).*`)
	restRegexp              = regexp.MustCompile(`.*rest\(("[a-zA-Z0-9-/]+")*\).*`)
	restXMLRegexp           = regexp.MustCompile(`^\s*rest\s*{.*`)
	groovyLanguageRegexp    = regexp.MustCompile(`.*\.groovy\s*\(.*\).*`)
	jsonPathLanguageRegexp  = regexp.MustCompile(`.*\.?(jsonpath|jsonpathWriteAsString)\s*\(.*\).*`)
	ognlRegexp              = regexp.MustCompile(`.*\.ognl\s*\(.*\).*`)
	mvelRegexp              = regexp.MustCompile(`.*\.mvel\s*\(.*\).*`)
	simpleLanguageRegexp    = regexp.MustCompile(`.*\.?simple\s*\(.*\).*`)
	xqueryRegexp            = regexp.MustCompile(`.*\.xquery\s*\(.*\).*`)
	xpathRegexp             = regexp.MustCompile(`.*\.?xpath\s*\(.*\).*`)
	xtokenizeRegexp         = regexp.MustCompile(`.*\.xtokenize\s*\(.*\).*`)

	sourceDependencies = struct {
		main    map[*regexp.Regexp]string
		quarkus map[*regexp.Regexp]string
	}{
		main: map[*regexp.Regexp]string{
			jsonLibraryRegexp:       "camel:jackson",
			jsonLanguageRegexp:      "camel:jackson",
			circuitBreakerRegexp:    "camel:hystrix",
			restConfigurationRegexp: "camel:rest",
			restRegexp:              "camel:rest",
			restXMLRegexp:           "camel:rest",
			groovyLanguageRegexp:    "camel:groovy",
			jsonPathLanguageRegexp:  "camel:jsonpath",
			ognlRegexp:              "camel:ognl",
			mvelRegexp:              "camel:mvel",
			simpleLanguageRegexp:    "camel:bean",
			xqueryRegexp:            "camel:saxon",
			xpathRegexp:             "camel:xpath",
			xtokenizeRegexp:         "camel:jaxp",
		},
		quarkus: map[*regexp.Regexp]string{
			xtokenizeRegexp: "camel-quarkus:core-xml",
		},
	}
)

// Inspector --
type Inspector interface {
	Extract(v1.SourceSpec, *Metadata) error
}

// InspectorForLanguage --
func InspectorForLanguage(catalog *camel.RuntimeCatalog, language v1.Language) Inspector {
	switch language {
	case v1.LanguageJavaSource:
		return &JavaSourceInspector{
			baseInspector: baseInspector{
				catalog: catalog,
			},
		}
	case v1.LanguageXML:
		return &XMLInspector{
			baseInspector: baseInspector{
				catalog: catalog,
			},
		}
	case v1.LanguageGroovy:
		return &GroovyInspector{
			baseInspector: baseInspector{
				catalog: catalog,
			},
		}
	case v1.LanguageJavaScript:
		return &JavaScriptInspector{
			baseInspector: baseInspector{
				catalog: catalog,
			},
		}
	case v1.LanguageKotlin:
		return &KotlinInspector{
			baseInspector: baseInspector{
				catalog: catalog,
			},
		}
	case v1.LanguageYaml:
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

func (i baseInspector) Extract(v1.SourceSpec, *Metadata) error {
	return nil
}

// discoverDependencies returns a list of dependencies required by the given source code
func (i *baseInspector) discoverDependencies(source v1.SourceSpec, meta *Metadata) {
	uris := util.StringSliceJoin(meta.FromURIs, meta.ToURIs)

	for _, uri := range uris {
		candidateComp := i.decodeComponent(uri)
		if candidateComp != "" {
			i.addDependency(candidateComp, meta)
		}
	}

	for pattern, dep := range sourceDependencies.main {
		if i.catalog.Runtime.Provider == v1.RuntimeProviderQuarkus {
			// Check whether quarkus has its own artifact that differs from the standard one
			if _, ok := sourceDependencies.quarkus[pattern]; ok {
				dep = sourceDependencies.quarkus[pattern]
			}
		}
		if pattern.MatchString(source.Content) {
			i.addDependency(dep, meta)
		}
	}

	for _, match := range languageRegexp.FindAllStringSubmatch(source.Content, -1) {
		if len(match) > 1 {
			if dependency, ok := i.catalog.GetLanguageDependency(match[1]); ok {
				i.addDependency(dependency, meta)
			}
		}
	}

	for _, match := range camelTypeRegexp.FindAllStringSubmatch(source.Content, -1) {
		if len(match) > 1 {
			if dependency, ok := i.catalog.GetJavaTypeDependency(match[1]); ok {
				i.addDependency(dependency, meta)
			}
		}
	}
}

func (i *baseInspector) addDependency(dependency string, meta *Metadata) {
	if i.catalog.Runtime.Provider == v1.RuntimeProviderQuarkus {
		if strings.HasPrefix(dependency, "camel:") {
			dependency = "camel-quarkus:" + strings.TrimPrefix(dependency, "camel:")
		}
	}
	meta.Dependencies.Add(dependency)
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
