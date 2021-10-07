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
	"os"
	"strings"

	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/defaults"
	coordination "k8s.io/api/coordination/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

const OperatorWatchNamespaceEnvVariable = "WATCH_NAMESPACE"
const operatorNamespaceEnvVariable = "NAMESPACE"
const operatorPodNameEnvVariable = "POD_NAME"

const OperatorLockName = "camel-k-lock"

var OperatorImage string

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

func IsOperatorHandler(object ctrl.Object) bool {
	if object == nil {
		return true
	}
	resourceID := object.GetLabels()[camelv1.OperatorIDLabel]
	operatorID := defaults.OperatorID()
	return resourceID == operatorID
}

// FilteringFuncs do preliminary checks to determine if certain events should be handled by the controller
// based on labels on the resources (e.g. camel.apache.org/operator.id) and the operator configuration,
// before handing the computation over to the user code.
type FilteringFuncs struct {
	// Create returns true if the Create event should be processed
	CreateFunc func(event.CreateEvent) bool

	// Delete returns true if the Delete event should be processed
	DeleteFunc func(event.DeleteEvent) bool

	// Update returns true if the Update event should be processed
	UpdateFunc func(event.UpdateEvent) bool

	// Generic returns true if the Generic event should be processed
	GenericFunc func(event.GenericEvent) bool
}

func (f FilteringFuncs) Create(e event.CreateEvent) bool {
	if !IsOperatorHandler(e.Object) {
		return false
	}
	if f.CreateFunc != nil {
		return f.CreateFunc(e)
	}
	return true
}

func (f FilteringFuncs) Delete(e event.DeleteEvent) bool {
	if !IsOperatorHandler(e.Object) {
		return false
	}
	if f.DeleteFunc != nil {
		return f.DeleteFunc(e)
	}
	return true
}

func (f FilteringFuncs) Update(e event.UpdateEvent) bool {
	if !IsOperatorHandler(e.ObjectNew) {
		return false
	}
	if e.ObjectOld != nil && e.ObjectNew != nil &&
		e.ObjectOld.GetLabels()[camelv1.OperatorIDLabel] != e.ObjectNew.GetLabels()[camelv1.OperatorIDLabel] {
		// Always force reconciliation when the object becomes managed by the current operator
		return true
	}
	if f.UpdateFunc != nil {
		return f.UpdateFunc(e)
	}
	return true
}

func (f FilteringFuncs) Generic(e event.GenericEvent) bool {
	if !IsOperatorHandler(e.Object) {
		return false
	}
	if f.GenericFunc != nil {
		return f.GenericFunc(e)
	}
	return true
}

var _ predicate.Predicate = FilteringFuncs{}
