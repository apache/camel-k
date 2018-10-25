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
)

var (
	singleQuotedFrom = regexp.MustCompile("from\\s*\\(\\s*'([a-z0-9-]+:[^']+)'\\s*\\)")
	doubleQuotedFrom = regexp.MustCompile("from\\s*\\(\\s*\"([a-z0-9-]+:[^\"]+)\"\\s*\\)")
	singleQuotedTo   = regexp.MustCompile("\\.to\\s*\\(\\s*'([a-z0-9-]+:[^']+)'\\s*\\)")
	singleQuotedToD  = regexp.MustCompile("\\.toD\\s*\\(\\s*'([a-z0-9-]+:[^']+)'\\s*\\)")
	singleQuotedToF  = regexp.MustCompile("\\.toF\\s*\\(\\s*'([a-z0-9-]+:[^']+)'[^)]*\\)")
	doubleQuotedTo   = regexp.MustCompile("\\.to\\s*\\(\\s*\"([a-z0-9-]+:[^\"]+)\"\\s*\\)")
	doubleQuotedToD  = regexp.MustCompile("\\.toD\\s*\\(\\s*\"([a-z0-9-]+:[^\"]+)\"\\s*\\)")
	doubleQuotedToF  = regexp.MustCompile("\\.toF\\s*\\(\\s*\"([a-z0-9-]+:[^\"]+)\"[^)]*\\)")
	xmlTagFrom       = regexp.MustCompile("<\\s*from\\s+[^>]*uri\\s*=\\s*\"([a-z0-9-]+:[^\"]+)\"[^>]*>")
	xmlTagTo         = regexp.MustCompile("<\\s*to\\s+[^>]*uri\\s*=\\s*\"([a-z0-9-]+:[^\"]+)\"[^>]*>")
	xmlTagToD        = regexp.MustCompile("<\\s*toD\\s+[^>]*uri\\s*=\\s*\"([a-z0-9-]+:[^\"]+)\"[^>]*>")
)

// discoverFromURIs returns all uris used in a from clause
func discoverFromURIs(source v1alpha1.SourceSpec, language v1alpha1.Language) []string {
	fromRegexps := getFromRegexpsForLanguage(language)
	return findAllDistinctStringSubmatch(source.Content, fromRegexps...)
}

// discoverToURIs returns all uris used in a to clause
func discoverToURIs(source v1alpha1.SourceSpec, language v1alpha1.Language) []string {
	toRegexps := getToRegexpsForLanguage(language)
	return findAllDistinctStringSubmatch(source.Content, toRegexps...)
}

func getFromRegexpsForLanguage(language v1alpha1.Language) []*regexp.Regexp {
	switch language {
	case v1alpha1.LanguageJavaSource:
		return []*regexp.Regexp{doubleQuotedFrom}
	case v1alpha1.LanguageXML:
		return []*regexp.Regexp{xmlTagFrom}
	case v1alpha1.LanguageGroovy:
		return []*regexp.Regexp{singleQuotedFrom, doubleQuotedFrom}
	case v1alpha1.LanguageJavaScript:
		return []*regexp.Regexp{singleQuotedFrom, doubleQuotedFrom}
	case v1alpha1.LanguageKotlin:
		return []*regexp.Regexp{doubleQuotedFrom}
	}
	return []*regexp.Regexp{}
}

func getToRegexpsForLanguage(language v1alpha1.Language) []*regexp.Regexp {
	switch language {
	case v1alpha1.LanguageJavaSource:
		return []*regexp.Regexp{doubleQuotedTo, doubleQuotedToD, doubleQuotedToF}
	case v1alpha1.LanguageXML:
		return []*regexp.Regexp{xmlTagTo, xmlTagToD}
	case v1alpha1.LanguageGroovy:
		return []*regexp.Regexp{singleQuotedTo, doubleQuotedTo, singleQuotedToD, doubleQuotedToD, singleQuotedToF, doubleQuotedToF}
	case v1alpha1.LanguageJavaScript:
		return []*regexp.Regexp{singleQuotedTo, doubleQuotedTo, singleQuotedToD, doubleQuotedToD, singleQuotedToF, doubleQuotedToF}
	case v1alpha1.LanguageKotlin:
		return []*regexp.Regexp{doubleQuotedTo, doubleQuotedToD, doubleQuotedToF}
	}
	return []*regexp.Regexp{}
}

func findAllDistinctStringSubmatch(data string, regexps ...*regexp.Regexp) []string {
	candidates := make([]string, 0)
	alreadyFound := make(map[string]bool)
	for _, reg := range regexps {
		hits := reg.FindAllStringSubmatch(data, -1)
		for _, hit := range hits {
			if hit != nil && len(hit) > 1 {
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