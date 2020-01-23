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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (in *Artifact) String() string {
	return in.ID
}

func (in *ConfigurationSpec) String() string {
	return fmt.Sprintf("%s=%s", in.Type, in.Value)
}

// NewErrorFailure --
func NewErrorFailure(err error) *Failure {
	return &Failure{
		Reason: err.Error(),
		Time:   metav1.Now(),
	}
}
