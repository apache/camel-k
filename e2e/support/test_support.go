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

package support

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/google/uuid"
	"github.com/onsi/gomega"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	eventing "knative.dev/eventing/pkg/apis/eventing/v1beta1"
	messaging "knative.dev/eventing/pkg/apis/messaging/v1beta1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"

	"github.com/apache/camel-k/e2e/support/util"
	"github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/cmd"
	"github.com/apache/camel-k/pkg/install"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/log"
	"github.com/apache/camel-k/pkg/util/openshift"

	// let's enable addons in all tests
	_ "github.com/apache/camel-k/addons"
)

const kubeConfigEnvVar = "KUBECONFIG"

var TestTimeoutShort = 1 * time.Minute
var TestTimeoutMedium = 5 * time.Minute
var TestTimeoutLong = 10 * time.Minute

var TestContext context.Context
var TestClient client.Client

// KamelHooks contains hooks useful to add option to kamel commands at runtime
var KamelHooks []func([]string) []string

var TestImageName = defaults.ImageName
var TestImageVersion = defaults.Version

func init() {
	// Register some resources used in e2e tests only
	client.FastMapperAllowedAPIGroups["project.openshift.io"] = true
	client.FastMapperAllowedAPIGroups["route.openshift.io"] = true
	client.FastMapperAllowedAPIGroups["eventing.knative.dev"] = true
	client.FastMapperAllowedAPIGroups["messaging.knative.dev"] = true
	client.FastMapperAllowedAPIGroups["serving.knative.dev"] = true

	var err error
	TestContext = context.TODO()
	TestClient, err = NewTestClient()
	if err != nil {
		panic(err)
	}

	// Defaults for testing
	imageName := os.Getenv("CAMEL_K_TEST_IMAGE_NAME")
	if imageName != "" {
		TestImageName = imageName
	}
	imageVersion := os.Getenv("CAMEL_K_TEST_IMAGE_VERSION")
	if imageVersion != "" {
		TestImageVersion = imageVersion
	}

	// Timeouts
	var duration time.Duration
	if value, ok := os.LookupEnv("CAMEL_K_TEST_TIMEOUT_SHORT"); ok {
		if duration, err = time.ParseDuration(value); err == nil {
			TestTimeoutShort = duration
		} else {
			fmt.Printf("Can't parse CAMEL_K_TEST_TIMEOUT_SHORT. Using default value: %s", TestTimeoutShort)
		}
	}

	if value, ok := os.LookupEnv("CAMEL_K_TEST_TIMEOUT_MEDIUM"); ok {
		if duration, err = time.ParseDuration(value); err == nil {
			TestTimeoutMedium = duration
		} else {
			fmt.Printf("Can't parse CAMEL_K_TEST_TIMEOUT_MEDIUM. Using default value: %s", TestTimeoutMedium)
		}
	}

	if value, ok := os.LookupEnv("CAMEL_K_TEST_TIMEOUT_LONG"); ok {
		if duration, err = time.ParseDuration(value); err == nil {
			TestTimeoutLong = duration
		} else {
			fmt.Printf("Can't parse CAMEL_K_TEST_TIMEOUT_LONG. Using default value: %s", TestTimeoutLong)
		}
	}

	gomega.SetDefaultEventuallyTimeout(TestTimeoutShort)

}

func NewTestClient() (client.Client, error) {
	return client.NewOutOfClusterClient(os.Getenv(kubeConfigEnvVar))
}

func Kamel(args ...string) *cobra.Command {
	return KamelWithContext(TestContext, args...)
}

func KamelWithContext(ctx context.Context, args ...string) *cobra.Command {
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
	for _, hook := range KamelHooks {
		args = hook(args)
	}
	c.SetArgs(args)
	return c
}

/*
	Curryied utility functions for testing
*/

