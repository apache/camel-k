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
	"bufio"
	"bytes"
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

	"github.com/google/uuid"
	"github.com/onsi/gomega"
	"github.com/spf13/cobra"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/batch/v1beta1"
	coordination "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	eventing "knative.dev/eventing/pkg/apis/eventing/v1"
	messaging "knative.dev/eventing/pkg/apis/messaging/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"

	"github.com/apache/camel-k/e2e/support/util"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/cmd"
	"github.com/apache/camel-k/pkg/install"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/kubernetes"
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
var testClient client.Client

func TestClient() client.Client {
	if testClient != nil {
		return testClient
	}
	var err error
	testClient, err = NewTestClient()
	if err != nil {
		panic(err)
	}
	return testClient
}

func SyncClient() client.Client {
	var err error
	testClient, err = NewTestClient()
	if err != nil {
		panic(err)
	}
	return testClient
}

// KamelHooks contains hooks useful to add option to kamel commands at runtime
var KamelHooks []func([]string) []string

var TestImageName = defaults.ImageName
var TestImageVersion = defaults.Version

func init() {
	// Register some resources used in e2e tests only
	client.FastMapperAllowedAPIGroups["coordination.k8s.io"] = true
	client.FastMapperAllowedAPIGroups["project.openshift.io"] = true
	client.FastMapperAllowedAPIGroups["route.openshift.io"] = true
	client.FastMapperAllowedAPIGroups["eventing.knative.dev"] = true
	client.FastMapperAllowedAPIGroups["messaging.knative.dev"] = true
	client.FastMapperAllowedAPIGroups["serving.knative.dev"] = true
	client.FastMapperAllowedAPIGroups["operators.coreos.com"] = true
	client.FastMapperAllowedAPIGroups["policy"] = true

	var err error
	TestContext = context.TODO()

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
			RunE: func(cmd *cobra.Command, args []string) error {
				externalBin := exec.CommandContext(ctx, kamelBin, args...)
				var stdout io.Reader
				stdout, err = externalBin.StdoutPipe()
				if err != nil {
					panic(err)
				}
				err := externalBin.Start()
				if err != nil {
					return err
				}
				_, err = io.Copy(c.OutOrStdout(), stdout)
				if err != nil {
					return err
				}
				err = externalBin.Wait()
				if err != nil {
					return err
				}
				return nil
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

func IntegrationLogs(ns, name string) func() string {
	return func() string {
		pod := IntegrationPod(ns, name)()
		if pod == nil {
			return ""
		}

		options := corev1.PodLogOptions{
			TailLines: pointer.Int64Ptr(100),
		}

		if len(pod.Spec.Containers) > 1 {
			options.Container = pod.Spec.Containers[0].Name
		}

		return Logs(ns, pod.Name, options)()
	}
}

func Logs(ns, podName string, options corev1.PodLogOptions) func() string {
	return func() string {
		byteReader, err := TestClient().CoreV1().Pods(ns).GetLogs(podName, &options).Stream(TestContext)
		if err != nil {
			log.Error(err, "Error while reading container logs")
			return ""
		}
		defer func() {
			if err := byteReader.Close(); err != nil {
				log.Error(err, "Error closing the stream")
			}
		}()

		bytes, err := ioutil.ReadAll(byteReader)
		if err != nil {
			log.Error(err, "Error while reading container logs")
			return ""
		}
		return string(bytes)
	}
}

func StructuredLogs(ns, podName string, options corev1.PodLogOptions, ignoreParseErrors bool) []util.LogEntry {
	byteReader, err := TestClient().CoreV1().Pods(ns).GetLogs(podName, &options).Stream(TestContext)
	if err != nil {
		log.Error(err, "Error while reading container logs")
		return nil
	}
	defer func() {
		if err := byteReader.Close(); err != nil {
			log.Error(err, "Error closing the stream")
		}
	}()

	entries := make([]util.LogEntry, 0)
	scanner := bufio.NewScanner(byteReader)
	for scanner.Scan() {
		entry := util.LogEntry{}
		t := scanner.Text()
		err := json.Unmarshal([]byte(t), &entry)
		if err != nil {
			if ignoreParseErrors {
				continue
			} else {
				log.Errorf(err, "Unable to parse structured content: %s", t)
				return nil
			}
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		log.Error(err, "Error while scanning container logs")
		return nil
	}

	return entries
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
		pods := IntegrationPods(ns, name)()
		if len(pods) == 0 {
			return nil
		}
		return &pods[0]
	}
}

func IntegrationPods(ns string, name string) func() []corev1.Pod {
	return func() []corev1.Pod {
		lst := corev1.PodList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
		}
		err := TestClient().List(TestContext, &lst,
			ctrl.InNamespace(ns),
			ctrl.MatchingLabels{
				v1.IntegrationLabel: name,
			})
		if err != nil {
			panic(err)
		}
		return lst.Items
	}
}

func IntegrationSpecReplicas(ns string, name string) func() *int32 {
	return func() *int32 {
		it := Integration(ns, name)()
		if it == nil {
			return nil
		}
		return it.Spec.Replicas
	}
}

func IntegrationStatusReplicas(ns string, name string) func() *int32 {
	return func() *int32 {
		it := Integration(ns, name)()
		if it == nil {
			return nil
		}
		return it.Status.Replicas
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

func Lease(ns string, name string) func() *coordination.Lease {
	return func() *coordination.Lease {
		lease := coordination.Lease{}
		key := ctrl.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		err := TestClient().Get(TestContext, key, &lease)
		if err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			panic(err)
		}
		return &lease
	}
}

func Nodes() func() []corev1.Node {
	return func() []corev1.Node {
		nodes := &corev1.NodeList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "NodeList",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
		}
		err := TestClient().List(TestContext, nodes)
		if err != nil {
			panic(err)
		}
		return nodes.Items
	}
}

func Node(name string) func() *corev1.Node {
	return func() *corev1.Node {
		node := &corev1.Node{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Node",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}
		err := TestClient().Get(TestContext, ctrl.ObjectKeyFromObject(node), node)
		if err != nil {
			panic(err)
		}
		return node
	}
}

func Service(ns string, name string) func() *corev1.Service {
	return func() *corev1.Service {
		svc := corev1.Service{}
		key := ctrl.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		err := TestClient().Get(TestContext, key, &svc)
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
		key := ctrl.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		err := TestClient().Get(TestContext, key, &route)
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
		err := TestClient().List(TestContext, &lst,
			ctrl.InNamespace(ns),
			ctrl.MatchingLabels{
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
		key := ctrl.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		if err := TestClient().Get(TestContext, key, &it); err != nil && !k8serrors.IsNotFound(err) {
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
		if it.Status.IntegrationKit == nil {
			return ""
		}
		return it.Status.IntegrationKit.Name
	}
}

func UpdateIntegration(ns string, name string, upd func(it *v1.Integration)) error {
	it := Integration(ns, name)()
	if it == nil {
		return fmt.Errorf("no integration named %s found", name)
	}
	upd(it)
	return TestClient().Update(TestContext, it)
}

func ScaleIntegration(ns string, name string, replicas int32) error {
	return UpdateIntegration(ns, name, func(it *v1.Integration) {
		it.Spec.Replicas = &replicas
	})
}

func Kits(ns string, filters ...func(*v1.IntegrationKit) bool) func() []v1.IntegrationKit {
	return func() []v1.IntegrationKit {
		list := v1.NewIntegrationKitList()
		if err := TestClient().List(TestContext, &list, ctrl.InNamespace(ns)); err != nil {
			panic(err)
		}

		if len(filters) == 0 {
			filters = []func(*v1.IntegrationKit) bool{
				func(kit *v1.IntegrationKit) bool {
					return true
				},
			}
		}

		var kits []v1.IntegrationKit
	kits:
		for _, kit := range list.Items {
			for _, filter := range filters {
				if !filter(&kit) {
					continue kits
				}
			}
			kits = append(kits, kit)
		}

		return kits
	}
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
		key := ctrl.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		if err := TestClient().Get(TestContext, key, &cm); err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			log.Error(err, "Error while retrieving configmap "+name)
			return nil
		}
		return &cm
	}
}

func NewPlainTextConfigmap(ns string, name string, data map[string]string) error {
	cm := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
		Data: data,
	}
	return TestClient().Create(TestContext, &cm)
}

func NewBinaryConfigmap(ns string, name string, data map[string][]byte) error {
	cm := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
		BinaryData: data,
	}
	return TestClient().Create(TestContext, &cm)
}

func NewPlainTextSecret(ns string, name string, data map[string]string) error {
	sec := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
		StringData: data,
	}
	return TestClient().Create(TestContext, &sec)
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
		key := ctrl.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		if err := TestClient().Get(TestContext, key, &answer); err != nil && k8serrors.IsNotFound(err) {
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
		key := ctrl.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		if err := TestClient().Get(TestContext, key, &answer); err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			log.Errorf(err, "Error while retrieving deployment %s", name)
			return nil
		}
		return &answer
	}
}

