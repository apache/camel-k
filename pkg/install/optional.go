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

package install

import (
	"context"

	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/platform"
	logutil "github.com/apache/camel-k/v2/pkg/util/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OperatorStartupOptionalTools tries to install optional tools at operator startup and warns if something goes wrong.
func OperatorStartupOptionalTools(ctx context.Context, c client.Client, namespace string, operatorNamespace string, log logutil.Logger) {
	// Try to register the OpenShift CLI Download link if possible
	if err := OpenShiftConsoleDownloadLink(ctx, c); err != nil {
		log.Info("Cannot install OpenShift CLI download link: skipping.")
		log.Debug("Error while installing OpenShift CLI download link", "error", err)
	}
	// Check the presence of a registry service configuration, and, if it exists, calculate the address
	registryServiceName := platform.GetEnvOrDefault("REGISTRY_SVC_NAME", "")
	if registryServiceName != "" {
		registryServiceNamespace := platform.GetEnvOrDefault("REGISTRY_SVC_NAMESPACE", "")
		if registryServiceNamespace == "" {
			// fallback to operator namespace
			registryServiceNamespace = platform.GetOperatorNamespace()
		}
		svc, err := c.CoreV1().Services(registryServiceNamespace).Get(ctx, registryServiceName, metav1.GetOptions{})
		if err != nil {
			log.Error(err, "Could not get any container registry %s in namespace %s. "+
				"If you're targeting Minikube, make sure to enable the registry addon first!",
				registryServiceNamespace, registryServiceName)
		}
		platform.SingletonPlatform.Registry.Address = svc.Spec.ClusterIP
	}
}