func IntegrationLogs(ns string, name string) func() string {
	return func() string {
		pod := IntegrationPod(ns, name)()
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
		byteReader, err := TestClient.CoreV1().Pods(ns).GetLogs(pod.Name, &logOptions).Stream(TestContext)
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

func IntegrationPodPhase(ns string, name string) func() corev1.PodPhase {
	return func() corev1.PodPhase {
		pod := IntegrationPod(ns, name)()
		if pod == nil {
			return ""
		}
		return pod.Status.Phase
	}
}

func IntegrationPodImage(ns string, name string) func() string {
	return func() string {
		pod := IntegrationPod(ns, name)()
		if pod == nil || len(pod.Spec.Containers) == 0 {
			return ""
		}
		return pod.Spec.Containers[0].Image
	}
}

func IntegrationPod(ns string, name string) func() *corev1.Pod {
	return func() *corev1.Pod {
		lst := corev1.PodList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
		}
		err := TestClient.List(TestContext, &lst,
			k8sclient.InNamespace(ns),
			k8sclient.MatchingLabels{
				v1.IntegrationLabel: name,
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

func IntegrationCondition(ns string, name string, conditionType v1.IntegrationConditionType) func() corev1.ConditionStatus {
	return func() corev1.ConditionStatus {
		it := Integration(ns, name)()
		if it == nil {
			return "IntegrationMissing"
		}
		c := it.Status.GetCondition(conditionType)
		if c == nil {
			return "ConditionMissing"
		}
		return c.Status
	}
}

func ConfigMap(ns string, name string) func() *corev1.ConfigMap {
	return func() *corev1.ConfigMap {
		cm := corev1.ConfigMap{}
		key := k8sclient.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		err := TestClient.Get(TestContext, key, &cm)
		if err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			panic(err)
		}
		return &cm
	}
}

func Service(ns string, name string) func() *corev1.Service {
	return func() *corev1.Service {
		svc := corev1.Service{}
		key := k8sclient.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		err := TestClient.Get(TestContext, key, &svc)
		if err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			panic(err)
		}
		return &svc
	}
}

func Route(ns string, name string) func() *routev1.Route {
	return func() *routev1.Route {
		route := routev1.Route{}
		key := k8sclient.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		err := TestClient.Get(TestContext, key, &route)
		if err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			panic(err)
		}
		return &route
	}
}

func IntegrationCronJob(ns string, name string) func() *v1beta1.CronJob {
	return func() *v1beta1.CronJob {
		lst := v1beta1.CronJobList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "CronJob",
				APIVersion: v1beta1.SchemeGroupVersion.String(),
			},
		}
		err := TestClient.List(TestContext, &lst,
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

func Integration(ns string, name string) func() *v1.Integration {
	return func() *v1.Integration {
		it := v1.NewIntegration(ns, name)
		key := k8sclient.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		if err := TestClient.Get(TestContext, key, &it); err != nil && !k8serrors.IsNotFound(err) {
			panic(err)
		} else if err != nil && k8serrors.IsNotFound(err) {
			return nil
		}
		return &it
	}
}

func IntegrationVersion(ns string, name string) func() string {
	return func() string {
		it := Integration(ns, name)()
		if it == nil {
			return ""
		}
		return it.Status.Version
	}
}

func IntegrationProfile(ns string, name string) func() v1.TraitProfile {
	return func() v1.TraitProfile {
		it := Integration(ns, name)()
		if it == nil {
			return ""
		}
		return it.Status.Profile
	}
}

func IntegrationPhase(ns string, name string) func() v1.IntegrationPhase {
	return func() v1.IntegrationPhase {
		it := Integration(ns, name)()
		if it == nil {
			return ""
		}
		return it.Status.Phase
	}
}

func IntegrationSpecProfile(ns string, name string) func() v1.TraitProfile {
	return func() v1.TraitProfile {
		it := Integration(ns, name)()
		if it == nil {
			return ""
		}
		return it.Spec.Profile
	}
}

func IntegrationKit(ns string, name string) func() string {
	return func() string {
		it := Integration(ns, name)()
		if it == nil {
			return ""
		}
		return it.Status.Kit
	}
}

func SetIntegrationVersion(ns string, name string, version string) error {
	it := Integration(ns, name)()
	if it == nil {
		return fmt.Errorf("no integration named %s found", name)
	}
	it.Status.Version = version
	return TestClient.Status().Update(TestContext, it)
}

func UpdateIntegration(ns string, name string, upd func(it *v1.Integration)) error {
	it := Integration(ns, name)()
	if it == nil {
		return fmt.Errorf("no integration named %s found", name)
	}
	upd(it)
	return TestClient.Update(TestContext, it)
}

func Kits(ns string) func() []v1.IntegrationKit {
	return func() []v1.IntegrationKit {
		lst := v1.NewIntegrationKitList()
		if err := TestClient.List(TestContext, &lst, k8sclient.InNamespace(ns)); err != nil {
			panic(err)
		}
		return lst.Items
	}
}

func KitsWithVersion(ns string, version string) func() int {
	return func() int {
		count := 0
		for _, k := range Kits(ns)() {
			if k.Status.Version == version {
				count++
			}
		}
		return count
	}
}

func SetAllKitsVersion(ns string, version string) error {
	for _, k := range Kits(ns)() {
		kit := k
		kit.Status.Version = version
		if err := TestClient.Status().Update(TestContext, &kit); err != nil {
			return err
		}
	}
	return nil
}

func OperatorImage(ns string) func() string {
	return func() string {
		pod := OperatorPod(ns)()
		if pod != nil {
			if len(pod.Spec.Containers) > 0 {
				return pod.Spec.Containers[0].Image
			}
		}
		return ""
	}
}

func OperatorPodPhase(ns string) func() corev1.PodPhase {
	return func() corev1.PodPhase {
		pod := OperatorPod(ns)()
		if pod == nil {
			return ""
		}
		return pod.Status.Phase
	}
}

func Configmap(ns string, name string) func() *corev1.ConfigMap {
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
		if err := TestClient.Get(TestContext, key, &cm); err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			log.Error(err, "Error while retrieving configmap "+name)
			return nil
		}
		return &cm
	}
}

func KnativeService(ns string, name string) func() *servingv1.Service {
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
		if err := TestClient.Get(TestContext, key, &answer); err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			log.Errorf(err, "Error while retrieving knative service %s", name)
			return nil
		}
		return &answer
	}
}

