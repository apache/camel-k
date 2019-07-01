// +build integration

// To enable compilation of this file in Goland, go to "Settings -> Go -> Vendoring & Build Tags -> Custom Tags" and add "integration"

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

package e2e

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/apache/camel-k/pkg/util/indentedwriter"

	"io/ioutil"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/cmd"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/log"
	"github.com/apache/camel-k/pkg/util/openshift"
	"github.com/google/uuid"
	"github.com/onsi/gomega"
	projectv1 "github.com/openshift/api/project/v1"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var testContext context.Context
var testClient client.Client

func init() {
	var err error
	testContext = context.TODO()
	testClient, err = newTestClient()
	if err != nil {
		panic(err)
	}

	// Defaults for testing
	gomega.SetDefaultEventuallyTimeout(60 * time.Second)
}

func newTestClient() (client.Client, error) {
	return client.NewOutOfClusterClient("")
}

func kamel(args ...string) *cobra.Command {
	c, err := cmd.NewKamelCommand(testContext)
	if err != nil {
		panic(err)
	}
	c.SetArgs(args)
	return c
}

/*
	Curryied utility functions for testing
*/

func integrationLogs(ns string, name string) func() string {
	return func() string {
		pod := integrationPod(ns, name)()
		if pod == nil {
			return ""
		}
		containerName := ""
		if len(pod.Spec.Containers) > 1 {
			containerName = pod.Spec.Containers[0].Name
		}
		tail := int64(100)
		logOptions := v1.PodLogOptions{
			Follow:    false,
			Container: containerName,
			TailLines: &tail,
		}
		byteReader, err := testClient.CoreV1().Pods(ns).GetLogs(pod.Name, &logOptions).Context(testContext).Stream()
		if err != nil {
			log.Error(err, "Error while reading the pod logs")
			return ""
		}
		defer func() {
			if err := byteReader.Close(); err != nil {
				log.Error(err, "Error closing the stream")
			}
		}()

		bytes, err := ioutil.ReadAll(byteReader)
		if err != nil {
			log.Error(err, "Error while reading the pod logs content")
			return ""
		}
		return string(bytes)
	}
}

func integrationPodPhase(ns string, name string) func() v1.PodPhase {
	return func() v1.PodPhase {
		pod := integrationPod(ns, name)()
		if pod == nil {
			return ""
		}
		return pod.Status.Phase
	}
}

func integrationPod(ns string, name string) func() *v1.Pod {
	return func() *v1.Pod {
		lst := v1.PodList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: v1.SchemeGroupVersion.String(),
			},
		}
		opts := k8sclient.ListOptions{
			LabelSelector: labels.SelectorFromSet(labels.Set{
				"camel.apache.org/integration": name,
			}),
			Namespace: ns,
		}
		if err := testClient.List(testContext, &opts, &lst); err != nil {
			panic(err)
		}
		if len(lst.Items) == 0 {
			return nil
		}
		return &lst.Items[0]
	}
}

func operatorImage(ns string) func() string {
	return func() string {
		pod := operatorPod(ns)()
		if pod != nil {
			if len(pod.Spec.Containers) > 0 {
				return pod.Spec.Containers[0].Image
			}
		}
		return ""
	}
}

func configmap(ns string, name string) func() *v1.ConfigMap {
	return func() *v1.ConfigMap {
		cm := v1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: metav1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      name,
			},
		}
		key := k8sclient.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		if err := testClient.Get(testContext, key, &cm); err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			log.Error(err, "Error while retrieving configmap "+name)
			return nil
		}
		return &cm
	}
}

func build(ns string, name string) func() *v1alpha1.Build {
	return func() *v1alpha1.Build {
		build := v1alpha1.NewBuild(ns, name)
		key := k8sclient.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		if err := testClient.Get(testContext, key, &build); err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			log.Error(err, "Error while retrieving build "+name)
			return nil
		}
		return &build
	}
}

