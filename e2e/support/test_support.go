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
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	consoleV1 "github.com/openshift/api/console/v1"
	"github.com/stretchr/testify/require"

	"github.com/google/uuid"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/format"

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
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	eventing "knative.dev/eventing/pkg/apis/eventing/v1"
	messaging "knative.dev/eventing/pkg/apis/messaging/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/apache/camel-k/v2/e2e/support/util"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	v1alpha1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/cmd"
	"github.com/apache/camel-k/v2/pkg/install"
	"github.com/apache/camel-k/v2/pkg/platform"
	v2util "github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/log"
	"github.com/apache/camel-k/v2/pkg/util/openshift"
	"github.com/apache/camel-k/v2/pkg/util/patch"
	configv1 "github.com/openshift/api/config/v1"
	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	// let's enable addons in all tests
	_ "github.com/apache/camel-k/v2/addons"
)

const kubeConfigEnvVar = "KUBECONFIG"
const ciPID = "/tmp/ci-k8s-pid"

// v1.Build,          v1.Integration
// v1.IntegrationKit, v1.IntegrationPlatform, v1.IntegrationProfile
// v1.Kamelet,  v1.Pipe,
// v1alpha1.Kamelet, v1alpha1.KameletBinding
const ExpectedCRDs = 9

// camel-k-operator,
// camel-k-operator-events,
// camel-k-operator-leases,
// camel-k-operator-podmonitors,
// camel-k-operator-strimzi,
// camel-k-operator-keda,
// camel-k-operator-knative
const ExpectedKubePromoteRoles = 7

// camel-k-edit
// camel-k-operator-custom-resource-definitions
// camel-k-operator-bind-addressable-resolver
// camel-k-operator-local-registry
const ExpectedKubeClusterRoles = 4

// camel-k-operator-openshift
const ExpectedOSPromoteRoles = 1

// camel-k-operator-console-openshift
const ExpectedOSClusterRoles = 1

var TestDefaultNamespace = "default"

var TestTimeoutShort = 1 * time.Minute
var TestTimeoutMedium = 3 * time.Minute
var TestTimeoutLong = 5 * time.Minute

// TestTimeoutVeryLong should be used only for testing native builds.
var TestTimeoutVeryLong = 15 * time.Minute

var NoOlmOperatorImage string

var testContext = context.TODO()
var testClient client.Client

func init() {
	// This line prevents controller-runtime from complaining about log.SetLogger never being called
	logf.SetLogger(zap.New(zap.UseDevMode(true)))
}

// Only panic the test if absolutely necessary and there is
// no test locus. In most cases, the test should fail gracefully
// using the test locus to error out and fail now.
func failTest(t *testing.T, err error) {
	if t != nil {
		t.Helper()
		t.Error(err)
		t.FailNow()
	} else {
		panic(err)
	}
}

func TestContext() context.Context {
	return testContext
}

func TestClient(t *testing.T) client.Client {

	if testClient != nil {
		return testClient
	}

	var err error
	testClient, err = NewTestClient()
	if err != nil {
		failTest(t, err)
	}
	return testClient
}