func DeploymentCondition(ns string, name string, conditionType appsv1.DeploymentConditionType) func() appsv1.DeploymentCondition {
	return func() appsv1.DeploymentCondition {
		deployment := Deployment(ns, name)()

		condition := appsv1.DeploymentCondition{
			Status: corev1.ConditionUnknown,
		}

		for _, c := range deployment.Status.Conditions {
			if c.Type == conditionType {
				condition = c
				break
			}
		}

		return condition
	}
}

func Build(ns string, name string) func() *v1.Build {
	return func() *v1.Build {
		build := v1.NewBuild(ns, name)
		key := ctrl.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		if err := TestClient().Get(TestContext, key, &build); err != nil && k8serrors.IsNotFound(err) {
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
		if err := TestClient().List(TestContext, &lst, ctrl.InNamespace(ns)); err != nil {
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
		err := TestClient().Delete(TestContext, pl)
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
		return TestClient().Status().Update(TestContext, p)
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
		err := TestClient().List(TestContext, &lst,
			ctrl.InNamespace(ns),
			ctrl.MatchingLabels{
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

func Role(ns string) func() *rbacv1.Role {
	return func() *rbacv1.Role {
		lst := rbacv1.RoleList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Role",
				APIVersion: rbacv1.SchemeGroupVersion.String(),
			},
		}
		err := TestClient().List(TestContext, &lst,
			ctrl.InNamespace(ns),
			ctrl.MatchingLabels{
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

func RoleBinding(ns string) func() *rbacv1.RoleBinding {
	return func() *rbacv1.RoleBinding {
		lst := rbacv1.RoleBindingList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "RoleBinding",
				APIVersion: metav1.SchemeGroupVersion.String(),
			},
		}
		err := TestClient().List(TestContext, &lst,
			ctrl.InNamespace(ns),
			ctrl.MatchingLabels{
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
		err := TestClient().List(TestContext, &lst,
			ctrl.InNamespace(ns),
			ctrl.MatchingLabels{
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

func KameletList(ns string) func() []v1alpha1.Kamelet {
	return func() []v1alpha1.Kamelet {
		lst := v1alpha1.NewKameletList()
		err := TestClient().List(TestContext, &lst, ctrl.InNamespace(ns))
		if err != nil {
			panic(err)
		}
		return lst.Items
	}
}

func Kamelet(name string, ns string) func() *v1alpha1.Kamelet {
	return func() *v1alpha1.Kamelet {
		it := v1alpha1.NewKamelet(ns, name)
		key := ctrl.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		if err := TestClient().Get(TestContext, key, &it); err != nil && !k8serrors.IsNotFound(err) {
			panic(err)
		} else if err != nil && k8serrors.IsNotFound(err) {
			return nil
		}
		return &it
	}
}

/*
	Tekton
*/

func CreateOperatorServiceAccount(ns string) error {
	return install.Resource(TestContext, TestClient(), ns, true, install.IdentityResourceCustomizer, "/manager/operator-service-account.yaml")
}

func CreateOperatorRole(ns string) (err error) {
	oc, err := openshift.IsOpenShift(TestClient())
	if err != nil {
		panic(err)
	}
	err = install.Resource(TestContext, TestClient(), ns, true, install.IdentityResourceCustomizer, "/rbac/operator-role-kubernetes.yaml")
	if err != nil {
		return err
	}
	if oc {
		return install.Resource(TestContext, TestClient(), ns, true, install.IdentityResourceCustomizer, "/rbac/operator-role-openshift.yaml")
	}
	return nil
}

func CreateOperatorRoleBinding(ns string) error {
	oc, err := openshift.IsOpenShift(TestClient())
	if err != nil {
		panic(err)
	}
	err = install.Resource(TestContext, TestClient(), ns, true, install.IdentityResourceCustomizer, "/rbac/operator-role-binding.yaml")
	if err != nil {
		return err
	}
	if oc {
		return install.Resource(TestContext, TestClient(), ns, true, install.IdentityResourceCustomizer, "/rbac/operator-role-binding-openshift.yaml")
	}
	return nil
}

func CreateKamelPod(ns string, name string, command ...string) error {
	args := command
	for _, hook := range KamelHooks {
		args = hook(args)
	}
	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: corev1.SchemeGroupVersion.String(),
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
	return TestClient().Create(TestContext, &pod)
}

/*
	Knative
*/

func CreateKnativeChannel(ns string, name string) func() error {
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
		return TestClient().Create(TestContext, &channel)
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
				Definition: &v1alpha1.JSONSchemaProps{
					Properties: map[string]v1alpha1.JSONSchemaProp{
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
		return TestClient().Create(TestContext, &kamelet)
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
		return kubernetes.ReplaceResource(TestContext, TestClient(), &kb)
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

func asEndpointProperties(props map[string]string) *v1alpha1.EndpointProperties {
	bytes, err := json.Marshal(props)
	if err != nil {
		panic(err)
	}
	return &v1alpha1.EndpointProperties{
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
		if err := TestClient().List(TestContext, &lst, ctrl.InNamespace(ns)); err != nil && k8serrors.IsUnauthorized(err) {
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
			if err := util.Dump(TestContext, TestClient(), ns, t); err != nil {
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
	nsKey := ctrl.ObjectKeyFromObject(&nsRef)
	if err := TestClient().Get(TestContext, nsKey, &nsRef); err != nil {
		panic(err)
	}

	nsRef.SetLabels(make(map[string]string, 0))
	if err := TestClient().Update(TestContext, &nsRef); err != nil {
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
	if err := TestClient().Delete(TestContext, &broker); err != nil {
		panic(err)
	}
}

func DeleteTestNamespace(t *testing.T, ns ctrl.Object) {
	var oc bool
	var err error
	if oc, err = openshift.IsOpenShift(TestClient()); err != nil {
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
		if err := TestClient().Delete(TestContext, prj); err != nil {
			t.Logf("Warning: cannot delete test project %q", prj.Name)
		}
	} else {
		if err := TestClient().Delete(TestContext, ns); err != nil {
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

func NewTestNamespace(injectKnativeBroker bool) ctrl.Object {
	var err error
	var oc bool
	var obj ctrl.Object

	brokerLabel := "eventing.knative.dev/injection"
	name := "test-" + uuid.New().String()

	if oc, err = openshift.IsOpenShift(TestClient()); err != nil {
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

	if err = TestClient().Create(TestContext, obj); err != nil {
		panic(err)
	}
	// workaround https://github.com/openshift/origin/issues/3819
	if injectKnativeBroker && oc {
		// use Kubernetes API - https://access.redhat.com/solutions/2677921
		var namespace *corev1.Namespace
		if namespace, err = TestClient().CoreV1().Namespaces().Get(TestContext, name, metav1.GetOptions{}); err != nil {
			panic(err)
		} else {
			if _, ok := namespace.GetLabels()[brokerLabel]; !ok {
				namespace.SetLabels(map[string]string{
					brokerLabel: "enabled",
				})
				if err = TestClient().Update(TestContext, namespace); err != nil {
					panic("Unable to label project with knative-eventing-injection. This operation needs update permission on the project.")
				}
			}
		}
	}
	return obj
}

func GetOutputString(command *cobra.Command) string {
	var buf bytes.Buffer
	reader, writer, err := os.Pipe()
	if err != nil {
		panic(err)
	}

	command.SetOut(writer)
	command.Execute()

	writer.Close()
	defer reader.Close()

	buf.ReadFrom(reader)

	return buf.String()
}

func GetOutputStringAsync(cmd *cobra.Command) func() string {
	var buffer bytes.Buffer
	stdout := bufio.NewWriter(&buffer)

	cmd.SetOut(stdout)
	go cmd.Execute()

	return func() string {
		return buffer.String()
	}
}
