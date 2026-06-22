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

	camelv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	coordination "k8s.io/api/coordination/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/apache/camel-k/v2/pkg/util/log"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	OperatorWatchNamespaceEnvVariable = "WATCH_NAMESPACE"
	// OperatorWatchNamespaceSelectorEnvVariable holds a Kubernetes label selector. When set, the
	// operator dynamically discovers and watches every namespace whose labels match the selector,
	// in addition to any namespaces listed in WATCH_NAMESPACE. It is the gate for the dynamic
	// multi-namespace mode.
	OperatorWatchNamespaceSelectorEnvVariable = "WATCH_NAMESPACE_SELECTOR"
	operatorNamespaceEnvVariable              = "NAMESPACE"
	operatorPodNameEnvVariable                = "POD_NAME"
	OperatorBuildStrategyEnvVar               = "BUILD_STRATEGY"
)

const OperatorLockName = "camel-k-lock"

var OperatorImage string

// IsCurrentOperatorGlobal returns true if the operator is configured to watch all namespaces.
//
// The operator is global only when it has no explicit scope at all: WATCH_NAMESPACE is empty/unset
// AND no WATCH_NAMESPACE_SELECTOR is set. A non-empty WATCH_NAMESPACE (single or comma-separated
// list) or a non-empty selector both put the operator in a (multi-)namespace-scoped, i.e. local, mode.
func IsCurrentOperatorGlobal() bool {
	if selector, envSet := os.LookupEnv(OperatorWatchNamespaceSelectorEnvVariable); envSet && strings.TrimSpace(selector) != "" {
		log.Debug("Operator is local to namespaces matching a label selector")

		return false
	}

	if watchNamespace, envSet := os.LookupEnv(OperatorWatchNamespaceEnvVariable); !envSet || strings.TrimSpace(watchNamespace) == "" {
		log.Debug("Operator is global to all namespaces")

		return true
	}

	log.Debug("Operator is local to namespace")

	return false
}

// GetOperatorWatchNamespace returns the raw value of the WATCH_NAMESPACE environment variable.
// It may be empty (global), a single namespace, or a comma-separated list of namespaces.
func GetOperatorWatchNamespace() string {
	if namespace, envSet := os.LookupEnv(OperatorWatchNamespaceEnvVariable); envSet {
		return namespace
	}

	return ""
}

// GetWatchNamespaces returns the explicit, de-duplicated list of namespaces the operator is
// statically configured to watch via WATCH_NAMESPACE. WATCH_NAMESPACE may contain a single
// namespace or a comma-separated list; surrounding whitespace and empty entries are ignored.
// An empty result means no namespace was statically configured (global mode, or selector-only
// dynamic mode).
func GetWatchNamespaces() []string {
	raw := GetOperatorWatchNamespace()
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	seen := make(map[string]bool)
	namespaces := make([]string, 0)
	for _, ns := range strings.Split(raw, ",") {
		ns = strings.TrimSpace(ns)
		if ns == "" || seen[ns] {
			continue
		}
		seen[ns] = true
		namespaces = append(namespaces, ns)
	}

	return namespaces
}

// GetWatchNamespaceSelector returns the trimmed WATCH_NAMESPACE_SELECTOR label selector used to
// dynamically discover namespaces to watch. An empty result means dynamic discovery is disabled.
func GetWatchNamespaceSelector() string {
	if selector, envSet := os.LookupEnv(OperatorWatchNamespaceSelectorEnvVariable); envSet {
		return strings.TrimSpace(selector)
	}

	return ""
}

// GetOperatorNamespace returns the namespace where the current operator is located (if set).
func GetOperatorNamespace() string {
	if podNamespace, envSet := os.LookupEnv(operatorNamespaceEnvVariable); envSet {
		return podNamespace
	}

	return ""
}

// GetOperatorPodName returns the pod that is running the current operator (if any).
func GetOperatorPodName() string {
	if podName, envSet := os.LookupEnv(operatorPodNameEnvVariable); envSet {
		return podName
	}

	return ""
}

// GetOperatorLockName returns the name of the lock lease that is electing a leader on the particular namespace.
func GetOperatorLockName(operatorID string) string {
	return operatorID + "-lock"
}

// IsNamespaceLocked tells if the namespace contains a lock indicating that an operator owns it.
func IsNamespaceLocked(ctx context.Context, c ctrl.Reader, namespace string) (bool, error) {
	if namespace == "" {
		return false, nil
	}

	lease := coordination.Lease{}
	if err := c.Get(ctx, ctrl.ObjectKey{Namespace: namespace, Name: OperatorLockName}, &lease); err == nil || !k8serrors.IsNotFound(err) {
		return true, err
	}

	return false, nil
}

