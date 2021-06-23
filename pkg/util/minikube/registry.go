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

// Package minikube contains utilities for Minikube deployments
package minikube

import (
	"context"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/camel-k/pkg/client"
)

const (
	registryNamespace = "kube-system"
)

// FindRegistry returns the Minikube addon registry location if any
func FindRegistry(ctx context.Context, c client.Client) (*string, error) {
	svcs := corev1.ServiceList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
	}
	err := c.List(ctx, &svcs,
		k8sclient.InNamespace(registryNamespace),
		k8sclient.MatchingLabels{
			"kubernetes.io/minikube-addons": "registry",
		})
	if err != nil {
		return nil, err
	}
	if len(svcs.Items) == 0 {
		return nil, nil
	}
	svc := svcs.Items[0]
	ip := svc.Spec.ClusterIP
	portStr := ""
	if len(svc.Spec.Ports) > 0 {
		port := svc.Spec.Ports[0].Port
		if port > 0 && port != 80 {
			portStr = ":" + strconv.FormatInt(int64(port), 10)
		}
	}
	registry := ip + portStr
	return &registry, nil
}
