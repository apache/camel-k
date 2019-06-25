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
	"context"
	"time"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/cmd"
	"github.com/apache/camel-k/pkg/util/log"
	"github.com/apache/camel-k/pkg/util/openshift"
	"github.com/google/uuid"
	"github.com/onsi/gomega"
	projectv1 "github.com/openshift/api/project/v1"
	"github.com/spf13/cobra"
	"io/ioutil"
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

/*
	Namespace testing functions
*/

func withNewTestNamespace(doRun func(string)) {
	ns := newTestNamespace()
	defer deleteTestNamespace(ns)

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
