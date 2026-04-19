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
	// If the container registry is configured as MINIKUBE, we try to get the proper container registry, providing a warning notice as well
	if platform.SingletonPlatform.Registry.Address == "MINIKUBE" {
		log.Info("WARN: container registry is configured to use Minikube container registry extension. " +
			"Mind that this is fine only for development purposes, move to a real container registry instead!")
		svc, err := c.CoreV1().Services("kube-system").Get(ctx, "registry", metav1.GetOptions{})
		if err != nil {
			log.Error(err, "Could not get a Minikube container registry. Make sure to enable the addon properly.")
		}
		platform.SingletonPlatform.Registry.Address = svc.Spec.ClusterIP
		platform.SingletonPlatform.Registry.Insecure = true
		log.Info("Container registry address setting changed to " + platform.SingletonPlatform.Registry.Address + " (insecure=true)")
	}
}
