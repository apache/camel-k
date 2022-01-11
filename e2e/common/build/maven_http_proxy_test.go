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

package build

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	rand2 "math/rand"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func TestMavenProxy(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		hostname := fmt.Sprintf("%s.%s.svc", "proxy", ns)
		tlsMountPath := "/etc/tls/private"

		// Generate the TLS certificate
		serialNumber := big.NewInt(rand2.Int63())
		cert := &x509.Certificate{
			SerialNumber: serialNumber,
			Subject: pkix.Name{
				Organization: []string{"Camel K test"},
			},
			DNSNames:              []string{hostname},
			NotBefore:             time.Now(),
			NotAfter:              time.Now().AddDate(1, 0, 0),
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
			BasicConstraintsValid: true,
		}

		// generate the certificate private key
		certPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		Expect(err).To(BeNil())

		privateKeyBytes := x509.MarshalPKCS1PrivateKey(certPrivateKey)
		// encode for storing into a Secret
		privateKeyPem := pem.EncodeToMemory(
			&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: privateKeyBytes,
			},
		)
		certBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &certPrivateKey.PublicKey, certPrivateKey)
		Expect(err).To(BeNil())

		// encode for storing into a Secret
		certPem := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certBytes,
		})

		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      "tls-secret",
			},
			Type: corev1.SecretTypeTLS,
			Data: map[string][]byte{
				corev1.TLSCertKey:       certPem,
				corev1.TLSPrivateKeyKey: privateKeyPem,
			},
		}
		Expect(TestClient().Create(TestContext, secret)).To(Succeed())

		// HTTPD configuration
		config := &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      "httpd-config",
			},
			Data: map[string]string{
				"httpd.conf": fmt.Sprintf(`
ServerRoot "/etc/httpd

PidFile /var/run/httpd/httpd.pid"

LoadModule mpm_event_module /usr/local/apache2/modules/mod_mpm_event.so
LoadModule authn_core_module /usr/local/apache2/modules/mod_authn_core.so
LoadModule authz_core_module /usr/local/apache2/modules/mod_authz_core.so
LoadModule proxy_module /usr/local/apache2/modules/mod_proxy.so
LoadModule proxy_http_module /usr/local/apache2/modules/mod_proxy_http.so
LoadModule proxy_connect_module /usr/local/apache2/modules/mod_proxy_connect.so
LoadModule headers_module /usr/local/apache2/modules/mod_headers.so
LoadModule setenvif_module /usr/local/apache2/modules/mod_setenvif.so
LoadModule version_module /usr/local/apache2/modules/mod_version.so
LoadModule log_config_module /usr/local/apache2/modules/mod_log_config.so
LoadModule env_module /usr/local/apache2/modules/mod_env.so
LoadModule unixd_module /usr/local/apache2/modules/mod_unixd.so
LoadModule status_module /usr/local/apache2/modules/mod_status.so
LoadModule autoindex_module /usr/local/apache2/modules/mod_autoindex.so
LoadModule ssl_module /usr/local/apache2/modules/mod_ssl.so

Mutex posixsem

LogFormat "%%h %%l %%u %%t \"%%r\" %%>s %%b" common
CustomLog /dev/stdout common
ErrorLog /dev/stderr

LogLevel warn

Listen 8080
Listen 8443

ServerName %s

ProxyRequests On
ProxyVia Off

<VirtualHost *:8443>
  SSLEngine on

  SSLCertificateFile "%s/%s"
  SSLCertificateKeyFile "%s/%s"

  AllowEncodedSlashes NoDecode
</VirtualHost>
`,
					hostname, tlsMountPath, corev1.TLSCertKey, tlsMountPath, corev1.TLSPrivateKeyKey,
				),
			},
		}
		Expect(TestClient().Create(TestContext, config)).To(Succeed())

		// Deploy HTTPD
		// $ curl --proxy-cacert ca.crt --proxy https://proxy.http-proxy.svc:443 https://www.google.com
		// https://github.com/curl/curl/pull/1127
		deployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: appsv1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      "proxy",
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "proxy",
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "proxy",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:    "httpd",
								Image:   "httpd:2.4.46",
								Command: []string{"httpd", "-f", "/etc/httpd/httpd.conf", "-DFOREGROUND"},
								Ports: []corev1.ContainerPort{
									{
										Name:          "http",
										ContainerPort: 8080,
									},
									{
										Name:          "https",
										ContainerPort: 8443,
									},
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "tls",
										MountPath: tlsMountPath,
										ReadOnly:  true,
									},
									{
										Name:      "httpd-conf",
										MountPath: "/etc/httpd",
										ReadOnly:  true,
									},
									{
										Name:      "httpd-run",
										MountPath: "/var/run/httpd",
									},
								},
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: "tls",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										SecretName: secret.Name,
									},
								},
							},
							{
								Name: "httpd-conf",
								VolumeSource: corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: config.Name,
										},
									},
								},
							},
							{
								Name: "httpd-run",
								VolumeSource: corev1.VolumeSource{
									EmptyDir: &corev1.EmptyDirVolumeSource{},
								},
							},
						},
					},
				},
			},
		}
		Expect(TestClient().Create(TestContext, deployment)).To(Succeed())

		service := &corev1.Service{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Service",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      deployment.Name,
			},
			Spec: corev1.ServiceSpec{
				Selector: deployment.Spec.Template.Labels,
				Ports: []corev1.ServicePort{
					{
						Name:       "http",
						Port:       80,
						TargetPort: intstr.FromString("http"),
					},
					{
						Name:       "https",
						Port:       443,
						TargetPort: intstr.FromString("https"),
					},
				},
			},
		}
		Expect(TestClient().Create(TestContext, service)).To(Succeed())

		// Wait for the Deployment to become ready
		Eventually(Deployment(ns, deployment.Name), TestTimeoutMedium).Should(PointTo(MatchFields(IgnoreExtras,
			Fields{
				"Status": MatchFields(IgnoreExtras,
					Fields{
						"ReadyReplicas": Equal(int32(1)),
					}),
			}),
		))

		svc := Service("default", "kubernetes")()
		Expect(svc).NotTo(BeNil())

		// It may be needed to populate the values from the cluster, machine and service network CIDRs
		noProxy := []string{
			".cluster.local",
			".svc",
			"localhost",
		}
		noProxy = append(noProxy, svc.Spec.ClusterIPs...)

		// Install Camel K with the HTTP proxy
		Expect(Kamel("install", "-n", ns,
			"--operator-env-vars", fmt.Sprintf("HTTP_PROXY=http://%s", hostname),
			// TODO: enable TLS for the HTTPS proxy when Maven supports it
			// "--operator-env-vars", fmt.Sprintf("HTTPS_PROXY=https://%s", hostname),
			// "--maven-ca-secret", secret.Name+"/"+corev1.TLSCertKey,
			"--operator-env-vars", "NO_PROXY="+strings.Join(noProxy, ","),
		).Execute()).To(Succeed())

		Eventually(PlatformPhase(ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		// Run the Integration
		name := "java"
		Expect(Kamel("run", "-n", ns, "files/Java.java", "--name", name).Execute()).To(Succeed())

		Eventually(IntegrationPodPhase(ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		proxies := corev1.PodList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
		}
		err = TestClient().List(TestContext, &proxies,
			ctrl.InNamespace(ns),
			ctrl.MatchingLabels(deployment.Spec.Selector.MatchLabels),
		)
		Expect(err).To(Succeed())
		Expect(proxies.Items).To(HaveLen(1))

		logs := Logs(ns, proxies.Items[0].Name, corev1.PodLogOptions{})()
		Expect(logs).NotTo(BeEmpty())
		Expect(logs).To(ContainSubstring("\"CONNECT repo.maven.apache.org:443 HTTP/1.1\" 200"))

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		Expect(TestClient().Delete(TestContext, deployment)).To(Succeed())
		Expect(TestClient().Delete(TestContext, service)).To(Succeed())
		Expect(TestClient().Delete(TestContext, secret)).To(Succeed())
		Expect(TestClient().Delete(TestContext, config)).To(Succeed())
	})
}