// IsOperatorAllowedOnNamespace returns true if the current operator is allowed to react on changes in the given namespace.
func IsOperatorAllowedOnNamespace(ctx context.Context, c ctrl.Reader, namespace string) (bool, error) {
	// allow all local operators
	if !IsCurrentOperatorGlobal() {
		return true, nil
	}

	// allow global operators that use a proper operator id
	if defaults.OperatorID() != "" {
		log.Debugf("Operator ID: %s", defaults.OperatorID())

		return true, nil
	}

	operatorNamespace := GetOperatorNamespace()
	if operatorNamespace == namespace {
		// Global operator is allowed on its own namespace
		return true, nil
	}
	alreadyOwned, err := IsNamespaceLocked(ctx, c, namespace)
	if err != nil {
		log.Debugf("Error occurred while testing whether namespace is locked: %v", err)

		return false, err
	}

	log.Debugf("Lock status of namespace %s: %t", namespace, alreadyOwned)

	return !alreadyOwned, nil
}

// IsOperatorHandler checks on resource operator id annotation and this operator instance id.
// Operators matching the annotation operator id are allowed to reconcile.
// For legacy resources that are missing a proper operator id annotation the default global operator or the local
// operator in this namespace are candidates for reconciliation.
func IsOperatorHandler(object ctrl.Object) bool {
	if object == nil {
		return true
	}
	resourceID := camelv1.GetOperatorIDAnnotation(object)
	operatorID := defaults.OperatorID()

	// allow operator with matching id to handle the resource
	if resourceID == operatorID {
		return true
	}

	// check if we are dealing with resource that is missing a proper operator id annotation
	if resourceID == "" {
		// allow default global operator to handle legacy resources (missing proper operator id annotations)
		if operatorID == DefaultPlatformName {
			return true
		}

		// allow local operators to handle legacy resources (missing proper operator id annotations)
		if !IsCurrentOperatorGlobal() {
			return true
		}
	}

	return false
}

// IsOperatorHandlerConsideringLock uses normal IsOperatorHandler checks and adds additional check for legacy resources
// that are missing a proper operator id annotation. In general two kind of operators race for reconcile these legacy resources.
// The local operator for this namespace and the default global operator instance. Based on the existence of a namespace
// lock the current local operator has precedence. When no lock exists the default global operator should reconcile.
func IsOperatorHandlerConsideringLock(ctx context.Context, c ctrl.Reader, namespace string, object ctrl.Object) bool {
	isHandler := IsOperatorHandler(object)
	if !isHandler {
		return false
	}

	resourceID := camelv1.GetOperatorIDAnnotation(object)
	// add additional check on resources missing an operator id
	if resourceID == "" {
		operatorNamespace := GetOperatorNamespace()
		if operatorNamespace == namespace {
			// Global operator is allowed on its own namespace
			return true
		}

		if locked, err := IsNamespaceLocked(ctx, c, namespace); err != nil || locked {
			// namespace is locked so local operators do have precedence
			return !IsCurrentOperatorGlobal()
		}
	}

	return true
}

// FilteringFuncs do preliminary checks to determine if certain events should be handled by the controller
// based on labels on the resources (e.g. camel.apache.org/operator.id) and the operator configuration,
// before handing the computation over to the user code.
type FilteringFuncs[T ctrl.Object] struct {
	// Create returns true if the Create event should be processed
	CreateFunc func(event.TypedCreateEvent[T]) bool

	// Delete returns true if the Delete event should be processed
	DeleteFunc func(event.TypedDeleteEvent[T]) bool

	// Update returns true if the Update event should be processed
	UpdateFunc func(event.TypedUpdateEvent[T]) bool

	// Generic returns true if the Generic event should be processed
	GenericFunc func(event.TypedGenericEvent[T]) bool
}

func (f FilteringFuncs[T]) Create(e event.TypedCreateEvent[T]) bool {
	if !IsOperatorHandler(e.Object) {
		return false
	}
	if f.CreateFunc != nil {
		return f.CreateFunc(e)
	}

	return true
}

func (f FilteringFuncs[T]) Delete(e event.TypedDeleteEvent[T]) bool {
	if !IsOperatorHandler(e.Object) {
		return false
	}
	if f.DeleteFunc != nil {
		return f.DeleteFunc(e)
	}

	return true
}

func (f FilteringFuncs[T]) Update(e event.TypedUpdateEvent[T]) bool {
	if !IsOperatorHandler(e.ObjectNew) {
		return false
	}
	if camelv1.GetOperatorIDAnnotation(e.ObjectOld) != camelv1.GetOperatorIDAnnotation(e.ObjectNew) {
		// Always force reconciliation when the object becomes managed by the current operator
		return true
	}
	if camelv1.GetIntegrationProfileAnnotation(e.ObjectOld) != camelv1.GetIntegrationProfileAnnotation(e.ObjectNew) {
		// Always force reconciliation when the object gets attached to a new integration profile
		return true
	}
	//nolint:staticcheck
	if camelv1.GetIntegrationProfileNamespaceAnnotation(e.ObjectOld) != camelv1.GetIntegrationProfileNamespaceAnnotation(e.ObjectNew) {
		// Always force reconciliation when the object gets attached to a new integration profile
		return true
	}
	if f.UpdateFunc != nil {
		return f.UpdateFunc(e)
	}

	return true
}

func (f FilteringFuncs[T]) Generic(e event.TypedGenericEvent[T]) bool {
	if !IsOperatorHandler(e.Object) {
		return false
	}
	if f.GenericFunc != nil {
		return f.GenericFunc(e)
	}

	return true
}

var _ predicate.Predicate = FilteringFuncs[ctrl.Object]{}
