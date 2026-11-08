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

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util"
)

var (
	replaceURIDoubleQuotedFrom  = regexp.MustCompile(`from\s*\(\s*"(?:timer|cron|quartz)[^"]*"\s*\)`)
	replaceURIDoubleQuotedFromF = regexp.MustCompile(`fromF\s*\(\s*"(?:timer|cron|quartz)[^"]*"\s*\)`)
	bindToRegistryClassRegexp = regexp.MustCompile(`(?s)@(?:org\.apache\.camel\.)?BindToRegistry\s*(?:\(\s*(?:"([^"]*)"|[^)]*\bvalue\s*=\s*"([^"]*)")?[^)]*\))?\s*(?:(?:public|protected|private|abstract|final|static|strictfp)\s+)*(?:class|interface|enum|record)\s+([A-Za-z_$][A-Za-z0-9_$]*)\b`)
	bindToRegistryMethodRegexp = regexp.MustCompile(`(?s)@(?:org\.apache\.camel\.)?BindToRegistry\s*(?:\(\s*(?:"([^"]*)"|[^)]*\bvalue\s*=\s*"([^"]*)")?[^)]*\))?\s*(?:(?:public|protected|private|abstract|final|static|strictfp|synchronized|native)\s+)*([A-Za-z_$][A-Za-z0-9_$.<>?,\[\] ]*)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*\(`)
	bindToRegistryFieldRegexp = regexp.MustCompile(`(?s)@(?:org\.apache\.camel\.)?BindToRegistry\s*(?:\(\s*(?:"([^"]*)"|[^)]*\bvalue\s*=\s*"([^"]*)")?[^)]*\))?\s*(?:(?:public|protected|private|abstract|final|static|transient|volatile)\s+)*([A-Za-z_$][A-Za-z0-9_$.<>?,\[\] ]*)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*(?:=|;)`)
	javaPackageRegexp = regexp.MustCompile(`(?m)^\s*package\s+([A-Za-z_$][A-Za-z0-9_$.]*)\s*;`)
)

// JavaSourceInspector inspects Java DSL spec.
type JavaSourceInspector struct {
	baseInspector
}

// Extract extracts all metadata from source spec.
func (i JavaSourceInspector) Extract(source v1.SourceSpec, meta *Metadata) error {
	parseJavaBeanDefinitions(source.Content, meta)

	from := util.FindAllDistinctStringSubmatch(
		source.Content,
		doubleQuotedFrom,
		doubleQuotedFromF,
	)
	to := util.FindAllDistinctStringSubmatch(
		source.Content,
		doubleQuotedTo,
		doubleQuotedToD,
		doubleQuotedToF,
		doubleQuotedWireTap,
	)
	kameletEips := util.FindAllDistinctStringSubmatch(
		source.Content,
		doubleQuotedKameletEip)

	hasRest := restRegexp.MatchString(source.Content)

	return i.extract(source, meta, from, to, kameletEips, hasRest)
}

// parseJavaBeanDefinitions records beans declared with @BindToRegistry.
// It reads only annotation names and declared Java types; it does not evaluate Java code.
func parseJavaBeanDefinitions(content string, meta *Metadata) {
	packageName := ""
	if match := javaPackageRegexp.FindStringSubmatch(content); len(match) == 2 {
		packageName = match[1]
	}

	for _, match := range bindToRegistryClassRegexp.FindAllStringSubmatch(content, -1) {
		if len(match) != 4 {
			continue
		}
		name := bindToRegistryName(match[1], match[2], decapitalizeJavaName(match[3]))
		meta.Beans = append(meta.Beans, BeanDefinition{
			Name:       name,
			Type:       qualifyJavaBeanType(match[3], packageName),
			Properties: make(map[string]string),
		})
	}

	for _, match := range bindToRegistryMethodRegexp.FindAllStringSubmatch(content, -1) {
		if len(match) != 5 {
			continue
		}
		beanType := strings.TrimSpace(match[3])
		if beanType == "" || beanType == "void" {
			continue
		}
		name := bindToRegistryName(match[1], match[2], match[4])
		meta.Beans = append(meta.Beans, BeanDefinition{
			Name:       name,
			Type:       qualifyJavaBeanType(beanType, packageName),
			Properties: make(map[string]string),
		})
	}

	for _, match := range bindToRegistryFieldRegexp.FindAllStringSubmatch(content, -1) {
		if len(match) != 5 {
			continue
		}
		beanType := strings.TrimSpace(match[3])
		name := bindToRegistryName(match[1], match[2], match[4])
		meta.Beans = append(meta.Beans, BeanDefinition{
			Name:       name,
			Type:       qualifyJavaBeanType(beanType, packageName),
			Properties: make(map[string]string),
		})
	}
}

func bindToRegistryName(positional, named, fallback string) string {
	name := strings.TrimSpace(positional)
	if name == "" {
		name = strings.TrimSpace(named)
	}
	if name == "" {
		return fallback
	}
	return name
}

func qualifyJavaBeanType(beanType, packageName string) string {
	beanType = strings.TrimSpace(beanType)
	if packageName == "" || beanType == "" || strings.ContainsAny(beanType, ".<>[], ") {
		return beanType
	}
	return packageName + "." + beanType
}

func decapitalizeJavaName(name string) string {
	runes := []rune(name)
	if len(runes) == 0 {
		return name
	}
	if len(runes) > 1 && unicode.IsUpper(runes[0]) && unicode.IsUpper(runes[1]) {
		return name
	}
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// ReplaceFromURI parses the source content and replace the `from` URI configuration with the a new URI. Returns true if it applies a replacement.
func (i JavaSourceInspector) ReplaceFromURI(source *v1.SourceSpec, newFromURI string) (bool, error) {
	return replaceFromURIDoubleQuotesOnly(source, newFromURI)
}

func replaceFromURIDoubleQuotesOnly(source *v1.SourceSpec, newFromURI string) (bool, error) {
	originalContent := source.Content

	source.Content = replaceURIDoubleQuotedFrom.ReplaceAllString(source.Content, fmt.Sprintf(`from("%s")`, newFromURI))
	source.Content = replaceURIDoubleQuotedFromF.ReplaceAllString(source.Content, fmt.Sprintf(`fromF('%s')`, newFromURI))

	return originalContent != source.Content, nil
}