func RefreshClient(t *testing.T) client.Client {

	var err error
	testClient, err = NewTestClient()
	if err != nil {
		failTest(t, err)
	}
	testContext = context.TODO()
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

	if value, ok := os.LookupEnv("CAMEL_K_TEST_TIMEOUT_VERY_LONG"); ok {
		if duration, err = time.ParseDuration(value); err == nil {
			TestTimeoutVeryLong = duration
		} else {
			fmt.Printf("Can't parse CAMEL_K_TEST_TIMEOUT_VERY_LONG. Using default value: %s", TestTimeoutVeryLong)
		}
	}

	if imageNoOlm, ok := os.LookupEnv("CAMEL_K_TEST_NO_OLM_OPERATOR_IMAGE"); ok {
		if imageNoOlm != "" {
			NoOlmOperatorImage = imageNoOlm
		} else {
			fmt.Printf("Can't parse CAMEL_K_TEST_NO_OLM_OPERATOR_IMAGE. Using default value from kamel")
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

func Kamel(t *testing.T, ctx context.Context, args ...string) *cobra.Command {
	return KamelWithContext(t, ctx, args...)
}

func KamelRun(t *testing.T, ctx context.Context, namespace string, args ...string) *cobra.Command {
	return KamelRunWithID(t, ctx, platform.DefaultPlatformName, namespace, args...)
}

func KamelRunWithID(t *testing.T, ctx context.Context, operatorID string, namespace string, args ...string) *cobra.Command {
	return KamelRunWithContext(t, ctx, operatorID, namespace, args...)
}

func KamelRunWithContext(t *testing.T, ctx context.Context, operatorID string, namespace string, args ...string) *cobra.Command {
	return KamelCommandWithContext(t, ctx, "run", operatorID, namespace, args...)
}

func KamelBind(t *testing.T, ctx context.Context, namespace string, args ...string) *cobra.Command {
	return KamelBindWithID(t, ctx, platform.DefaultPlatformName, namespace, args...)
}

func KamelBindWithID(t *testing.T, ctx context.Context, operatorID string, namespace string, args ...string) *cobra.Command {
	return KamelBindWithContext(t, ctx, operatorID, namespace, args...)
}

func KamelBindWithContext(t *testing.T, ctx context.Context, operatorID string, namespace string, args ...string) *cobra.Command {
	return KamelCommandWithContext(t, ctx, "bind", operatorID, namespace, args...)
}

func KamelCommandWithContext(t *testing.T, ctx context.Context, command string, operatorID string, namespace string, args ...string) *cobra.Command {
	var cmdArgs []string

	cmdArgs = []string{command, "-n", namespace, "--operator-id", operatorID}

	cmdArgs = append(cmdArgs, args...)
	return KamelWithContext(t, ctx, cmdArgs...)
}

func KamelWithContext(t *testing.T, ctx context.Context, args ...string) *cobra.Command {
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
			failTest(t, e)
		}
		fmt.Printf("Using external kamel binary on path %s\n", kamelBin)
		c = &cobra.Command{
			DisableFlagParsing: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				externalBin := exec.CommandContext(ctx, kamelBin, args...)
				var stdout, stderr io.Reader
				stdout, err = externalBin.StdoutPipe()
				if err != nil {
					failTest(t, err)
				}
				stderr, err = externalBin.StderrPipe()
				if err != nil {
					failTest(t, err)
				}
				err := externalBin.Start()
				if err != nil {
					return err
				}
				_, err = io.Copy(c.OutOrStdout(), stdout)
				if err != nil {
					return err
				}
				_, err = io.Copy(c.ErrOrStderr(), stderr)
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
		failTest(t, err)
	}
	for _, hook := range KamelHooks {
		args = hook(args)
	}
	c.SetArgs(args)
	return c
}

func Make(t *testing.T, rule string, args ...string) *exec.Cmd {
	return MakeWithContext(t, rule, args...)
}

func MakeWithContext(t *testing.T, rule string, args ...string) *exec.Cmd {
	makeArgs := os.Getenv("CAMEL_K_TEST_MAKE_ARGS")
	defaultArgs := strings.Fields(makeArgs)
	args = append(defaultArgs, args...)

	defaultDir := "."
	makeDir := os.Getenv("CAMEL_K_TEST_MAKE_DIR")
	if makeDir == "" {
		makeDir = defaultDir
	} else if makeDir != defaultDir {
		fmt.Printf("Using alternative make directory on path: %s\n", makeDir)
	}

	if fi, e := os.Stat(makeDir); e != nil && os.IsNotExist(e) {
		failTest(t, e)
	} else if !fi.Mode().IsDir() {
		failTest(t, e)
	}

	args = append([]string{"-C", makeDir, rule}, args...)
	return exec.Command("make", args...)
}

func Kubectl(args ...string) *exec.Cmd {
	return KubectlWithContext(args...)
}

func KubectlWithContext(args ...string) *exec.Cmd {
	return exec.Command("kubectl", args...)
}

// =============================================================================
// Curried utility functions for testing
// =============================================================================

func IntegrationLogs(t *testing.T, ctx context.Context, ns, name string) func() string {
	return func() string {
		pod := IntegrationPod(t, ctx, ns, name)()
		if pod == nil {
			return ""
		}

		options := corev1.PodLogOptions{
			TailLines: pointer.Int64(100),
		}

		for _, container := range pod.Status.ContainerStatuses {
			if !container.Ready || container.State.Waiting != nil {
				// avoid logs watch fail due to container creating state
				return ""
			}
		}

		if len(pod.Spec.Containers) > 1 {
			options.Container = pod.Spec.Containers[0].Name
		}

		return Logs(t, ctx, ns, pod.Name, options)()
	}
}

// TailedLogs Retrieve the Logs from the Pod defined by its name in the given namespace ns. The number of lines numLines from the end of the logs to show.
func TailedLogs(t *testing.T, ctx context.Context, ns, name string, numLines int64) func() string {
	return func() string {
		options := corev1.PodLogOptions{
			TailLines: pointer.Int64(numLines),
		}

		return Logs(t, ctx, ns, name, options)()
	}
}

func Logs(t *testing.T, ctx context.Context, ns, podName string, options corev1.PodLogOptions) func() string {
	return func() string {
		byteReader, err := TestClient(t).CoreV1().Pods(ns).GetLogs(podName, &options).Stream(ctx)
		if err != nil {
			log.Error(err, "Error while reading container logs")
			return ""
		}
		defer func() {
			if err := byteReader.Close(); err != nil {
				log.Error(err, "Error closing the stream")
			}
		}()

		logBytes, err := io.ReadAll(byteReader)
		if err != nil {
			log.Error(err, "Error while reading container logs")
			return ""
		}
		return string(logBytes)
	}
}

func StructuredLogs(t *testing.T, ctx context.Context, ns, podName string, options *corev1.PodLogOptions, ignoreParseErrors bool) ([]util.LogEntry, error) {

	stream, err := TestClient(t).CoreV1().Pods(ns).GetLogs(podName, options).Stream(ctx)
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

func IntegrationPodPhase(t *testing.T, ctx context.Context, ns string, name string) func() corev1.PodPhase {
	return func() corev1.PodPhase {
		pod := IntegrationPod(t, ctx, ns, name)()
		if pod == nil {
			return ""
		}
		return pod.Status.Phase
	}
}

func IntegrationPodImage(t *testing.T, ctx context.Context, ns string, name string) func() string {
	return func() string {
		pod := IntegrationPod(t, ctx, ns, name)()
		if pod == nil || len(pod.Spec.Containers) == 0 {
			return ""
		}
		return pod.Spec.Containers[0].Image
	}
}

func IntegrationPod(t *testing.T, ctx context.Context, ns string, name string) func() *corev1.Pod {
	return func() *corev1.Pod {
		pods := IntegrationPods(t, ctx, ns, name)()
		if len(pods) == 0 {
			return nil
		}
		return &pods[0]
	}
}

func IntegrationPodHas(t *testing.T, ctx context.Context, ns string, name string, predicate func(pod *corev1.Pod) bool) func() bool {
	return func() bool {
		pod := IntegrationPod(t, ctx, ns, name)()
		if pod == nil {
			return false
		}
		return predicate(pod)
	}
}

func IntegrationPods(t *testing.T, ctx context.Context, ns string, name string) func() []corev1.Pod {
	return func() []corev1.Pod {
		lst := corev1.PodList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
		}
		err := TestClient(t).List(ctx, &lst,
			ctrl.InNamespace(ns),
			ctrl.MatchingLabels{
				v1.IntegrationLabel: name,
			})
		if err != nil {
			failTest(t, err)
		}
		return lst.Items
	}
}

func IntegrationPodsNumbers(t *testing.T, ctx context.Context, ns string, name string) func() *int32 {
	return func() *int32 {
		i := int32(len(IntegrationPods(t, ctx, ns, name)()))
		return &i
	}
}

func IntegrationSpecReplicas(t *testing.T, ctx context.Context, ns string, name string) func() *int32 {
	return func() *int32 {
		it := Integration(t, ctx, ns, name)()
		if it == nil {
			return nil
		}
		return it.Spec.Replicas
	}
}

func IntegrationGeneration(t *testing.T, ctx context.Context, ns string, name string) func() *int64 {
	return func() *int64 {
		it := Integration(t, ctx, ns, name)()
		if it == nil {
			return nil
		}
		return &it.Generation
	}
}

func IntegrationObservedGeneration(t *testing.T, ctx context.Context, ns string, name string) func() *int64 {
	return func() *int64 {
		it := Integration(t, ctx, ns, name)()
		if it == nil {
			return nil
		}
		return &it.Status.ObservedGeneration
	}
}

func IntegrationStatusReplicas(t *testing.T, ctx context.Context, ns string, name string) func() *int32 {
	return func() *int32 {
		it := Integration(t, ctx, ns, name)()
		if it == nil {
			return nil
		}
		return it.Status.Replicas
	}
}

func IntegrationStatusImage(t *testing.T, ctx context.Context, ns string, name string) func() string {
	return func() string {
		it := Integration(t, ctx, ns, name)()
		if it == nil {
			return ""
		}
		return it.Status.Image
	}
}

func IntegrationAnnotations(t *testing.T, ctx context.Context, ns string, name string) func() map[string]string {
	return func() map[string]string {
		it := Integration(t, ctx, ns, name)()
		if it == nil {
			return map[string]string{}
		}
		return it.Annotations
	}
}

func IntegrationCondition(t *testing.T, ctx context.Context, ns string, name string, conditionType v1.IntegrationConditionType) func() *v1.IntegrationCondition {
	return func() *v1.IntegrationCondition {
		it := Integration(t, ctx, ns, name)()
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

func HealthCheckResponse(podRegexp string, healthName string) func(*v1.IntegrationCondition) *v1.HealthCheckResponse {
	re := regexp.MustCompile(podRegexp)

	return func(c *v1.IntegrationCondition) *v1.HealthCheckResponse {
		if c == nil {
			return nil
		}

		for p := range c.Pods {
			if re.MatchString(c.Pods[p].Name) {
				continue
			}

			for h := range c.Pods[p].Health {
				if c.Pods[p].Health[h].Name == healthName {
					return &c.Pods[p].Health[h]
				}
			}

		}

		return nil
	}
}

func HealthCheckData(r *v1.HealthCheckResponse) (map[string]interface{}, error) {
	if r == nil {
		return nil, nil
	}
	if r.Data == nil {
		return nil, nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal(r.Data, &data); err != nil {
		return nil, err
	}

	return data, nil
}

func IntegrationConditionStatus(t *testing.T, ctx context.Context, ns string, name string, conditionType v1.IntegrationConditionType) func() corev1.ConditionStatus {
	return func() corev1.ConditionStatus {
		c := IntegrationCondition(t, ctx, ns, name, conditionType)()
		if c == nil {
			return "Unknown"
		}
		return c.Status
	}
}

func AssignIntegrationToOperator(t *testing.T, ctx context.Context, ns, name, operator string) error {
	it := Integration(t, ctx, ns, name)()
	if it == nil {
		return fmt.Errorf("cannot assign integration %q to operator: integration not found", name)
	}

	it.SetOperatorID(operator)
	return TestClient(t).Update(ctx, it)
}

func Annotations(object metav1.Object) map[string]string {
	return object.GetAnnotations()
}

func IntegrationSpec(it *v1.Integration) *v1.IntegrationSpec {
	return &it.Spec
}

func Lease(t *testing.T, ctx context.Context, ns string, name string) func() *coordination.Lease {
	return func() *coordination.Lease {
		lease := coordination.Lease{}
		key := ctrl.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		err := TestClient(t).Get(ctx, key, &lease)
		if err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			failTest(t, err)
		}
		return &lease
	}
}

func Nodes(t *testing.T, ctx context.Context) func() []corev1.Node {
	return func() []corev1.Node {
		nodes := &corev1.NodeList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "NodeList",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
		}
		if err := TestClient(t).List(ctx, nodes); err != nil {
			failTest(t, err)
		}
		return nodes.Items
	}
}

func Node(t *testing.T, ctx context.Context, name string) func() *corev1.Node {
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
		err := TestClient(t).Get(ctx, ctrl.ObjectKeyFromObject(node), node)
		if err != nil {
			failTest(t, err)
		}
		return node
	}
}

func Service(t *testing.T, ctx context.Context, ns string, name string) func() *corev1.Service {
	return func() *corev1.Service {
		svc := corev1.Service{}
		key := ctrl.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		err := TestClient(t).Get(ctx, key, &svc)
		if err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			failTest(t, err)
		}
		return &svc
	}
}

func ServiceType(t *testing.T, ctx context.Context, ns string, name string) func() corev1.ServiceType {
	return func() corev1.ServiceType {
		svc := Service(t, ctx, ns, name)()
		if svc == nil {
			return ""
		}
		return svc.Spec.Type
	}
}

// ServicesByType Find the service in the given namespace with the given type
func ServicesByType(t *testing.T, ctx context.Context, ns string, svcType corev1.ServiceType) func() []corev1.Service {
	return func() []corev1.Service {
		svcs := []corev1.Service{}

		svcList, err := TestClient(t).CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
		if err != nil && k8serrors.IsNotFound(err) {
			return svcs
		} else if err != nil {
			failTest(t, err)
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

func Route(t *testing.T, ctx context.Context, ns string, name string) func() *routev1.Route {
	return func() *routev1.Route {
		route := routev1.Route{}
		key := ctrl.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		err := TestClient(t).Get(ctx, key, &route)
		if err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			failTest(t, err)
		}
		return &route
	}
}

func RouteFull(t *testing.T, ctx context.Context, ns string, name string) func() *routev1.Route {
	return func() *routev1.Route {
		answer := routev1.Route{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Route",
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
		err := TestClient(t).Get(ctx, key, &answer)
		if err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			failTest(t, err)
		}
		return &answer
	}
}

func RouteStatus(t *testing.T, ctx context.Context, ns string, name string) func() string {
	return func() string {
		route := Route(t, ctx, ns, name)()
		if route == nil || len(route.Status.Ingress) == 0 {
			return ""
		}
		return string(route.Status.Ingress[0].Conditions[0].Status)
	}
}

func IntegrationCronJob(t *testing.T, ctx context.Context, ns string, name string) func() *batchv1.CronJob {
	return func() *batchv1.CronJob {
		lst := batchv1.CronJobList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "CronJob",
				APIVersion: batchv1.SchemeGroupVersion.String(),
			},
		}
		if err := TestClient(t).List(ctx, &lst,
			ctrl.InNamespace(ns),
			ctrl.MatchingLabels{
				"camel.apache.org/integration": name,
			}); err != nil {
			failTest(t, err)
		}
		if len(lst.Items) == 0 {
			return nil
		}
		return &lst.Items[0]
	}
}

func Integrations(t *testing.T, ctx context.Context, ns string) func() *v1.IntegrationList {
	return func() *v1.IntegrationList {
		lst := v1.NewIntegrationList()
		if err := TestClient(t).List(ctx, &lst, ctrl.InNamespace(ns)); err != nil {
			failTest(t, err)
		}

		return &lst
	}
}

func NumIntegrations(t *testing.T, ctx context.Context, ns string) func() int {
	return func() int {
		lst := Integrations(t, ctx, ns)()
		return len(lst.Items)
	}
}

func Integration(t *testing.T, ctx context.Context, ns string, name string) func() *v1.Integration {
	return func() *v1.Integration {
		it := v1.NewIntegration(ns, name)
		key := ctrl.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		if err := TestClient(t).Get(ctx, key, &it); err != nil && !k8serrors.IsNotFound(err) {
			failTest(t, err)
		} else if err != nil && k8serrors.IsNotFound(err) {
			return nil
		}
		return &it
	}
}

func IntegrationVersion(t *testing.T, ctx context.Context, ns string, name string) func() string {
	return func() string {
		it := Integration(t, ctx, ns, name)()
		if it == nil {
			return ""
		}
		return it.Status.Version
	}
}

func IntegrationTraitProfile(t *testing.T, ctx context.Context, ns string, name string) func() v1.TraitProfile {
	return func() v1.TraitProfile {
		it := Integration(t, ctx, ns, name)()
		if it == nil {
			return ""
		}
		return it.Status.Profile
	}
}

func IntegrationPhase(t *testing.T, ctx context.Context, ns string, name string) func() v1.IntegrationPhase {
	return func() v1.IntegrationPhase {
		it := Integration(t, ctx, ns, name)()
		if it == nil {
			return ""
		}
		return it.Status.Phase
	}
}

func IntegrationSpecProfile(t *testing.T, ctx context.Context, ns string, name string) func() v1.TraitProfile {
	return func() v1.TraitProfile {
		it := Integration(t, ctx, ns, name)()
		if it == nil {
			return ""
		}
		return it.Spec.Profile
	}
}

func IntegrationStatusCapabilities(t *testing.T, ctx context.Context, ns string, name string) func() []string {
	return func() []string {
		it := Integration(t, ctx, ns, name)()
		if it == nil || &it.Status == nil {
			return nil
		}
		return it.Status.Capabilities
	}
}

func IntegrationSpecSA(t *testing.T, ctx context.Context, ns string, name string) func() string {
	return func() string {
		it := Integration(t, ctx, ns, name)()
		if it == nil {
			return ""
		}
		return it.Spec.ServiceAccountName
	}
}

func IntegrationKit(t *testing.T, ctx context.Context, ns string, name string) func() string {
	return func() string {
		it := Integration(t, ctx, ns, name)()
		if it == nil {
			return ""
		}
		if it.Status.IntegrationKit == nil {
			return ""
		}
		return it.Status.IntegrationKit.Name
	}
}

func IntegrationKitNamespace(t *testing.T, ctx context.Context, integrationNamespace string, name string) func() string {
	return func() string {
		it := Integration(t, ctx, integrationNamespace, name)()
		if it == nil {
			return ""
		}
		if it.Status.IntegrationKit == nil {
			return ""
		}
		return it.Status.IntegrationKit.Namespace
	}
}

func IntegrationKitLayout(t *testing.T, ctx context.Context, ns string, name string) func() string {
	return func() string {
		it := Integration(t, ctx, ns, name)()
		if it == nil {
			return ""
		}
		if it.Status.IntegrationKit == nil {
			return ""
		}
		kit := Kit(t, ctx, it.Status.IntegrationKit.Namespace, it.Status.IntegrationKit.Name)()
		return kit.Labels[v1.IntegrationKitLayoutLabel]
	}
}

func IntegrationKitStatusPhase(t *testing.T, ctx context.Context, ns string, name string) func() v1.IntegrationKitPhase {
	return func() v1.IntegrationKitPhase {
		it := Integration(t, ctx, ns, name)()
		if it == nil {
			return v1.IntegrationKitPhaseNone
		}
		if it.Status.IntegrationKit == nil {
			return v1.IntegrationKitPhaseNone
		}
		kit := Kit(t, ctx, it.Status.IntegrationKit.Namespace, it.Status.IntegrationKit.Name)()
		return kit.Status.Phase
	}
}

func Kit(t *testing.T, ctx context.Context, ns, name string) func() *v1.IntegrationKit {
	return func() *v1.IntegrationKit {
		kit := v1.NewIntegrationKit(ns, name)
		if err := TestClient(t).Get(ctx, ctrl.ObjectKeyFromObject(kit), kit); err != nil && !k8serrors.IsNotFound(err) {
			failTest(t, err)
		} else if err != nil && k8serrors.IsNotFound(err) {
			return nil
		}
		return kit
	}
}

func KitPhase(t *testing.T, ctx context.Context, ns, name string) func() v1.IntegrationKitPhase {
	return func() v1.IntegrationKitPhase {
		kit := Kit(t, ctx, ns, name)()
		if kit == nil {
			return v1.IntegrationKitPhaseNone
		}
		return kit.Status.Phase
	}
}

func KitImage(t *testing.T, ctx context.Context, ns, name string) func() string {
	return func() string {
		kit := Kit(t, ctx, ns, name)()
		if kit == nil {
			return ""
		}
		return kit.Status.Image
	}
}

func KitRootImage(t *testing.T, ctx context.Context, ns, name string) func() string {
	return func() string {
		kit := Kit(t, ctx, ns, name)()
		if kit == nil {
			return ""
		}
		return kit.Status.RootImage
	}
}

func KitCondition(t *testing.T, ctx context.Context, ns string, name string, conditionType v1.IntegrationKitConditionType) func() *v1.IntegrationKitCondition {
	return func() *v1.IntegrationKitCondition {
		kt := Kit(t, ctx, ns, name)()
		if kt == nil {
			return nil
		}
		return kt.Status.GetCondition(conditionType)
	}
}

func UpdateIntegration(t *testing.T, ctx context.Context, ns string, name string, mutate func(it *v1.Integration)) error {
	it := Integration(t, ctx, ns, name)()
	if it == nil {
		return fmt.Errorf("no integration named %s found", name)
	}
	target := it.DeepCopy()
	mutate(target)
	return TestClient(t).Update(ctx, target)
}

func PatchIntegration(t *testing.T, ctx context.Context, ns string, name string, mutate func(it *v1.Integration)) error {
	it := Integration(t, ctx, ns, name)()
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
	return TestClient(t).Patch(ctx, target, ctrl.RawPatch(types.MergePatchType, p))
}

func ScaleIntegration(t *testing.T, ctx context.Context, ns string, name string, replicas int32) error {
	return PatchIntegration(t, ctx, ns, name, func(it *v1.Integration) {
		it.Spec.Replicas = &replicas
	})
}

func Pipe(t *testing.T, ctx context.Context, ns string, name string) func() *v1.Pipe {
	return func() *v1.Pipe {
		klb := v1.NewPipe(ns, name)
		key := ctrl.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		if err := TestClient(t).Get(ctx, key, &klb); err != nil && !k8serrors.IsNotFound(err) {
			failTest(t, err)
		} else if err != nil && k8serrors.IsNotFound(err) {
			return nil
		}
		return &klb
	}
}

func PipePhase(t *testing.T, ctx context.Context, ns string, name string) func() v1.PipePhase {
	return func() v1.PipePhase {
		klb := Pipe(t, ctx, ns, name)()
		if klb == nil {
			return ""
		}
		return klb.Status.Phase
	}
}

func PipeSpecReplicas(t *testing.T, ctx context.Context, ns string, name string) func() *int32 {
	return func() *int32 {
		klb := Pipe(t, ctx, ns, name)()
		if klb == nil {
			return nil
		}
		return klb.Spec.Replicas
	}
}

func PipeStatusReplicas(t *testing.T, ctx context.Context, ns string, name string) func() *int32 {
	return func() *int32 {
		klb := Pipe(t, ctx, ns, name)()
		if klb == nil {
			return nil
		}
		return klb.Status.Replicas
	}
}

func PipeCondition(t *testing.T, ctx context.Context, ns string, name string, conditionType v1.PipeConditionType) func() *v1.PipeCondition {
	return func() *v1.PipeCondition {
		kb := Pipe(t, ctx, ns, name)()
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

func PipeConditionStatusExtract(c *v1.PipeCondition) corev1.ConditionStatus {
	if c == nil {
		return ""
	}
	return c.Status
}

func PipeConditionReason(c *v1.PipeCondition) string {
	if c == nil {
		return ""
	}
	return c.Reason
}

func PipeConditionMessage(c *v1.PipeCondition) string {
	if c == nil {
		return ""
	}
	return c.Message
}

func PipeConditionStatus(t *testing.T, ctx context.Context, ns string, name string, conditionType v1.PipeConditionType) func() corev1.ConditionStatus {
	return func() corev1.ConditionStatus {
		klb := Pipe(t, ctx, ns, name)()
		if klb == nil {
			return "PipeMissing"
		}
		c := klb.Status.GetCondition(conditionType)
		if c == nil {
			return "ConditionMissing"
		}
		return c.Status
	}
}

func UpdatePipe(t *testing.T, ctx context.Context, ns string, name string, upd func(it *v1.Pipe)) error {
	klb := Pipe(t, ctx, ns, name)()
	if klb == nil {
		return fmt.Errorf("no Pipe named %s found", name)
	}
	target := klb.DeepCopy()
	upd(target)
	// For some reason, full patch fails on some clusters
	p, err := patch.MergePatch(klb, target)
	if err != nil {
		return err
	} else if len(p) == 0 {
		return nil
	}
	return TestClient(t).Patch(ctx, target, ctrl.RawPatch(types.MergePatchType, p))
}

func ScalePipe(t *testing.T, ctx context.Context, ns string, name string, replicas int32) error {
	return UpdatePipe(t, ctx, ns, name, func(klb *v1.Pipe) {
		klb.Spec.Replicas = &replicas
	})
}

func AssignPipeToOperator(t *testing.T, ctx context.Context, ns, name, operator string) error {
	klb := Pipe(t, ctx, ns, name)()
	if klb == nil {
		return fmt.Errorf("cannot assign Pipe %q to operator:Pipe not found", name)
	}

	klb.SetOperatorID(operator)
	return TestClient(t).Update(ctx, klb)
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

func KitWithVersionPrefix(versionPrefix string) KitFilter {
	return &kitFilter{
		filter: func(kit *v1.IntegrationKit) bool {
			return strings.HasPrefix(kit.Status.Version, versionPrefix)
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

func Kits(t *testing.T, ctx context.Context, ns string, options ...interface{}) func() []v1.IntegrationKit {
	filters := make([]KitFilter, 0)
	listOptions := []ctrl.ListOption{ctrl.InNamespace(ns)}
	for _, option := range options {
		switch o := option.(type) {
		case KitFilter:
			filters = append(filters, o)
		case ctrl.ListOption:
			listOptions = append(listOptions, o)
		default:
			failTest(t, fmt.Errorf("unsupported kits option %q", o))
		}
	}

	return func() []v1.IntegrationKit {
		list := v1.NewIntegrationKitList()
		if err := TestClient(t).List(ctx, &list, listOptions...); err != nil {
			failTest(t, err)
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

func DeleteKits(t *testing.T, ctx context.Context, ns string) error {
	kits := Kits(t, ctx, ns)()
	if len(kits) == 0 {
		return nil
	}
	for _, kit := range kits {
		if err := TestClient(t).Delete(ctx, &kit); err != nil {
			return err
		}
	}

	return nil
}

func DeleteIntegrations(t *testing.T, ctx context.Context, ns string) func() (int, error) {
	return func() (int, error) {
		integrations := Integrations(t, ctx, ns)()
		if len(integrations.Items) == 0 {
			return 0, nil
		}

		if err := Kamel(t, ctx, "delete", "--all", "-n", ns).Execute(); err != nil {
			return 0, err
		}

		return NumIntegrations(t, ctx, ns)(), nil
	}
}

func OperatorImage(t *testing.T, ctx context.Context, ns string) func() string {
	return func() string {
		pod := OperatorPod(t, ctx, ns)()
		if pod != nil {
			if len(pod.Spec.Containers) > 0 {
				return pod.Spec.Containers[0].Image
			}
		}
		return ""
	}
}

func OperatorPodSecurityContext(t *testing.T, ctx context.Context, ns string) func() *corev1.SecurityContext {
	return func() *corev1.SecurityContext {
		pod := OperatorPod(t, ctx, ns)()
		if pod == nil || pod.Spec.Containers == nil || len(pod.Spec.Containers) == 0 {
			return nil
		}
		return pod.Spec.Containers[0].SecurityContext
	}
}

func OperatorPodHas(t *testing.T, ctx context.Context, ns string, predicate func(pod *corev1.Pod) bool) func() bool {
	return func() bool {
		pod := OperatorPod(t, ctx, ns)()
		if pod == nil {
			return false
		}
		return predicate(pod)
	}
}

func OperatorPodPhase(t *testing.T, ctx context.Context, ns string) func() corev1.PodPhase {
	return func() corev1.PodPhase {
		pod := OperatorPod(t, ctx, ns)()
		if pod == nil {
			return ""
		}
		return pod.Status.Phase
	}
}

func OperatorEnvVarValue(t *testing.T, ctx context.Context, ns string, key string) func() string {
	return func() string {
		pod := OperatorPod(t, ctx, ns)()
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

func Configmap(t *testing.T, ctx context.Context, ns string, name string) func() *corev1.ConfigMap {
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
		if err := TestClient(t).Get(ctx, key, &cm); err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			log.Error(err, "Error while retrieving configmap "+name)
			return nil
		}
		return &cm
	}
}

func BuilderPod(t *testing.T, ctx context.Context, ns string, name string) func() *corev1.Pod {
	return func() *corev1.Pod {
		pod := corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
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
		if err := TestClient(t).Get(ctx, key, &pod); err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			log.Error(err, "Error while retrieving pod "+name)
			return nil
		}
		return &pod
	}
}

func BuilderPodPhase(t *testing.T, ctx context.Context, ns string, name string) func() corev1.PodPhase {
	return func() corev1.PodPhase {
		pod := BuilderPod(t, ctx, ns, name)()
		if pod == nil {
			return ""
		}
		return pod.Status.Phase
	}
}

func BuilderPodsCount(t *testing.T, ctx context.Context, ns string) func() int {
	return func() int {
		lst := corev1.PodList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
		}
		if err := TestClient(t).List(ctx, &lst,
			ctrl.InNamespace(ns),
			ctrl.MatchingLabels{
				"camel.apache.org/component": "builder",
			}); err != nil {
			failTest(t, err)
		}
		return len(lst.Items)
	}
}

func AutogeneratedConfigmapsCount(t *testing.T, ctx context.Context, ns string) func() int {
	return func() int {
		lst := corev1.ConfigMapList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
		}
		if err := TestClient(t).List(ctx, &lst,
			ctrl.InNamespace(ns),
			ctrl.MatchingLabels{
				kubernetes.ConfigMapAutogenLabel: "true",
			}); err != nil {
			failTest(t, err)
		}
		return len(lst.Items)
	}
}

func CreatePlainTextConfigmap(t *testing.T, ctx context.Context, ns string, name string, data map[string]string) error {
	return CreatePlainTextConfigmapWithLabels(t, ctx, ns, name, data, map[string]string{})
}

func CreatePlainTextConfigmapWithLabels(t *testing.T, ctx context.Context, ns string, name string, data map[string]string, labels map[string]string) error {
	cm := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
			Labels:    labels,
		},
		Data: data,
	}
	return TestClient(t).Create(ctx, &cm)
}

func CreatePlainTextConfigmapWithOwnerRefWithLabels(t *testing.T, ctx context.Context, ns string, name string, data map[string]string, orname string, uid types.UID, labels map[string]string) error {
	cm := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         v1.SchemeGroupVersion.String(),
				Kind:               "Integration",
				Name:               orname,
				UID:                uid,
				Controller:         pointer.Bool(true),
				BlockOwnerDeletion: pointer.Bool(true),
			},
			},
			Labels: labels,
		},
		Data: data,
	}
	return TestClient(t).Create(ctx, &cm)
}

func UpdatePlainTextConfigmap(t *testing.T, ctx context.Context, ns string, name string, data map[string]string) error {
	return UpdatePlainTextConfigmapWithLabels(t, ctx, ns, name, data, nil)
}

func UpdatePlainTextConfigmapWithLabels(t *testing.T, ctx context.Context, ns string, name string, data map[string]string, labels map[string]string) error {
	cm := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
			Labels:    labels,
		},
		Data: data,
	}
	return TestClient(t).Update(ctx, &cm)
}