func Deployment(ns string, name string) func() *appsv1.Deployment {
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
		if err := TestClient.Get(TestContext, key, &answer); err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			log.Errorf(err, "Error while retrieving deployment %s", name)
			return nil
		}
		return &answer
	}
}

func Build(ns string, name string) func() *v1.Build {
	return func() *v1.Build {
		build := v1.NewBuild(ns, name)
		key := k8sclient.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		if err := TestClient.Get(TestContext, key, &build); err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			log.Error(err, "Error while retrieving build "+name)
			return nil
		}
		return &build
	}
}

func Platform(ns string) func() *v1.IntegrationPlatform {
	return func() *v1.IntegrationPlatform {
		lst := v1.NewIntegrationPlatformList()
		if err := TestClient.List(TestContext, &lst, k8sclient.InNamespace(ns)); err != nil {
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

func DeletePlatform(ns string) func() bool {
	return func() bool {
		pl := Platform(ns)()
		if pl == nil {
			return true
		}
		err := TestClient.Delete(TestContext, pl)
		if err != nil {
			log.Error(err, "Got error while deleting the platform")
		}
		return false
	}
}

func SetPlatformVersion(ns string, version string) func() error {
	return func() error {
		p := Platform(ns)()
		if p == nil {
			return errors.New("no platform found")
		}
		p.Status.Version = version
		return TestClient.Status().Update(TestContext, p)
	}
}

func PlatformVersion(ns string) func() string {
	return func() string {
		p := Platform(ns)()
		if p == nil {
			return ""
		}
		return p.Status.Version
	}
}

func PlatformPhase(ns string) func() v1.IntegrationPlatformPhase {
	return func() v1.IntegrationPlatformPhase {
		p := Platform(ns)()
		if p == nil {
			return ""
		}
		return p.Status.Phase
	}
}

func PlatformProfile(ns string) func() v1.TraitProfile {
	return func() v1.TraitProfile {
		p := Platform(ns)()
		if p == nil {
			return ""
		}
		return p.Status.Profile
	}
}

func OperatorPod(ns string) func() *corev1.Pod {
	return func() *corev1.Pod {
		lst := corev1.PodList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: v1.SchemeGroupVersion.String(),
			},
		}
		err := TestClient.List(TestContext, &lst,
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

func OperatorTryPodForceKill(ns string, timeSeconds int) {
	pod := OperatorPod(ns)()
	if pod != nil {
		if err := TestClient.Delete(TestContext, pod, k8sclient.GracePeriodSeconds(timeSeconds)); err != nil {
			log.Error(err, "cannot forcefully kill the pod")
		}
	}
}

func ScaleOperator(ns string, replicas int32) func() error {
	return func() error {
		lst := appsv1.DeploymentList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: appsv1.SchemeGroupVersion.String(),
			},
		}
		err := TestClient.List(TestContext, &lst,
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
		err = TestClient.Update(TestContext, &operatorDeployment)
		if err != nil {
			return err
		}

		if replicas == 0 {
			// speedup scale down by killing the pod
			OperatorTryPodForceKill(ns, 10)
		}
		return nil
	}
}

func Role(ns string) func() *rbacv1.Role {
	return func() *rbacv1.Role {
		lst := rbacv1.RoleList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Role",
				APIVersion: rbacv1.SchemeGroupVersion.String(),
			},
		}
		err := TestClient.List(TestContext, &lst,
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

func Rolebinding(ns string) func() *rbacv1.RoleBinding {
	return func() *rbacv1.RoleBinding {
		lst := rbacv1.RoleBindingList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "RoleBinding",
				APIVersion: metav1.SchemeGroupVersion.String(),
			},
		}
		err := TestClient.List(TestContext, &lst,
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

func ServiceAccount(ns, name string) func() *corev1.ServiceAccount {
	return func() *corev1.ServiceAccount {
		lst := corev1.ServiceAccountList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ServiceAccount",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
		}
		err := TestClient.List(TestContext, &lst,
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

func CreateOperatorServiceAccount(ns string) error {
	return install.Resource(TestContext, TestClient, ns, true, install.IdentityResourceCustomizer, "operator-service-account.yaml")
}

func CreateOperatorRole(ns string) (err error) {
	var oc bool
	if oc, err = openshift.IsOpenShift(TestClient); err != nil {
		panic(err)
	}
	if oc {
		return install.Resource(TestContext, TestClient, ns, true, install.IdentityResourceCustomizer, "operator-role-openshift.yaml")
	}
	return install.Resource(TestContext, TestClient, ns, true, install.IdentityResourceCustomizer, "operator-role-kubernetes.yaml")
}

func CreateOperatorRoleBinding(ns string) error {
	return install.Resource(TestContext, TestClient, ns, true, install.IdentityResourceCustomizer, "operator-role-binding.yaml")
}

func CreateKamelPod(ns string, name string, command ...string) error {
	args := command
	for _, hook := range KamelHooks {
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
					Image:   TestImageName + ":" + TestImageVersion,
					Command: append([]string{"kamel"}, args...),
				},
			},
		},
	}
	return TestClient.Create(TestContext, &pod)
}

/*
	Knative
*/

func CreateKnativeChannelv1Alpha1(ns string, name string) func() error {
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
		return TestClient.Create(TestContext, &channel)
	}
}

func CreateKnativeChannelv1Beta1(ns string, name string) func() error {
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
		return TestClient.Create(TestContext, &channel)
	}
}

/*
	Kamelets
*/

func CreateTimerKamelet(ns string, name string) func() error {
	return func() error {
		kamelet := v1alpha1.Kamelet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      name,
			},
			Spec: v1alpha1.KameletSpec{
				Definition: v1alpha1.JSONSchemaProps{
					Properties: map[string]v1alpha1.JSONSchemaProps{
						"message": {
							Type: "string",
						},
					},
				},
				Flow: asFlow(map[string]interface{}{
					"from": map[string]interface{}{
						"uri": "timer:tick",
						"steps": []map[string]interface{}{
							{
								"set-body": map[string]interface{}{
									"constant": "{{message}}",
								},
							},
							{
								"to": "kamelet:sink",
							},
						},
					},
				}),
			},
		}
		return TestClient.Create(TestContext, &kamelet)
	}
}

