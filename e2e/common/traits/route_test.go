// +build integration

// To enable compilation of this file in Goland, go to "Settings -> Go -> Vendoring & Build Tags -> Custom Tags" and add "knative"

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
	"github.com/stretchr/testify/assert"

	rand2 "math/rand"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/apache/camel-k/e2e/support"
	"github.com/apache/camel-k/pkg/util/openshift"
)

const(
	secretName = "test-certificate"
	integrationName = "platform-http-server"
)

var waitBeforeHttpRequest = TestTimeoutShort/2

type keyCertificatePair struct {
	Key []byte
	Certificate []byte
}

// self-signed certificate used when making the https request
var certPem []byte

// the e2e tests run on openshift3 in CI and there is no object to retrieve the base domain name
// to create a valid hostname to later have it validate when doing the https request
// as this a e2e test to validate the route object and not the certificate itself 
// if the base domain name cannot be retrieved from dns/cluster we can safely skip TLS verification
// if running on openshift4, there is the dns/cluster object to retrieve the base domain name, 
// then in this case the http client validates the TLS certificate
var skipClientTLSVerification = true

func TestRunRoutes(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		ocp, err := openshift.IsOpenShift(TestClient())
		if !ocp {
			t.Skip("This test requires route object which is available on OpenShift only.")
			return
		}
		assert.Nil(t, err)

		Expect(Kamel("install", "-n", ns, "--trait-profile=openshift").Execute()).To(Succeed())

		// create a test secret of type tls with certificates
		// this secret is used to setupt the route TLS object across diferent tests
		secret, err := createSecret(ns)
		assert.Nil(t, err)

		// they refer to the certificates create in the secret and are reused the different tests
		refKey := secretName + "/" + corev1.TLSPrivateKeyKey
		refCert := secretName + "/" + corev1.TLSCertKey

		// =============================
		// Insecure Route / No TLS
		// =============================
		t.Run("Route unsecure http works", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/PlatformHttpServer.java").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, integrationName), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			route := Route(ns, integrationName)
			Eventually(route, TestTimeoutMedium).ShouldNot(BeNil())
			// must wait a little time after route is created, before doing an http request, 
			// otherwise the route is unavailable and the http request will fail
			time.Sleep(waitBeforeHttpRequest)
			url := fmt.Sprintf("http://%s/hello?name=Simple", route().Spec.Host)
			response, err := httpRequest(t, url, false)
			assert.Nil(t, err)
			assert.Equal(t, "Hello Simple", response)
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		// =============================
		// TLS Route Edge
		// =============================

		t.Run("Route Edge https works", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/PlatformHttpServer.java", "-t", "route.tls-termination=edge").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, integrationName), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			route := Route(ns, integrationName)
			Eventually(route, TestTimeoutMedium).ShouldNot(BeNil())
			// must wait a little time after route is created, before an http request, 
			// otherwise the route is unavailable and the http request will fail
			time.Sleep(waitBeforeHttpRequest)
			url := fmt.Sprintf("https://%s/hello?name=TLS_Edge", route().Spec.Host)
			response, err := httpRequest(t, url, true)
			assert.Nil(t, err)
			assert.Equal(t, "Hello TLS_Edge", response)
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		// =============================
		// TLS Route Edge with custom certificate
		// =============================

		t.Run("Route Edge (custom certificate) https works", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/PlatformHttpServer.java",
				"-t", "route.tls-termination=edge",
				"-t", "route.tls-certificate-secret=" + refCert,
				"-t", "route.tls-key-secret=" + refKey,
			).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, integrationName), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			route := Route(ns, integrationName)
			Eventually(route, TestTimeoutMedium).ShouldNot(BeNil())
			// must wait a little time after route is created, before an http request, 
			// otherwise the route is unavailable and the http request will fail
			time.Sleep(waitBeforeHttpRequest)
			code := "TLS_EdgeCustomCertificate"
			url := fmt.Sprintf("https://%s/hello?name=%s", route().Spec.Host, code)
			response, err := httpRequest(t, url, true)
			assert.Nil(t, err)
			assert.Equal(t, "Hello " + code, response)
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		// =============================
		// TLS Route Passthrough
		// =============================

		t.Run("Route passthrough https works", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/PlatformHttpServer.java",
				// the --resource mounts the certificates inside secret as files in the integration pod
				"--resource", "secret:" + secretName + "@/etc/ssl/" + secretName,
				// quarkus platform-http uses these two properties to setup the HTTP endpoint with TLS support
				"-p", "quarkus.http.ssl.certificate.file=/etc/ssl/" + secretName + "/tls.crt",
				"-p", "quarkus.http.ssl.certificate.key-file=/etc/ssl/" + secretName + "/tls.key",
				"-t", "route.tls-termination=passthrough",
				"-t", "container.port=8443",
			).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, integrationName), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			route := Route(ns, integrationName)
			Eventually(route, TestTimeoutMedium).ShouldNot(BeNil())
			// must wait a little time after route is created, before an http request, 
			// otherwise the route is unavailable and the http request will fail
			time.Sleep(waitBeforeHttpRequest)
			code := "TLS_Passthrough"
			url := fmt.Sprintf("https://%s/hello?name=%s", route().Spec.Host, code)
			response, err := httpRequest(t, url, true)
			assert.Nil(t, err)
			assert.Equal(t, "Hello " + code, response)
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		// =============================
		// TLS Route Reencrypt
		// =============================

		t.Run("Route Reencrypt https works", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/PlatformHttpServer.java",
				// the --resource mounts the certificates inside secret as files in the integration pod
				"--resource", "secret:" + secretName + "@/etc/ssl/" + secretName,
				// quarkus platform-http uses these two properties to setup the HTTP endpoint with TLS support
				"-p", "quarkus.http.ssl.certificate.file=/etc/ssl/" + secretName + "/tls.crt",
				"-p", "quarkus.http.ssl.certificate.key-file=/etc/ssl/" + secretName + "/tls.key",
				"-t", "route.tls-termination=reencrypt",
				// the destination CA certificate which the route service uses to validate the HTTP endpoint TLS certificate
				"-t", "route.tls-destination-ca-certificate-secret=" + refCert,
				"-t", "route.tls-certificate-secret=" + refCert,
				"-t", "route.tls-key-secret=" + refKey,
				"-t", "container.port=8443",
			).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, integrationName), TestTimeoutMedium).Should(Equal(corev1.PodRunning))

			route := Route(ns, integrationName)
			Eventually(route, TestTimeoutMedium).ShouldNot(BeNil())
			// must wait a little time after route is created, before an http request, 
			// otherwise the route is unavailable and the http request will fail
			time.Sleep(waitBeforeHttpRequest)
			code := "TLS_Reencrypt"
			url := fmt.Sprintf("https://%s/hello?name=%s", route().Spec.Host, code)
			response, err := httpRequest(t, url, true)
			assert.Nil(t, err)
			assert.Equal(t, "Hello " + code, response)
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})
		Expect(TestClient().Delete(TestContext, &secret)).To(Succeed())
	})
}