func CreateBinaryConfigmap(t *testing.T, ctx context.Context, ns string, name string, data map[string][]byte) error {
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
	return TestClient(t).Create(ctx, &cm)
}

func DeleteConfigmap(t *testing.T, ctx context.Context, ns string, name string) error {
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
	return TestClient(t).Delete(ctx, &cm)
}

func CreatePlainTextSecret(t *testing.T, ctx context.Context, ns string, name string, data map[string]string) error {
	return CreatePlainTextSecretWithLabels(t, ctx, ns, name, data, map[string]string{})
}

func UpdatePlainTextSecret(t *testing.T, ctx context.Context, ns string, name string, data map[string]string) error {
	return UpdatePlainTextSecretWithLabels(t, ctx, ns, name, data, nil)
}

func UpdatePlainTextSecretWithLabels(t *testing.T, ctx context.Context, ns string, name string, data map[string]string, labels map[string]string) error {
	sec := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
			Labels:    labels,
		},
		StringData: data,
	}
	return TestClient(t).Update(ctx, &sec)
}

func CreatePlainTextSecretWithLabels(t *testing.T, ctx context.Context, ns string, name string, data map[string]string, labels map[string]string) error {
	sec := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
			Labels:    labels,
		},
		StringData: data,
	}
	return TestClient(t).Create(ctx, &sec)
}

