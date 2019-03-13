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

package kubernetes

import (
	"testing"
)

func TestSanitizeName(t *testing.T) {
	cases := []map[string]string{
		{"input": "./abc.java", "expect": "abc"},
		{"input": "../../abc.java", "expect": "abc"},
		{"input": "/path/to/abc.js", "expect": "abc"},
		{"input": "abc.xml", "expect": "abc"},
		{"input": "./path/to/abc.kts", "expect": "abc"},
		{"input": "fooToBar.groovy", "expect": "foo-to-bar"},
		{"input": "foo-to-bar", "expect": "foo-to-bar"},
		{"input": "http://foo.bar.com/cheese/wine/beer/abc.java", "expect": "abc"},
		{"input": "http://foo.bar.com/cheese", "expect": "cheese"},
		{"input": "http://foo.bar.com", "expect": "foo"},
	}

	for _, c := range cases {
		if name := SanitizeName(c["input"]); name != c["expect"] {
			t.Errorf("result of %s should be %s, instead of %s", c["input"], c["expect"], name)
		}
	}
}
