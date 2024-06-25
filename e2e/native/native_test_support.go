//go:build integration
// +build integration

// To enable compilation of this file in Goland, go to "Settings -> Go -> Vendoring & Build Tags -> Custom Tags" and add "integration"

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

package native

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
)

func getContainerCommand() func(pod *corev1.Pod) string {
	return func(pod *corev1.Pod) string {
		cmd := strings.Join(pod.Spec.Containers[0].Command, " ")
		cmd = cmd + strings.Join(pod.Spec.Containers[0].Args, " ")
		return cmd
	}
}
