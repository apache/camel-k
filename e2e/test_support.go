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
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/apache/camel-k/e2e/util"
	"github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/cmd"
	"github.com/apache/camel-k/pkg/install"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/log"
	"github.com/apache/camel-k/pkg/util/openshift"
	"github.com/google/uuid"
	"github.com/onsi/gomega"
	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	eventing "knative.dev/eventing/pkg/apis/eventing/v1alpha1"
	messaging "knative.dev/eventing/pkg/apis/messaging/v1alpha1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	// let's enable addons in all tests
	_ "github.com/apache/camel-k/addons"
)

var testTimeoutShort = 1 * time.Minute
var testTimeoutMedium = 5 * time.Minute
var testTimeoutLong = 10 * time.Minute

var testContext context.Context
var testClient client.Client

// kamelHooks contains hooks useful to add option to kamel commands at runtime
var kamelHooks []func([]string) []string

var testImageName = defaults.ImageName
var testImageVersion = defaults.Version

func init() {
	// Register some resources used in e2e tests only
	client.FastMapperAllowedAPIGroups["project.openshift.io"] = true
	client.FastMapperAllowedAPIGroups["route.openshift.io"] = true
	client.FastMapperAllowedAPIGroups["eventing.knative.dev"] = true
	client.FastMapperAllowedAPIGroups["messaging.knative.dev"] = true
	client.FastMapperAllowedAPIGroups["serving.knative.dev"] = true

	var err error
	testContext = context.TODO()
	testClient, err = newTestClient()
	if err != nil {
		panic(err)
	}

	// Defaults for testing
	imageName := os.Getenv("CAMEL_K_TEST_IMAGE_NAME")
	if imageName != "" {
		testImageName = imageName
	}
	imageVersion := os.Getenv("CAMEL_K_TEST_IMAGE_VERSION")
	if imageVersion != "" {
		testImageVersion = imageVersion
	}

	// Timeouts
	var duration time.Duration
	if value, ok := os.LookupEnv("CAMEL_K_TEST_TIMEOUT_SHORT"); ok {
		if duration, err = time.ParseDuration(value); err == nil {
			testTimeoutShort = duration
		} else {
			fmt.Printf("Can't parse CAMEL_K_TEST_TIMEOUT_SHORT. Using default value: %s", testTimeoutShort)
		}
	}

	if value, ok := os.LookupEnv("CAMEL_K_TEST_TIMEOUT_MEDIUM"); ok {
		if duration, err = time.ParseDuration(value); err == nil {
			testTimeoutMedium = duration
		} else {
			fmt.Printf("Can't parse CAMEL_K_TEST_TIMEOUT_MEDIUM. Using default value: %s", testTimeoutMedium)
		}
	}

	if value, ok := os.LookupEnv("CAMEL_K_TEST_TIMEOUT_LONG"); ok {
		if duration, err = time.ParseDuration(value); err == nil {
			testTimeoutLong = duration
		} else {
			fmt.Printf("Can't parse CAMEL_K_TEST_TIMEOUT_LONG. Using default value: %s", testTimeoutLong)
		}
	}

	gomega.SetDefaultEventuallyTimeout(testTimeoutShort)

}

func newTestClient() (client.Client, error) {
	return client.NewOutOfClusterClient(os.Getenv(k8sutil.KubeConfigEnvVar))
}

func kamel(args ...string) *cobra.Command {
	return kamelWithContext(testContext, args...)
}

