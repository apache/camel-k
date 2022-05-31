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

package modeline

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/mattn/go-shellwords"
)

var (
	commonModelineRegexp = regexp.MustCompile(`^\s*//\s*camel-k\s*:\s*([^\s]+.*)$`)
	yamlModelineRegexp   = regexp.MustCompile(`^\s*#+\s*camel-k\s*:\s*([^\s]+.*)$`)
	xmlModelineRegexp    = regexp.MustCompile(`^.*<!--\s*camel-k\s*:\s*([^\s]+[^>]*)-->.*$`)
)

func Parse(name, content string) (res []Option, err error) {
	lang := inferLanguage(name)
	if lang == "" {
		return nil, fmt.Errorf("unsupported file type %s", name)
	}
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		res = append(res, getModelineOptions(scanner.Text(), lang)...)
	}
	return res, scanner.Err()
}

func getModelineOptions(line string, lang v1.Language) (res []Option) {
	reg := modelineRegexp(lang)
	if !reg.MatchString(line) {
		return nil
	}
	strs := reg.FindStringSubmatch(line)
	if len(strs) == 2 {
		tokens, _ := shellwords.Parse(strs[1])
		for _, token := range tokens {
			if len(strings.Trim(token, "\t\n\f\r ")) == 0 {
				continue
			}
			eq := strings.Index(token, "=")
			var name, value string
			if eq > 0 {
				name = token[0:eq]
				value = token[eq+1:]
			} else {
				name = token
				value = ""
			}
			opt := Option{
				Name:  name,
				Value: value,
			}
			res = append(res, opt)
		}
	}
	return res
}

func modelineRegexp(lang v1.Language) *regexp.Regexp {
	switch lang {
	case v1.LanguageYaml:
		return yamlModelineRegexp
	case v1.LanguageXML:
		return xmlModelineRegexp
	default:
		return commonModelineRegexp
	}
}

func inferLanguage(fileName string) v1.Language {
	for _, l := range v1.Languages {
		if strings.HasSuffix(fileName, fmt.Sprintf(".%s", string(l))) {
			return l
		}
	}
	if strings.HasSuffix(fileName, ".yml") {
		return v1.LanguageYaml
	}
	return ""
}
