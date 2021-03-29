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
	"testing"
	"time"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	. "github.com/apache/camel-k/e2e/support"
)

func TestMavenCASecret(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		hostname := fmt.Sprintf("%s.%s.svc", "nexus", ns)
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

		// generate certPem private key
		certPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		Expect(err).To(BeNil())

		privateKeyBytes := x509.MarshalPKCS1PrivateKey(certPrivateKey)
		// encode for storing into secret
		privateKeyPem := pem.EncodeToMemory(
			&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: privateKeyBytes,
			},
		)
		certBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &certPrivateKey.PublicKey, certPrivateKey)
		Expect(err).To(BeNil())

		// encode for storing into secret
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
LoadModule headers_module /usr/local/apache2/modules/mod_headers.so
LoadModule setenvif_module /usr/local/apache2/modules/mod_setenvif.so
LoadModule version_module /usr/local/apache2/modules/mod_version.so
LoadModule log_config_module /usr/local/apache2/modules/mod_log_config.so
LoadModule env_module /usr/local/apache2/modules/mod_env.so
LoadModule unixd_module /usr/local/apache2/modules/mod_unixd.so
LoadModule status_module /usr/local/apache2/modules/mod_status.so
LoadModule autoindex_module /usr/local/apache2/modules/mod_autoindex.so
LoadModule ssl_module /usr/local/apache2/modules/mod_ssl.so

ErrorLog /proc/self/fd/2

LogLevel warn

Listen 8443

ProxyRequests Off
ProxyPreserveHost On

<Directory />
  Options FollowSymLinks
  AllowOverride All
  Require all granted
</Directory>

<VirtualHost *:8443>
  SSLEngine on

  SSLCertificateFile "%s/%s"
  SSLCertificateKeyFile "%s/%s"

  AllowEncodedSlashes NoDecode

  ServerName %s
  ProxyPass / http://localhost:8081/ nocanon
  ProxyPassReverse / http://localhost:8081/
  RequestHeader set X-Forwarded-Proto "https"
</VirtualHost>
`,
					tlsMountPath, corev1.TLSCertKey, tlsMountPath, corev1.TLSPrivateKeyKey, hostname,
				),
			},
		}
		Expect(TestClient().Create(TestContext, config)).To(Succeed())

		// Deploy Nexus
		// https://help.sonatype.com/repomanager3/installation/run-behind-a-reverse-proxy
		deployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: appsv1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      "nexus",
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"camel-k": "maven-test-nexus",
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"camel-k": "maven-test-nexus",
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
							{
								Name:  "nexus",
								Image: "sonatype/nexus3:3.30.0",
								Ports: []corev1.ContainerPort{
									{
										Name:          "nexus",
										ContainerPort: 8081,
									},
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "nexus",
										MountPath: "/nexus-data",
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
							{
								Name: "nexus",
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
						Name:       "https",
						Port:       443,
						TargetPort: intstr.FromString("https"),
					},
				},
			},
		}
		Expect(TestClient().Create(TestContext, service)).To(Succeed())

		// Wait for the Deployment to become ready
		Eventually(Deployment(ns, deployment.Name)).Should(PointTo(MatchFields(IgnoreExtras,
			Fields{
				"Status": MatchFields(IgnoreExtras,
					Fields{
						"ReadyReplicas": Equal(int32(1)),
					}),
			}),
		))

		// Install Camel K with the Maven CA secret
		Expect(Kamel("install", "-n", ns,
			"--maven-repository", fmt.Sprintf(`https://%s/repository/maven-public/@id=central@snapshots`, hostname),
			"--maven-ca-secret", secret.Name+"/"+corev1.TLSCertKey,
		).Execute()).To(Succeed())

		// Run the Integration
		name := "java"
		Expect(Kamel("run", "-n", ns, "files/Java.java",
			"--name", name,
		).Execute()).To(Succeed())

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