func kamelWithContext(ctx context.Context, args ...string) *cobra.Command {
	var c *cobra.Command
	var err error

	kamelArgs := os.Getenv("KAMEL_ARGS")
	kamelDefaultArgs := strings.Fields(kamelArgs)
	args = append(kamelDefaultArgs, args...)

	kamelBin := os.Getenv("KAMEL_BIN")
	if kamelBin != "" {
		if _, e := os.Stat(kamelBin); e != nil && os.IsNotExist(e) {
			panic(e)
		}
		fmt.Printf("Using external kamel binary on path %s\n", kamelBin)
		c = &cobra.Command{
			DisableFlagParsing: true,
			Run: func(cmd *cobra.Command, args []string) {

				externalBin := exec.Command(kamelBin, args...)
				var stdout io.Reader
				stdout, err = externalBin.StdoutPipe()
				if err != nil {
					panic(err)
				}

				externalBin.Start()
				io.Copy(c.OutOrStdout(), stdout)
				externalBin.Wait()

			},
		}
	} else {
		c, err = cmd.NewKamelCommand(ctx)
	}
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
		logOptions := corev1.PodLogOptions{
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

func integrationPodPhase(ns string, name string) func() corev1.PodPhase {
	return func() corev1.PodPhase {
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

func integrationPod(ns string, name string) func() *corev1.Pod {
	return func() *corev1.Pod {
		lst := corev1.PodList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
		}
		err := testClient.List(testContext, &lst,
			k8sclient.InNamespace(ns),
			k8sclient.MatchingLabels{
				"camel.apache.org/integration": name,
			})
		if err != nil {
			panic(err)
		}
		if len(lst.Items) == 0 {
			return nil
		}
		return &lst.Items[0]
	}
}

func configMap(ns string, name string) func() *corev1.ConfigMap {
	return func() *corev1.ConfigMap {
		cm := corev1.ConfigMap{}
		key := k8sclient.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		err := testClient.Get(testContext, key, &cm)
		if err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			panic(err)
		}
		return &cm
	}
}

func service(ns string, name string) func() *corev1.Service {
	return func() *corev1.Service {
		svc := corev1.Service{}
		key := k8sclient.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		err := testClient.Get(testContext, key, &svc)
		if err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			panic(err)
		}
		return &svc
	}
}

func route(ns string, name string) func() *routev1.Route {
	return func() *routev1.Route {
		route := routev1.Route{}
		key := k8sclient.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		err := testClient.Get(testContext, key, &route)
		if err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			panic(err)
		}
		return &route
	}
}

func integrationCronJob(ns string, name string) func() *v1beta1.CronJob {
	return func() *v1beta1.CronJob {
		lst := v1beta1.CronJobList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "CronJob",
				APIVersion: v1beta1.SchemeGroupVersion.String(),
			},
		}
		err := testClient.List(testContext, &lst,
			k8sclient.InNamespace(ns),
			k8sclient.MatchingLabels{
				"camel.apache.org/integration": name,
			})
		if err != nil {
			panic(err)
		}
		if len(lst.Items) == 0 {
			return nil
		}
		return &lst.Items[0]
	}
}

