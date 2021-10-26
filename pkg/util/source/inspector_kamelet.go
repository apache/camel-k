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
)

var kameletNameRegexp = regexp.MustCompile("kamelet:(?://)?([a-z0-9-.]+(/[a-z0-9-.]+)?)(?:$|[^a-z0-9-.].*)")

func ExtractKamelets(uris []string) (kamelets []string) {
	for _, uri := range uris {
		kamelet := ExtractKamelet(uri)
		if kamelet != "" {
			kamelets = append(kamelets, kamelet)
		}
	}
	return
}

func ExtractKamelet(uri string) (kamelet string) {
	matches := kameletNameRegexp.FindStringSubmatch(uri)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func AddKamelet(meta *Metadata, content string) {
	if maybeKamelet := ExtractKamelet(content); maybeKamelet != "" {
		meta.Kamelets = append(meta.Kamelets, maybeKamelet)
	}
}
