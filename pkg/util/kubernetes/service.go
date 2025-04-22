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
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// GetClusterTypeServiceURI will return the URL of the Service in the cluster type format.
func GetClusterTypeServiceURI(svc *corev1.Service) string {
	url := fmt.Sprintf("http://%s.%s.svc.cluster.local", svc.Name, svc.Namespace)
loop:
	for _, port := range svc.Spec.Ports {
		if port.Port != 80 { // Assuming HTTP default port
			url += fmt.Sprintf(":%d", port.Port)
			break loop
		}
	}

	return url
}
