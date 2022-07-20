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

package label

import (
	"fmt"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// The AdditionalLabels is a big string consisting of key=value, set at build time
// when set are to be added to Deployments, CronJob and KNativeService whose pods
// should expose these labels

// AdditionalLabels are labels=values, they MUST be set as key=value separated by comma ,
// example: myKey1=myValue1,myKey2=myValue2
// Also it supports replacing a value for the integration name at runtime, just use the value as
// "token_integration_name"
// example: myKey1=myValue1,myKey2=token_integration_name
var AdditionalLabels = ""

var FixedLabels = map[string]string{}

// parses the labels early on and fail fast if there are errors.
func init() {
	checkAdditionalLabels()
}

func checkAdditionalLabels() {
	if len(AdditionalLabels) > 0 {
		var err error
		FixedLabels, err = labels.ConvertSelectorToLabelsMap(AdditionalLabels)
		if err != nil {
			// as this should be used only in build time, it's ok to fail fast
			panic(fmt.Sprintf("Error parsing AdditionalLabels %s, Error: %s\n", AdditionalLabels, err))
		}
	}
}

// parses the AdditionalLabels variable and returns as map[string]string.
func AddLabels(integration string) map[string]string {
	definitiveLabels := labels.Set{
		v1.IntegrationLabel: integration,
	}
	for k, v := range FixedLabels {
		if v == "token_integration_name" {
			definitiveLabels[k] = integration
		} else {
			definitiveLabels[k] = v
		}
	}
	return definitiveLabels
}
