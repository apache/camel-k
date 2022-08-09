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
	"encoding/json"
	"fmt"
	"os"
	"strings"

	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/envvar"
	"github.com/apache/camel-k/pkg/util/knative"
	coordination "k8s.io/api/coordination/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	OperatorWatchNamespaceEnvVariable = "WATCH_NAMESPACE"
	operatorNamespaceEnvVariable      = "NAMESPACE"
	operatorPodNameEnvVariable        = "POD_NAME"
)

const OperatorLockName = "camel-k-lock"

var OperatorImage string

// IsCurrentOperatorGlobal returns true if the operator is configured to watch all namespaces.
func IsCurrentOperatorGlobal() bool {
	if watchNamespace, envSet := os.LookupEnv(OperatorWatchNamespaceEnvVariable); !envSet || strings.TrimSpace(watchNamespace) == "" {
		return true
	}
	return false
}

// GetOperatorWatchNamespace returns the namespace the operator watches.
func GetOperatorWatchNamespace() string {
	if namespace, envSet := os.LookupEnv(OperatorWatchNamespaceEnvVariable); envSet {
		return namespace
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

// GetOperatorLockName returns the name of the lock lease that is electing a leader on the particular namepsace.
func GetOperatorLockName(operatorID string) string {
	return fmt.Sprintf("%s-lock", operatorID)
}

// IsNamespaceLocked tells if the namespace contains a lock indicating that an operator owns it.
func IsNamespaceLocked(ctx context.Context, c ctrl.Reader, namespace string) (bool, error) {
	if namespace == "" {
		return false, nil
	}

	platforms, err := ListPrimaryPlatforms(ctx, c, namespace)
	if err != nil {
		return true, err
	}

	for _, platform := range platforms.Items {
		lease := coordination.Lease{}

		var operatorLockName string
		if platform.Name != "" {
			operatorLockName = GetOperatorLockName(platform.Name)
		} else {
			operatorLockName = OperatorLockName
		}

		if err := c.Get(ctx, ctrl.ObjectKey{Namespace: namespace, Name: operatorLockName}, &lease); err == nil || !k8serrors.IsNotFound(err) {
			return true, err
		}
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

// EnableKnativeBindInNamespace sets the "bindings.knative.dev/include=true" label to the namespace if ALL of the conditions are met:
// * Knative is installed and enabled in the IntegrationPlatform namespace
// * There aren't any of these labels bindings.knative.dev/include bindings.knative.dev/exclude in the IntegrationPlatform namespace
// * The environment variable SINK_BINDING_SELECTION_MODE should be set to inclusion in deploy/eventing-webhook of knative-eventing namespace
// Returns true if the label was set in the namespace
// see https://knative.dev/v1.3-docs/eventing/custom-event-source/sinkbinding/create-a-sinkbinding/
func EnableKnativeBindInNamespace(ctx context.Context, client client.Client, namespace string) (bool, error) {
	ns, err := client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	// if Knative is not enabled in this namespace, doesn't proceed
	isKnative := knative.IsEnabledInNamespace(ctx, client, namespace)
	if !isKnative {
		return false, nil
	}

	// if there are sinkbinding labels in the namespace, camel-k-operator respects it and doesn't proceed
	sinkbindingLabelsExists := ns.Labels["bindings.knative.dev/include"] != "" || ns.Labels["bindings.knative.dev/exclude"] != ""
	if sinkbindingLabelsExists {
		return false, nil
	}

	eventingWebhook, err := client.AppsV1().Deployments("knative-eventing").Get(ctx, "eventing-webhook", metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	inclusionMode := false
	for _, c := range eventingWebhook.Spec.Template.Spec.Containers {
		if c.Name == "eventing-webhook" {
			if sinkBindingSelectionMode := envvar.Get(c.Env, "SINK_BINDING_SELECTION_MODE"); sinkBindingSelectionMode != nil {
				inclusionMode = sinkBindingSelectionMode.Value == "inclusion"
			}
			break
		}
	}
	// only proceeds if the sinkbinding mode is inclusion
	if !inclusionMode {
		return false, err
	}
	var jsonLabelPatch = map[string]interface{}{
		"metadata": map[string]interface{}{
			"labels": map[string]string{"bindings.knative.dev/include": "true"},
		},
	}
	patch, err := json.Marshal(jsonLabelPatch)
	if err != nil {
		return false, err
	}
	_, err = client.CoreV1().Namespaces().Patch(ctx, namespace, types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
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
		camelv1.GetOperatorIDAnnotation(e.ObjectOld) != camelv1.GetOperatorIDAnnotation(e.ObjectNew) {
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
