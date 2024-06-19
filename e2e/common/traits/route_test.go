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

package traits

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	. "github.com/apache/camel-k/v2/e2e/support"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/openshift"
)

const (
	secretName      = "test-certificate"
	integrationName = "platform-http-server"
)

var waitBeforeHttpRequest = TestTimeoutShort / 2

type keyCertificatePair struct {
	Key         []byte
	Certificate []byte
}

// self-signed certificate used when making the https request
var certPem []byte

// the e2e tests run on OpenShift 3 and there is no object to retrieve the base domain name
// to create a valid hostname to later have it validated when doing the HTTPS request.
// As this e2e test validates the route object and not the certificate itself,
// if the base domain name cannot be retrieved from dns/cluster, we can safely skip TLS verification.
// if running on openshift4, there is the dns/cluster object to retrieve the base domain name,
// then in this case the HTTP client validates the TLS certificate.
var skipClientTLSVerification = true

func TestRunRoutes(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		ocp, err := openshift.IsOpenShift(TestClient(t))
		if !ocp {
			t.Skip("This test requires route object which is available on OpenShift only.")
			return
		}
		require.NoError(t, err)

		// create a test secret of type tls with certificates
		// this secret is used to setupt the route TLS object across diferent tests
		secret, err := createSecret(t, ctx, ns)
		require.NoError(t, err)

		// they refer to the certificates create in the secret and are reused the different tests
		refKey := secretName + "/" + corev1.TLSPrivateKeyKey
		refCert := secretName + "/" + corev1.TLSCertKey

		// =============================
		// Insecure Route / No TLS
		// =============================
		t.Run("Route unsecure http works", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/PlatformHttpServer.java").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, integrationName), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(Route(t, ctx, ns, integrationName), TestTimeoutMedium).ShouldNot(BeNil())
			// must wait a little time after route is created, before doing an http request,
			// otherwise the route is unavailable and the http request will fail
			time.Sleep(waitBeforeHttpRequest)
			route := Route(t, ctx, ns, integrationName)
			url := fmt.Sprintf("http://%s/hello?name=Simple", route().Spec.Host)
			g.Eventually(httpRequest(url, false), TestTimeoutShort).Should(Equal("Hello Simple"))
			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).Should(BeNil())
			g.Eventually(Route(t, ctx, ns, integrationName), TestTimeoutMedium).Should(BeNil())
		})

		// =============================
		// TLS Route Edge
		// =============================
		t.Run("Route Edge https works", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/PlatformHttpServer.java", "-t", "route.tls-termination=edge").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, integrationName), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(Route(t, ctx, ns, integrationName), TestTimeoutMedium).ShouldNot(BeNil())
			// must wait a little time after route is created, before an http request,
			// otherwise the route is unavailable and the http request will fail
			time.Sleep(waitBeforeHttpRequest)
			route := Route(t, ctx, ns, integrationName)
			url := fmt.Sprintf("https://%s/hello?name=TLS_Edge", route().Spec.Host)
			g.Eventually(httpRequest(url, true), TestTimeoutShort).Should(Equal("Hello TLS_Edge"))
			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).Should(BeNil())
			g.Eventually(Route(t, ctx, ns, integrationName), TestTimeoutMedium).Should(BeNil())
		})

		// =============================
		// TLS Route Edge with custom certificate
		// =============================
		t.Run("Route Edge (custom certificate) https works", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/PlatformHttpServer.java", "-t", "route.tls-termination=edge", "-t", "route.tls-certificate-secret="+refCert, "-t", "route.tls-key-secret="+refKey).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, integrationName), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(Route(t, ctx, ns, integrationName), TestTimeoutMedium).ShouldNot(BeNil())
			// must wait a little time after route is created, before an http request,
			// otherwise the route is unavailable and the http request will fail
			time.Sleep(waitBeforeHttpRequest)
			route := Route(t, ctx, ns, integrationName)
			code := "TLS_EdgeCustomCertificate"
			url := fmt.Sprintf("https://%s/hello?name=%s", route().Spec.Host, code)
			g.Eventually(httpRequest(url, true), TestTimeoutShort).Should(Equal("Hello " + code))
			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).Should(BeNil())
			g.Eventually(Route(t, ctx, ns, integrationName), TestTimeoutMedium).Should(BeNil())
		})

		// =============================
		// TLS Route Passthrough
		// =============================
		t.Run("Route passthrough https works", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/PlatformHttpServer.java", "--resource", "secret:"+secretName+"@/etc/ssl/"+secretName, "-p", "quarkus.http.ssl.certificate.files=/etc/ssl/"+secretName+"/tls.crt", "-p", "quarkus.http.ssl.certificate.key-files=/etc/ssl/"+secretName+"/tls.key", "-t", "route.tls-termination=passthrough", "-t", "container.port=8443").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, integrationName), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(Route(t, ctx, ns, integrationName), TestTimeoutMedium).ShouldNot(BeNil())
			// must wait a little time after route is created, before an http request,
			// otherwise the route is unavailable and the http request will fail
			time.Sleep(waitBeforeHttpRequest)
			route := Route(t, ctx, ns, integrationName)
			code := "TLS_Passthrough"
			url := fmt.Sprintf("https://%s/hello?name=%s", route().Spec.Host, code)
			g.Eventually(httpRequest(url, true), TestTimeoutShort).Should(Equal("Hello " + code))
			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).Should(BeNil())
			g.Eventually(Route(t, ctx, ns, integrationName), TestTimeoutMedium).Should(BeNil())
		})

		// =============================
		// TLS Route Reencrypt
		// =============================
		t.Run("Route Reencrypt https works", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/PlatformHttpServer.java", "--resource", "secret:"+secretName+"@/etc/ssl/"+secretName, "-p", "quarkus.http.ssl.certificate.files=/etc/ssl/"+secretName+"/tls.crt", "-p", "quarkus.http.ssl.certificate.key-files=/etc/ssl/"+secretName+"/tls.key", "-t", "route.tls-termination=reencrypt", "-t", "route.tls-destination-ca-certificate-secret="+refCert, "-t", "route.tls-certificate-secret="+refCert, "-t", "route.tls-key-secret="+refKey, "-t", "container.port=8443").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, integrationName), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(Route(t, ctx, ns, integrationName), TestTimeoutMedium).ShouldNot(BeNil())
			// must wait a little time after route is created, before an http request,
			// otherwise the route is unavailable and the http request will fail
			route := Route(t, ctx, ns, integrationName)
			time.Sleep(waitBeforeHttpRequest)
			code := "TLS_Reencrypt"
			url := fmt.Sprintf("https://%s/hello?name=%s", route().Spec.Host, code)
			g.Eventually(httpRequest(url, true), TestTimeoutShort).Should(Equal("Hello " + code))
			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).Should(BeNil())
			g.Eventually(Route(t, ctx, ns, integrationName), TestTimeoutMedium).Should(BeNil())
		})

		t.Run("Route annotations added", func(t *testing.T) {
			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).Should(BeNil())
			g.Expect(KamelRun(t, ctx, ns, "files/PlatformHttpServer.java", "-t", "route.annotations.'haproxy.router.openshift.io/balance'=roundrobin").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, integrationName), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(Route(t, ctx, ns, integrationName), TestTimeoutMedium).ShouldNot(BeNil())
			// must wait a little time after route is created, before an http request,
			// otherwise the route is unavailable and the http request will fail
			time.Sleep(waitBeforeHttpRequest)
			route := RouteFull(t, ctx, ns, integrationName)()
			var annotations = route.ObjectMeta.Annotations
			g.Expect(annotations["haproxy.router.openshift.io/balance"]).To(Equal("roundrobin"))

			// check integration schema does not contains unwanted default trait value.
			g.Eventually(UnstructuredIntegration(t, ctx, ns, integrationName)).ShouldNot(BeNil())
			unstructuredIntegration := UnstructuredIntegration(t, ctx, ns, integrationName)()
			routeTrait, _, _ := unstructured.NestedMap(unstructuredIntegration.Object, "spec", "traits", "route")
			g.Expect(routeTrait).ToNot(BeNil())
			g.Expect(len(routeTrait)).To(Equal(1))
			g.Expect(routeTrait["annotations"]).ToNot(BeNil())
		})
		g.Expect(TestClient(t).Delete(ctx, &secret)).To(Succeed())
	})
}

