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

package knative

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/camel-k/v2/pkg/apis/duck/knative/apis"
	eventing "github.com/apache/camel-k/v2/pkg/apis/duck/knative/eventing/v1"
	messaging "github.com/apache/camel-k/v2/pkg/apis/duck/knative/messaging/v1"
	serving "github.com/apache/camel-k/v2/pkg/apis/duck/knative/serving/v1"
	sources "github.com/apache/camel-k/v2/pkg/apis/duck/knative/sources/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	util "github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

const knativeService = "Service"

func CreateSubscription(channelReference corev1.ObjectReference, serviceName string, path string) *messaging.Subscription {
	return &messaging.Subscription{
		APIVersion: messaging.SchemeGroupVersion.String(),
		Kind:       "Subscription",
		Namespace:  channelReference.Namespace,
		Name:       channelReference.Name + "-" + serviceName,
		Spec: messaging.SubscriptionSpec{
			Channel: apis.KReference{
				APIVersion: channelReference.GroupVersionKind().GroupVersion().String(),
				Kind:       channelReference.Kind,
				Name:       channelReference.Name,
			},
			Subscriber: &apis.Destination{
				Ref: &apis.KReference{
					APIVersion: serving.SchemeGroupVersion.String(),
					Kind:       knativeService,
					Name:       serviceName,
				},
				URI: &apis.URL{
					Path: path,
				},
			},
		},
	}
}

// CreateServiceTrigger create Knative trigger with arbitrary Kubernetes Service as a subscriber - usually used when no Knative Serving is available on the cluster.
func CreateServiceTrigger(brokerReference corev1.ObjectReference, serviceName string, eventType string, path string, attributes map[string]string) (*eventing.Trigger, error) {
	subscriberRef := apis.KReference{
		APIVersion: "v1",
		Kind:       knativeService,
		Name:       serviceName,
	}

	return CreateTrigger(brokerReference, subscriberRef, eventType, path, attributes)
}

// CreateKnativeServiceTrigger create Knative trigger with Knative Serving Service as a subscriber - default option when Knative Serving is available on the cluster.
func CreateKnativeServiceTrigger(brokerReference corev1.ObjectReference, serviceName string, eventType string, path string, attributes map[string]string) (*eventing.Trigger, error) {
	subscriberRef := apis.KReference{
		APIVersion: serving.SchemeGroupVersion.String(),
		Kind:       knativeService,
		Name:       serviceName,
	}

	return CreateTrigger(brokerReference, subscriberRef, eventType, path, attributes)
}

func CreateTrigger(brokerReference corev1.ObjectReference, subscriberRef apis.KReference, eventType string, path string, attributes map[string]string) (*eventing.Trigger, error) {
	trigger := eventing.Trigger{
		APIVersion: eventing.SchemeGroupVersion.String(),
		Kind:       "Trigger",
		Namespace:  brokerReference.Namespace,
		Name:       GetTriggerName(brokerReference.Name, subscriberRef.Name, eventType),
		Spec: eventing.TriggerSpec{
			Broker: brokerReference.Name,
			Subscriber: apis.Destination{
				Ref: &subscriberRef,
				URI: &apis.URL{
					Path: path,
				},
			},
		},
	}

	if len(attributes) > 0 {
		trigger.Spec.Filter = &eventing.TriggerFilter{
			Attributes: attributes,
		}
	}

	return &trigger, nil
}

func GetTriggerName(brokerName string, subscriberName string, eventType string) string {
	nameSuffix := ""
	if eventType != "" {
		nameSuffix = "-" + util.SanitizeLabel(eventType)
	}

	return brokerName + "-" + subscriberName + nameSuffix
}

func CreateSinkBinding(source corev1.ObjectReference, target corev1.ObjectReference) *sources.SinkBinding {
	return &sources.SinkBinding{
		APIVersion: sources.SchemeGroupVersion.String(),
		Kind:       "SinkBinding",
		Namespace:  source.Namespace,
		Name:       source.Name,
		Spec: sources.SinkBindingSpec{
			BindingSpec: apis.BindingSpec{
				Subject: apis.Reference{
					APIVersion: source.APIVersion,
					Kind:       source.Kind,
					Name:       source.Name,
				},
			},
			SourceSpec: apis.SourceSpec{
				Sink: apis.Destination{
					Ref: &apis.KReference{
						APIVersion: target.APIVersion,
						Kind:       target.Kind,
						Name:       target.Name,
					},
				},
			},
		},
	}
}

