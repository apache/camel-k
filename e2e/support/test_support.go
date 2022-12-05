//go:build integration
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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	consoleV1 "github.com/openshift/api/console/v1"
	"github.com/stretchr/testify/assert"

	"github.com/google/uuid"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	coordination "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	eventing "knative.dev/eventing/pkg/apis/eventing/v1"
	messaging "knative.dev/eventing/pkg/apis/messaging/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	configv1 "github.com/openshift/api/config/v1"
	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"

	"github.com/apache/camel-k/e2e/support/util"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/cmd"
	"github.com/apache/camel-k/pkg/install"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/log"
	"github.com/apache/camel-k/pkg/util/openshift"
	"github.com/apache/camel-k/pkg/util/patch"

	// let's enable addons in all tests
	_ "github.com/apache/camel-k/addons"
)

const kubeConfigEnvVar = "KUBECONFIG"

var TestTimeoutShort = 1 * time.Minute
var TestTimeoutMedium = 5 * time.Minute
var TestTimeoutLong = 10 * time.Minute

// TestTimeoutVeryLong should be used only for testing native builds.
var TestTimeoutVeryLong = 90 * time.Minute

var NoOlmOperatorImage string

var TestContext context.Context
var testClient client.Client

var testLocus *testing.T

func setTestLocus(t *testing.T) {
	testLocus = t
}

//
// Only panic the test if absolutely necessary and there is
// no test locus. In most cases, the test should fail gracefully
// using the test locus to error out and fail now.
//
func failTest(err error) {
	if testLocus != nil {
		testLocus.Error(err)
		testLocus.FailNow()
	} else {
		panic(err)
	}
}

func TestClient() client.Client {
	if testClient != nil {
		return testClient
	}
	var err error
	testClient, err = NewTestClient()
	if err != nil {
		failTest(err)
	}
	return testClient
}