func BindKameletTo(ns, name, from string, to corev1.ObjectReference, properties map[string]string) func() error {
	return func() error {
		kb := v1alpha1.KameletBinding{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      name,
			},
			Spec: v1alpha1.KameletBindingSpec{
				Source: v1alpha1.Endpoint{
					Ref: &corev1.ObjectReference{
						Kind:       "Kamelet",
						APIVersion: v1alpha1.SchemeGroupVersion.String(),
						Name:       from,
					},
					Properties: asEndpointProperties(properties),
				},
				Sink: v1alpha1.Endpoint{
					Ref:        &to,
					Properties: asEndpointProperties(map[string]string{}),
				},
			},
		}
		return kubernetes.ReplaceResource(TestContext, TestClient, &kb)
	}
}

func asFlow(source map[string]interface{}) *v1.Flow {
	bytes, err := json.Marshal(source)
	if err != nil {
		panic(err)
	}
	return &v1.Flow{
		RawMessage: bytes,
	}
}

func asEndpointProperties(props map[string]string) v1alpha1.EndpointProperties {
	bytes, err := json.Marshal(props)
	if err != nil {
		panic(err)
	}
	return v1alpha1.EndpointProperties{
		RawMessage: bytes,
	}
}

/*
	Namespace testing functions
*/

