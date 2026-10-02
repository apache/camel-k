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

package apis

import (
	"encoding/json"
	"net/url"
	"strconv"

	"k8s.io/apimachinery/pkg/api/equality"
)

// URL is an alias of url.URL with custom JSON marshalling: it is serialized as a plain string.
type URL url.URL //nolint: recvcheck

func init() {
	// url.URL has an unexported field (url.Userinfo) that breaks the reflection based
	// semantic equality, hence the custom equality function.
	_ = equality.Semantic.AddFunc(func(a, b URL) bool {
		return a.String() == b.String()
	})
}

// ParseURL attempts to parse the given string as a URL. An empty string yields a nil URL.
func ParseURL(u string) (*URL, error) {
	if u == "" {
		return nil, nil
	}
	pu, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	return (*URL)(pu), nil
}

// MarshalJSON implements json.Marshaler.
func (u URL) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(u.String())), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (u *URL) UnmarshalJSON(b []byte) error {
	var ref string
	if err := json.Unmarshal(b, &ref); err != nil {
		return err
	}
	r, err := ParseURL(ref)
	if err != nil {
		return err
	}
	if r != nil {
		*u = *r
	} else {
		*u = URL{}
	}

	return nil
}

// String returns the string representation of the URL, or an empty string for a nil URL.
func (u *URL) String() string {
	if u == nil {
		return ""
	}
	uu := url.URL(*u)

	return uu.String()
}
