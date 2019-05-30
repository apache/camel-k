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

package platform

import (
	"context"
	"errors"
	"os"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const operatorNamespaceEnvVariable = "NAMESPACE"
const operatorPodNameEnvVariable = "POD_NAME"

// GetCurrentOperatorImage returns the image currently used by the running operator if present (when running out of cluster, it may be absent).
func GetCurrentOperatorImage(ctx context.Context, c client.Client) (string, error) {
	var podNamespace string
	var podName string
	var envSet bool
	if podNamespace, envSet = os.LookupEnv(operatorNamespaceEnvVariable); !envSet || podNamespace == "" {
		return "", nil
	}
	if podName, envSet = os.LookupEnv(operatorPodNameEnvVariable); !envSet || podName == "" {
		return "", nil
	}

	podKey := client.ObjectKey{
		Namespace: podNamespace,
		Name:      podName,
	}
	pod := v1.Pod{}

	if err := c.Get(ctx, podKey, &pod); err != nil {
		return "", err
	}
	if len(pod.Spec.Containers) == 0 {
		return "", errors.New("no containers found in operator pod")
	}
	return pod.Spec.Containers[0].Image, nil
}
