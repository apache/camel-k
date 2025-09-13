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
	kameletsv1 "github.com/apache/camel-kamelets/crds/pkg/apis/camel/v1"
)

var kameletNameRegexp = regexp.MustCompile("kamelet:(?://)?([a-z0-9-.]+(/[a-z0-9-.]+)?)(?:$|[^a-z0-9-.].*)")
var kameletVersionRegexp = regexp.MustCompile(kameletsv1.KameletVersionProperty + "=([a-z0-9-.]+)")

func ExtractKamelet(uri string) string {
	matches := kameletNameRegexp.FindStringSubmatch(uri)
	if len(matches) > 1 {
		version := kameletVersionRegexp.FindString(uri)
		if version != "" {
			return fmt.Sprintf("%s?%s", matches[1], version)
		}
		return matches[1]
	}
	return ""
}

func AddKamelet(meta *Metadata, content string) {
	if maybeKamelet := ExtractKamelet(content); maybeKamelet != "" {
		meta.Kamelets = append(meta.Kamelets, maybeKamelet)
	}
}

// ToIntegrationSource converts the Kamelet source spec to an Integration source spec.
func ToIntegrationSource(source kameletsv1.SourceSpec) v1.SourceSpec {
	itSource := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name:        source.DataSpec.Name,
			Path:        source.DataSpec.Path,
			Content:     source.DataSpec.Content,
			RawContent:  source.DataSpec.RawContent,
			ContentRef:  source.DataSpec.ContentRef,
			ContentKey:  source.DataSpec.ContentKey,
			ContentType: source.DataSpec.ContentType,
			Compression: source.DataSpec.Compression,
		},
		Language:      v1.Language(string(source.Language)),
		Loader:        source.Loader,
		Interceptors:  source.Interceptors,
		Type:          v1.SourceType(string(source.Type)),
		PropertyNames: source.PropertyNames,
		FromKamelet:   source.FromKamelet,
	}

	return itSource
}