// GetAddressableReference looks up the resource among all given types and returns an object reference to it.
func GetAddressableReference(ctx context.Context, c client.Client,
	possibleReferences []corev1.ObjectReference, namespace string, name string) (*corev1.ObjectReference, error) {
	for _, ref := range possibleReferences {
		sink := ref.DeepCopy()
		sink.Namespace = namespace
		_, err := getSinkURI(ctx, c, sink, namespace)
		if err != nil && (k8serrors.IsNotFound(err) || util.IsUnknownAPIError(err)) {
			continue
		} else if err != nil {
			return nil, err
		}

		return sink, nil
	}

	return nil, k8serrors.NewNotFound(schema.GroupResource{}, name)
}

// GetSinkURL returns the sink as *url.URL.
func GetSinkURL(ctx context.Context, c client.Client, sink *corev1.ObjectReference, namespace string) (*url.URL, error) {
	res, err := getSinkURI(ctx, c, sink, namespace)
	if err != nil {
		return nil, err
	}

	return url.Parse(res)
}

// getSinkURI retrieves the sink URI from the object referenced by the given ObjectReference.
//
// Method taken from https://github.com/knative/eventing-contrib/blob/master/pkg/controller/sinks/sinks.go
func getSinkURI(ctx context.Context, c client.Client, sink *corev1.ObjectReference, namespace string) (string, error) {
	if sink == nil {
		return "", errors.New("sink ref is nil")
	}

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(sink.GroupVersionKind())
	err := c.Get(ctx, ctrl.ObjectKey{Namespace: namespace, Name: sink.Name}, u)
	if err != nil {
		return "", err
	}

	objIdentifier := fmt.Sprintf("\"%s/%s\" (%s)", u.GetNamespace(), u.GetName(), u.GroupVersionKind())
	// Special case v1/Service allowing it to be addressable
	if u.GroupVersionKind().Kind == knativeService && u.GroupVersionKind().Group == "" && u.GroupVersionKind().Version == "v1" {
		return fmt.Sprintf("http://%s.%s.svc/", u.GetName(), u.GetNamespace()), nil
	}

	addressURL, found, err := unstructured.NestedString(u.Object, "status", "address", "url")
	if err != nil {
		return "", fmt.Errorf("failed to deserialize sink %s: %w", objIdentifier, err)
	}
	if !found || addressURL == "" {
		return "", fmt.Errorf("sink %s does not contain address or URL", objIdentifier)
	}

	parsedURL, err := url.Parse(addressURL)
	if err != nil {
		return "", fmt.Errorf("failed to deserialize sink %s: %w", objIdentifier, err)
	}
	if parsedURL.Host == "" {
		return "", fmt.Errorf("sink %s contains an empty hostname", objIdentifier)
	}

	return parsedURL.String(), nil
}

// EnableKnativeBindInNamespace sets the "bindings.knative.dev/include=true" label to the namespace, only
// if there aren't any of these labels bindings.knative.dev/include bindings.knative.dev/exclude in the namespace
// Returns true if the label was set in the namespace
// https://knative.dev/docs/eventing/custom-event-source/sinkbinding/create-a-sinkbinding
func EnableKnativeBindInNamespace(ctx context.Context, client client.Client, namespace string) (bool, error) {
	ns, err := client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	// if there are sinkbinding labels in the namespace, camel-k-operator respects it and doesn't proceed
	sinkbindingLabelsExists := ns.Labels["bindings.knative.dev/include"] != "" || ns.Labels["bindings.knative.dev/exclude"] != ""
	if sinkbindingLabelsExists {
		return false, nil
	}

	var jsonLabelPatch = map[string]any{
		"metadata": map[string]any{
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
