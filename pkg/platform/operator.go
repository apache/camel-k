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
	"strings"

	coordination "k8s.io/api/coordination/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

const OperatorWatchNamespaceEnvVariable = "WATCH_NAMESPACE"
const operatorNamespaceEnvVariable = "NAMESPACE"
const operatorPodNameEnvVariable = "POD_NAME"

const OperatorLockName = "camel-k-lock"

// GetCurrentOperatorImage returns the image currently used by the running operator if present (when running out of cluster, it may be absent).
func GetCurrentOperatorImage(ctx context.Context, c ctrl.Reader) (string, error) {
	ns := GetOperatorNamespace()
	name := GetOperatorPodName()
	if ns == "" || name == "" {
		return "", nil
	}

	pod := v1.Pod{}
	if err := c.Get(ctx, ctrl.ObjectKey{Namespace: ns, Name: name}, &pod); err != nil && k8serrors.IsNotFound(err) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	if len(pod.Spec.Containers) == 0 {
		return "", errors.New("no containers found in operator pod")
	}
	return pod.Spec.Containers[0].Image, nil
}

// IsCurrentOperatorGlobal returns true if the operator is configured to watch all namespaces
func IsCurrentOperatorGlobal() bool {
	if watchNamespace, envSet := os.LookupEnv(OperatorWatchNamespaceEnvVariable); !envSet || strings.TrimSpace(watchNamespace) == "" {
		return true
	}
	return false
}

// GetOperatorWatchNamespace returns the namespace the operator watches
func GetOperatorWatchNamespace() string {
	if namespace, envSet := os.LookupEnv(OperatorWatchNamespaceEnvVariable); envSet {
		return namespace
	}
	return ""
}

// GetOperatorNamespace returns the namespace where the current operator is located (if set)
func GetOperatorNamespace() string {
	if podNamespace, envSet := os.LookupEnv(operatorNamespaceEnvVariable); envSet {
		return podNamespace
	}
	return ""
}

// GetOperatorPodName returns the pod that is running the current operator (if any)
func GetOperatorPodName() string {
	if podName, envSet := os.LookupEnv(operatorPodNameEnvVariable); envSet {
		return podName
	}
	return ""
}

// IsNamespaceLocked tells if the namespace contains a lock indicating that an operator owns it
func IsNamespaceLocked(ctx context.Context, c ctrl.Reader, namespace string) (bool, error) {
	if namespace == "" {
		return false, nil
	}

	lease := coordination.Lease{}
	if err := c.Get(ctx, ctrl.ObjectKey{Namespace: namespace, Name: OperatorLockName}, &lease); err != nil && k8serrors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return true, err
	}
	return true, nil
}

// IsOperatorAllowedOnNamespace returns true if the current operator is allowed to react on changes in the given namespace
func IsOperatorAllowedOnNamespace(ctx context.Context, c ctrl.Reader, namespace string) (bool, error) {
	if !IsCurrentOperatorGlobal() {
		return true, nil
	}
	operatorNamespace := GetOperatorNamespace()
	if operatorNamespace == namespace {
		// Global operator is allowed on its own namespace
		return true, nil
	}
	alreadyOwned, err := IsNamespaceLocked(ctx, c, namespace)
	if err != nil {
		return false, err
	}
	return !alreadyOwned, nil
}