// SecretByName Finds a secret in the given namespace by name or prefix of name
func SecretByName(t *testing.T, ctx context.Context, ns string, prefix string) func() *corev1.Secret {
	return func() *corev1.Secret {
		secretList, err := TestClient(t).CoreV1().Secrets(ns).List(ctx, metav1.ListOptions{})
		if err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			failTest(t, err)
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

func DeleteSecret(t *testing.T, ctx context.Context, ns string, name string) error {
	sec := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
	}
	return TestClient(t).Delete(ctx, &sec)
}

func CreateSecretDecoded(t *testing.T, ctx context.Context, ns string, pathToFile string, secretName string, certName string) error {
	bytes, _ := os.ReadFile(pathToFile)
	block, _ := pem.Decode(bytes)

	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      secretName,
		},
		Data: map[string][]byte{
			certName: block.Bytes,
		},
	}
	return TestClient(t).Create(ctx, &secret)
}

func KnativeService(t *testing.T, ctx context.Context, ns string, name string) func() *servingv1.Service {
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
		if err := TestClient(t).Get(ctx, key, &answer); err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			log.Errorf(err, "Error while retrieving knative service %s", name)
			return nil
		}
		return &answer
	}
}
func DeploymentWithIntegrationLabel(t *testing.T, ctx context.Context, ns string, label string) func() *appsv1.Deployment {
	return func() *appsv1.Deployment {
		lst := appsv1.DeploymentList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: appsv1.SchemeGroupVersion.String(),
			},
		}
		if err := TestClient(t).List(ctx, &lst, ctrl.InNamespace(ns), ctrl.MatchingLabels{v1.IntegrationLabel: label}); err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			log.Errorf(err, "Error while retrieving deployment %s", label)
			return nil
		}
		if len(lst.Items) == 0 {
			return nil
		}
		return &lst.Items[0]
	}
}