func httpRequest(url string, tlsEnabled bool) func() (string, error) {
	return func() (string, error) {
		client, err := httpClient(tlsEnabled, 3*time.Second)
		if err != nil {
			return "", err
		}
		response, err := client.Get(url)
		if err != nil {
			return "", err
		}
		defer response.Body.Close()

		buf := new(bytes.Buffer)
		_, err = buf.ReadFrom(response.Body)
		if err != nil {
			return "", err
		}
		return buf.String(), nil
	}
}

func httpClient(tlsEnabled bool, timeout time.Duration) (*http.Client, error) {
	var client http.Client
	if tlsEnabled {
		var transCfg http.Transport
		if skipClientTLSVerification {
			transCfg = http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			}
		} else {
			certPool := x509.NewCertPool()
			certPool.AppendCertsFromPEM(certPem)
			transCfg = http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: certPool,
				},
			}
		}
		client = http.Client{Transport: &transCfg}
	} else {
		client = http.Client{}
	}
	client.Timeout = timeout
	return &client, nil
}

func createSecret(t *testing.T, ctx context.Context, ns string) (corev1.Secret, error) {
	keyCertPair := generateSampleKeyAndCertificate(t, ctx, ns)
	sec := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      secretName,
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSPrivateKeyKey: keyCertPair.Key,
			corev1.TLSCertKey:       keyCertPair.Certificate,
		},
	}
	return sec, TestClient(t).Create(ctx, &sec)
}

func generateSampleKeyAndCertificate(t *testing.T, ctx context.Context, ns string) keyCertificatePair {
	serialNumber := big.NewInt(util.RandomInt63())
	domainName, err := ClusterDomainName(t, ctx)
	if err != nil {
		fmt.Printf("Error retrieving cluster domain object, then the http client request will skip TLS validation: %s\n", err)
		skipClientTLSVerification = true
	}
	var dnsHostname string
	if len(domainName) > 0 {
		dnsHostname = integrationName + "-" + ns + "." + domainName
	} else {
		dnsHostname = integrationName + "-" + ns
	}
	x509Certificate := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Camel K test"},
		},
		IsCA:                  true,
		DNSNames:              []string{dnsHostname},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// generate the private key
	certPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Printf("Error generating private key: %s\n", err)
	}

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(certPrivateKey)
	// encode for storing into secret
	privateKeyPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privateKeyBytes,
		},
	)
	certBytes, err := x509.CreateCertificate(rand.Reader, &x509Certificate, &x509Certificate, &certPrivateKey.PublicKey, certPrivateKey)
	if err != nil {
		fmt.Printf("Error generating certificate: %s\n", err)
	}

	// encode for storing into secret
	certPem = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	return keyCertificatePair{
		Key:         privateKeyPem,
		Certificate: certPem,
	}
}
