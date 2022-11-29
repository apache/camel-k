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
	"fmt"
	"regexp"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/camel"
)

type catalog2deps func(*camel.RuntimeCatalog) []string

const (
	defaultJSONDataFormat = "jackson"
)

var (
	singleQuotedFrom        = regexp.MustCompile(`from\s*\(\s*'([a-zA-Z0-9-]+:[^']+)'`)
	singleQuotedFromF       = regexp.MustCompile(`fromF\s*\(\s*'([a-zA-Z0-9-]+:[^']+)'`)
	doubleQuotedFrom        = regexp.MustCompile(`from\s*\(\s*"([a-zA-Z0-9-]+:[^"]+)"`)
	doubleQuotedFromF       = regexp.MustCompile(`fromF\s*\(\s*"([a-zA-Z0-9-]+:[^"]+)"`)
	singleQuotedTo          = regexp.MustCompile(`\.to\s*\(\s*'([a-zA-Z0-9-]+:[^']+)'`)
	singleQuotedToD         = regexp.MustCompile(`\.toD\s*\(\s*'([a-zA-Z0-9-]+:[^']+)'`)
	singleQuotedToF         = regexp.MustCompile(`\.toF\s*\(\s*'([a-zA-Z0-9-]+:[^']+)'`)
	singleQuotedWireTap     = regexp.MustCompile(`\.wireTap\s*\(\s*'([a-zA-Z0-9-]+:[^']+)'`)
	doubleQuotedTo          = regexp.MustCompile(`\.to\s*\(\s*"([a-zA-Z0-9-]+:[^"]+)"`)
	doubleQuotedToD         = regexp.MustCompile(`\.toD\s*\(\s*"([a-zA-Z0-9-]+:[^"]+)"`)
	doubleQuotedToF         = regexp.MustCompile(`\.toF\s*\(\s*"([a-zA-Z0-9-]+:[^"]+)"`)
	doubleQuotedWireTap     = regexp.MustCompile(`\.wireTap\s*\(\s*"([a-zA-Z0-9-]+:[^"]+)"`)
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
	singleQuotedKameletEip  = regexp.MustCompile(`kamelet\s*\(\s*'(?://)?([a-z0-9-.]+(/[a-z0-9-.]+)?)(?:$|[^a-z0-9-.].*)'`)
	doubleQuotedKameletEip  = regexp.MustCompile(`kamelet\s*\(\s*"(?://)?([a-z0-9-.]+(/[a-z0-9-.]+)?)(?:$|[^a-z0-9-.].*)"`)

	sourceCapabilities = map[*regexp.Regexp][]string{
		circuitBreakerRegexp: {v1.CapabilityCircuitBreaker},
	}

	sourceDependencies = map[*regexp.Regexp]catalog2deps{
		jsonLibraryRegexp: func(catalog *camel.RuntimeCatalog) []string {
			res := make([]string, 0)
			if jsonDF := catalog.GetArtifactByDataFormat(defaultJSONDataFormat); jsonDF != nil {
				res = append(res, jsonDF.GetDependencyID())
			}
			return res
		},
		jsonLanguageRegexp: func(catalog *camel.RuntimeCatalog) []string {
			res := make([]string, 0)
			if jsonDF := catalog.GetArtifactByDataFormat(defaultJSONDataFormat); jsonDF != nil {
				res = append(res, jsonDF.GetDependencyID())
			}
			return res
		},
		restConfigurationRegexp: func(catalog *camel.RuntimeCatalog) []string {
			deps := make([]string, 0)
			if c, ok := catalog.CamelCatalogSpec.Runtime.Capabilities[v1.CapabilityRest]; ok {
				for _, d := range c.Dependencies {
					deps = append(deps, d.GetDependencyID())
				}
			}
			return deps
		},
		restRegexp: func(catalog *camel.RuntimeCatalog) []string {
			deps := make([]string, 0)
			if c, ok := catalog.CamelCatalogSpec.Runtime.Capabilities[v1.CapabilityRest]; ok {
				for _, d := range c.Dependencies {
					deps = append(deps, d.GetDependencyID())
				}
			}
			return deps
		},
		restClosureRegexp: func(catalog *camel.RuntimeCatalog) []string {
			deps := make([]string, 0)
			if c, ok := catalog.CamelCatalogSpec.Runtime.Capabilities[v1.CapabilityRest]; ok {
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

// Inspector is the common interface for language specific inspector implementations.
type Inspector interface {
	Extract(v1.SourceSpec, *Metadata) error
}

// InspectorForLanguage is the factory function to return a new inspector for the given language
// with the catalog.
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

func (i *baseInspector) extract(source v1.SourceSpec, meta *Metadata,
	from, to, kameletEips []string, hasRest bool) error {
	meta.FromURIs = append(meta.FromURIs, from...)
	meta.ToURIs = append(meta.ToURIs, to...)

	for _, k := range kameletEips {
		AddKamelet(meta, "kamelet:"+k)
	}

	if err := i.discoverCapabilities(source, meta); err != nil {
		return err
	}
	if err := i.discoverDependencies(source, meta); err != nil {
		return err
	}
	i.discoverKamelets(meta)

	if hasRest {
		meta.AddRequiredCapability(v1.CapabilityRest)
	}

	meta.ExposesHTTPServices = hasRest || i.containsHTTPURIs(meta.FromURIs)
	meta.PassiveEndpoints = i.hasOnlyPassiveEndpoints(meta.FromURIs)

	return nil
}

// discoverCapabilities returns a list of dependencies required by the given source code.
func (i *baseInspector) discoverCapabilities(source v1.SourceSpec, meta *Metadata) error {
	uris := util.StringSliceJoin(meta.FromURIs, meta.ToURIs)

	for _, uri := range uris {
		if i.getURIPrefix(uri) == "platform-http" {
			meta.AddRequiredCapability(v1.CapabilityPlatformHTTP)
		}
	}

	for pattern, capabilities := range sourceCapabilities {
		if !pattern.MatchString(source.Content) {
			continue
		}

		for _, capability := range capabilities {
			meta.AddRequiredCapability(capability)
		}
	}

	// validate capabilities
	var err error
	meta.RequiredCapabilities.Each(func(capability string) bool {
		if !i.catalog.HasCapability(capability) {
			err = fmt.Errorf("capability %s not supported in camel catalog runtime version %s",
				capability, i.catalog.GetRuntimeVersion())
			return false
		}
		return true
	})

	return err
}

// discoverDependencies returns a list of dependencies required by the given source code.
func (i *baseInspector) discoverDependencies(source v1.SourceSpec, meta *Metadata) error {
	for _, uri := range meta.FromURIs {
		// consumer
		if err := i.addDependencies(uri, meta, true); err != nil {
			return err
		}
	}

	for _, uri := range meta.ToURIs {
		// producer
		if err := i.addDependencies(uri, meta, false); err != nil {
			return err
		}
	}

	for pattern, supplier := range sourceDependencies {
		if !pattern.MatchString(source.Content) {
			continue
		}

		for _, dep := range supplier(i.catalog) {
			meta.AddDependency(dep)
		}
	}

	for _, match := range languageRegexp.FindAllStringSubmatch(source.Content, -1) {
		if len(match) > 1 {
			if dependency, ok := i.catalog.GetLanguageDependency(match[1]); ok {
				meta.AddDependency(dependency)
			}
		}
	}

	for _, match := range camelTypeRegexp.FindAllStringSubmatch(source.Content, -1) {
		if len(match) > 1 {
			if dependency, ok := i.catalog.GetJavaTypeDependency(match[1]); ok {
				meta.AddDependency(dependency)
			}
		}
	}

	return nil
}

// discoverKamelets inspects endpoints and extract kamelets.
func (i *baseInspector) discoverKamelets(meta *Metadata) {
	for _, uri := range meta.FromURIs {
		AddKamelet(meta, uri)
	}
	for _, uri := range meta.ToURIs {
		AddKamelet(meta, uri)
	}
}

func (i *baseInspector) addDependencies(uri string, meta *Metadata, consumer bool) error {
	candidateComp, scheme := i.catalog.DecodeComponent(uri)
	if candidateComp == nil || scheme == nil {
		return fmt.Errorf("component not found for uri %q in camel catalog runtime version %s",
			uri, i.catalog.GetRuntimeVersion())
	}

	meta.AddDependency(candidateComp.GetDependencyID())
	var deps []string
	if consumer {
		deps = candidateComp.GetConsumerDependencyIDs(scheme.ID)
	} else {
		deps = candidateComp.GetProducerDependencyIDs(scheme.ID)
	}
	for _, dep := range deps {
		meta.AddDependency(dep)
	}

	// some components require additional dependency resolution based on URI
	if err := i.addDependenciesFromURI(uri, scheme, meta); err != nil {
		return err
	}

	return nil
}

func (i *baseInspector) addDependenciesFromURI(uri string, scheme *v1.CamelScheme, meta *Metadata) error {
	if scheme.ID == "dataformat" {
		// dataformat:name:(marshal|unmarshal)[?options]
		parts := strings.Split(uri, ":")
		if len(parts) < 3 {
			return fmt.Errorf("invalid dataformat uri: %s", uri)
		}
		name := parts[1]
		df := i.catalog.GetArtifactByDataFormat(name)
		if df == nil {
			return fmt.Errorf("dataformat %q not found: %s", name, uri)
		}
		meta.AddDependency(df.GetDependencyID())
	}

	return nil
}

// hasOnlyPassiveEndpoints returns true if the source has no endpoint that needs to remain always active.
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