func NumPods(ns string) func() int {
	return func() int {
		lst := corev1.PodList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: v1.SchemeGroupVersion.String(),
			},
		}
		if err := TestClient.List(TestContext, &lst, k8sclient.InNamespace(ns)); err != nil && k8serrors.IsUnauthorized(err) {
			return 0
		} else if err != nil {
			log.Error(err, "Error while listing the pods")
			return 0
		}
		return len(lst.Items)
	}
}

func WithNewTestNamespace(t *testing.T, doRun func(string)) {
	ns := NewTestNamespace(false)
	defer DeleteTestNamespace(t, ns)
	defer UserCleanup()

	InvokeUserTestCode(t, ns.GetName(), doRun)
}

func WithNewTestNamespaceWithKnativeBroker(t *testing.T, doRun func(string)) {
	ns := NewTestNamespace(true)
	defer DeleteTestNamespace(t, ns)
	defer DeleteKnativeBroker(ns)
	defer UserCleanup()

	InvokeUserTestCode(t, ns.GetName(), doRun)
}

func UserCleanup() {
	userCmd := os.Getenv("KAMEL_TEST_CLEANUP")
	if userCmd != "" {
		fmt.Printf("Executing user cleanup command: %s\n", userCmd)
		cmdSplit := strings.Split(userCmd, " ")
		command := exec.Command(cmdSplit[0], cmdSplit[1:]...)
		command.Stderr = os.Stderr
		command.Stdout = os.Stdout
		if err := command.Run(); err != nil {
			fmt.Printf("An error occurred during user cleanup command execution: %v\n", err)
		} else {
			fmt.Printf("User cleanup command completed successfully\n")
		}
	}
}

func InvokeUserTestCode(t *testing.T, ns string, doRun func(string)) {
	defer func() {
		if t.Failed() {

			if err := util.Dump(TestContext, TestClient, ns, t); err != nil {
				t.Logf("Error while dumping namespace %s: %v\n", ns, err)
			}
		}
	}()

	gomega.RegisterTestingT(t)
	doRun(ns)
}

func DeleteKnativeBroker(ns metav1.Object) {
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
	if err := TestClient.Get(TestContext, nsKey, &nsRef); err != nil {
		panic(err)
	}

	nsRef.SetLabels(make(map[string]string, 0))
	if err := TestClient.Update(TestContext, &nsRef); err != nil {
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
	if err := TestClient.Delete(TestContext, &broker); err != nil {
		panic(err)
	}
}

func DeleteTestNamespace(t *testing.T, ns metav1.Object) {
	var oc bool
	var err error
	if oc, err = openshift.IsOpenShift(TestClient); err != nil {
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
		if err := TestClient.Delete(TestContext, prj); err != nil {
			t.Logf("Warning: cannot delete test project %q", prj.Name)
		}
	} else {
		if err := TestClient.Delete(TestContext, ns.(runtime.Object)); err != nil {
			t.Logf("Warning: cannot delete test namespace %q", ns.GetName())
		}
	}

	// Wait for all pods to be deleted
	pods := NumPods(ns.GetName())()
	for i := 0; pods > 0 && i < 60; i++ {
		time.Sleep(1 * time.Second)
		pods = NumPods(ns.GetName())()
	}
	if pods > 0 {
		t.Logf("Warning: some pods are still running in namespace %q after deletion (%d)", ns.GetName(), pods)
	}
}

func NewTestNamespace(injectKnativeBroker bool) metav1.Object {
	var err error
	var oc bool
	var obj runtime.Object

	brokerLabel := "knative-eventing-injection"
	name := "test-" + uuid.New().String()

	if oc, err = openshift.IsOpenShift(TestClient); err != nil {
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

	if err = TestClient.Create(TestContext, obj); err != nil {
		panic(err)
	}
	// workaround https://github.com/openshift/origin/issues/3819
	if injectKnativeBroker && oc {
		// use Kubernetes API - https://access.redhat.com/solutions/2677921
		var namespace *corev1.Namespace
		if namespace, err = TestClient.CoreV1().Namespaces().Get(TestContext, name, metav1.GetOptions{}); err != nil {
			panic(err)
		} else {
			if _, ok := namespace.GetLabels()[brokerLabel]; !ok {
				namespace.SetLabels(map[string]string{
					brokerLabel: "enabled",
				})
				if err = TestClient.Update(TestContext, namespace); err != nil {
					panic("Unable to label project with knative-eventing-injection. This operation needs update permission on the project.")
				}
			}
		}
	}
	return obj.(metav1.Object)
}
