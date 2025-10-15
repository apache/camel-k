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
	"net/url"
	"regexp"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

var kameletNameRegexp = regexp.MustCompile("kamelet:(?://)?([a-z0-9-.]+(/[a-z0-9-.]+)?)(?:$|[^a-z0-9-.].*)")

func ExtractKamelet(uri string) string {
	matches := kameletNameRegexp.FindStringSubmatch(uri)
	if len(matches) > 1 {
		version := getKameletParam(uri, v1.KameletVersionProperty)
		namespace := getKameletParam(uri, v1.KameletNamespaceProperty)
		return GetKameletQuerystring(matches[1], version, namespace)
	}
	return ""
}

func AddKamelet(meta *Metadata, content string) {
	if maybeKamelet := ExtractKamelet(content); maybeKamelet != "" {
		meta.Kamelets = append(meta.Kamelets, maybeKamelet)
	}
}

// getKameletParam parses the URI and return the query parameter or an empty value if not found.
func getKameletParam(uri, param string) string {
	parsedURL, err := url.Parse(uri)
	if err != nil {
		return ""
	}

	queryParams := parsedURL.Query()
	return queryParams.Get(param)
}

// GetKameletQuerystring returns a kamelet name appended with its version and namespace (if provided).
func GetKameletQuerystring(name, version, namespace string) string {
	if version != "" || namespace != "" {
		var querystring string
		if version != "" {
			querystring = v1.KameletVersionProperty + "=" + version
		}
		if namespace != "" {
			if querystring != "" {
				querystring += "&"
			}
			querystring += v1.KameletNamespaceProperty + "=" + namespace
		}
		return fmt.Sprintf("%s?%s", name, querystring)
	}

	return name
}