func integration(ns string, name string) func() *v1.Integration {
	return func() *v1.Integration {
		it := v1.NewIntegration(ns, name)
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

func integrationProfile(ns string, name string) func() v1.TraitProfile {
	return func() v1.TraitProfile {
		it := integration(ns, name)()
		if it == nil {
			return ""
		}
		return it.Status.Profile
	}
}

func integrationPhase(ns string, name string) func() v1.IntegrationPhase {
	return func() v1.IntegrationPhase {
		it := integration(ns, name)()
		if it == nil {
			return ""
		}
		return it.Status.Phase
	}
}

func integrationSpecProfile(ns string, name string) func() v1.TraitProfile {
	return func() v1.TraitProfile {
		it := integration(ns, name)()
		if it == nil {
			return ""
		}
		return it.Spec.Profile
	}
}

func integrationKit(ns string, name string) func() string {
	return func() string {
		it := integration(ns, name)()
		if it == nil {
			return ""
		}
		return it.Status.Kit
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

func updateIntegration(ns string, name string, upd func(it *v1.Integration)) error {
	it := integration(ns, name)()
	if it == nil {
		return fmt.Errorf("no integration named %s found", name)
	}
	upd(it)
	return testClient.Update(testContext, it)
}

func kits(ns string) func() []v1.IntegrationKit {
	return func() []v1.IntegrationKit {
		lst := v1.NewIntegrationKitList()
		if err := testClient.List(testContext, &lst, k8sclient.InNamespace(ns)); err != nil {
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

func operatorPodPhase(ns string) func() corev1.PodPhase {
	return func() corev1.PodPhase {
		pod := operatorPod(ns)()
		if pod == nil {
			return ""
		}
		return pod.Status.Phase
	}
}

func configmap(ns string, name string) func() *corev1.ConfigMap {
	return func() *corev1.ConfigMap {
		cm := corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: corev1.SchemeGroupVersion.String(),
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

func knativeService(ns string, name string) func() *servingv1.Service {
	return func() *servingv1.Service {
		answer := servingv1.Service{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Service",
				APIVersion: servingv1.SchemeGroupVersion.String(),
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
		if err := testClient.Get(testContext, key, &answer); err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			log.Errorf(err, "Error while retrieving knative service %s", name)
			return nil
		}
		return &answer
	}
}

func deployment(ns string, name string) func() *appsv1.Deployment {
	return func() *appsv1.Deployment {
		answer := appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: appsv1.SchemeGroupVersion.String(),
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
		if err := testClient.Get(testContext, key, &answer); err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			log.Errorf(err, "Error while retrieving deployment %s", name)
			return nil
		}
		return &answer
	}
}

func build(ns string, name string) func() *v1.Build {
	return func() *v1.Build {
		build := v1.NewBuild(ns, name)
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

func platform(ns string) func() *v1.IntegrationPlatform {
	return func() *v1.IntegrationPlatform {
		lst := v1.NewIntegrationPlatformList()
		if err := testClient.List(testContext, &lst, k8sclient.InNamespace(ns)); err != nil {
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

func deletePlatform(ns string) func() bool {
	return func() bool {
		pl := platform(ns)()
		if pl == nil {
			return true
		}
		err := testClient.Delete(testContext, pl)
		if err != nil {
			log.Error(err, "Got error while deleting the platform")
		}
		return false
	}
}

func setPlatformVersion(ns string, version string) func() error {
	return func() error {
		p := platform(ns)()
		if p == nil {
			return errors.New("no platform found")
		}
		p.Status.Version = version
		return testClient.Status().Update(testContext, p)
	}
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

func platformPhase(ns string) func() v1.IntegrationPlatformPhase {
	return func() v1.IntegrationPlatformPhase {
		p := platform(ns)()
		if p == nil {
			return ""
		}
		return p.Status.Phase
	}
}

func platformProfile(ns string) func() v1.TraitProfile {
	return func() v1.TraitProfile {
		p := platform(ns)()
		if p == nil {
			return ""
		}
		return p.Status.Profile
	}
}

func operatorPod(ns string) func() *corev1.Pod {
	return func() *corev1.Pod {
		lst := corev1.PodList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: v1.SchemeGroupVersion.String(),
			},
		}
		err := testClient.List(testContext, &lst,
			k8sclient.InNamespace(ns),
			k8sclient.MatchingLabels{
				"camel.apache.org/component": "operator",
			})
		if err != nil {
			panic(err)
		}
		if len(lst.Items) == 0 {
			return nil
		}
		return &lst.Items[0]
	}
}

func operatorTryPodForceKill(ns string, timeSeconds int) {
	pod := operatorPod(ns)()
	if pod != nil {
		if err := testClient.Delete(testContext, pod, k8sclient.GracePeriodSeconds(timeSeconds)); err != nil {
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
		err := testClient.List(testContext, &lst,
			k8sclient.InNamespace(ns),
			k8sclient.MatchingLabels{
				"camel.apache.org/component": "operator",
			})
		if err != nil {
			return err
		}
		if len(lst.Items) == 0 {
			return errors.New("camel k operator not found")
		} else if len(lst.Items) > 1 {
			return errors.New("too many camel k operators")
		}

		operatorDeployment := lst.Items[0]
		operatorDeployment.Spec.Replicas = &replicas
		err = testClient.Update(testContext, &operatorDeployment)
		if err != nil {
			return err
		}

		if replicas == 0 {
			// speedup scale down by killing the pod
			operatorTryPodForceKill(ns, 10)
		}
		return nil
	}
}

func role(ns string) func() *rbacv1.Role {
	return func() *rbacv1.Role {
		lst := rbacv1.RoleList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Role",
				APIVersion: rbacv1.SchemeGroupVersion.String(),
			},
		}
		err := testClient.List(testContext, &lst,
			k8sclient.InNamespace(ns),
			k8sclient.MatchingLabels{
				"app": "camel-k",
			})
		if err != nil {
			panic(err)
		}
		if len(lst.Items) == 0 {
			return nil
		}
		return &lst.Items[0]
	}
}

func rolebinding(ns string) func() *rbacv1.RoleBinding {
	return func() *rbacv1.RoleBinding {
		lst := rbacv1.RoleBindingList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "RoleBinding",
				APIVersion: metav1.SchemeGroupVersion.String(),
			},
		}
		err := testClient.List(testContext, &lst,
			k8sclient.InNamespace(ns),
			k8sclient.MatchingLabels{
				"app": "camel-k",
			})
		if err != nil {
			panic(err)
		}
		if len(lst.Items) == 0 {
			return nil
		}
		return &lst.Items[0]
	}
}

func clusterrole(ns string) func() *rbacv1.ClusterRole {
	return func() *rbacv1.ClusterRole {
		lst := rbacv1.ClusterRoleList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ClusterRole",
				APIVersion: rbacv1.SchemeGroupVersion.String(),
			},
		}
		err := testClient.List(testContext, &lst,
			k8sclient.InNamespace(ns),
			k8sclient.MatchingLabels{
				"app": "camel-k",
			})
		if err != nil {
			panic(err)
		}
		if len(lst.Items) == 0 {
			return nil
		}
		return &lst.Items[0]
	}
}

func serviceaccount(ns, name string) func() *corev1.ServiceAccount {
	return func() *corev1.ServiceAccount {
		lst := corev1.ServiceAccountList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ServiceAccount",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
		}
		err := testClient.List(testContext, &lst,
			k8sclient.InNamespace(ns),
			k8sclient.MatchingLabels{
				"app": "camel-k",
			})
		if err != nil {
			panic(err)
		}
		if len(lst.Items) == 0 {
			return nil
		}
		return &lst.Items[0]
	}
}

/*
	Tekton
*/

func createOperatorServiceAccount(ns string) error {
	return install.Resource(testContext, testClient, ns, true, install.IdentityResourceCustomizer, "operator-service-account.yaml")
}

func createOperatorRole(ns string) (err error) {
	var oc bool
	if oc, err = openshift.IsOpenShift(testClient); err != nil {
		panic(err)
	}
	if oc {
		return install.Resource(testContext, testClient, ns, true, install.IdentityResourceCustomizer, "operator-role-openshift.yaml")
	}
	return install.Resource(testContext, testClient, ns, true, install.IdentityResourceCustomizer, "operator-role-kubernetes.yaml")
}

func createOperatorRoleBinding(ns string) error {
	return install.Resource(testContext, testClient, ns, true, install.IdentityResourceCustomizer, "operator-role-binding.yaml")
}

func createKamelPod(ns string, name string, command ...string) error {
	args := command
	for _, hook := range kamelHooks {
		args = hook(args)
	}
	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "camel-k-operator",
			RestartPolicy:      corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:    "kamel-runner",
					Image:   testImageName + ":" + testImageVersion,
					Command: append([]string{"kamel"}, args...),
				},
			},
		},
	}
	return testClient.Create(testContext, &pod)
}

/*
	Knative
*/

func createKnativeChannel(ns string, name string) func() error {
	return func() error {
		channel := messaging.InMemoryChannel{
			TypeMeta: metav1.TypeMeta{
				Kind:       "InMemoryChannel",
				APIVersion: messaging.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      name,
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
		lst := corev1.PodList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: v1.SchemeGroupVersion.String(),
			},
		}
		if err := testClient.List(testContext, &lst, k8sclient.InNamespace(ns)); err != nil && k8serrors.IsUnauthorized(err) {
			return 0
		} else if err != nil {
			log.Error(err, "Error while listing the pods")
			return 0
		}
		return len(lst.Items)
	}
}

func withNewTestNamespace(t *testing.T, doRun func(string)) {
	ns := newTestNamespace(false)
	defer deleteTestNamespace(ns)

	invokeUserTestCode(t, ns.GetName(), doRun)
}

func withNewTestNamespaceWithKnativeBroker(t *testing.T, doRun func(string)) {
	ns := newTestNamespace(true)
	defer deleteTestNamespace(ns)
	defer deleteKnativeBroker(ns)

	invokeUserTestCode(t, ns.GetName(), doRun)
}

func invokeUserTestCode(t *testing.T, ns string, doRun func(string)) {
	defer func() {
		if t.Failed() {

			if err := util.Dump(testClient, ns, t); err != nil {
				t.Logf("Error while dumping namespace %s: %v\n", ns, err)
			}
		}
	}()

	gomega.RegisterTestingT(t)
	doRun(ns)
}

func deleteKnativeBroker(ns metav1.Object) {
	nsRef := corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: ns.GetName(),
		},
	}
	nsKey, err := k8sclient.ObjectKeyFromObject(&nsRef)
	if err != nil {
		panic(err)
	}
	if err := testClient.Get(testContext, nsKey, &nsRef); err != nil {
		panic(err)
	}

	nsRef.SetLabels(make(map[string]string, 0))
	if err := testClient.Update(testContext, &nsRef); err != nil {
		panic(err)
	}
	broker := eventing.Broker{
		TypeMeta: metav1.TypeMeta{
			APIVersion: eventing.SchemeGroupVersion.String(),
			Kind:       "Broker",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns.GetName(),
			Name:      "default",
		},
	}
	if err := testClient.Delete(testContext, &broker); err != nil {
		panic(err)
	}
}

