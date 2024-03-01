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

package cli

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"text/template"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/install"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/openshift"
	consolev1 "github.com/openshift/api/console/v1"
)

func TestBasicInstallation(t *testing.T) {
	WithNewTestNamespace(t, func(g *WithT, ns string) {
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		g.Expect(KamelInstallWithID(t, operatorID, ns).Execute()).To(Succeed())
		g.Eventually(OperatorPod(t, ns)).ShouldNot(BeNil())
		g.Eventually(Platform(t, ns)).ShouldNot(BeNil())
		g.Eventually(PlatformConditionStatus(t, ns, v1.IntegrationPlatformConditionTypeCreated), TestTimeoutShort).
			Should(Equal(corev1.ConditionTrue))

		// Check if restricted security context has been applyed
		operatorPod := OperatorPod(t, ns)()
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.RunAsNonRoot).To(Equal(kubernetes.DefaultOperatorSecurityContext().RunAsNonRoot))
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.Capabilities).To(Equal(kubernetes.DefaultOperatorSecurityContext().Capabilities))
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.SeccompProfile).To(Equal(kubernetes.DefaultOperatorSecurityContext().SeccompProfile))
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.AllowPrivilegeEscalation).To(Equal(kubernetes.DefaultOperatorSecurityContext().AllowPrivilegeEscalation))

		t.Run("run yaml", func(t *testing.T) {
			g.Expect(KamelRunWithID(t, operatorID, ns, "files/yaml.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ns, "yaml", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ns, "yaml"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			// Check if file exists in operator pod
			g.Expect(OperatorPod(t, ns)().Name).NotTo(Equal(""))
			g.Expect(OperatorPod(t, ns)().Spec.Containers[0].Name).NotTo(Equal(""))

			req := TestClient(t).CoreV1().RESTClient().Post().
				Resource("pods").
				Name(OperatorPod(t, ns)().Name).
				Namespace(ns).
				SubResource("exec").
				Param("container", OperatorPod(t, ns)().Spec.Containers[0].Name)

			req.VersionedParams(&corev1.PodExecOptions{
				Container: OperatorPod(t, ns)().Spec.Containers[0].Name,
				Command:   []string{"test", "-e", defaults.LocalRepository + "/org/apache/camel/k"},
				Stdin:     false,
				Stdout:    true,
				Stderr:    true,
				TTY:       false,
			}, scheme.ParameterCodec)

			exec, err := remotecommand.NewSPDYExecutor(TestClient(t).GetConfig(), "POST", req.URL())
			g.Expect(err).To(BeNil())

			// returns an error if file does not exists
			execErr := exec.Stream(remotecommand.StreamOptions{
				Stdout: os.Stdout,
				Stderr: os.Stderr,
				Tty:    false,
			})
			g.Expect(execErr).To(BeNil())

		})

		g.Expect(Kamel(t, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

func TestAlternativeImageInstallation(t *testing.T) {
	WithNewTestNamespace(t, func(g *WithT, ns string) {
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		g.Expect(KamelInstallWithID(t, operatorID, ns, "--olm=false", "--operator-image", "x/y:latest").Execute()).To(Succeed())
		g.Eventually(OperatorImage(t, ns)).Should(Equal("x/y:latest"))
	})
}

func TestKitMainInstallation(t *testing.T) {
	WithNewTestNamespace(t, func(g *WithT, ns string) {
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		g.Expect(KamelInstallWithID(t, operatorID, ns).Execute()).To(Succeed())
		g.Expect(Kamel(t, "kit", "create", "timer", "-d", "camel:timer", "-x", operatorID, "-n", ns).Execute()).To(Succeed())
		g.Eventually(Build(t, ns, "timer"), TestTimeoutMedium).ShouldNot(BeNil())
	})
}

func TestMavenRepositoryInstallation(t *testing.T) {
	WithNewTestNamespace(t, func(g *WithT, ns string) {
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		g.Expect(KamelInstallWithID(t, operatorID, ns, "--maven-repository", "https://my.repo.org/public/").Execute()).To(Succeed())
		configmapName := fmt.Sprintf("%s-maven-settings", operatorID)
		g.Eventually(Configmap(t, ns, configmapName)).Should(Not(BeNil()))
		g.Eventually(func() string {
			return Configmap(t, ns, configmapName)().Data["settings.xml"]
		}).Should(ContainSubstring("https://my.repo.org/public/"))
	})
}

/*
 * Expert mode where user does not want a registry configured
 * so the Platform will have an empty Registry structure
 */
func TestSkipRegistryInstallation(t *testing.T) {
	WithNewTestNamespace(t, func(g *WithT, ns string) {
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		g.Expect(KamelInstallWithID(t, operatorID, ns, "--skip-registry-setup").Execute()).To(Succeed())
		g.Eventually(Platform(t, ns)).ShouldNot(BeNil())
		g.Eventually(func() v1.RegistrySpec {
			return Platform(t, ns)().Spec.Build.Registry
		}, TestTimeoutMedium).Should(Equal(v1.RegistrySpec{}))
	})
}

type templateArgs struct {
	Version  string
	Platform string
}

func TestConsoleCliDownload(t *testing.T) {
	ocp, err := openshift.IsOpenShift(TestClient(t))
	require.NoError(t, err)

	ok, err := kubernetes.IsAPIResourceInstalled(TestClient(t), "console.openshift.io/v1", reflect.TypeOf(consolev1.ConsoleCLIDownload{}).Name())
	require.NoError(t, err)

	if !ocp || !ok {
		t.Skip("This test requires ConsoleCliDownload object which is available on OpenShift 4+ only.")
		return
	}

	name := GetEnvOrDefault("CAMEL_K_CONSOLE_CLI_DOWNLOAD_NAME", install.KamelCLIDownloadName)
	downloadUrlTemplate := GetEnvOrDefault("CAMEL_K_CONSOLE_CLI_DOWNLOAD_URL_TEMPLATE", "https://github.com/apache/camel-k/v2/releases/download/v{{ .Version}}/camel-k-client-{{ .Version}}-{{ .Platform}}-64bit.tar.gz")

	args := templateArgs{Version: defaults.Version}
	templt, err := template.New("downloadLink").Parse(downloadUrlTemplate)
	require.NoError(t, err)

	WithNewTestNamespace(t, func(g *WithT, ns string) {
		// make sure there is no preinstalled CliDownload resource
		cliDownload := ConsoleCLIDownload(t, name)()
		if cliDownload != nil {
			g.Expect(TestClient(t).Delete(TestContext, cliDownload)).To(Succeed())
		}

		operatorID := fmt.Sprintf("camel-k-%s", ns)
		g.Expect(KamelInstallWithID(t, operatorID, ns).Execute()).To(Succeed())
		g.Eventually(ConsoleCLIDownload(t, name), TestTimeoutMedium).Should(Not(BeNil()))

		cliDownload = ConsoleCLIDownload(t, name)()
		links := cliDownload.Spec.Links

		for _, link := range links {
			var buf bytes.Buffer
			if strings.Contains(link.Text, "for Linux") {
				args.Platform = "linux"
			} else if strings.Contains(link.Text, "for Mac") {
				args.Platform = "mac"
			} else if strings.Contains(link.Text, "for Windows") {
				args.Platform = "windows"
			}

			templt.Execute(&buf, args)
			g.Expect(link.Href).To(MatchRegexp(buf.String()))
		}
	})
}

func TestInstallSkipDefaultKameletsInstallation(t *testing.T) {
	WithNewTestNamespace(t, func(g *WithT, ns string) {
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		g.Expect(KamelInstallWithIDAndKameletCatalog(t, operatorID, ns, "--skip-default-kamelets-setup").Execute()).To(Succeed())
		g.Eventually(OperatorPod(t, ns)).ShouldNot(BeNil())
		g.Expect(KameletList(t, ns)()).Should(BeEmpty())
	})
}

func TestInstallDebugLogging(t *testing.T) {
	WithNewTestNamespace(t, func(g *WithT, ns string) {
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		g.Expect(KamelInstallWithID(t, operatorID, ns, "-z", "debug").Execute()).To(Succeed())

		g.Eventually(OperatorEnvVarValue(t, ns, "LOG_LEVEL"), TestTimeoutLong).Should(Equal("debug"))
	})
}