func Deployment(t *testing.T, ctx context.Context, ns string, name string) func() *appsv1.Deployment {
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
		if err := TestClient(t).Get(ctx, key, &answer); err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			log.Errorf(err, "Error while retrieving deployment %s", name)
			return nil
		}
		return &answer
	}
}

func DeploymentCondition(t *testing.T, ctx context.Context, ns string, name string, conditionType appsv1.DeploymentConditionType) func() appsv1.DeploymentCondition {
	return func() appsv1.DeploymentCondition {
		deployment := Deployment(t, ctx, ns, name)()

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

func Build(t *testing.T, ctx context.Context, ns, name string) func() *v1.Build {
	return func() *v1.Build {
		build := v1.NewBuild(ns, name)
		if err := TestClient(t).Get(ctx, ctrl.ObjectKeyFromObject(build), build); err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			log.Error(err, "Error while retrieving build "+name)
			return nil
		}
		return build
	}
}

func BuildConfig(t *testing.T, ctx context.Context, ns, name string) func() v1.BuildConfiguration {
	return func() v1.BuildConfiguration {
		build := Build(t, ctx, ns, name)()
		if build != nil {
			return *build.BuilderConfiguration()
		}
		return v1.BuildConfiguration{}
	}
}

func BuildPhase(t *testing.T, ctx context.Context, ns, name string) func() v1.BuildPhase {
	return func() v1.BuildPhase {
		build := Build(t, ctx, ns, name)()
		if build != nil {
			return build.Status.Phase
		}
		return v1.BuildPhaseNone
	}
}

func BuildConditions(t *testing.T, ctx context.Context, ns, name string) func() []v1.BuildCondition {
	return func() []v1.BuildCondition {
		build := Build(t, ctx, ns, name)()
		if build != nil && &build.Status != nil && build.Status.Conditions != nil {
			return build.Status.Conditions
		}
		return nil
	}
}

func BuildCondition(t *testing.T, ctx context.Context, ns string, name string, conditionType v1.BuildConditionType) func() *v1.BuildCondition {
	return func() *v1.BuildCondition {
		build := Build(t, ctx, ns, name)()
		if build != nil && &build.Status != nil && build.Status.Conditions != nil {
			return build.Status.GetCondition(conditionType)
		}
		return &v1.BuildCondition{}
	}
}

func BuildFailureRecoveryAttempt(t *testing.T, ctx context.Context, ns, name string) func() int {
	return func() int {
		build := Build(t, ctx, ns, name)()
		if build != nil {
			return build.Status.Failure.Recovery.Attempt
		}
		return 0
	}
}

func BuildsRunning(predicates ...func() v1.BuildPhase) func() int {
	return func() int {
		runningBuilds := 0
		for _, predicate := range predicates {
			if predicate() == v1.BuildPhaseRunning {
				runningBuilds++
			}
		}
		return runningBuilds
	}
}

func HasPlatform(t *testing.T, ctx context.Context, ns string) func() bool {
	return func() bool {
		lst := v1.NewIntegrationPlatformList()
		if err := TestClient(t).List(ctx, &lst, ctrl.InNamespace(ns)); err != nil {
			return false
		}
		return len(lst.Items) > 0
	}
}

func Platform(t *testing.T, ctx context.Context, ns string) func() *v1.IntegrationPlatform {
	return func() *v1.IntegrationPlatform {
		lst := v1.NewIntegrationPlatformList()
		if err := TestClient(t).List(ctx, &lst, ctrl.InNamespace(ns)); err != nil {
			failTest(t, err)
		}
		if len(lst.Items) == 0 {
			return nil
		}
		if len(lst.Items) > 1 {
			failTest(t, fmt.Errorf("multiple integration platforms found in namespace %q", ns))
		}
		return &lst.Items[0]
	}
}

func PlatformByName(t *testing.T, ctx context.Context, ns string, name string) func() *v1.IntegrationPlatform {
	return func() *v1.IntegrationPlatform {
		lst := v1.NewIntegrationPlatformList()
		if err := TestClient(t).List(ctx, &lst, ctrl.InNamespace(ns)); err != nil {
			failTest(t, err)
		}
		for _, p := range lst.Items {
			if p.Name == name {
				return &p
			}
		}
		return nil
	}
}

func IntegrationProfileByName(t *testing.T, ctx context.Context, ns string, name string) func() *v1.IntegrationProfile {
	return func() *v1.IntegrationProfile {
		lst := v1.NewIntegrationProfileList()
		if err := TestClient(t).List(ctx, &lst, ctrl.InNamespace(ns)); err != nil {
			failTest(t, err)
		}
		for _, pc := range lst.Items {
			if pc.Name == name {
				return &pc
			}
		}
		return nil
	}
}

func CamelCatalog(t *testing.T, ctx context.Context, ns, name string) func() *v1.CamelCatalog {
	return func() *v1.CamelCatalog {
		cat := v1.CamelCatalog{}
		key := ctrl.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		if err := TestClient(t).Get(ctx, key, &cat); err != nil && !k8serrors.IsNotFound(err) {
			failTest(t, err)
		} else if err != nil && k8serrors.IsNotFound(err) {
			return nil
		}
		return &cat
	}
}

func IntegrationProfile(t *testing.T, ctx context.Context, ns string) func() *v1.IntegrationProfile {
	return func() *v1.IntegrationProfile {
		lst := v1.NewIntegrationProfileList()
		if err := TestClient(t).List(ctx, &lst, ctrl.InNamespace(ns)); err != nil {
			failTest(t, err)
		}
		if len(lst.Items) == 0 {
			return nil
		}
		if len(lst.Items) > 1 {
			failTest(t, fmt.Errorf("multiple integration profiles found in namespace %q", ns))
		}
		return &lst.Items[0]
	}
}

func CreateIntegrationProfile(t *testing.T, ctx context.Context, profile *v1.IntegrationProfile) error {
	return TestClient(t).Create(ctx, profile)
}

func UpdateIntegrationProfile(t *testing.T, ctx context.Context, ns string, upd func(ipr *v1.IntegrationProfile)) error {
	ipr := IntegrationProfile(t, ctx, ns)()
	if ipr == nil {
		return fmt.Errorf("unable to locate Integration Profile in %s", ns)
	}
	target := ipr.DeepCopy()
	upd(target)
	// For some reason, full patch fails on some clusters
	p, err := patch.MergePatch(ipr, target)
	if err != nil {
		return err
	} else if len(p) == 0 {
		return nil
	}
	return TestClient(t).Patch(ctx, target, ctrl.RawPatch(types.MergePatchType, p))
}

func DeleteCamelCatalog(t *testing.T, ctx context.Context, ns, name string) func() bool {
	return func() bool {
		cat := CamelCatalog(t, ctx, ns, name)()
		if cat == nil {
			return true
		}
		if err := TestClient(t).Delete(ctx, cat); err != nil {
			log.Error(err, "Got error while deleting the catalog")
		}
		return true
	}
}

func DefaultCamelCatalogPhase(t *testing.T, ctx context.Context, ns string) func() v1.CamelCatalogPhase {
	return func() v1.CamelCatalogPhase {
		catalogName := fmt.Sprintf("camel-catalog-%s", strings.ToLower(defaults.DefaultRuntimeVersion))
		c := CamelCatalog(t, ctx, ns, catalogName)()
		if c == nil {
			return ""
		}
		return c.Status.Phase
	}
}

func CamelCatalogPhase(t *testing.T, ctx context.Context, ns, name string) func() v1.CamelCatalogPhase {
	return func() v1.CamelCatalogPhase {
		c := CamelCatalog(t, ctx, ns, name)()
		if c == nil {
			return ""
		}
		return c.Status.Phase
	}
}

func CamelCatalogCondition(t *testing.T, ctx context.Context, ns, name string, conditionType v1.CamelCatalogConditionType) func() *v1.CamelCatalogCondition {
	return func() *v1.CamelCatalogCondition {
		c := CamelCatalog(t, ctx, ns, name)()
		if c == nil {
			return nil
		}
		for _, condition := range c.Status.Conditions {
			if condition.Type == conditionType {
				return &condition
			}
		}
		return nil
	}
}

func CamelCatalogImage(t *testing.T, ctx context.Context, ns, name string) func() string {
	return func() string {
		c := CamelCatalog(t, ctx, ns, name)()
		if c == nil {
			return ""
		}
		return c.Status.Image
	}
}

func CamelCatalogList(t *testing.T, ctx context.Context, ns string) func() []v1.CamelCatalog {
	return func() []v1.CamelCatalog {
		lst := v1.NewCamelCatalogList()
		if err := TestClient(t).List(ctx, &lst, ctrl.InNamespace(ns)); err != nil {
			failTest(t, err)
		}
		return lst.Items
	}
}

func DeletePlatform(t *testing.T, ctx context.Context, ns string) func() bool {
	return func() bool {
		pl := Platform(t, ctx, ns)()
		if pl == nil {
			return true
		}
		if err := TestClient(t).Delete(ctx, pl); err != nil {
			log.Error(err, "Got error while deleting the platform")
		}
		return false
	}
}

func UpdatePlatform(t *testing.T, ctx context.Context, ns string, upd func(ip *v1.IntegrationPlatform)) error {
	ip := PlatformByName(t, ctx, ns, platform.DefaultPlatformName)()
	if ip == nil {
		return fmt.Errorf("unable to locate Integration Platform %s in %s", platform.DefaultPlatformName, ns)
	}
	target := ip.DeepCopy()
	upd(target)
	// For some reason, full patch fails on some clusters
	p, err := patch.MergePatch(ip, target)
	if err != nil {
		return err
	} else if len(p) == 0 {
		return nil
	}
	return TestClient(t).Patch(ctx, target, ctrl.RawPatch(types.MergePatchType, p))
}

func PlatformVersion(t *testing.T, ctx context.Context, ns string) func() string {
	return func() string {
		p := Platform(t, ctx, ns)()
		if p == nil {
			return ""
		}
		return p.Status.Version
	}
}

func PlatformPhase(t *testing.T, ctx context.Context, ns string) func() v1.IntegrationPlatformPhase {
	return func() v1.IntegrationPlatformPhase {
		p := Platform(t, ctx, ns)()
		if p == nil {
			return ""
		}
		return p.Status.Phase
	}
}

func SelectedPlatformPhase(t *testing.T, ctx context.Context, ns string, name string) func() v1.IntegrationPlatformPhase {
	return func() v1.IntegrationPlatformPhase {
		p := PlatformByName(t, ctx, ns, name)()
		if p == nil {
			return ""
		}
		return p.Status.Phase
	}
}

func SelectedIntegrationProfilePhase(t *testing.T, ctx context.Context, ns string, name string) func() v1.IntegrationProfilePhase {
	return func() v1.IntegrationProfilePhase {
		pc := IntegrationProfileByName(t, ctx, ns, name)()
		if pc == nil {
			return ""
		}
		return pc.Status.Phase
	}
}

func PlatformHas(t *testing.T, ctx context.Context, ns string, predicate func(pl *v1.IntegrationPlatform) bool) func() bool {
	return func() bool {
		pl := Platform(t, ctx, ns)()
		if pl == nil {
			return false
		}
		return predicate(pl)
	}
}

func PlatformCondition(t *testing.T, ctx context.Context, ns string, conditionType v1.IntegrationPlatformConditionType) func() *v1.IntegrationPlatformCondition {
	return func() *v1.IntegrationPlatformCondition {
		p := Platform(t, ctx, ns)()
		if p == nil {
			return nil
		}
		return p.Status.GetCondition(conditionType)
	}
}

func PlatformConditionStatus(t *testing.T, ctx context.Context, ns string, conditionType v1.IntegrationPlatformConditionType) func() corev1.ConditionStatus {
	return func() corev1.ConditionStatus {
		c := PlatformCondition(t, ctx, ns, conditionType)()
		if c == nil {
			return "Unknown"
		}
		return c.Status
	}
}

func PlatformProfile(t *testing.T, ctx context.Context, ns string) func() v1.TraitProfile {
	return func() v1.TraitProfile {
		p := Platform(t, ctx, ns)()
		if p == nil {
			return ""
		}
		return p.Status.Profile
	}
}

func PlatformTimeout(t *testing.T, ctx context.Context, ns string) func() *metav1.Duration {
	return func() *metav1.Duration {
		p := Platform(t, ctx, ns)()
		if p == nil {
			return &metav1.Duration{}
		}
		return p.Status.Build.Timeout
	}
}

func AssignPlatformToOperator(t *testing.T, ctx context.Context, ns, operator string) error {
	pl := Platform(t, ctx, ns)()
	if pl == nil {
		return errors.New("cannot assign platform to operator: no platform found")
	}

	pl.SetOperatorID(operator)
	return TestClient(t).Update(ctx, pl)
}

func GetExpectedCRDs(releaseVersion string) int {
	switch releaseVersion {
	case "2.2.0":
		return 8
	case defaults.Version:
		return ExpectedCRDs
	}

	return ExpectedCRDs
}

func CRDs(t *testing.T) func() []metav1.APIResource {
	return func() []metav1.APIResource {

		kinds := []string{
			reflect.TypeOf(v1.Build{}).Name(),
			reflect.TypeOf(v1.Integration{}).Name(),
			reflect.TypeOf(v1.IntegrationKit{}).Name(),
			reflect.TypeOf(v1.IntegrationPlatform{}).Name(),
			reflect.TypeOf(v1.IntegrationProfile{}).Name(),
			reflect.TypeOf(v1.Kamelet{}).Name(),
			reflect.TypeOf(v1.Pipe{}).Name(),
			reflect.TypeOf(v1alpha1.KameletBinding{}).Name(),
		}

		versions := []string{"v1", "v1alpha1"}
		present := []metav1.APIResource{}

		for _, version := range versions {
			lst, err := TestClient(t).Discovery().ServerResourcesForGroupVersion("camel.apache.org/" + version)
			if err != nil && k8serrors.IsNotFound(err) {
				return nil
			} else if err != nil {
				failTest(t, err)
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

func ConsoleCLIDownload(t *testing.T, ctx context.Context, name string) func() *consoleV1.ConsoleCLIDownload {
	return func() *consoleV1.ConsoleCLIDownload {
		cliDownload := consoleV1.ConsoleCLIDownload{}
		if err := TestClient(t).Get(ctx, ctrl.ObjectKey{Name: name}, &cliDownload); err != nil && !k8serrors.IsNotFound(err) {
			failTest(t, err)
		} else if err != nil && k8serrors.IsNotFound(err) {
			return nil
		}
		return &cliDownload
	}
}

func operatorPods(t *testing.T, ctx context.Context, ns string) []corev1.Pod {
	lst := corev1.PodList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: v1.SchemeGroupVersion.String(),
		},
	}
	opts := []ctrl.ListOption{
		ctrl.MatchingLabels{
			"camel.apache.org/component": "operator",
		},
	}
	if ns != "" {
		opts = append(opts, ctrl.InNamespace(ns))
	}
	if err := TestClient(t).List(ctx, &lst, opts...); err != nil {
		failTest(t, err)
	}
	if len(lst.Items) == 0 {
		return nil
	}
	return lst.Items
}

func OperatorPod(t *testing.T, ctx context.Context, ns string) func() *corev1.Pod {
	return func() *corev1.Pod {
		pods := operatorPods(t, ctx, ns)
		if len(pods) > 0 {
			return &pods[0]
		}
		return nil
	}
}

// Return the first global operator Pod found in the cluster (if any).
func OperatorPodGlobal(t *testing.T, ctx context.Context) func() *corev1.Pod {
	return func() *corev1.Pod {
		pods := operatorPods(t, ctx, "")
		for _, pod := range pods {
			for _, envVar := range pod.Spec.Containers[0].Env {
				if envVar.Name == "WATCH_NAMESPACE" {
					if envVar.Value == "" {
						return &pod
					}
				}
			}
		}
		return nil
	}
}

// Pod Find one pod filtered by namespace ns and label app.kubernetes.io/name value appName.
func Pod(t *testing.T, ctx context.Context, ns string, appName string) func() (*corev1.Pod, error) {
	return func() (*corev1.Pod, error) {
		lst := corev1.PodList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: v1.SchemeGroupVersion.String(),
			},
		}
		if err := TestClient(t).List(ctx, &lst,
			ctrl.InNamespace(ns),
			ctrl.MatchingLabels{
				"app.kubernetes.io/name": appName,
			}); err != nil {
			return nil, err
		}
		if len(lst.Items) == 0 {
			return nil, nil
		}
		return &lst.Items[0], nil
	}
}

func OperatorTryPodForceKill(t *testing.T, ctx context.Context, ns string, timeSeconds int) {
	pod := OperatorPod(t, ctx, ns)()
	if pod != nil {
		if err := TestClient(t).Delete(ctx, pod, ctrl.GracePeriodSeconds(timeSeconds)); err != nil {
			log.Error(err, "cannot forcefully kill the pod")
		}
	}
}

func ScaleOperator(t *testing.T, ctx context.Context, ns string, replicas int32) error {
	operator, err := TestClient(t).AppsV1().Deployments(ns).Get(ctx, "camel-k-operator", metav1.GetOptions{})
	if err != nil {
		return err
	}
	operator.Spec.Replicas = &replicas
	_, err = TestClient(t).AppsV1().Deployments(ns).Update(ctx, operator, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	if replicas == 0 {
		// speedup scale down by killing the pod
		OperatorTryPodForceKill(t, ctx, ns, 10)
	}
	return nil
}

func ClusterRole(t *testing.T, ctx context.Context) func() []rbacv1.ClusterRole {
	return func() []rbacv1.ClusterRole {
		lst := rbacv1.ClusterRoleList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ClusterRole",
				APIVersion: rbacv1.SchemeGroupVersion.String(),
			},
		}
		if err := TestClient(t).List(ctx, &lst,
			ctrl.MatchingLabels{
				"app": "camel-k",
			}); err != nil {
			failTest(t, err)
		}
		if len(lst.Items) == 0 {
			return nil
		}
		return lst.Items
	}
}

