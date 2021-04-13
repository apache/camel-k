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

type catalog2deps func(*camel.RuntimeCatalog) []string

const (
	defaultJsonDataFormat = "json-jackson"
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
	restRegexp              = regexp.MustCompile(`.*rest\s*\([^)]*\).*`)
	restClosureRegexp       = regexp.MustCompile(`.*rest\s*{\s*`)
	groovyLanguageRegexp    = regexp.MustCompile(`.*\.groovy\s*\(.*\).*`)
	jsonPathLanguageRegexp  = regexp.MustCompile(`.*\.?(jsonpath|jsonpathWriteAsString)\s*\(.*\).*`)
	ognlRegexp              = regexp.MustCompile(`.*\.ognl\s*\(.*\).*`)
	mvelRegexp              = regexp.MustCompile(`.*\.mvel\s*\(.*\).*`)
	xqueryRegexp            = regexp.MustCompile(`.*\.xquery\s*\(.*\).*`)
	xpathRegexp             = regexp.MustCompile(`.*\.?xpath\s*\(.*\).*`)
	xtokenizeRegexp         = regexp.MustCompile(`.*\.xtokenize\s*\(.*\).*`)

	sourceCapabilities = map[*regexp.Regexp][]string{
		circuitBreakerRegexp: {v1.CapabilityCircuitBreaker},
	}

	sourceDependencies = map[*regexp.Regexp]catalog2deps{
		jsonLibraryRegexp: func(catalog *camel.RuntimeCatalog) []string {
			res := make([]string, 0)
			if jsonDF := catalog.GetArtifactByDataFormat(defaultJsonDataFormat); jsonDF != nil {
				res = append(res, jsonDF.GetDependencyID())
			}
			return res
		},
		jsonLanguageRegexp: func(catalog *camel.RuntimeCatalog) []string {
			res := make([]string, 0)
			if jsonDF := catalog.GetArtifactByDataFormat(defaultJsonDataFormat); jsonDF != nil {
				res = append(res, jsonDF.GetDependencyID())
			}
			return res
		},
		restConfigurationRegexp: func(catalog *camel.RuntimeCatalog) []string {
			deps := make([]string, 0)
			if c, ok := catalog.CamelCatalogSpec.Runtime.Capabilities["rest"]; ok {
				for _, d := range c.Dependencies {
					deps = append(deps, d.GetDependencyID())
				}
			}
			return deps
		},
		restRegexp: func(catalog *camel.RuntimeCatalog) []string {
			deps := make([]string, 0)
			if c, ok := catalog.CamelCatalogSpec.Runtime.Capabilities["rest"]; ok {
				for _, d := range c.Dependencies {
					deps = append(deps, d.GetDependencyID())
				}
			}
			return deps
		},
		restClosureRegexp: func(catalog *camel.RuntimeCatalog) []string {
			deps := make([]string, 0)
			if c, ok := catalog.CamelCatalogSpec.Runtime.Capabilities["rest"]; ok {
				for _, d := range c.Dependencies {
					deps = append(deps, d.GetDependencyID())
				}
			}
			return deps
		},
		groovyLanguageRegexp: func(catalog *camel.RuntimeCatalog) []string {
			if dependency, ok := catalog.GetLanguageDependency("groovy"); ok {
				return []string{dependency}
			}

			return []string{}
		},
		jsonPathLanguageRegexp: func(catalog *camel.RuntimeCatalog) []string {
			if dependency, ok := catalog.GetLanguageDependency("jsonpath"); ok {
				return []string{dependency}
			}

			return []string{}
		},
		ognlRegexp: func(catalog *camel.RuntimeCatalog) []string {
			if dependency, ok := catalog.GetLanguageDependency("ognl"); ok {
				return []string{dependency}
			}

			return []string{}
		},
		mvelRegexp: func(catalog *camel.RuntimeCatalog) []string {
			if dependency, ok := catalog.GetLanguageDependency("mvel"); ok {
				return []string{dependency}
			}

			return []string{}
		},
		xqueryRegexp: func(catalog *camel.RuntimeCatalog) []string {
			if dependency, ok := catalog.GetLanguageDependency("xquery"); ok {
				return []string{dependency}
			}

			return []string{}
		},
		xpathRegexp: func(catalog *camel.RuntimeCatalog) []string {
			if dependency, ok := catalog.GetLanguageDependency("xpath"); ok {
				return []string{dependency}
			}

			return []string{}
		},
		xtokenizeRegexp: func(catalog *camel.RuntimeCatalog) []string {
			if dependency, ok := catalog.GetLanguageDependency("xtokenize"); ok {
				return []string{dependency}
			}

			return []string{}
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
func (i *baseInspector) discoverCapabilities(source v1.SourceSpec, meta *Metadata) {
	uris := util.StringSliceJoin(meta.FromURIs, meta.ToURIs)

	for _, uri := range uris {
		if i.getURIPrefix(uri) == "platform-http" {
			meta.RequiredCapabilities.Add(v1.CapabilityPlatformHTTP)
		}
	}

	for pattern, capabilities := range sourceCapabilities {
		if !pattern.MatchString(source.Content) {
			continue
		}

		for _, capability := range capabilities {
			meta.RequiredCapabilities.Add(capability)
		}
	}
}

// discoverDependencies returns a list of dependencies required by the given source code
func (i *baseInspector) discoverDependencies(source v1.SourceSpec, meta *Metadata) {
	for _, uri := range meta.FromURIs {
		candidateComp, scheme := i.decodeComponent(uri)
		if candidateComp != nil {
			i.addDependency(candidateComp.GetDependencyID(), meta)
			if scheme != nil {
				for _, dep := range candidateComp.GetConsumerDependencyIDs(scheme.ID) {
					i.addDependency(dep, meta)
				}
			}
		}
	}

	for _, uri := range meta.ToURIs {
		candidateComp, scheme := i.decodeComponent(uri)
		if candidateComp != nil {
			i.addDependency(candidateComp.GetDependencyID(), meta)
			if scheme != nil {
				for _, dep := range candidateComp.GetProducerDependencyIDs(scheme.ID) {
					i.addDependency(dep, meta)
				}
			}
		}
	}

	for pattern, supplier := range sourceDependencies {
		if !pattern.MatchString(source.Content) {
			continue
		}

		for _, dep := range supplier(i.catalog) {
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
	meta.Dependencies.Add(dependency)
}

func (i *baseInspector) decodeComponent(uri string) (*v1.CamelArtifact, *v1.CamelScheme) {
	uriSplit := strings.SplitN(uri, ":", 2)
	if len(uriSplit) < 2 {
		return nil, nil
	}
	uriStart := uriSplit[0]
	scheme, ok := i.catalog.GetScheme(uriStart)
	var schemeRef *v1.CamelScheme
	if ok {
		schemeRef = &scheme
	}
	return i.catalog.GetArtifactByScheme(uriStart), schemeRef
}

// hasOnlyPassiveEndpoints returns true if the source has no endpoint that needs to remain always active
func (i *baseInspector) hasOnlyPassiveEndpoints(fromURIs []string) bool {
	passivePlusHTTP := make(map[string]bool)
	i.catalog.VisitSchemes(func(id string, scheme v1.CamelScheme) bool {
		if scheme.HTTP || scheme.Passive {
			passivePlusHTTP[id] = true
		}

		return true
	})

	return i.containsOnlyURIsIn(fromURIs, passivePlusHTTP)
}

func (i *baseInspector) containsOnlyURIsIn(fromURI []string, allowed map[string]bool) bool {
	for _, uri := range fromURI {
		if uri == "kamelet:source" {
			continue
		}
		prefix := i.getURIPrefix(uri)
		if enabled, ok := allowed[prefix]; !ok || !enabled {
			return false
		}
	}
	return true
}

func (i *baseInspector) getURIPrefix(uri string) string {
	parts := strings.SplitN(uri, ":", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func (i *baseInspector) containsHTTPURIs(fromURI []string) bool {
	for _, uri := range fromURI {
		prefix := i.getURIPrefix(uri)
		scheme, ok := i.catalog.GetScheme(prefix)

		if !ok {
			// scheme does not exists
			continue
		}

		if scheme.HTTP {
			return true
		}
	}

	return false
}
