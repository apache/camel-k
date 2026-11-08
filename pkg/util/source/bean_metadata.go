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
	"strings"
)

// isSensitiveBeanProperty avoids retaining credentials in source metadata.
func isSensitiveBeanProperty(key string) bool {
	normalized := strings.ToLower(strings.NewReplacer("-", "", "_", "", ".", "").Replace(key))
	for _, sensitive := range []string{"password", "passwd", "username", "user", "secret", "token", "credential", "privatekey", "clientkey", "accesskey"} {
		if strings.Contains(normalized, sensitive) {
			return true
		}
	}

	return false
}

// scalarBeanProperty returns supported scalar bean property values.
// Nested objects and sequences are intentionally left unresolved.
func scalarBeanProperty(value any) (string, bool) {
	switch value.(type) {
	case string, bool, int, int64, uint64, float32, float64:
		valueString := fmt.Sprint(value)
		if containsSensitiveURLData(valueString) {
			return "", false
		}

		return valueString, true
	default:
		return "", false
	}
}

func containsSensitiveURLData(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	if parsed.User != nil {
		return true
	}
	for key := range parsed.Query() {
		if isSensitiveBeanProperty(key) {
			return true
		}
	}

	return false
}