func Role(t *testing.T, ctx context.Context, ns string) func() []rbacv1.Role {
	return func() []rbacv1.Role {
		lst := rbacv1.RoleList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Role",
				APIVersion: rbacv1.SchemeGroupVersion.String(),
			},
		}
		if err := TestClient(t).List(ctx, &lst,
			ctrl.InNamespace(ns),
			ctrl.MatchingLabels{
				"app": "camel-k",
			}); err != nil {
			failTest(t, err)
		}
		if len(lst.Items) == 0 {
			return nil
		}
		return lst.Items
	}
}

func RoleBinding(t *testing.T, ctx context.Context, ns string) func() *rbacv1.RoleBinding {
	return func() *rbacv1.RoleBinding {
		lst := rbacv1.RoleBindingList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "RoleBinding",
				APIVersion: metav1.SchemeGroupVersion.String(),
			},
		}
		if err := TestClient(t).List(ctx, &lst,
			ctrl.InNamespace(ns),
			ctrl.MatchingLabels{
				"app": "camel-k",
			}); err != nil {
			failTest(t, err)
		}
		if len(lst.Items) == 0 {
			return nil
		}
		return &lst.Items[0]
	}
}

func ServiceAccount(t *testing.T, ctx context.Context, ns, name string) func() *corev1.ServiceAccount {
	return func() *corev1.ServiceAccount {
		lst := corev1.ServiceAccountList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ServiceAccount",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
		}
		if err := TestClient(t).List(ctx, &lst,
			ctrl.InNamespace(ns),
			ctrl.MatchingLabels{
				"app": "camel-k",
			}); err != nil {
			failTest(t, err)
		}
		if len(lst.Items) == 0 {
			return nil
		}
		return &lst.Items[0]
	}
}

