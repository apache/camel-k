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

package keda

import (
	"net/url"
	"strings"

	"github.com/apache/camel-k/v2/pkg/util/uri"
)

// ParseComponentURI extracts the component scheme, path value, and query parameters from a Camel URI.
func ParseComponentURI(rawURI string) (string, string, map[string]string, error) {
	scheme := uri.GetComponent(rawURI)
	if scheme == "" {
		return "", "", nil, nil
	}

	params := make(map[string]string)

	// extract path
	remainder := strings.TrimPrefix(rawURI, scheme+":")
	var pathValue string
	if before, after, ok := strings.Cut(remainder, "?"); ok {
		pathValue = before
		queryString := after

		values, parseErr := url.ParseQuery(queryString)
		if parseErr != nil {
			return "", "", nil, parseErr
		}
		for k, v := range values {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}
	} else {
		pathValue = remainder
	}

	return scheme, pathValue, params, nil
}
