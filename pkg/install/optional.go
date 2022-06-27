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
	"strings"

	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/defaults"
	logutil "github.com/apache/camel-k/pkg/util/log"
)

// OperatorStartupOptionalTools tries to install optional tools at operator startup and warns if something goes wrong.
func OperatorStartupOptionalTools(ctx context.Context, c client.Client, namespace string, operatorNamespace string, log logutil.Logger) {
	// Try to register the OpenShift CLI Download link if possible
	if err := OpenShiftConsoleDownloadLink(ctx, c); err != nil {
		log.Info("Cannot install OpenShift CLI download link: skipping.")
		log.Debug("Error while installing OpenShift CLI download link", "error", err)
	}

	// Try to install Kamelet Catalog automatically
	var kameletNamespace string
	globalOperator := false
	if namespace != "" && !strings.Contains(namespace, ",") {
		kameletNamespace = namespace
	} else {
		kameletNamespace = operatorNamespace
		globalOperator = true
	}

	if kameletNamespace != "" {
		if defaults.InstallDefaultKamelets() {
			if err := KameletCatalog(ctx, c, kameletNamespace); err != nil {
				log.Info("Cannot install bundled Kamelet Catalog: skipping.")
				log.Debug("Error while installing bundled Kamelet Catalog", "error", err)
			}
		} else {
			log.Info("Kamelet Catalog installation is disabled")
		}

		if globalOperator {
			// Make sure that Kamelets installed in operator namespace can be used by others
			if err := KameletViewerRole(ctx, c, kameletNamespace); err != nil {
				log.Info("Cannot install global Kamelet viewer role: skipping.")
				log.Debug("Error while installing global Kamelet viewer role", "error", err)
			}
		}
	}

	// Try to bind the Knative Addressable resolver aggregated ClusterRole to the operator ServiceAccount
	if err := BindKnativeAddressableResolverClusterRole(ctx, c, namespace, operatorNamespace); err != nil {
		log.Info("Cannot bind the Knative addressable resolver aggregated ClusterRole: skipping.")
		log.Debug("Error while binding the Knative Addressable resolver aggregated ClusterRole", "error", err)
	}
}
