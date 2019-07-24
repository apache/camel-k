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

	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/log"
	"github.com/pkg/errors"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	eventing "github.com/knative/eventing/pkg/apis/eventing/v1alpha1"
	serving "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IsEnabledInNamespace returns true if we can list some basic knative objects in the given namespace
func IsEnabledInNamespace(ctx context.Context, c k8sclient.Reader, namespace string) bool {
	channels := eventing.ChannelList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Channel",
			APIVersion: eventing.SchemeGroupVersion.String(),
		},
	}
	if err := c.List(ctx, &k8sclient.ListOptions{Namespace: namespace}, &channels); err != nil {
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
func CreateSubscription(namespace string, channel string, name string) eventing.Subscription {
	return eventing.Subscription{
		TypeMeta: metav1.TypeMeta{
			APIVersion: eventing.SchemeGroupVersion.String(),
			Kind:       "Subscription",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      channel + "-" + name,
		},
		Spec: eventing.SubscriptionSpec{
			Channel: corev1.ObjectReference{
				APIVersion: eventing.SchemeGroupVersion.String(),
				Kind:       "Channel",
				Name:       channel,
			},
			Subscriber: &eventing.SubscriberSpec{
				Ref: &corev1.ObjectReference{
					APIVersion: serving.SchemeGroupVersion.String(),
					Kind:       "Service",
					Name:       name,
				},
			},
		},
	}
}

// GetService --
func GetService(ctx context.Context, client client.Client, namespace string, name string) (*serving.Service, error) {
	service := serving.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: serving.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
	key := k8sclient.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
	if err := client.Get(ctx, key, &service); err != nil {
		return nil, errors.Wrap(err, "could not retrieve service "+name+" in namespace "+namespace)
	}
	return &service, nil
}

// GetChannel --
func GetChannel(ctx context.Context, client client.Client, namespace string, name string) (*eventing.Channel, error) {
	channel := eventing.Channel{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Channel",
			APIVersion: eventing.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
	key := k8sclient.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
	if err := client.Get(ctx, key, &channel); err != nil {
		return nil, errors.Wrap(err, "could not retrieve channel "+name+" in namespace "+namespace)
	}
	return &channel, nil
}