func deleteTestNamespace(ns metav1.Object) {
	var oc bool
	var err error
	if oc, err = openshift.IsOpenShift(testClient); err != nil {
		panic(err)
	} else if oc {
		prj := &projectv1.Project{
			TypeMeta: metav1.TypeMeta{
				APIVersion: projectv1.GroupVersion.String(),
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
	gomega.Eventually(numPods(ns.GetName()), 60*time.Second).Should(gomega.Equal(0))
}

func newTestNamespace(injectKnativeBroker bool) metav1.Object {
	var err error
	var oc bool
	var obj runtime.Object

	brokerLabel := "knative-eventing-injection"
	name := "test-" + uuid.New().String()

	if oc, err = openshift.IsOpenShift(testClient); err != nil {
		panic(err)
	} else if oc {
		obj = &projectv1.ProjectRequest{
			TypeMeta: metav1.TypeMeta{
				APIVersion: projectv1.GroupVersion.String(),
				Kind:       "ProjectRequest",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}
	} else {
		obj = &corev1.Namespace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}
	}

	if injectKnativeBroker {
		mo := obj.(metav1.Object)
		mo.SetLabels(map[string]string{
			brokerLabel: "enabled",
		})
	}

	if err = testClient.Create(testContext, obj); err != nil {
		panic(err)
	}
	// workaround https://github.com/openshift/origin/issues/3819
	if injectKnativeBroker && oc {
		// use Kubernetes API - https://access.redhat.com/solutions/2677921
		var namespace *corev1.Namespace
		if namespace, err = testClient.CoreV1().Namespaces().Get(name, metav1.GetOptions{}); err != nil {
			panic(err)
		} else {
			if _, ok := namespace.GetLabels()[brokerLabel]; !ok {
				namespace.SetLabels(map[string]string{
					brokerLabel: "enabled",
				})
				if err = testClient.Update(testContext, namespace); err != nil {
					panic("Unable to label project with knative-eventing-injection. This operation needs update permission on the project.")
				}
			}
		}
	}
	return obj.(metav1.Object)
}
