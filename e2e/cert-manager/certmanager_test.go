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

package certmanager

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/certmanager"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestIngressCertManagerAutoDiscovery verifies that the Ingress trait, when
// tls-cert-manager-auto is enabled, discovers the self-signed ClusterIssuer
// installed by setup/setup.sh, annotates the generated Ingress accordingly,
// and that cert-manager actually issues a certificate into the derived
// "<service>-tls" secret for the configured host.
func TestIngressCertManagerAutoDiscovery(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		host := "cert-manager-it.example.com"
		integrationName := "platform-http-server"

		g.Expect(KamelRun(t, ctx, ns, "files/PlatformHttpServer.java",
			"-t", "ingress.enabled=true",
			"-t", "ingress.host="+host,
			"-t", "ingress.tls-cert-manager-auto=true",
		).Execute()).To(Succeed())

		g.Eventually(IntegrationConditionStatus(t, ctx, ns, integrationName,
			v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))

		// The Ingress trait should have auto-discovered the ClusterIssuer installed
		// by setup.sh and annotated the Ingress accordingly.
		g.Eventually(func() (string, error) {
			ingress, err := TestClient(t).NetworkingV1().Ingresses(ns).Get(ctx, integrationName, metav1.GetOptions{})
			if err != nil {
				return "", err
			}

			return ingress.Annotations[certmanager.AnnotationClusterIssuer], nil
		}, TestTimeoutShort).Should(Equal("selfsigned-cluster-issuer"))

		// cert-manager should pick up the annotation and populate the derived secret.
		g.Eventually(SecretByName(t, ctx, ns, integrationName+"-tls"), TestTimeoutMedium).ShouldNot(BeNil())

		certSecret := SecretByName(t, ctx, ns, integrationName+"-tls")()
		g.Expect(certSecret.Data).To(HaveKey("tls.crt"))
		g.Expect(certSecret.Data).To(HaveKey("tls.key"))

		block, _ := pem.Decode(certSecret.Data["tls.crt"])
		g.Expect(block).NotTo(BeNil())
		cert, err := x509.ParseCertificate(block.Bytes)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(cert.DNSNames).To(ContainElement(host))
		g.Expect(time.Now()).To(BeTemporally("<", cert.NotAfter))
	})
}