func KameletList(t *testing.T, ctx context.Context, ns string) func() []v1.Kamelet {
	return func() []v1.Kamelet {
		lst := v1.NewKameletList()
		if err := TestClient(t).List(ctx, &lst, ctrl.InNamespace(ns)); err != nil {
			failTest(t, err)
		}
		return lst.Items
	}
}

func Kamelet(t *testing.T, ctx context.Context, name string, ns string) func() *v1.Kamelet {
	return func() *v1.Kamelet {
		it := v1.NewKamelet(ns, name)
		key := ctrl.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		if err := TestClient(t).Get(ctx, key, &it); err != nil && !k8serrors.IsNotFound(err) {
			failTest(t, err)
		} else if err != nil && k8serrors.IsNotFound(err) {
			return nil
		}
		return &it
	}
}

func KameletLabels(kamelet *v1.Kamelet) map[string]string {
	if kamelet == nil {
		return map[string]string{}
	}
	return kamelet.GetLabels()
}

func ClusterDomainName(t *testing.T, ctx context.Context) (string, error) {
	dns := configv1.DNS{}
	key := ctrl.ObjectKey{
		Name: "cluster",
	}
	err := TestClient(t).Get(ctx, key, &dns)
	if err != nil {
		return "", err
	}
	return dns.Spec.BaseDomain, nil
}

/*
	Tekton
*/

func CreateOperatorServiceAccount(t *testing.T, ctx context.Context, ns string) error {
	return install.Resource(ctx, TestClient(t), ns, true, install.IdentityResourceCustomizer, "/config/manager/operator-service-account.yaml")
}

func CreateOperatorRole(t *testing.T, ctx context.Context, ns string) (err error) {
	oc, err := openshift.IsOpenShift(TestClient(t))
	if err != nil {
		failTest(t, err)
	}
	customizer := install.IdentityResourceCustomizer
	if oc {
		// Remove Ingress permissions as it's not needed on OpenShift
		// This should ideally be removed from the common RBAC manifest.
		customizer = install.RemoveIngressRoleCustomizer
	}
	err = install.Resource(ctx, TestClient(t), ns, true, customizer, "/config/rbac/namespaced/operator-role.yaml")
	if err != nil {
		return err
	}
	if oc {
		return install.Resource(ctx, TestClient(t), ns, true, install.IdentityResourceCustomizer, "/config/rbac/openshift/namespaced/operator-role-openshift.yaml")
	}
	return nil
}

func CreateOperatorRoleBinding(t *testing.T, ctx context.Context, ns string) error {
	oc, err := openshift.IsOpenShift(TestClient(t))
	if err != nil {
		failTest(t, err)
	}
	err = install.Resource(ctx, TestClient(t), ns, true, install.IdentityResourceCustomizer, "/config/rbac/namespaced/operator-role-binding.yaml")
	if err != nil {
		return err
	}
	if oc {
		return install.Resource(ctx, TestClient(t), ns, true, install.IdentityResourceCustomizer, "/config/rbac/openshift/namespaced/operator-role-binding-openshift.yaml")
	}
	return nil
}

// CreateKamelPodWithIntegrationSource generates and deploy a Pod from current Camel K controller image that will run a `kamel xxxx` command.
// The integration parameter represent an Integration source file contained in a ConfigMap or Secret defined and mounted on the as a Volume.
func CreateKamelPodWithIntegrationSource(t *testing.T, ctx context.Context, ns string, name string, integration v1.ValueSource, command ...string) error {

	var volumes []corev1.Volume
	if integration.SecretKeyRef != nil {
		volumes = []corev1.Volume{
			{
				Name: "integration-source-volume",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: integration.SecretKeyRef.Name,
					},
				},
			},
		}
	} else {
		volumes = []corev1.Volume{
			{
				Name: "integration-source-volume",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: integration.ConfigMapKeyRef.LocalObjectReference,
					},
				},
			},
		}
	}

	var volumeMounts []corev1.VolumeMount
	volumeMounts = []corev1.VolumeMount{
		{
			Name:      "integration-source-volume",
			MountPath: "/tmp/",
			ReadOnly:  true,
		},
	}

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
					Name:         "kamel-runner",
					Image:        TestImageName + ":" + TestImageVersion,
					Command:      append([]string{"kamel"}, args...),
					VolumeMounts: volumeMounts,
				},
			},
			Volumes: volumes,
		},
	}
	return TestClient(t).Create(ctx, &pod)
}

/*
	Knative
*/

func CreateKnativeChannel(t *testing.T, ctx context.Context, ns string, name string) func() error {
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
		return TestClient(t).Create(ctx, &channel)
	}
}

func CreateKnativeBroker(t *testing.T, ctx context.Context, ns string, name string) func() error {
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
		return TestClient(t).Create(ctx, &broker)
	}
}

/*
	Kamelets
*/

func CreateKamelet(t *testing.T, ctx context.Context, ns string, name string, template map[string]interface{}, properties map[string]v1.JSONSchemaProp, labels map[string]string) func() error {
	return CreateKameletWithID(t, platform.DefaultPlatformName, ctx, ns, name, template, properties, labels)
}

func CreateKameletWithID(t *testing.T, operatorID string, ctx context.Context, ns string, name string, template map[string]interface{}, properties map[string]v1.JSONSchemaProp, labels map[string]string) func() error {
	return func() error {
		kamelet := v1.Kamelet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      name,
				Labels:    labels,
			},
			Spec: v1.KameletSpec{
				Definition: &v1.JSONSchemaProps{
					Properties: properties,
				},
				Template: asTemplate(t, template),
			},
		}

		kamelet.SetOperatorID(operatorID)
		return TestClient(t).Create(ctx, &kamelet)
	}
}

func CreateTimerKamelet(t *testing.T, ctx context.Context, ns string, name string) func() error {
	return CreateTimerKameletWithID(t, ctx, platform.DefaultPlatformName, ns, name)
}

func CreateTimerKameletWithID(t *testing.T, ctx context.Context, operatorID string, ns string, name string) func() error {
	props := map[string]v1.JSONSchemaProp{
		"message": {
			Type: "string",
		},
	}

	flow := map[string]interface{}{
		"from": map[string]interface{}{
			"uri": "timer:tick",
			"steps": []map[string]interface{}{
				{
					"setBody": map[string]interface{}{
						"constant": "{{message}}",
					},
				},
				{
					"to": "kamelet:sink",
				},
			},
		},
	}

	return CreateKameletWithID(t, operatorID, ctx, ns, name, flow, props, nil)
}

