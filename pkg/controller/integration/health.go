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
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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

func proxyGetHTTPProbe(ctx context.Context, c kubernetes.Interface, p *corev1.Probe, pod *corev1.Pod, container *corev1.Container) ([]byte, error) {
	if p.HTTPGet == nil {
		return nil, fmt.Errorf("missing probe handler for %s/%s", pod.Namespace, pod.Name)
	}

	// We have to extract the port number from the HTTP probe in case it's a named port,
	// as Pod proxying via the API Server does not work with named ports.
	port, err := extractPortNumber(p.HTTPGet.Port, container)
	if err != nil {
		return nil, err
	}

	probeCtx, cancel := context.WithTimeout(ctx, time.Duration(p.TimeoutSeconds)*time.Second)
	defer cancel()
	params := make(map[string]string)
	return c.CoreV1().Pods(pod.Namespace).
		ProxyGet(strings.ToLower(string(p.HTTPGet.Scheme)), pod.Name, strconv.Itoa(port), p.HTTPGet.Path, params).
		DoRaw(probeCtx)
}

func extractPortNumber(port intstr.IntOrString, container *corev1.Container) (int, error) {
	number := -1
	var err error
	switch port.Type {
	case intstr.Int:
		number = port.IntValue()
	case intstr.String:
		if number, err = findPortByName(container, port.StrVal); err != nil {
			// Last ditch effort - maybe it was an int stored as string?
			if number, err = strconv.Atoi(port.StrVal); err != nil {
				return number, err
			}
		}
	default:
		return number, fmt.Errorf("intOrString had no kind: %+v", port)
	}
	if number > 0 && number < 65536 {
		return number, nil
	}
	return number, fmt.Errorf("invalid port number: %v", number)
}

// findPortByName is a helper function to look up a port in a container by name.
func findPortByName(container *corev1.Container, portName string) (int, error) {
	for _, port := range container.Ports {
		if port.Name == portName {
			return int(port.ContainerPort), nil
		}
	}
	return 0, fmt.Errorf("port %s not found", portName)
}
