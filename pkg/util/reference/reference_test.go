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

package reference

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestExpressions(t *testing.T) {
	emptyPrefix := ""
	tests := []struct {
		defaultPrefix *string
		name          string
		error         bool
		ref           corev1.ObjectReference
		stringRef     string
		properties    map[string]string
	}{
		{
			name:  "lowercase:source",
			error: true,
		},
		{
			name:  "PostgreSQL/ns/name",
			error: true,
		},
		{
			defaultPrefix: &emptyPrefix,
			name:          "source",
			error:         true,
		},
		{
			name: "source",
			ref: corev1.ObjectReference{
				Kind:       "Kamelet",
				APIVersion: "camel.apache.org/v1alpha1",
				Name:       "source",
			},
			stringRef: "camel.apache.org/v1alpha1:Kamelet:source",
		},
		{
			name: "ns1/source",
			ref: corev1.ObjectReference{
				Kind:       "Kamelet",
				APIVersion: "camel.apache.org/v1alpha1",
				Namespace:  "ns1",
				Name:       "source",
			},
			stringRef: "camel.apache.org/v1alpha1:Kamelet:ns1/source",
		},
		{
			name: "v1:Secret:ns1/scr2",
			ref: corev1.ObjectReference{
				Kind:       "Secret",
				APIVersion: "v1",
				Namespace:  "ns1",
				Name:       "scr2",
			},
			stringRef: "v1:Secret:ns1/scr2",
		},
		{
			name: "ksvc:service",
			ref: corev1.ObjectReference{
				Kind:       "Service",
				APIVersion: "serving.knative.dev/v1",
				Name:       "service",
			},
			stringRef: "serving.knative.dev/v1:Service:service",
		},
		{
			name: "channel:ns3/ch2",
			ref: corev1.ObjectReference{
				Kind:       "Channel",
				APIVersion: "messaging.knative.dev/v1",
				Namespace:  "ns3",
				Name:       "ch2",
			},
			stringRef: "messaging.knative.dev/v1:Channel:ns3/ch2",
		},
		{
			name: "broker:default",
			ref: corev1.ObjectReference{
				Kind:       "Broker",
				APIVersion: "eventing.knative.dev/v1",
				Name:       "default",
			},
			stringRef: "eventing.knative.dev/v1:Broker:default",
		},
		{
			name: "PostgreSQL:ns1/db",
			ref: corev1.ObjectReference{
				Kind:      "PostgreSQL",
				Namespace: "ns1",
				Name:      "db",
			},
			stringRef: "PostgreSQL:ns1/db",
		},
		{
			name: "postgres.org/v1alpha1:PostgreSQL:ns1/db",
			ref: corev1.ObjectReference{
				APIVersion: "postgres.org/v1alpha1",
				Kind:       "PostgreSQL",
				Namespace:  "ns1",
				Name:       "db",
			},
			stringRef: "postgres.org/v1alpha1:PostgreSQL:ns1/db",
		},
		{
			name: "postgres.org/v1alpha1:PostgreSQL:ns1/db?user=user1&password=pwd2&host=192.168.2.2&special=%201&special2=a=1",
			ref: corev1.ObjectReference{
				APIVersion: "postgres.org/v1alpha1",
				Kind:       "PostgreSQL",
				Namespace:  "ns1",
				Name:       "db",
			},
			stringRef: "postgres.org/v1alpha1:PostgreSQL:ns1/db",
			properties: map[string]string{
				"user":     "user1",
				"password": "pwd2",
				"host":     "192.168.2.2",
				"special":  " 1",
				"special2": "a=1",
			},
		},
		{
			name: "source?a=b&b=c&d=e",
			ref: corev1.ObjectReference{
				Kind:       "Kamelet",
				APIVersion: "camel.apache.org/v1alpha1",
				Name:       "source",
			},
			stringRef: "camel.apache.org/v1alpha1:Kamelet:source",
			properties: map[string]string{
				"a": "b",
				"b": "c",
				"d": "e",
			},
		},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("%d-%s", i, tc.name), func(t *testing.T) {

			var converter *Converter
			if tc.defaultPrefix != nil {
				converter = NewConverter(*tc.defaultPrefix)
			} else {
				// Using kamelet: prefix by default in the tests
				converter = NewConverter(KameletPrefix)
			}

			ref, err := converter.FromString(tc.name)
			if tc.error {
				assert.Error(t, err)
			} else {
				asString, err2 := converter.ToString(ref)
				assert.NoError(t, err2)

				props, err3 := converter.PropertiesFromString(tc.name)
				assert.NoError(t, err3)
				assert.Equal(t, tc.properties, props)

				assert.NoError(t, err)
				assert.Equal(t, tc.ref, ref)
				assert.Equal(t, tc.stringRef, asString)
			}
		})
	}

}