func operatorPod(ns string) func() *v1.Pod {
	return func() *v1.Pod {
		lst := v1.PodList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: v1.SchemeGroupVersion.String(),
			},
		}
		opts := k8sclient.ListOptions{
			LabelSelector: labels.SelectorFromSet(labels.Set{
				"camel.apache.org/component": "operator",
			}),
			Namespace: ns,
		}
		if err := testClient.List(testContext, &opts, &lst); err != nil {
			panic(err)
		}
		if len(lst.Items) == 0 {
			return nil
		}
		return &lst.Items[0]
	}
}

func buildPods(ns string) []v1.Pod {
	lst := v1.PodList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: v1.SchemeGroupVersion.String(),
		},
	}

	selector, err := labels.Parse("openshift.io/build.name")
	if err != nil {
		panic(err)
	}

	opts := k8sclient.ListOptions{
		LabelSelector: selector,
		Namespace:     ns,
	}
	if err := testClient.List(testContext, &opts, &lst); err != nil {
		panic(err)
	}
	if len(lst.Items) == 0 {
		return nil
	}
	return lst.Items
}

/*
	Namespace testing functions
*/

func withNewTestNamespace(doRun func(string)) {
	ns := newTestNamespace()
	defer deleteTestNamespace(ns)

	ctx, cancel := context.WithCancel(context.Background())

	go dumpStats(ctx, ns.GetName())
	defer cancel()

	doRun(ns.GetName())
}

func deleteTestNamespace(ns metav1.Object) {
	var oc bool
	var err error
	if oc, err = openshift.IsOpenShift(testClient); err != nil {
		panic(err)
	} else if oc {
		prj := &projectv1.Project{
			TypeMeta: metav1.TypeMeta{
				APIVersion: projectv1.SchemeGroupVersion.String(),
				Kind:       "Project",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: ns.GetName(),
			},
		}
		if err := testClient.Delete(testContext, prj); err != nil {
			log.Error(err, "cannot delete test project", "name", prj.Name)
		}
	} else {
		if err := testClient.Delete(testContext, ns.(runtime.Object)); err != nil {
			log.Error(err, "cannot delete test namespace", "name", ns.GetName())
		}
	}
}

func newTestNamespace() metav1.Object {
	var err error
	var oc bool
	var obj runtime.Object

	name := "test-" + uuid.New().String()

	if oc, err = openshift.IsOpenShift(testClient); err != nil {
		panic(err)
	} else if oc {
		obj = &projectv1.ProjectRequest{
			TypeMeta: metav1.TypeMeta{
				APIVersion: projectv1.SchemeGroupVersion.String(),
				Kind:       "ProjectRequest",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}
	} else {
		obj = &v1.Namespace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}
	}
	if err = testClient.Create(testContext, obj); err != nil {
		panic(err)
	}
	return obj.(metav1.Object)
}