func DeleteKamelet(t *testing.T, ctx context.Context, ns string, name string) error {
	kamelet := v1.Kamelet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
	}
	return TestClient(t).Delete(ctx, &kamelet)
}

func asTemplate(t *testing.T, source map[string]interface{}) *v1.Template {
	bytes, err := json.Marshal(source)
	if err != nil {
		failTest(t, err)
	}
	return &v1.Template{
		RawMessage: bytes,
	}
}

// nolint: staticcheck
func AsTraitConfiguration(t *testing.T, props map[string]string) *traitv1.Configuration {
	bytes, err := json.Marshal(props)
	if err != nil {
		failTest(t, err)
	}
	return &traitv1.Configuration{
		RawMessage: bytes,
	}
}

/*
	Namespace testing functions
*/

func Pods(t *testing.T, ctx context.Context, ns string) func() []corev1.Pod {
	return func() []corev1.Pod {
		lst := corev1.PodList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: v1.SchemeGroupVersion.String(),
			},
		}
		if err := TestClient(t).List(ctx, &lst, ctrl.InNamespace(ns)); err != nil {
			if !k8serrors.IsUnauthorized(err) {
				log.Error(err, "Error while listing the pods")
			}
			return nil
		}
		return lst.Items
	}
}

func WithNewTestNamespace(t *testing.T, doRun func(context.Context, *gomega.WithT, string)) {
	ns := NewTestNamespace(t, testContext, false)
	defer deleteTestNamespace(t, testContext, ns)
	defer userCleanup(t)

	invokeUserTestCode(t, testContext, ns.GetName(), doRun)
}

func WithGlobalOperatorNamespace(t *testing.T, test func(context.Context, *gomega.WithT, string)) {
	ocp, err := openshift.IsOpenShift(TestClient(t))
	require.NoError(t, err)
	if ocp {
		// global operators are always installed in the openshift-operators namespace
		invokeUserTestCode(t, testContext, "openshift-operators", test)
	} else {
		// create new namespace for the global operator
		WithNewTestNamespace(t, test)
	}
}

func WithNewTestNamespaceWithKnativeBroker(t *testing.T, doRun func(context.Context, *gomega.WithT, string)) {
	ns := NewTestNamespace(t, testContext, true)
	defer deleteTestNamespace(t, testContext, ns)
	defer deleteKnativeBroker(t, testContext, ns)
	defer userCleanup(t)

	invokeUserTestCode(t, testContext, ns.GetName(), doRun)
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

func invokeUserTestCode(t *testing.T, ctx context.Context, ns string, doRun func(context.Context, *gomega.WithT, string)) {
	defer func() {
		DumpNamespace(t, ctx, ns)

		osns := os.Getenv("CAMEL_K_GLOBAL_OPERATOR_NS")

		// Try to clean up namespace
		if ns != osns && HasPlatform(t, ctx, ns)() {
			t.Logf("Clean up test namespace: %s", ns)

			if err := Kamel(t, ctx, "uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles").Execute(); err != nil {
				t.Logf("Error while cleaning up namespace %s: %v\n", ns, err)
			}

			t.Logf("Successfully cleaned up test namespace: %s", ns)
		}
	}()

	g := gomega.NewWithT(t)
	doRun(ctx, g, ns)
}

func deleteKnativeBroker(t *testing.T, ctx context.Context, ns metav1.Object) {
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
	if err := TestClient(t).Get(ctx, nsKey, &nsRef); err != nil {
		failTest(t, err)
	}

	nsRef.SetLabels(make(map[string]string, 0))
	if err := TestClient(t).Update(ctx, &nsRef); err != nil {
		failTest(t, err)
	}
	broker := eventing.Broker{
		TypeMeta: metav1.TypeMeta{
			APIVersion: eventing.SchemeGroupVersion.String(),
			Kind:       "Broker",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns.GetName(),
			Name:      TestDefaultNamespace,
		},
	}
	if err := TestClient(t).Delete(ctx, &broker); err != nil {
		failTest(t, err)
	}
}

func deleteTestNamespace(t *testing.T, ctx context.Context, ns ctrl.Object) {
	value, saveNS := os.LookupEnv("CAMEL_K_TEST_SAVE_FAILED_TEST_NAMESPACE")
	if t != nil && t.Failed() && saveNS && value == "true" {
		t.Logf("Warning: retaining failed test project %q", ns.GetName())
		return
	}

	var oc bool
	var err error
	if oc, err = openshift.IsOpenShift(TestClient(t)); err != nil {
		failTest(t, err)
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
		if err := TestClient(t).Delete(ctx, prj); err != nil {
			t.Logf("Warning: cannot delete test project %q", prj.Name)
		}
	} else {
		if err := TestClient(t).Delete(ctx, ns); err != nil {
			t.Logf("Warning: cannot delete test namespace %q", ns.GetName())
		}
	}

	// Wait for all pods to be deleted
	pods := Pods(t, ctx, ns.GetName())()
	for i := 0; len(pods) > 0 && i < 60; i++ {
		time.Sleep(1 * time.Second)
		pods = Pods(t, ctx, ns.GetName())()
	}
	if len(pods) > 0 {
		names := []string{}
		for _, pod := range pods {
			names = append(names, pod.Name)
		}
		t.Logf("Warning: some pods are still running in namespace %q after deletion", ns.GetName())
		t.Logf("Warning: %d running pods: %s", len(pods), names)
	}
}

func testNamespaceExists(t *testing.T, ctx context.Context, ns string) (bool, error) {
	_, err := TestClient(t).CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return false, nil
		} else {
			return false, err
		}
	}

	return true, nil
}

func DumpNamespace(t *testing.T, ctx context.Context, ns string) {
	if t.Failed() {
		if err := util.Dump(ctx, TestClient(t), ns, t); err != nil {
			t.Logf("Error while dumping namespace %s: %v\n", ns, err)
		}
	}
}

func DeleteNamespace(t *testing.T, ctx context.Context, ns string) error {
	nsObj, err := TestClient(t).CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
	if err != nil {
		return err
	}

	deleteTestNamespace(t, ctx, nsObj)

	return nil
}

func NewTestNamespace(t *testing.T, ctx context.Context, injectKnativeBroker bool) ctrl.Object {
	name := os.Getenv("CAMEL_K_TEST_NS")
	if name == "" {
		name = "test-" + uuid.New().String()
	}

	if exists, err := testNamespaceExists(t, ctx, name); err != nil {
		failTest(t, err)
	} else if exists {
		fmt.Println("Warning: namespace ", name, " already exists so using different namespace name")
		name = fmt.Sprintf("%s-%d", name, time.Now().Second())
	}

	return NewNamedTestNamespace(t, ctx, name, injectKnativeBroker)
}

func NewNamedTestNamespace(t *testing.T, ctx context.Context, name string, injectKnativeBroker bool) ctrl.Object {
	brokerLabel := "eventing.knative.dev/injection"
	c := TestClient(t)

	if oc, err := openshift.IsOpenShift(TestClient(t)); err != nil {
		failTest(t, err)
	} else if oc {
		httpCli, err := rest.HTTPClientFor(c.GetConfig())
		if err != nil {
			failTest(t, err)
		}
		rest, err := apiutil.RESTClientForGVK(
			schema.GroupVersionKind{Group: projectv1.GroupName, Version: projectv1.GroupVersion.Version}, false,
			c.GetConfig(), serializer.NewCodecFactory(c.GetScheme()), httpCli)
		if err != nil {
			failTest(t, err)
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
			Do(ctx).
			Into(project)
		if err != nil {
			failTest(t, err)
		}
		// workaround https://github.com/openshift/origin/issues/3819
		if injectKnativeBroker {
			// use Kubernetes API - https://access.redhat.com/solutions/2677921
			if namespace, err := TestClient(t).CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{}); err != nil {
				failTest(t, err)
			} else {
				if _, ok := namespace.GetLabels()[brokerLabel]; !ok {
					namespace.SetLabels(map[string]string{
						brokerLabel: "enabled",
					})
					if err = TestClient(t).Update(ctx, namespace); err != nil {
						failTest(t, errors.New("Unable to label project with knative-eventing-injection. This operation needs update permission on the project."))
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
		if err := TestClient(t).Create(ctx, namespace); err != nil {
			failTest(t, err)
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

func CreateLogKamelet(t *testing.T, ctx context.Context, ns string, name string) func() error {
	flow := map[string]interface{}{
		"from": map[string]interface{}{
			"uri": "kamelet:source",
			"steps": []map[string]interface{}{
				{
					"to": "log:{{loggerName}}",
				},
			},
		},
	}

	props := map[string]v1.JSONSchemaProp{
		"loggerName": {
			Type: "string",
		},
	}

	return CreateKamelet(t, ctx, ns, name, flow, props, nil)
}

func GetCIProcessID() string {
	id, err := os.ReadFile(ciPID)
	if err != nil {
		return ""
	}
	return string(id)
}

func SaveCIProcessID(id string) {
	err := os.WriteFile(ciPID, []byte(id), 0644)
	if err != nil {
		panic(err)
	}
}

func DeleteCIProcessID() {
	err := os.Remove(ciPID)
	if err != nil {
		panic(err)
	}
}

func RandomizedSuffixName(name string) string {
	return name + strings.ToLower(v2util.RandomString(5))
}
