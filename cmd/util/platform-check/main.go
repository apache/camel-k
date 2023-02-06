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

package main

import (
	"fmt"
	"os"
	"strings"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func main() {
	fmt.Println(platformID())
}

func platformID() string {
	client := APIClient()
	_, apiResourceLists, err := client.Discovery().ServerGroupsAndResources()
	exitOnError(err)

	for _, apiResList := range apiResourceLists {
		// Should be version independent just in case image api is ever upgraded
		if strings.Contains(apiResList.GroupVersion, "image.openshift.io") {
			return "openshift"
		}
	}

	return "kubernetes"
}

func exitOnError(err error) {
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
}

func RestConfig() *rest.Config {
	restConfig, err := config.GetConfig()
	exitOnError(err)

	return restConfig
}

func APIClient() kubernetes.Interface {
	apiClient, err := kubernetes.NewForConfig(RestConfig())
	exitOnError(err)

	return apiClient
}
