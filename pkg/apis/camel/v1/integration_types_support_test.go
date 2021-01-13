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

package v1

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllLanguages(t *testing.T) {
	assert.Contains(t, Languages, LanguageJavaSource)
	assert.Contains(t, Languages, LanguageJavaScript)
	assert.Contains(t, Languages, LanguageGroovy)
	assert.Contains(t, Languages, LanguageKotlin)
	assert.Contains(t, Languages, LanguageXML)
	assert.Contains(t, Languages, LanguageYaml)
}

func TestLanguageFromName(t *testing.T) {
	for _, l := range Languages {
		language := l
		t.Run(string(language), func(t *testing.T) {
			code := SourceSpec{
				DataSpec: DataSpec{
					Name: fmt.Sprintf("code.%s", language),
				},
			}

			if language != code.InferLanguage() {
				t.Errorf("got %s, want %s", code.InferLanguage(), language)
			}
		})
	}
}

func TestLanguageAlreadySet(t *testing.T) {
	code := SourceSpec{
		DataSpec: DataSpec{
			Name: "Request.java",
		},
		Language: LanguageJavaScript,
	}
	assert.Equal(t, LanguageJavaScript, code.InferLanguage())
}

func TestAddDependency(t *testing.T) {
	integration := IntegrationSpec{}
	integration.AddDependency("camel-file")
	assert.Equal(t, integration.Dependencies, []string{"camel:file"})

	integration = IntegrationSpec{}
	integration.AddDependency("mvn:com.my:company")
	assert.Equal(t, integration.Dependencies, []string{"mvn:com.my:company"})

	integration = IntegrationSpec{}
	integration.AddDependency("file:dep")
	assert.Equal(t, integration.Dependencies, []string{"file:dep"})
}
