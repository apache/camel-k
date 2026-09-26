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
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VolatileTime wraps metav1.Time. Unlike metav1.Time, two VolatileTime values are always
// semantically equal, so that a change of a condition transition time alone is not
// considered a status change.
type VolatileTime struct { //nolint: recvcheck
	Inner metav1.Time `json:",inline"`
}

func init() {
	_ = equality.Semantic.AddFunc(func(VolatileTime, VolatileTime) bool {
		return true
	})
}

// MarshalJSON implements json.Marshaler.
func (t VolatileTime) MarshalJSON() ([]byte, error) {
	return t.Inner.MarshalJSON()
}

// UnmarshalJSON implements json.Unmarshaler.
func (t *VolatileTime) UnmarshalJSON(b []byte) error {
	return t.Inner.UnmarshalJSON(b)
}