func SyncClient() client.Client {
	var err error
	testClient, err = NewTestClient()
	if err != nil {
		failTest(err)
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
	client.FastMapperAllowedAPIGroups["config.openshift.io"] = true
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

	if imageNoOlm, ok := os.LookupEnv("CAMEL_K_TEST_NO_OLM_OPERATOR_IMAGE"); ok {
		if imageNoOlm != "" {
			NoOlmOperatorImage = imageNoOlm
		} else {
			fmt.Printf("Can't parse CAMEL_K_TEST_NO_OLM_OPERATOR_IMAGE. Using default value from kamel")
		}
	}

	if value, ok := os.LookupEnv("CAMEL_K_TEST_TIMEOUT_LONG"); ok {
		if duration, err = time.ParseDuration(value); err == nil {
			TestTimeoutLong = duration
		} else {
			fmt.Printf("Can't parse CAMEL_K_TEST_TIMEOUT_LONG. Using default value: %s", TestTimeoutLong)
		}
	}

	// Gomega settings
	gomega.SetDefaultEventuallyTimeout(TestTimeoutShort)
	// Disable object truncation on test results
	format.MaxLength = 0

}

func NewTestClient() (client.Client, error) {
	return client.NewOutOfClusterClient(os.Getenv(kubeConfigEnvVar))
}

func Kamel(args ...string) *cobra.Command {
	return KamelWithContext(TestContext, args...)
}

func KamelInstall(namespace string, args ...string) *cobra.Command {
	return KamelInstallWithID(platform.DefaultPlatformName, namespace, args...)
}

func KamelInstallWithID(operatorID string, namespace string, args ...string) *cobra.Command {
	return KamelInstallWithContext(TestContext, operatorID, namespace, args...)
}

func KamelInstallWithContext(ctx context.Context, operatorID string, namespace string, args ...string) *cobra.Command {
	var installArgs []string

	globalTest := os.Getenv("CAMEL_K_FORCE_GLOBAL_TEST") == "true"
	if globalTest {
		fmt.Printf("Executing as global test\n")

		if err := verifyGlobalOperator(); err != nil {
			failTest(err)
		}

		// Have a global operator pod watching all namespaces
		// so ensure an integration platform is installed in target namespace
		installArgs = []string{"install", "--skip-operator-setup", "-n", namespace}
	} else {
		// NOT global so proceed with local namespaced kamel install using the operator id
		installArgs = []string{"install", "-n", namespace, "--operator-id", operatorID}
	}

	logLevel := os.Getenv("CAMEL_K_TEST_LOG_LEVEL")
	if len(logLevel) > 0 {
		fmt.Printf("Setting log-level to %s\n", logLevel)
		installArgs = append(installArgs, "--log-level", logLevel)
	}

	mvnCLIOptions := os.Getenv("CAMEL_K_TEST_MAVEN_CLI_OPTIONS")
	if len(mvnCLIOptions) > 0 {
		// Split the string by spaces
		mvnCLIArr := strings.Split(mvnCLIOptions, " ")
		for _, mc := range mvnCLIArr {
			mc = strings.Trim(mc, " ")
			if len(mc) == 0 {
				continue
			}

			fmt.Printf("Adding maven cli option %s\n", mc)
			installArgs = append(installArgs, "--maven-cli-option", mc)
		}
	}

	installArgs = append(installArgs, args...)
	return KamelWithContext(ctx, installArgs...)
}

func KamelRun(namespace string, args ...string) *cobra.Command {
	return KamelRunWithID(platform.DefaultPlatformName, namespace, args...)
}

func KamelRunWithID(operatorID string, namespace string, args ...string) *cobra.Command {
	return KamelRunWithContext(TestContext, operatorID, namespace, args...)
}

func KamelRunWithContext(ctx context.Context, operatorID string, namespace string, args ...string) *cobra.Command {
	return KamelCommandWithContext(ctx, "run", operatorID, namespace, args...)
}

func KamelBind(namespace string, args ...string) *cobra.Command {
	return KamelBindWithID(platform.DefaultPlatformName, namespace, args...)
}

func KamelBindWithID(operatorID string, namespace string, args ...string) *cobra.Command {
	return KamelBindWithContext(TestContext, operatorID, namespace, args...)
}

func KamelBindWithContext(ctx context.Context, operatorID string, namespace string, args ...string) *cobra.Command {
	return KamelCommandWithContext(ctx, "bind", operatorID, namespace, args...)
}

func KamelCommandWithContext(ctx context.Context, command string, operatorID string, namespace string, args ...string) *cobra.Command {
	var cmdArgs []string

	globalTest := os.Getenv("CAMEL_K_FORCE_GLOBAL_TEST") == "true"
	if globalTest {
		fmt.Printf("Running as globally managed resource\n")

		if err := verifyGlobalOperator(); err != nil {
			failTest(err)
		}

		// Have a global operator reconciling the integration
		cmdArgs = []string{command, "-n", namespace}
	} else {
		// NOT global so proceed with local namespaced operator reconciling the integration
		cmdArgs = []string{command, "-n", namespace, "--operator-id", operatorID}
	}

	cmdArgs = append(cmdArgs, args...)
	return KamelWithContext(ctx, cmdArgs...)
}

func verifyGlobalOperator() error {
	opns := os.Getenv("CAMEL_K_GLOBAL_OPERATOR_NS")
	if opns == "" {
		return errors.New("No operator namespace defined in CAMEL_K_GLOBAL_OPERATOR_NS")
	}

	oppod := OperatorPod(opns)()
	if oppod == nil {
		return fmt.Errorf("No operator pod detected in namespace %s. Operator install is a pre-requisite of the test", opns)
	}

	return nil
}

func KamelWithContext(ctx context.Context, args ...string) *cobra.Command {
	var c *cobra.Command
	var err error

	if os.Getenv("CAMEL_K_TEST_LOG_LEVEL") == "debug" {
		fmt.Printf("Executing kamel with command %+q\n", args)
		fmt.Println("Printing stack for KamelWithContext")
		debug.PrintStack()
	}

	kamelArgs := os.Getenv("KAMEL_ARGS")
	kamelDefaultArgs := strings.Fields(kamelArgs)
	args = append(kamelDefaultArgs, args...)

	kamelBin := os.Getenv("KAMEL_BIN")
	if kamelBin != "" {
		if _, e := os.Stat(kamelBin); e != nil && os.IsNotExist(e) {
			failTest(e)
		}
		fmt.Printf("Using external kamel binary on path %s\n", kamelBin)
		c = &cobra.Command{
			DisableFlagParsing: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				externalBin := exec.CommandContext(ctx, kamelBin, args...)
				var stdout io.Reader
				stdout, err = externalBin.StdoutPipe()
				if err != nil {
					failTest(err)
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
		// Use modeline CLI as it's closer to the real usage
		c, args, err = cmd.NewKamelWithModelineCommand(ctx, append([]string{"kamel"}, args...))
	}
	if err != nil {
		failTest(err)
	}
	for _, hook := range KamelHooks {
		args = hook(args)
	}
	c.SetArgs(args)
	return c
}

func Make(rule string, args ...string) *exec.Cmd {
	return MakeWithContext(TestContext, rule, args...)
}

func MakeWithContext(ctx context.Context, rule string, args ...string) *exec.Cmd {
	makeArgs := os.Getenv("MAKE_ARGS")
	defaultArgs := strings.Fields(makeArgs)
	args = append(defaultArgs, args...)

	makeDir := os.Getenv("MAKE_DIR")
	if makeDir == "" {
		makeDir = "../../../../install"
	} else {
		fmt.Printf("Using alternative make directory on path %s\n", makeDir)
	}

	if fi, e := os.Stat(makeDir); e != nil && os.IsNotExist(e) {
		failTest(e)
	} else if !fi.Mode().IsDir() {
		failTest(e)
	}

	args = append([]string{"-C", makeDir, rule}, args...)

	return exec.Command("make", args...)
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
			TailLines: pointer.Int64(100),
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

func StructuredLogs(ns, podName string, options *corev1.PodLogOptions, ignoreParseErrors bool) ([]util.LogEntry, error) {

	stream, err := TestClient().CoreV1().Pods(ns).GetLogs(podName, options).Stream(TestContext)
	if err != nil {
		msg := "Error while reading container logs"
		log.Error(err, msg)
		return nil, fmt.Errorf("%s: %w\n", msg, err)
	}
	defer func() {
		if err := stream.Close(); err != nil {
			log.Error(err, "Error closing the stream")
		}
	}()

	entries := make([]util.LogEntry, 0)
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		entry := util.LogEntry{}
		t := scanner.Text()
		err := json.Unmarshal([]byte(t), &entry)
		if err != nil {
			if ignoreParseErrors {
				fmt.Printf("Warning: Ignoring parse error for logging line: %q\n", t)
				continue
			} else {
				msg := fmt.Sprintf("Unable to parse structured content: %s", t)
				log.Errorf(err, msg)
				return nil, fmt.Errorf("%s %w\n", msg, err)
			}
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		msg := "Error while scanning container logs"
		log.Error(err, msg)
		return nil, fmt.Errorf("%s %w\n", msg, err)
	}

	if len(entries) == 0 {
		msg := "Error fetched zero log entries"
		log.Error(err, msg)
		return nil, fmt.Errorf("%s %w\n", msg, err)
	}

	return entries, nil
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

func IntegrationPodHas(ns string, name string, predicate func(pod *corev1.Pod) bool) func() bool {
	return func() bool {
		pod := IntegrationPod(ns, name)()
		if pod == nil {
			return false
		}
		return predicate(pod)
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
			failTest(err)
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

func IntegrationGeneration(ns string, name string) func() *int64 {
	return func() *int64 {
		it := Integration(ns, name)()
		if it == nil {
			return nil
		}
		return &it.Generation
	}
}

func IntegrationObservedGeneration(ns string, name string) func() *int64 {
	return func() *int64 {
		it := Integration(ns, name)()
		if it == nil {
			return nil
		}
		return &it.Status.ObservedGeneration
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

func IntegrationStatusImage(ns string, name string) func() string {
	return func() string {
		it := Integration(ns, name)()
		if it == nil {
			return ""
		}
		return it.Status.Image
	}
}

func IntegrationAnnotations(ns string, name string) func() map[string]string {
	return func() map[string]string {
		it := Integration(ns, name)()
		if it == nil {
			return map[string]string{}
		}
		return it.Annotations
	}
}

func IntegrationCondition(ns string, name string, conditionType v1.IntegrationConditionType) func() *v1.IntegrationCondition {
	return func() *v1.IntegrationCondition {
		it := Integration(ns, name)()
		if it == nil {
			return nil
		}
		return it.Status.GetCondition(conditionType)
	}
}

func IntegrationConditionReason(c *v1.IntegrationCondition) string {
	if c == nil {
		return ""
	}
	return c.Reason
}

func IntegrationConditionMessage(c *v1.IntegrationCondition) string {
	if c == nil {
		return ""
	}
	return c.Message
}

func IntegrationConditionStatus(ns string, name string, conditionType v1.IntegrationConditionType) func() corev1.ConditionStatus {
	return func() corev1.ConditionStatus {
		c := IntegrationCondition(ns, name, conditionType)()
		if c == nil {
			return "Unknown"
		}
		return c.Status
	}
}

func AssignIntegrationToOperator(ns, name, operator string) error {
	it := Integration(ns, name)()
	if it == nil {
		return fmt.Errorf("cannot assign integration %q to operator: integration not found", name)
	}

	it.SetOperatorID(operator)
	return TestClient().Update(TestContext, it)
}

func Annotations(object metav1.Object) map[string]string {
	return object.GetAnnotations()
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
			failTest(err)
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
			failTest(err)
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
			failTest(err)
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
			failTest(err)
		}
		return &svc
	}
}

func ServiceType(ns string, name string) func() corev1.ServiceType {
	return func() corev1.ServiceType {
		svc := Service(ns, name)()
		if svc == nil {
			return ""
		}
		return svc.Spec.Type
	}
}

//
// Find the service in the given namespace with the given type
//
func ServicesByType(ns string, svcType corev1.ServiceType) func() []corev1.Service {
	return func() []corev1.Service {
		svcs := []corev1.Service{}

		svcList, err := TestClient().CoreV1().Services(ns).List(TestContext, metav1.ListOptions{})
		if err != nil && k8serrors.IsNotFound(err) {
			return svcs
		} else if err != nil {
			failTest(err)
		}

		if len(svcList.Items) == 0 {
			return svcs
		}

		for _, svc := range svcList.Items {
			if svc.Spec.Type == svcType {
				svcs = append(svcs, svc)
			}
		}

		return svcs
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
			failTest(err)
		}
		return &route
	}
}

func RouteStatus(ns string, name string) func() string {
	return func() string {
		route := Route(ns, name)()
		if route == nil || len(route.Status.Ingress) == 0 {
			return ""
		}
		return string(route.Status.Ingress[0].Conditions[0].Status)
	}
}

func IntegrationCronJob(ns string, name string) func() *batchv1.CronJob {
	return func() *batchv1.CronJob {
		lst := batchv1.CronJobList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "CronJob",
				APIVersion: batchv1.SchemeGroupVersion.String(),
			},
		}
		err := TestClient().List(TestContext, &lst,
			ctrl.InNamespace(ns),
			ctrl.MatchingLabels{
				"camel.apache.org/integration": name,
			})
		if err != nil {
			failTest(err)
		}
		if len(lst.Items) == 0 {
			return nil
		}
		return &lst.Items[0]
	}
}

func Integrations(ns string) func() *v1.IntegrationList {
	return func() *v1.IntegrationList {
		lst := v1.NewIntegrationList()
		if err := TestClient().List(TestContext, &lst, ctrl.InNamespace(ns)); err != nil {
			failTest(err)
		}

		return &lst
	}
}

func NumIntegrations(ns string) func() int {
	return func() int {
		lst := Integrations(ns)()
		return len(lst.Items)
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
			failTest(err)
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

func Kit(ns, name string) func() *v1.IntegrationKit {
	return func() *v1.IntegrationKit {
		kit := v1.NewIntegrationKit(ns, name)
		if err := TestClient().Get(TestContext, ctrl.ObjectKeyFromObject(kit), kit); err != nil && !k8serrors.IsNotFound(err) {
			failTest(err)
		} else if err != nil && k8serrors.IsNotFound(err) {
			return nil
		}
		return kit
	}
}

func KitPhase(ns, name string) func() v1.IntegrationKitPhase {
	return func() v1.IntegrationKitPhase {
		kit := Kit(ns, name)()
		if kit == nil {
			return v1.IntegrationKitPhaseNone
		}
		return kit.Status.Phase
	}
}

func UpdateIntegration(ns string, name string, mutate func(it *v1.Integration)) error {
	it := Integration(ns, name)()
	if it == nil {
		return fmt.Errorf("no integration named %s found", name)
	}
	target := it.DeepCopy()
	mutate(target)
	return TestClient().Update(TestContext, target)
}

func PatchIntegration(ns string, name string, mutate func(it *v1.Integration)) error {
	it := Integration(ns, name)()
	if it == nil {
		return fmt.Errorf("no integration named %s found", name)
	}
	target := it.DeepCopy()
	mutate(target)
	p, err := patch.MergePatch(it, target)
	if err != nil {
		return err
	} else if len(p) == 0 {
		return nil
	}
	return TestClient().Patch(TestContext, target, ctrl.RawPatch(types.MergePatchType, p))
}

func ScaleIntegration(ns string, name string, replicas int32) error {
	return PatchIntegration(ns, name, func(it *v1.Integration) {
		it.Spec.Replicas = &replicas
	})
}

func KameletBinding(ns string, name string) func() *v1alpha1.KameletBinding {
	return func() *v1alpha1.KameletBinding {
		klb := v1alpha1.NewKameletBinding(ns, name)
		key := ctrl.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		if err := TestClient().Get(TestContext, key, &klb); err != nil && !k8serrors.IsNotFound(err) {
			failTest(err)
		} else if err != nil && k8serrors.IsNotFound(err) {
			return nil
		}
		return &klb
	}
}

func KameletBindingSpecReplicas(ns string, name string) func() *int32 {
	return func() *int32 {
		klb := KameletBinding(ns, name)()
		if klb == nil {
			return nil
		}
		return klb.Spec.Replicas
	}
}

func KameletBindingStatusReplicas(ns string, name string) func() *int32 {
	return func() *int32 {
		klb := KameletBinding(ns, name)()
		if klb == nil {
			return nil
		}
		return klb.Status.Replicas
	}
}

func KameletBindingCondition(ns string, name string, conditionType v1alpha1.KameletBindingConditionType) func() *v1alpha1.KameletBindingCondition {
	return func() *v1alpha1.KameletBindingCondition {
		kb := KameletBinding(ns, name)()
		if kb == nil {
			return nil
		}
		c := kb.Status.GetCondition(conditionType)
		if c == nil {
			return nil
		}
		return c
	}
}

func KameletBindingConditionReason(c *v1alpha1.KameletBindingCondition) string {
	if c == nil {
		return ""
	}
	return c.Reason
}

func KameletBindingConditionMessage(c *v1alpha1.KameletBindingCondition) string {
	if c == nil {
		return ""
	}
	return c.Message
}

func KameletBindingConditionStatus(ns string, name string, conditionType v1alpha1.KameletBindingConditionType) func() corev1.ConditionStatus {
	return func() corev1.ConditionStatus {
		klb := KameletBinding(ns, name)()
		if klb == nil {
			return "KameletBindingMissing"
		}
		c := klb.Status.GetCondition(conditionType)
		if c == nil {
			return "ConditionMissing"
		}
		return c.Status
	}
}

func UpdateKameletBinding(ns string, name string, upd func(it *v1alpha1.KameletBinding)) error {
	klb := KameletBinding(ns, name)()
	if klb == nil {
		return fmt.Errorf("no kamelet binding named %s found", name)
	}
	target := klb.DeepCopy()
	upd(target)
	// For some reasons, full patch fails on some clusters
	p, err := patch.MergePatch(klb, target)
	if err != nil {
		return err
	} else if len(p) == 0 {
		return nil
	}
	return TestClient().Patch(TestContext, target, ctrl.RawPatch(types.MergePatchType, p))
}

func ScaleKameletBinding(ns string, name string, replicas int32) error {
	return UpdateKameletBinding(ns, name, func(klb *v1alpha1.KameletBinding) {
		klb.Spec.Replicas = &replicas
	})
}

func AssignKameletBindingToOperator(ns, name, operator string) error {
	klb := KameletBinding(ns, name)()
	if klb == nil {
		return fmt.Errorf("cannot assign kamelet binding %q to operator: kamelet binding not found", name)
	}

	klb.SetOperatorID(operator)
	return TestClient().Update(TestContext, klb)
}

type KitFilter interface {
	Match(*v1.IntegrationKit) bool
}

func KitWithPhase(phase v1.IntegrationKitPhase) KitFilter {
	return &kitFilter{
		filter: func(kit *v1.IntegrationKit) bool {
			return kit.Status.Phase == phase
		},
	}
}

func KitWithVersion(version string) KitFilter {
	return &kitFilter{
		filter: func(kit *v1.IntegrationKit) bool {
			return kit.Status.Version == version
		},
	}
}

func KitWithLabels(kitLabels map[string]string) ctrl.ListOption {
	return ctrl.MatchingLabelsSelector{
		Selector: labels.Set(kitLabels).AsSelector(),
	}
}

type kitFilter struct {
	filter func(*v1.IntegrationKit) bool
}

func (f *kitFilter) Match(kit *v1.IntegrationKit) bool {
	return f.filter(kit)
}

func Kits(ns string, options ...interface{}) func() []v1.IntegrationKit {
	filters := make([]KitFilter, 0)
	listOptions := []ctrl.ListOption{ctrl.InNamespace(ns)}
	for _, option := range options {
		switch o := option.(type) {
		case KitFilter:
			filters = append(filters, o)
		case ctrl.ListOption:
			listOptions = append(listOptions, o)
		default:
			failTest(fmt.Errorf("unsupported kits option %q", o))
		}
	}

	return func() []v1.IntegrationKit {
		list := v1.NewIntegrationKitList()
		if err := TestClient().List(TestContext, &list, listOptions...); err != nil {
			failTest(err)
		}

		var kits []v1.IntegrationKit
	kits:
		for _, kit := range list.Items {
			for _, filter := range filters {
				if !filter.Match(&kit) {
					continue kits
				}
			}
			kits = append(kits, kit)
		}

		return kits
	}
}

func DeleteKits(ns string) error {
	kits := Kits(ns)()
	if len(kits) == 0 {
		return nil
	}
	for _, kit := range kits {
		if err := TestClient().Delete(TestContext, &kit); err != nil {
			return err
		}
	}

	return nil
}

func DeleteIntegrations(ns string) func() (int, error) {
	return func() (int, error) {
		integrations := Integrations(ns)()
		if len(integrations.Items) == 0 {
			return 0, nil
		}

		if err := Kamel("delete", "--all", "-n", ns).Execute(); err != nil {
			return 0, err
		}

		return NumIntegrations(ns)(), nil
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

func OperatorEnvVarValue(ns string, key string) func() string {
	return func() string {
		pod := OperatorPod(ns)()
		if pod == nil || len(pod.Spec.Containers) == 0 {
			return ""
		}
		envvars := pod.Spec.Containers[0].Env
		for _, v := range envvars {
			if v.Name == key {
				return v.Value
			}
		}

		return ""
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

func AutogeneratedConfigmapsCount(ns string) func() int {
	return func() int {
		lst := corev1.ConfigMapList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
		}
		err := TestClient().List(TestContext, &lst,
			ctrl.InNamespace(ns),
			ctrl.MatchingLabels{
				kubernetes.ConfigMapAutogenLabel: "true",
			})
		if err != nil {
			failTest(err)
		}
		return len(lst.Items)
	}
}

func CreatePlainTextConfigmap(ns string, name string, data map[string]string) error {
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

func UpdatePlainTextConfigmap(ns string, name string, data map[string]string) error {
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
	return TestClient().Update(TestContext, &cm)
}

func CreateBinaryConfigmap(ns string, name string, data map[string][]byte) error {
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

func CreatePlainTextSecret(ns string, name string, data map[string]string) error {
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

//
// Finds a secret in the given namespace by name or prefix of name
//
func SecretByName(ns string, prefix string) func() *corev1.Secret {
	return func() *corev1.Secret {
		secretList, err := TestClient().CoreV1().Secrets(ns).List(TestContext, metav1.ListOptions{})
		if err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			failTest(err)
		}

		if len(secretList.Items) == 0 {
			return nil
		}

		for _, secret := range secretList.Items {
			if strings.HasPrefix(secret.Name, prefix) {
				return &secret
			}
		}

		return nil
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

func Build(ns, name string) func() *v1.Build {
	return func() *v1.Build {
		build := v1.NewBuild(ns, name)
		if err := TestClient().Get(TestContext, ctrl.ObjectKeyFromObject(build), build); err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			log.Error(err, "Error while retrieving build "+name)
			return nil
		}
		return build
	}
}

func BuildPhase(ns, name string) func() v1.BuildPhase {
	return func() v1.BuildPhase {
		build := Build(ns, name)()
		if build != nil {
			return build.Status.Phase
		}
		return v1.BuildPhaseNone
	}
}

func HasPlatform(ns string) func() bool {
	return func() bool {
		lst := v1.NewIntegrationPlatformList()
		if err := TestClient().List(TestContext, &lst, ctrl.InNamespace(ns)); err != nil {
			return false
		}
		return len(lst.Items) > 0
	}
}

func Platform(ns string) func() *v1.IntegrationPlatform {
	return func() *v1.IntegrationPlatform {
		lst := v1.NewIntegrationPlatformList()
		if err := TestClient().List(TestContext, &lst, ctrl.InNamespace(ns)); err != nil {
			failTest(err)
		}
		if len(lst.Items) == 0 {
			return nil
		}
		if len(lst.Items) > 1 {
			var pl *v1.IntegrationPlatform
			for _, p := range lst.Items {
				p := p
				if platform.IsSecondary(&p) {
					continue
				}
				if pl != nil {
					failTest(fmt.Errorf("multiple primary integration platforms found in namespace %q", ns))
				}
				pl = &p
			}
			if pl == nil {
				failTest(fmt.Errorf("multiple integration platforms found in namespace %q but no one is primary", ns))
			}
			return pl
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
		if err := TestClient().Delete(TestContext, pl); err != nil {
			log.Error(err, "Got error while deleting the platform")
		}
		return false
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

func PlatformCondition(
	ns string,
	conditionType v1.IntegrationPlatformConditionType,
) func() *v1.IntegrationPlatformCondition {
	return func() *v1.IntegrationPlatformCondition {
		p := Platform(ns)()
		if p == nil {
			return nil
		}
		return p.Status.GetCondition(conditionType)
	}
}

func PlatformConditionStatus(
	ns string,
	conditionType v1.IntegrationPlatformConditionType,
) func() corev1.ConditionStatus {
	return func() corev1.ConditionStatus {
		c := PlatformCondition(ns, conditionType)()
		if c == nil {
			return "Unknown"
		}
		return c.Status
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

func AssignPlatformToOperator(ns, operator string) error {
	pl := Platform(ns)()
	if pl == nil {
		return errors.New("cannot assign platform to operator: no platform found")
	}

	pl.SetOperatorID(operator)
	return TestClient().Update(TestContext, pl)
}

func ConfigureSecondaryPlatformWith(ns string, customizer func(pl *v1.IntegrationPlatform)) error {
	pl := Platform(ns)()
	if pl == nil {
		return errors.New("cannot find primary platform")
	}

	v1.SetAnnotation(&pl.ObjectMeta, v1.SecondaryPlatformAnnotation, "true")
	pl.ObjectMeta.ResourceVersion = ""
	pl.Name = ""
	pl.Status = v1.IntegrationPlatformStatus{}
	customizer(pl)
	return TestClient().Create(TestContext, pl)
}

func CRDs() func() []metav1.APIResource {
	return func() []metav1.APIResource {

		kinds := []string{
			reflect.TypeOf(v1.Build{}).Name(),
			reflect.TypeOf(v1.Integration{}).Name(),
			reflect.TypeOf(v1.IntegrationKit{}).Name(),
			reflect.TypeOf(v1.IntegrationPlatform{}).Name(),
			reflect.TypeOf(v1alpha1.Kamelet{}).Name(),
			reflect.TypeOf(v1alpha1.KameletBinding{}).Name(),
		}

		versions := []string{"v1", "v1alpha1"}
		present := []metav1.APIResource{}

		for _, version := range versions {
			lst, err := TestClient().Discovery().ServerResourcesForGroupVersion("camel.apache.org/" + version)
			if err != nil && k8serrors.IsNotFound(err) {
				return nil
			} else if err != nil {
				failTest(err)
			}

			for _, res := range lst.APIResources {
				if strings.Contains(res.Name, "/") {
					continue // ignore sub types like status
				}

				for _, k := range kinds {
					if k == res.Kind {
						present = append(present, res)
					}
				}
			}
		}

		return present
	}
}

func ConsoleCLIDownload(name string) func() *consoleV1.ConsoleCLIDownload {
	return func() *consoleV1.ConsoleCLIDownload {
		cliDownload := consoleV1.ConsoleCLIDownload{}
		if err := TestClient().Get(TestContext, ctrl.ObjectKey{Name: name}, &cliDownload); err != nil && !k8serrors.IsNotFound(err) {
			failTest(err)
		} else if err != nil && k8serrors.IsNotFound(err) {
			return nil
		}
		return &cliDownload
	}
}

func OperatorPod(ns string) func() *corev1.Pod {
	namespace := ns

	globalTest := os.Getenv("CAMEL_K_FORCE_GLOBAL_TEST") == "true"
	opns := os.Getenv("CAMEL_K_GLOBAL_OPERATOR_NS")
	if globalTest && len(opns) > 0 {
		// Use the global operator pod instead of given namespace
		namespace = opns
	}

	return func() *corev1.Pod {
		lst := corev1.PodList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: v1.SchemeGroupVersion.String(),
			},
		}
		err := TestClient().List(TestContext, &lst,
			ctrl.InNamespace(namespace),
			ctrl.MatchingLabels{
				"camel.apache.org/component": "operator",
			})
		if err != nil {
			failTest(err)
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
		if err := TestClient().Delete(TestContext, pod, ctrl.GracePeriodSeconds(timeSeconds)); err != nil {
			log.Error(err, "cannot forcefully kill the pod")
		}
	}
}

func ScaleOperator(ns string, replicas int32) error {
	operator, err := TestClient().AppsV1().Deployments(ns).Get(TestContext, "camel-k-operator", metav1.GetOptions{})
	if err != nil {
		return err
	}
	operator.Spec.Replicas = &replicas
	_, err = TestClient().AppsV1().Deployments(ns).Update(TestContext, operator, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	if replicas == 0 {
		// speedup scale down by killing the pod
		OperatorTryPodForceKill(ns, 10)
	}
	return nil
}

func ClusterRole() func() []rbacv1.ClusterRole {
	return func() []rbacv1.ClusterRole {
		lst := rbacv1.ClusterRoleList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ClusterRole",
				APIVersion: rbacv1.SchemeGroupVersion.String(),
			},
		}
		err := TestClient().List(TestContext, &lst,
			ctrl.MatchingLabels{
				"app": "camel-k",
			})
		if err != nil {
			failTest(err)
		}
		if len(lst.Items) == 0 {
			return nil
		}
		return lst.Items
	}
}

func Role(ns string) func() []rbacv1.Role {
	return func() []rbacv1.Role {
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
			failTest(err)
		}
		if len(lst.Items) == 0 {
			return nil
		}
		return lst.Items
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
			failTest(err)
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
			failTest(err)
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
			failTest(err)
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
			failTest(err)
		} else if err != nil && k8serrors.IsNotFound(err) {
			return nil
		}
		return &it
	}
}

func KameletLabels(kamelet *v1alpha1.Kamelet) map[string]string {
	if kamelet == nil {
		return map[string]string{}
	}
	return kamelet.GetLabels()
}

func ClusterDomainName() (string, error) {
	dns := configv1.DNS{}
	key := ctrl.ObjectKey{
		Name: "cluster",
	}
	err := TestClient().Get(TestContext, key, &dns)
	if err != nil {
		return "", err
	}
	return dns.Spec.BaseDomain, nil
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
		failTest(err)
	}
	customizer := install.IdentityResourceCustomizer
	if oc {
		// Remove Ingress permissions as it's not needed on OpenShift
		// This should ideally be removed from the common RBAC manifest.
		customizer = install.RemoveIngressRoleCustomizer
	}
	err = install.Resource(TestContext, TestClient(), ns, true, customizer, "/rbac/operator-role.yaml")
	if err != nil {
		return err
	}
	if oc {
		return install.Resource(TestContext, TestClient(), ns, true, install.IdentityResourceCustomizer, "/rbac/openshift/operator-role-openshift.yaml")
	}
	return nil
}

func CreateOperatorRoleBinding(ns string) error {
	oc, err := openshift.IsOpenShift(TestClient())
	if err != nil {
		failTest(err)
	}
	err = install.Resource(TestContext, TestClient(), ns, true, install.IdentityResourceCustomizer, "/rbac/operator-role-binding.yaml")
	if err != nil {
		return err
	}
	if oc {
		return install.Resource(TestContext, TestClient(), ns, true, install.IdentityResourceCustomizer, "/rbac/openshift/operator-role-binding-openshift.yaml")
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

func CreateKnativeBroker(ns string, name string) func() error {
	return func() error {
		broker := eventing.Broker{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Broker",
				APIVersion: eventing.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      name,
			},
		}
		return TestClient().Create(TestContext, &broker)
	}
}

/*
	Kamelets
*/

func CreateKamelet(ns string, name string, template map[string]interface{}, properties map[string]v1alpha1.JSONSchemaProp, labels map[string]string) func() error {
	return func() error {
		kamelet := v1alpha1.Kamelet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      name,
				Labels:    labels,
			},
			Spec: v1alpha1.KameletSpec{
				Definition: &v1alpha1.JSONSchemaProps{
					Properties: properties,
				},
				Template: asTemplate(template),
			},
		}
		return TestClient().Create(TestContext, &kamelet)
	}
}

func CreateTimerKamelet(ns string, name string) func() error {
	props := map[string]v1alpha1.JSONSchemaProp{
		"message": {
			Type: "string",
		},
	}

	flow := map[string]interface{}{
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
	}

	return CreateKamelet(ns, name, flow, props, nil)
}

func BindKameletTo(ns, name string, annotations map[string]string, from, to corev1.ObjectReference,
	sourceProperties, sinkProperties map[string]string) func() error {
	return BindKameletToWithErrorHandler(ns, name, annotations, from, to, sourceProperties, sinkProperties, nil)
}

func BindKameletToWithErrorHandler(ns, name string, annotations map[string]string, from, to corev1.ObjectReference,
	sourceProperties, sinkProperties map[string]string, errorHandler map[string]interface{}) func() error {
	return func() error {
		kb := v1alpha1.NewKameletBinding(ns, name)
		kb.Annotations = annotations
		kb.Spec = v1alpha1.KameletBindingSpec{
			Source: v1alpha1.Endpoint{
				Ref:        &from,
				Properties: asEndpointProperties(sourceProperties),
			},
			Sink: v1alpha1.Endpoint{
				Ref:        &to,
				Properties: asEndpointProperties(sinkProperties),
			},
		}
		if errorHandler != nil {
			kb.Spec.ErrorHandler = asErrorHandlerSpec(errorHandler)
		}
		_, err := kubernetes.ReplaceResource(TestContext, TestClient(), &kb)
		return err
	}
}

func asTemplate(source map[string]interface{}) *v1alpha1.Template {
	bytes, err := json.Marshal(source)
	if err != nil {
		failTest(err)
	}
	return &v1alpha1.Template{
		RawMessage: bytes,
	}
}

func asErrorHandlerSpec(source map[string]interface{}) *v1alpha1.ErrorHandlerSpec {
	bytes, err := json.Marshal(source)
	if err != nil {
		failTest(err)
	}
	return &v1alpha1.ErrorHandlerSpec{
		RawMessage: bytes,
	}
}

func asEndpointProperties(props map[string]string) *v1alpha1.EndpointProperties {
	bytes, err := json.Marshal(props)
	if err != nil {
		failTest(err)
	}
	return &v1alpha1.EndpointProperties{
		RawMessage: bytes,
	}
}

// nolint: staticcheck
func AsTraitConfiguration(props map[string]string) *traitv1.Configuration {
	bytes, err := json.Marshal(props)
	if err != nil {
		failTest(err)
	}
	return &traitv1.Configuration{
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
	setTestLocus(t)
	ns := newTestNamespace(false)
	defer deleteTestNamespace(t, ns)
	defer userCleanup(t)

	invokeUserTestCode(t, ns.GetName(), doRun)
}

func WithGlobalOperatorNamespace(t *testing.T, test func(string)) {
	setTestLocus(t)
	ocp, err := openshift.IsOpenShift(TestClient())
	assert.Nil(t, err)
	if ocp {
		// global operators are always installed in the openshift-operators namespace
		invokeUserTestCode(t, "openshift-operators", test)
	} else {
		// create new namespace for the global operator
		WithNewTestNamespace(t, test)
	}
}

func WithNewTestNamespaceWithKnativeBroker(t *testing.T, doRun func(string)) {
	setTestLocus(t)
	ns := newTestNamespace(true)
	defer deleteTestNamespace(t, ns)
	defer deleteKnativeBroker(ns)
	defer userCleanup(t)

	invokeUserTestCode(t, ns.GetName(), doRun)
}

func userCleanup(t *testing.T) {
	userCmd := os.Getenv("KAMEL_TEST_CLEANUP")
	if userCmd != "" {
		fmt.Printf("Executing user cleanup command: %s\n", userCmd)
		cmdSplit := strings.Split(userCmd, " ")
		command := exec.Command(cmdSplit[0], cmdSplit[1:]...)
		command.Stderr = os.Stderr
		command.Stdout = os.Stdout
		if err := command.Run(); err != nil {
			t.Logf("An error occurred during user cleanup command execution: %v\n", err)
		} else {
			t.Logf("User cleanup command completed successfully\n")
		}
	}
}

func invokeUserTestCode(t *testing.T, ns string, doRun func(string)) {
	globalTest := os.Getenv("CAMEL_K_FORCE_GLOBAL_TEST") == "true"

	defer func(isGlobal bool) {
		if t.Failed() {
			if err := util.Dump(TestContext, TestClient(), ns, t); err != nil {
				t.Logf("Error while dumping namespace %s: %v\n", ns, err)
			}
		}

		// Try to clean up namespace
		if !isGlobal && HasPlatform(ns)() {
			t.Logf("Clean up test namespace: %s", ns)

			if err := Kamel("uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles").Execute(); err != nil {
				t.Logf("Error while cleaning up namespace %s: %v\n", ns, err)
			}

			t.Logf("Successfully cleaned up test namespace: %s", ns)
		}
	}(globalTest)

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
	nsKey := ctrl.ObjectKeyFromObject(&nsRef)
	if err := TestClient().Get(TestContext, nsKey, &nsRef); err != nil {
		failTest(err)
	}

	nsRef.SetLabels(make(map[string]string, 0))
	if err := TestClient().Update(TestContext, &nsRef); err != nil {
		failTest(err)
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
		failTest(err)
	}
}

func deleteTestNamespace(t *testing.T, ns ctrl.Object) {
	value, saveNS := os.LookupEnv("CAMEL_K_TEST_SAVE_FAILED_TEST_NAMESPACE")
	if t.Failed() && saveNS && value == "true" {
		t.Logf("Warning: retaining failed test project %q", ns.GetName())
		return
	}

	var oc bool
	var err error
	if oc, err = openshift.IsOpenShift(TestClient()); err != nil {
		failTest(err)
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

func testNamespaceExists(ns string) (bool, error) {
	_, err := TestClient().CoreV1().Namespaces().Get(TestContext, ns, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return false, nil
		} else {
			return false, err
		}
	}

	return true, nil
}

func newTestNamespace(injectKnativeBroker bool) ctrl.Object {
	brokerLabel := "eventing.knative.dev/injection"
	name := os.Getenv("CAMEL_K_TEST_NS")
	if name == "" {
		name = "test-" + uuid.New().String()
	}
	c := TestClient()

	if exists, err := testNamespaceExists(name); err != nil {
		failTest(err)
	} else if exists {
		fmt.Println("Warning: namespace ", name, " already exists so using different namespace name")
		name = fmt.Sprintf("%s-%d", name, time.Now().Second())
	}

	if oc, err := openshift.IsOpenShift(TestClient()); err != nil {
		failTest(err)
	} else if oc {
		rest, err := apiutil.RESTClientForGVK(
			schema.GroupVersionKind{Group: projectv1.GroupName, Version: projectv1.GroupVersion.Version}, false,
			c.GetConfig(), serializer.NewCodecFactory(c.GetScheme()))
		if err != nil {
			failTest(err)
		}
		request := &projectv1.ProjectRequest{
			TypeMeta: metav1.TypeMeta{
				APIVersion: projectv1.GroupVersion.String(),
				Kind:       "ProjectRequest",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}
		project := &projectv1.Project{
			TypeMeta: metav1.TypeMeta{
				APIVersion: projectv1.GroupVersion.String(),
				Kind:       "Project",
			},
		}
		err = rest.Post().
			Resource("projectrequests").
			Body(request).
			Do(TestContext).
			Into(project)
		if err != nil {
			failTest(err)
		}
		// workaround https://github.com/openshift/origin/issues/3819
		if injectKnativeBroker {
			// use Kubernetes API - https://access.redhat.com/solutions/2677921
			if namespace, err := TestClient().CoreV1().Namespaces().Get(TestContext, name, metav1.GetOptions{}); err != nil {
				failTest(err)
			} else {
				if _, ok := namespace.GetLabels()[brokerLabel]; !ok {
					namespace.SetLabels(map[string]string{
						brokerLabel: "enabled",
					})
					if err = TestClient().Update(TestContext, namespace); err != nil {
						failTest(errors.New("Unable to label project with knative-eventing-injection. This operation needs update permission on the project."))
					}
				}
			}
		}
		return project
	} else {
		namespace := &corev1.Namespace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}
		if injectKnativeBroker {
			namespace.SetLabels(map[string]string{
				brokerLabel: "enabled",
			})
		}
		if err := TestClient().Create(TestContext, namespace); err != nil {
			failTest(err)
		}
		return namespace
	}

	return nil
}

func GetOutputString(command *cobra.Command) string {
	var buf bytes.Buffer

	command.SetOut(&buf)
	command.Execute()

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
