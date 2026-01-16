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

package common

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestJVMTrait(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		// Store a configmap holding a jar
		var cmData = make(map[string][]byte)
		// We calculate the expected content
		source, err := os.ReadFile("./files/jvm/sample-1.0.jar")
		require.NoError(t, err)
		cmData["sample-1.0.jar"] = source
		err = CreateBinaryConfigmap(t, ctx, ns, "my-deps", cmData)
		require.NoError(t, err)

		t.Run("JVM trait classpath", func(t *testing.T) {
			name := RandomizedSuffixName("classpath")
			g.Expect(KamelRun(t, ctx, ns,
				"./files/jvm/Classpath.java",
				"--name", name,
				"--resource", "configmap:my-deps",
				"-t", "jvm.classpath=/etc/camel/resources.d/_configmaps/my-deps/sample-1.0.jar").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Hello World!"))
		})

		t.Run("JVM trait classpath on deprecated path", func(t *testing.T) {
			name := RandomizedSuffixName("classpath")
			g.Expect(KamelRun(t, ctx, ns,
				"./files/jvm/Classpath.java",
				"--name", name,
				"-t", "mount.resources=configmap:my-deps/sample-1.0.jar@/etc/camel/resources",
				"-t", "jvm.classpath=/etc/camel/resources/my-deps/sample-1.0.jar").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Hello World!"))
		})

		t.Run("JVM trait classpath on specific classpath", func(t *testing.T) {
			name := RandomizedSuffixName("classpath")
			g.Expect(KamelRun(t, ctx, ns,
				"./files/jvm/Classpath.java",
				"--name", name,
				"-t", "mount.resources=configmap:my-deps/sample-1.0.jar@/etc/other/resources",
				"-t", "jvm.classpath=/etc/other/resources/my-deps/sample-1.0.jar").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Hello World!"))
		})

		t.Run("JVM trait multiple CA certs", func(t *testing.T) {
			// Test the new ca-certificates field with multiple certificates, each with its own password
			cert1Pem, err := generateSelfSignedCert()
			require.NoError(t, err)
			cert2Pem, err := generateSelfSignedCert()
			require.NoError(t, err)

			// Create secrets with both cert and password together
			caCert1Data := make(map[string]string)
			caCert1Data["ca.crt"] = string(cert1Pem)
			caCert1Data["password"] = "test-password-1"
			err = CreatePlainTextSecret(t, ctx, ns, "test-ca-cert-1", caCert1Data)
			require.NoError(t, err)

			caCert2Data := make(map[string]string)
			caCert2Data["ca.crt"] = string(cert2Pem)
			caCert2Data["password"] = "test-password-2"
			err = CreatePlainTextSecret(t, ctx, ns, "test-ca-cert-2", caCert2Data)
			require.NoError(t, err)

			name := RandomizedSuffixName("multicacert")
			g.Expect(KamelRun(t, ctx, ns,
				"./files/Java.java",
				"--name", name,
				"-t", "mount.configs=secret:test-ca-cert-1",
				"-t", "mount.configs=secret:test-ca-cert-2",
				// Using new ca-certificates field: each certificate with its own password path
				"-t", "jvm.ca-certificates[0].cert-path=/etc/camel/conf.d/_secrets/test-ca-cert-1/ca.crt",
				"-t", "jvm.ca-certificates[0].password-path=/etc/camel/conf.d/_secrets/test-ca-cert-1/password",
				"-t", "jvm.ca-certificates[1].cert-path=/etc/camel/conf.d/_secrets/test-ca-cert-2/ca.crt",
				"-t", "jvm.ca-certificates[1].password-path=/etc/camel/conf.d/_secrets/test-ca-cert-2/password",
			).Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong*2).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))

			pod := IntegrationPod(t, ctx, ns, name)()
			g.Expect(pod).NotTo(BeNil())
			initContainerNames := make([]string, 0)
			for _, c := range pod.Spec.InitContainers {
				initContainerNames = append(initContainerNames, c.Name)
			}
			g.Expect(initContainerNames).To(ContainElement("generate-truststore"))
		})

		t.Run("JVM trait CA cert with base truststore", func(t *testing.T) {
			// Test the new base-truststore field (replaces ca-cert-use-system-truststore)
			certPem, err := generateSelfSignedCert()
			require.NoError(t, err)

			// Create secret with cert and password
			caCertData := make(map[string]string)
			caCertData["ca.crt"] = string(certPem)
			caCertData["password"] = "test-password-456"
			err = CreatePlainTextSecret(t, ctx, ns, "test-ca-sys", caCertData)
			require.NoError(t, err)

			// Create secret for base truststore password (JDK cacerts uses "changeit")
			baseTsPassData := make(map[string]string)
			baseTsPassData["password"] = "changeit"
			err = CreatePlainTextSecret(t, ctx, ns, "base-ts-password", baseTsPassData)
			require.NoError(t, err)

			name := RandomizedSuffixName("syscacert")
			g.Expect(KamelRun(t, ctx, ns,
				"./files/Java.java",
				"--name", name,
				"-t", "mount.configs=secret:test-ca-sys",
				"-t", "mount.configs=secret:base-ts-password",
				// Using new base-truststore field with JDK cacerts as base
				"-t", "jvm.base-truststore.truststore-path=/opt/java/openjdk/lib/security/cacerts",
				"-t", "jvm.base-truststore.password-path=/etc/camel/conf.d/_secrets/base-ts-password/password",
				"-t", "jvm.ca-certificates[0].cert-path=/etc/camel/conf.d/_secrets/test-ca-sys/ca.crt",
				"-t", "jvm.ca-certificates[0].password-path=/etc/camel/conf.d/_secrets/test-ca-sys/password",
			).Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))

			pod := IntegrationPod(t, ctx, ns, name)()
			g.Expect(pod).NotTo(BeNil())
			initContainerNames := make([]string, 0)
			for _, c := range pod.Spec.InitContainers {
				initContainerNames = append(initContainerNames, c.Name)
			}
			g.Expect(initContainerNames).To(ContainElement("generate-truststore"))
		})

		t.Run("JVM trait CA cert single certificate", func(t *testing.T) {
			// Test single certificate with the new ca-certificates field
			certPem, err := generateSelfSignedCert()
			require.NoError(t, err)

			// Create secret with both cert and password
			caCertData := make(map[string]string)
			caCertData["ca.crt"] = string(certPem)
			caCertData["password"] = "changeit"
			err = CreatePlainTextSecret(t, ctx, ns, "test-ca-single", caCertData)
			require.NoError(t, err)

			name := RandomizedSuffixName("singlecert")
			g.Expect(KamelRun(t, ctx, ns,
				"./files/Java.java",
				"--name", name,
				"-t", "mount.configs=secret:test-ca-single",
				// Using new ca-certificates field with explicit password
				"-t", "jvm.ca-certificates[0].cert-path=/etc/camel/conf.d/_secrets/test-ca-single/ca.crt",
				"-t", "jvm.ca-certificates[0].password-path=/etc/camel/conf.d/_secrets/test-ca-single/password",
			).Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))

			pod := IntegrationPod(t, ctx, ns, name)()
			g.Expect(pod).NotTo(BeNil())
			initContainerNames := make([]string, 0)
			for _, c := range pod.Spec.InitContainers {
				initContainerNames = append(initContainerNames, c.Name)
			}
			g.Expect(initContainerNames).To(ContainElement("generate-truststore"))
		})
	})
}

// Helper to generate a self-signed certificate for testing
func generateSelfSignedCert() ([]byte, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Co"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes}), nil
}