func httpRequest(t *testing.T, url string, tlsEnabled bool) (string, error) {
	var client http.Client
	if tlsEnabled {
		var transCfg http.Transport
		if skipClientTLSVerification {
			transCfg = http.Transport{
				TLSClientConfig: &tls.Config {
					InsecureSkipVerify: true,
				},
			}
		} else {
			certPool := x509.NewCertPool()
			certPool.AppendCertsFromPEM(certPem)
			transCfg = http.Transport{
				TLSClientConfig: &tls.Config {
					RootCAs: certPool,
				},
			}
		}
		client = http.Client{Transport: &transCfg}
	} else {
		client = http.Client{}
	}
	response, err := client.Get(url)
	defer func() {
		if response != nil {
			_ = response.Body.Close()
		}
	}()
	if err != nil {
		fmt.Printf("Error making HTTP request. %s\n", err)
		return "", err
	}
	assert.Nil(t, err)
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(response.Body)
	if err != nil {
		fmt.Printf("Error reading the HTTP response. %s\n", err)
		return "", err
	}
	assert.Nil(t, err)
	return buf.String(), nil
}

func createSecret(ns string) (corev1.Secret, error) {
	keyCertPair := generateSampleKeyAndCertificate(ns)
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
			corev1.TLSCertKey: keyCertPair.Certificate,
		},
	}
	return sec, TestClient().Create(TestContext, &sec)
}

func generateSampleKeyAndCertificate(ns string) keyCertificatePair {
	serialNumber := big.NewInt(rand2.Int63())
	domainName, err := ClusterDomainName()
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
		IsCA: 				   true,
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

	return keyCertificatePair {
		Key: privateKeyPem,
		Certificate: certPem,
	}
}
