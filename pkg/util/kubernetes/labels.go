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
	"os"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

const CamelDashboardAppLabelEnvVar = "CAMEL_DASHBOARD_APP_LABEL"

// DeploymentLabels returns the fixed labels assigned to each Camel workload.
func DeploymentLabels(integrationName string) map[string]string {
	labels := map[string]string{
		// Required by Camel K
		v1.IntegrationLabel: integrationName,
	}
	camelDashboardLabel, ok := os.LookupEnv(CamelDashboardAppLabelEnvVar)
	if ok && camelDashboardLabel != "" {
		// Will automatically enable App discovery by Camel Dashboard
		labels[camelDashboardLabel] = integrationName
	}

	return labels
}
