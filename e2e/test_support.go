// +build integration knative

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
	"errors"
	"fmt"
	"time"

	"io/ioutil"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/cmd"
	"github.com/apache/camel-k/pkg/util/log"
	"github.com/apache/camel-k/pkg/util/openshift"
	"github.com/google/uuid"
	eventing "github.com/knative/eventing/pkg/apis/eventing/v1alpha1"
	"github.com/onsi/gomega"
	projectv1 "github.com/openshift/api/project/v1"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var testContext context.Context
var testClient client.Client

// kamelHooks contains hooks useful to add option to kamel commands at runtime
var kamelHooks []func([]string)[]string

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
	for _, hook := range kamelHooks {
		args = hook(args)
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

func integrationPodImage(ns string, name string) func() string {
	return func() string {
		pod := integrationPod(ns, name)()
		if pod == nil || len(pod.Spec.Containers) == 0 {
			return ""
		}
		return pod.Spec.Containers[0].Image
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

func integration(ns string, name string) func() *v1alpha1.Integration {
	return func() *v1alpha1.Integration {
		it := v1alpha1.NewIntegration(ns, name)
		key := k8sclient.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		if err := testClient.Get(testContext, key, &it); err != nil && !k8serrors.IsNotFound(err) {
			panic(err)
		} else if err != nil && k8serrors.IsNotFound(err) {
			return nil
		}
		return &it
	}
}

func integrationVersion(ns string, name string) func() string {
	return func() string {
		it := integration(ns, name)()
		if it == nil {
			return ""
		}
		return it.Status.Version
	}
}

func setIntegrationVersion(ns string, name string, version string) error {
	it := integration(ns, name)()
	if it == nil {
		return fmt.Errorf("no integration named %s found", name)
	}
	it.Status.Version = version
	return testClient.Status().Update(testContext, it)
}

func kits(ns string) func() []v1alpha1.IntegrationKit {
	return func() []v1alpha1.IntegrationKit {
		lst := v1alpha1.NewIntegrationKitList()
		opts := k8sclient.ListOptions{
			Namespace: ns,
		}
		if err := testClient.List(testContext, &opts, &lst); err != nil {
			panic(err)
		}
		return lst.Items
	}
}

func kitsWithVersion(ns string, version string) func() int {
	return func() int {
		count := 0
		for _, k := range kits(ns)() {
			if k.Status.Version == version {
				count++
			}
		}
		return count
	}
}

func setAllKitsVersion(ns string, version string) error {
	for _, k := range kits(ns)() {
		kit := k
		kit.Status.Version = version
		if err := testClient.Status().Update(testContext, &kit); err != nil {
			return err
		}
	}
	return nil
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

func operatorPodPhase(ns string) func() v1.PodPhase {
	return func() v1.PodPhase {
		pod := operatorPod(ns)()
		if pod == nil {
			return ""
		}
		return pod.Status.Phase
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

func platform(ns string) func() *v1alpha1.IntegrationPlatform {
	return func() *v1alpha1.IntegrationPlatform {
		lst := v1alpha1.NewIntegrationPlatformList()
		opts := k8sclient.ListOptions{
			Namespace: ns,
		}
		if err := testClient.List(testContext, &opts, &lst); err != nil {
			panic(err)
		}
		if len(lst.Items) == 0 {
			return nil
		}
		if len(lst.Items) > 1 {
			panic("multiple integration platforms found in namespace " + ns)
		}
		return &lst.Items[0]
	}
}

func setPlatformVersion(ns string, version string) error {
	p := platform(ns)()
	if p == nil {
		return errors.New("no platform found")
	}
	p.Status.Version = version
	return testClient.Status().Update(testContext, p)
}

func platformVersion(ns string) func() string {
	return func() string {
		p := platform(ns)()
		if p == nil {
			return ""
		}
		return p.Status.Version
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

func operatorTryPodForceKill(ns string) {
	pod := operatorPod(ns)()
	if pod != nil {
		opts := func(options *k8sclient.DeleteOptions) {
			zero := int64(0)
			options.GracePeriodSeconds = &zero
		}
		if err := testClient.Delete(testContext, pod, opts); err != nil {
			log.Error(err, "cannot forcefully kill the pod")
		}
	}
}

func scaleOperator(ns string, replicas int32) func() error {
	return func() error {
		lst := appsv1.DeploymentList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: appsv1.SchemeGroupVersion.String(),
			},
		}
		opts := k8sclient.ListOptions{
			LabelSelector: labels.SelectorFromSet(labels.Set{
				"camel.apache.org/component": "operator",
			}),
			Namespace: ns,
		}
		if err := testClient.List(testContext, &opts, &lst); err != nil {
			return err
		}
		if len(lst.Items) == 0 {
			return errors.New("camel k operator not found")
		} else if len(lst.Items) > 1 {
			return errors.New("too many camel k operators")
		}

		operatorDeployment := lst.Items[0]
		operatorDeployment.Spec.Replicas = &replicas
		err := testClient.Update(testContext, &operatorDeployment)
		if err != nil {
			return err
		}

		if replicas == 0 {
			// speedup scale down by killing the pod
			operatorTryPodForceKill(ns)
		}
		return nil
	}
}

/*
	Knative
 */

func createKnativeChannel(ns string, name string) func() error {
	return func() error {
		channel := eventing.Channel{
			TypeMeta: metav1.TypeMeta{
				Kind: "Channel",
				APIVersion: eventing.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name: name,
			},
			Spec: eventing.ChannelSpec{
				Provisioner: &v1.ObjectReference{
					APIVersion: "eventing.knative.dev/v1alpha1",
					Kind: "ClusterChannelProvisioner",
					Name: "in-memory",
				},
			},
		}
		return testClient.Create(testContext, &channel)
	}
}


/*
	Namespace testing functions
*/

func numPods(ns string) func() int {
	return func() int {
		lst := v1.PodList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: v1.SchemeGroupVersion.String(),
			},
		}
		opts := k8sclient.ListOptions{
			Namespace: ns,
		}
		if err := testClient.List(testContext, &opts, &lst); err != nil && k8serrors.IsUnauthorized(err) {
			return 0
		} else if err != nil {
			log.Error(err, "Error while listing the pods")
			return 0
		}
		return len(lst.Items)
	}
}

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

	// Wait for all pods to be deleted
	gomega.Eventually(numPods(ns.GetName()), 30*time.Second).Should(gomega.Equal(0))
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
