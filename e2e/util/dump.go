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
	"fmt"

	"github.com/apache/camel-k/pkg/client"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Dump prints all information about the given namespace to debug errors
func Dump(c client.Client, ns string) error {

	fmt.Printf("-------------------- start dumping namespace %s --------------------\n", ns)

	lst, err := c.CoreV1().Pods(ns).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	fmt.Printf("Found %d pods:\n", len(lst.Items))
	for _, pod := range lst.Items {
		fmt.Printf("name=%s\n", pod.Name)
		dumpConditions("  ", pod.Status.Conditions)
		fmt.Printf("  logs:\n")
		for _, container := range pod.Spec.Containers {
			pad := "    "
			fmt.Printf("%s%s\n", pad, container.Name)
			err := dumpLogs(c, fmt.Sprintf("%s> ", pad), ns, pod.Name, container.Name)
			if err != nil {
				fmt.Printf("%sERROR while reading the logs: %v\n", pad, err)
			}
		}
	}

	fmt.Printf("-------------------- end dumping namespace %s --------------------\n", ns)
	return nil
}

func dumpConditions(prefix string, conditions []v1.PodCondition) {
	for _, cond := range conditions {
		fmt.Printf("%scondition type=%s, status=%s, reason=%s, message=%q\n", prefix, cond.Type, cond.Status, cond.Reason, cond.Message)
	}
}

func dumpLogs(c client.Client, prefix string, ns string, name string, container string) error {
	lines := int64(50)
	stream, err := c.CoreV1().Pods(ns).GetLogs(name, &v1.PodLogOptions{
		Container: container,
		TailLines: &lines,
	}).Stream()
	if err != nil {
		return err
	}
	defer stream.Close()
	scanner := bufio.NewScanner(stream)
	printed := false
	for scanner.Scan() {
		printed = true
		fmt.Printf("%s%s\n", prefix, scanner.Text())
	}
	if !printed {
		fmt.Printf("%s[no logs available]\n", prefix)
	}
	return nil
}
