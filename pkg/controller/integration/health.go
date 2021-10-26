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

package integration

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type HealthCheckState string

const (
	HealthCheckStateDown HealthCheckState = "DOWN"
	HealthCheckStateUp   HealthCheckState = "UP"
)

type HealthCheck struct {
	Status HealthCheckState      `json:"state,omitempty"`
	Checks []HealthCheckResponse `json:"checks,omitempty"`
}

type HealthCheckResponse struct {
	Name   string                 `json:"name,omitempty"`
	Status HealthCheckState       `json:"state,omitempty"`
	Data   map[string]interface{} `json:"data,omitempty"`
}

func proxyGetHTTPProbe(ctx context.Context, c kubernetes.Interface, p *corev1.Probe, pod *corev1.Pod) ([]byte, error) {
	if p.HTTPGet == nil {
		return nil, fmt.Errorf("missing probe handler for %s/%s", pod.Namespace, pod.Name)
	}

	probeCtx, cancel := context.WithTimeout(ctx, time.Duration(p.TimeoutSeconds)*time.Second)
	defer cancel()
	params := make(map[string]string)
	return c.CoreV1().Pods(pod.Namespace).
		ProxyGet(strings.ToLower(string(p.HTTPGet.Scheme)), pod.Name, p.HTTPGet.Port.String(), p.HTTPGet.Path, params).
		DoRaw(probeCtx)
}
