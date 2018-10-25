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
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

// discoverLanguage discovers the code language from file extension if not set
func discoverLanguage(source v1alpha1.SourceSpec) v1alpha1.Language {
	if source.Language != "" {
		return source.Language
	}
	for _, l := range []v1alpha1.Language{
		v1alpha1.LanguageJavaSource,
		v1alpha1.LanguageJavaClass,
		v1alpha1.LanguageJavaScript,
		v1alpha1.LanguageGroovy,
		v1alpha1.LanguageJavaScript,
		v1alpha1.LanguageKotlin} {

		if strings.HasSuffix(source.Name, "."+string(l)) {
			return l
		}
	}
	return ""
}
