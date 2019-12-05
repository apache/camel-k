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
	"github.com/apache/camel-k/pkg/util/log"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	controller "sigs.k8s.io/controller-runtime/pkg/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	eventing "knative.dev/eventing/pkg/apis/eventing/v1alpha1"
	messaging "knative.dev/eventing/pkg/apis/messaging/v1alpha1"
	"knative.dev/pkg/apis/duck"
	duckv1alpha1 "knative.dev/pkg/apis/duck/v1alpha1"
	serving "knative.dev/serving/pkg/apis/serving/v1beta1"
)

// IsEnabledInNamespace returns true if we can list some basic knative objects in the given namespace
func IsEnabledInNamespace(ctx context.Context, c k8sclient.Reader, namespace string) bool {
	channels := messaging.ChannelList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Channel",
			APIVersion: eventing.SchemeGroupVersion.String(),
		},
	}
	if err := c.List(ctx, &channels, k8sclient.InNamespace(namespace)); err != nil {
		log.Infof("could not find knative in namespace %s, got error: %v", namespace, err)
		return false
	}
	return true
}

// IsInstalled returns true if we are connected to a cluster with Knative installed
func IsInstalled(ctx context.Context, c kubernetes.Interface) (bool, error) {
	// check knative eventing, since serving may be on v1beta1 in some clusters
	_, err := c.Discovery().ServerResourcesForGroupVersion("eventing.knative.dev/v1alpha1")
	if err != nil && k8serrors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

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
			Subscriber: &messaging.SubscriberSpec{
				Ref: &corev1.ObjectReference{
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
			Subscriber: &messaging.SubscriberSpec{
				Ref: &corev1.ObjectReference{
					APIVersion: serving.SchemeGroupVersion.String(),
					Kind:       "Service",
					Name:       serviceName,
				},
			},
		},
	}
	return &subs
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
	if u.GroupVersionKind().Kind == "Service" && u.GroupVersionKind().Version == "v1" {
		return fmt.Sprintf("http://%s.%s.svc/", u.GetName(), u.GetNamespace()), nil
	}

	t := duckv1alpha1.AddressableType{}
	err = duck.FromUnstructured(u, &t)
	if err != nil {
		return "", fmt.Errorf("failed to deserialize sink %s: %v", objIdentifier, err)
	}

	if t.Status.Address == nil {
		return "", fmt.Errorf("sink %s does not contain address", objIdentifier)
	}

	url := t.Status.Address.GetURL()
	if url.Host == "" {
		return "", fmt.Errorf("sink %s contains an empty hostname", objIdentifier)
	}
	return url.String(), nil
}