func dumpStats(ctx context.Context, namespace string) {
	ticker := time.NewTicker(5 * time.Second)

	var op *v1.Pod

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Context done, giving up")
			return
		case <-ticker.C:
			p, err := kubernetes.GetIntegrationPlatforms(ctx, testClient, namespace)
			if err != nil {
				fmt.Printf("Error while retrieving integration platform list from namespace %s: %s\n", namespace, err)
				continue
			}
			k, err := kubernetes.GetIntegrationKits(ctx, testClient, namespace)
			if err != nil {
				fmt.Printf("Error while retrieving integration kit list from namespace %s: %s\n", namespace, err)
				continue
			}
			i, err := kubernetes.GetIntegrations(ctx, testClient, namespace)
			if err != nil {
				fmt.Printf("Error while retrieving integration list from namespace %s: %s\n", namespace, err)
				continue
			}
			b, err := kubernetes.GetBuilds(ctx, testClient, namespace)
			if err != nil {
				fmt.Printf("Error while retrieving buil list from namespace %s: %s\n", namespace, err)
				continue
			}

			if len(p.Items) > 0 || len(k.Items) > 0 || len(i.Items) > 0 || len(b.Items) > 0 {
				fmt.Printf("\n")
				fmt.Printf("Namespace: %s\n", namespace)
				fmt.Printf("Resources:\n")

				fmt.Printf(indentedwriter.IndentedString(func(out io.Writer) {
					w := indentedwriter.NewWriter(out)
					w.Write(1, "Type\tName\tPhase\tReason\tSince Creation\n")

					for _, e := range p.Items {
						w.Write(1, "%s\t%s\t%s\t%s\t%s\n",
							e.TypeMeta.Kind,
							e.Name,
							e.Status.Phase,
							"",
							time.Since(e.CreationTimestamp.Time).Truncate(time.Second))
					}
					for _, e := range k.Items {
						reason := ""

						if e.Status.Failure != nil {
							reason = e.Status.Failure.Reason
						}

						w.Write(1, "%s\t%s\t%s\t%s\t%s\n",
							e.TypeMeta.Kind,
							e.Name,
							e.Status.Phase,
							reason,
							time.Since(e.CreationTimestamp.Time).Truncate(time.Second))

					}
					for _, e := range i.Items {
						reason := ""

						if e.Status.Failure != nil {
							reason = e.Status.Failure.Reason
						}

						w.Write(1, "%s\t%s\t%s\t%s\t%s\n",
							e.TypeMeta.Kind,
							e.Name,
							e.Status.Phase,
							reason,
							time.Since(e.CreationTimestamp.Time).Truncate(time.Second))
					}
					for _, e := range b.Items {
						reason := ""

						if e.Status.Failure != nil {
							reason = e.Status.Failure.Reason
						}

						w.Write(1, "%s\t%s\t%s\t%s\t%s\n",
							e.TypeMeta.Kind,
							e.Name,
							e.Status.Phase,
							reason,
							time.Since(e.CreationTimestamp.Time).Truncate(time.Second))
					}

					w.Flush()
				}))
			}

			if op == nil {
				op = operatorPod(namespace)()
			}

			if op != nil {
				containerName := ""
				if len(op.Spec.Containers) > 1 {
					containerName = op.Spec.Containers[0].Name
				}
				tail := int64(5)
				logOptions := v1.PodLogOptions{
					Follow:    false,
					Container: containerName,
					TailLines: &tail,
				}
				byteReader, err := testClient.CoreV1().Pods(namespace).GetLogs(op.Name, &logOptions).Context(testContext).Stream()
				if err != nil {
					fmt.Printf("Error while reading the pod logs: %s\n", err)
					continue
				}

				fmt.Printf("Operator Logs:\n")
				fmt.Printf(indentedwriter.IndentedString(func(out io.Writer) {
					w := indentedwriter.NewWriter(out)

					scanner := bufio.NewScanner(byteReader)
					for scanner.Scan() {
						w.Write(1, scanner.Text()+"\n")
					}
					if err := scanner.Err(); err != nil {
						fmt.Printf("Error while reading the pod logs content: %s\n", err)
					}
				}))

				if err := byteReader.Close(); err != nil {
					fmt.Printf("Error closing the stream: %s\n", err)
					continue
				}
			}

			buildPods := buildPods(namespace)

			if buildPods != nil && len(buildPods) > 0 {
				fmt.Printf("Build Logs:\n")
				for _, bp := range buildPods {
					containerName := ""
					if len(bp.Spec.Containers) > 1 {
						containerName = bp.Spec.Containers[0].Name
					}
					tail := int64(5)
					logOptions := v1.PodLogOptions{
						Follow:    false,
						Container: containerName,
						TailLines: &tail,
					}
					fmt.Printf(indentedwriter.IndentedString(func(out io.Writer) {
						w := indentedwriter.NewWriter(out)

						byteReader, err := testClient.CoreV1().Pods(namespace).GetLogs(bp.Name, &logOptions).Context(testContext).Stream()
						if err != nil {
							w.Write(1, "Error while reading the pod logs: %s\n", err)
							return
						}

						scanner := bufio.NewScanner(byteReader)
						for scanner.Scan() {
							w.Write(1, "%s: %s\n", bp.Name, scanner.Text())
						}
						if err := scanner.Err(); err != nil {
							w.Write(1, "Error while reading the pod logs content: %s\n", err)
						}
						if err := byteReader.Close(); err != nil {
							w.Write(1, "Error closing the stream: %s\n", err)
						}
					}))
				}
			}
		}
	}
}
