//go:build integration
// +build integration

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

package util

import (
	"bufio"
	"context"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	routev1 "github.com/openshift/api/route/v1"

	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/client/camel/clientset/versioned"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/openshift"
)

// Dump prints all information about the given namespace to debug errors
func Dump(ctx context.Context, c client.Client, ns string, t *testing.T) error {
	t.Logf("-------------------- start dumping namespace %s --------------------\n", ns)

	camelClient, err := versioned.NewForConfig(c.GetConfig())
	if err != nil {
		return err
	}
	pls, err := camelClient.CamelV1().IntegrationPlatforms(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	t.Logf("Found %d platforms:\n", len(pls.Items))
	for _, p := range pls.Items {
		ref := p
		pdata, err := kubernetes.ToYAMLNoManagedFields(&ref)
		if err != nil {
			return err
		}
		t.Logf("---\n%s\n---\n", string(pdata))
	}

	its, err := camelClient.CamelV1().Integrations(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	t.Logf("Found %d integrations:\n", len(its.Items))
	for _, integration := range its.Items {
		ref := integration
		pdata, err := kubernetes.ToYAMLNoManagedFields(&ref)
		if err != nil {
			return err
		}
		t.Logf("---\n%s\n---\n", string(pdata))
	}

	iks, err := camelClient.CamelV1().IntegrationKits(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	t.Logf("Found %d integration kits:\n", len(iks.Items))
	for _, ik := range iks.Items {
		ref := ik
		pdata, err := kubernetes.ToYAMLNoManagedFields(&ref)
		if err != nil {
			return err
		}
		t.Logf("---\n%s\n---\n", string(pdata))
	}

	builds, err := camelClient.CamelV1().Builds(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	t.Logf("Found %d builds:\n", len(builds.Items))
	for _, build := range builds.Items {
		data, err := kubernetes.ToYAMLNoManagedFields(&build)
		if err != nil {
			return err
		}
		t.Logf("---\n%s\n---\n", string(data))
	}

	cms, err := c.CoreV1().ConfigMaps(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	t.Logf("Found %d config maps:\n", len(cms.Items))
	for _, cm := range cms.Items {
		ref := cm
		pdata, err := kubernetes.ToYAMLNoManagedFields(&ref)
		if err != nil {
			return err
		}
		t.Logf("---\n%s\n---\n", string(pdata))
	}

	deployments, err := c.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	t.Logf("Found %d deployments:\n", len(iks.Items))
	for _, deployment := range deployments.Items {
		ref := deployment
		data, err := kubernetes.ToYAMLNoManagedFields(&ref)
		if err != nil {
			return err
		}
		t.Logf("---\n%s\n---\n", string(data))
	}

	lst, err := c.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	t.Logf("\nFound %d pods:\n", len(lst.Items))
	for _, pod := range lst.Items {
		t.Logf("name=%s\n", pod.Name)
		dumpConditions("  ", pod.Status.Conditions, t)
		t.Logf("  logs:\n")
		var allContainers []corev1.Container
		allContainers = append(allContainers, pod.Spec.InitContainers...)
		allContainers = append(allContainers, pod.Spec.Containers...)
		for _, container := range allContainers {
			pad := "    "
			t.Logf("%s%s\n", pad, container.Name)
			err := dumpLogs(ctx, c, fmt.Sprintf("%s> ", pad), ns, pod.Name, container.Name, t)
			if err != nil {
				t.Logf("%sERROR while reading the logs: %v\n", pad, err)
			}
		}
	}

	svcs, err := c.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	t.Logf("\nFound %d services:\n", len(svcs.Items))
	for _, svc := range svcs.Items {
		ref := svc
		data, err := kubernetes.ToYAMLNoManagedFields(&ref)
		if err != nil {
			return err
		}
		t.Logf("---\n%s\n---\n", string(data))
	}

	if ocp, err := openshift.IsOpenShift(c); err == nil && ocp {
		routes := routev1.RouteList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Route",
				APIVersion: routev1.SchemeGroupVersion.String(),
			},
		}
		err := c.List(ctx, &routes, ctrl.InNamespace(ns))
		if err != nil {
			return err
		}
		t.Logf("\nFound %d routes:\n", len(routes.Items))
		for _, route := range routes.Items {
			ref := route
			data, err := kubernetes.ToYAMLNoManagedFields(&ref)
			if err != nil {
				return err
			}
			t.Logf("---\n%s\n---\n", string(data))
		}
	}

	t.Logf("-------------------- end dumping namespace %s --------------------\n", ns)
	return nil
}

func dumpConditions(prefix string, conditions []corev1.PodCondition, t *testing.T) {
	for _, cond := range conditions {
		t.Logf("%scondition type=%s, status=%s, reason=%s, message=%q\n", prefix, cond.Type, cond.Status, cond.Reason, cond.Message)
	}
}

func dumpLogs(ctx context.Context, c client.Client, prefix string, ns string, name string, container string, t *testing.T) error {
	lines := int64(50)
	stream, err := c.CoreV1().Pods(ns).GetLogs(name, &corev1.PodLogOptions{
		Container: container,
		TailLines: &lines,
	}).Stream(ctx)
	if err != nil {
		return err
	}
	defer stream.Close()
	scanner := bufio.NewScanner(stream)
	printed := false
	for scanner.Scan() {
		printed = true
		t.Logf("%s%s\n", prefix, scanner.Text())
	}
	if !printed {
		t.Logf("%s[no logs available]\n", prefix)
	}
	return nil
}
