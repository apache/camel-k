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
	"fmt"
	"net/url"

	"github.com/apache/camel-k/pkg/client"
	kubernetesutils "github.com/apache/camel-k/pkg/util/kubernetes"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	eventing "knative.dev/eventing/pkg/apis/eventing/v1alpha1"
	messaging "knative.dev/eventing/pkg/apis/messaging/v1beta1"
	sourcesv1alpha1 "knative.dev/eventing/pkg/apis/sources/v1alpha1"
	"knative.dev/pkg/apis/duck"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	duckv1alpha1 "knative.dev/pkg/apis/duck/v1alpha1"
	"knative.dev/pkg/tracker"
	serving "knative.dev/serving/pkg/apis/serving/v1"
	controller "sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateSubscription ---
func CreateSubscription(channelReference corev1.ObjectReference, serviceName string) runtime.Object {
	subs := messaging.Subscription{
		TypeMeta: metav1.TypeMeta{
			APIVersion: messaging.SchemeGroupVersion.String(),
			Kind:       "Subscription",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: channelReference.Namespace,
			Name:      channelReference.Name + "-" + serviceName,
		},
		Spec: messaging.SubscriptionSpec{
			Channel: corev1.ObjectReference{
				APIVersion: channelReference.GroupVersionKind().GroupVersion().String(),
				Kind:       channelReference.Kind,
				Name:       channelReference.Name,
			},
			Subscriber: &duckv1.Destination{
				Ref: &duckv1.KReference{
					APIVersion: serving.SchemeGroupVersion.String(),
					Kind:       "Service",
					Name:       serviceName,
				},
			},
		},
	}

	return &subs
}

// CreateTrigger ---
func CreateTrigger(brokerReference corev1.ObjectReference, serviceName string, eventType string) runtime.Object {
	subs := eventing.Trigger{
		TypeMeta: metav1.TypeMeta{
			APIVersion: eventing.SchemeGroupVersion.String(),
			Kind:       "Trigger",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: brokerReference.Namespace,
			Name:      brokerReference.Name + "-" + serviceName + "-" + eventType,
		},
		Spec: eventing.TriggerSpec{
			Filter: &eventing.TriggerFilter{
				Attributes: &eventing.TriggerFilterAttributes{
					"type": eventType,
				},
			},
			Broker: brokerReference.Name,
			Subscriber: duckv1.Destination{
				Ref: &duckv1.KReference{
					APIVersion: serving.SchemeGroupVersion.String(),
					Kind:       "Service",
					Name:       serviceName,
				},
			},
		},
	}
	return &subs
}

// CreateSinkBinding ---
func CreateSinkBinding(source corev1.ObjectReference, target corev1.ObjectReference) runtime.Object {
	binding := sourcesv1alpha1.SinkBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: sourcesv1alpha1.SchemeGroupVersion.String(),
			Kind:       "SinkBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: source.Namespace,
			Name:      source.Name,
		},
		Spec: sourcesv1alpha1.SinkBindingSpec{
			BindingSpec: duckv1alpha1.BindingSpec{
				Subject: tracker.Reference{
					APIVersion: source.APIVersion,
					Kind:       source.Kind,
					Name:       source.Name,
				},
			},
			SourceSpec: duckv1.SourceSpec{
				Sink: duckv1.Destination{
					Ref: &duckv1.KReference{
						APIVersion: target.APIVersion,
						Kind:       target.Kind,
						Name:       target.Name,
					},
				},
			},
		},
	}

	return &binding
}

// GetAddressableReference looks up the resource among all given types and returns an object reference to it
func GetAddressableReference(ctx context.Context, c client.Client,
	possibleReferences []corev1.ObjectReference, namespace string, name string) (*corev1.ObjectReference, error) {

	for _, ref := range possibleReferences {
		sink := ref.DeepCopy()
		sink.Namespace = namespace
		_, err := GetSinkURI(ctx, c, sink, namespace)
		if err != nil && (k8serrors.IsNotFound(err) || kubernetesutils.IsUnknownAPIError(err)) {
			continue
		} else if err != nil {
			return nil, err
		}

		return sink, nil
	}
	return nil, k8serrors.NewNotFound(schema.GroupResource{}, name)
}

// GetSinkURL returns the sink as *url.URL
func GetSinkURL(ctx context.Context, c client.Client, sink *corev1.ObjectReference, namespace string) (*url.URL, error) {
	res, err := GetSinkURI(ctx, c, sink, namespace)
	if err != nil {
		return nil, err
	}
	return url.Parse(res)
}

// GetSinkURI retrieves the sink URI from the object referenced by the given
// ObjectReference.
//
// Method taken from https://github.com/knative/eventing-contrib/blob/master/pkg/controller/sinks/sinks.go
func GetSinkURI(ctx context.Context, c client.Client, sink *corev1.ObjectReference, namespace string) (string, error) {
	if sink == nil {
		return "", fmt.Errorf("sink ref is nil")
	}

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(sink.GroupVersionKind())
	err := c.Get(ctx, controller.ObjectKey{Namespace: namespace, Name: sink.Name}, u)
	if err != nil {
		return "", err
	}

	objIdentifier := fmt.Sprintf("\"%s/%s\" (%s)", u.GetNamespace(), u.GetName(), u.GroupVersionKind())
	// Special case v1/Service to allow it be addressable
	if u.GroupVersionKind().Kind == "Service" && u.GroupVersionKind().Group == "" && u.GroupVersionKind().Version == "v1" {
		return fmt.Sprintf("http://%s.%s.svc/", u.GetName(), u.GetNamespace()), nil
	}

	t := duckv1.AddressableType{}
	err = duck.FromUnstructured(u, &t)
	if err != nil {
		return "", fmt.Errorf("failed to deserialize sink %s: %v", objIdentifier, err)
	}

	if t.Status.Address == nil || t.Status.Address.URL == nil {
		return "", fmt.Errorf("sink %s does not contain address or URL", objIdentifier)
	}

	addressURL := t.Status.Address.URL
	if addressURL.Host == "" {
		return "", fmt.Errorf("sink %s contains an empty hostname", objIdentifier)
	}
	return addressURL.String(), nil
}
