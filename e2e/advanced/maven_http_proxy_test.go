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

package advanced

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/envvar"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

var httpdTlsMountPath = "/etc/tls/private"

func TestMavenProxy(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		hostname := fmt.Sprintf("%s.%s.svc", "proxy", ns)
		// Generate the TLS certificate
		serialNumber := big.NewInt(util.RandomInt63())
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
		g.Expect(err).To(BeNil())

		privateKeyBytes := x509.MarshalPKCS1PrivateKey(certPrivateKey)
		// encode for storing into a Secret
		privateKeyPem := pem.EncodeToMemory(
			&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: privateKeyBytes,
			},
		)
		certBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &certPrivateKey.PublicKey, certPrivateKey)
		g.Expect(err).To(BeNil())

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
		g.Expect(TestClient(t).Create(ctx, secret)).To(Succeed())
		// HTTPD ConfigMap
		config := newHTTPDConfigMap(ns, hostname)
		g.Expect(TestClient(t).Create(ctx, config)).To(Succeed())
		// HTTPD Deployment
		deployment := newHTTPDDeployment(ns, config.Name, secret.Name)
		g.Expect(TestClient(t).Create(ctx, deployment)).To(Succeed())
		service := newHTTPDService(deployment)
		g.Expect(TestClient(t).Create(ctx, service)).To(Succeed())

		// Wait for the Deployment to become ready
		g.Eventually(Deployment(t, ctx, ns, deployment.Name), TestTimeoutMedium).Should(PointTo(MatchFields(IgnoreExtras,
			Fields{
				"Status": MatchFields(IgnoreExtras,
					Fields{
						"ReadyReplicas": Equal(int32(1)),
					}),
			}),
		))

		svc := Service(t, ctx, TestDefaultNamespace, "kubernetes")()
		g.Expect(svc).NotTo(BeNil())

		// It may be needed to populate the values from the cluster, machine and service network CIDRs
		noProxy := []string{
			".cluster.local",
			".svc",
			"localhost",
		}
		noProxy = append(noProxy, svc.Spec.ClusterIPs...)

		InstallOperator(t, g, ns)

		// Check that operator pod has env_vars
		g.Eventually(OperatorPodHas(t, ctx, ns, func(op *corev1.Pod) bool {
			if envVar := envvar.Get(op.Spec.Containers[0].Env, "HTTP_PROXY"); envVar != nil {
				return envVar.Value == fmt.Sprintf("http://%s", hostname)
			}
			return false

		}), TestTimeoutShort).Should(BeTrue())
		g.Eventually(OperatorPodHas(t, ctx, ns, func(op *corev1.Pod) bool {
			if envVar := envvar.Get(op.Spec.Containers[0].Env, "NO_PROXY"); envVar != nil {
				return envVar.Value == strings.Join(noProxy, ",")
			}
			return false

		}), TestTimeoutShort).Should(BeTrue())

		// Run the Integration
		name := RandomizedSuffixName("java")
		g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "--name", name).Execute()).To(Succeed())

		g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		proxies := corev1.PodList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
		}
		err = TestClient(t).List(ctx, &proxies,
			ctrl.InNamespace(ns),
			ctrl.MatchingLabels(deployment.Spec.Selector.MatchLabels),
		)
		g.Expect(err).To(Succeed())
		g.Expect(proxies.Items).To(HaveLen(1))
	})
}

func TestMavenProxyNotPresent(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		//hostname := fmt.Sprintf("%s.%s.svc", "proxy-fake", ns)
		svc := Service(t, ctx, TestDefaultNamespace, "kubernetes")()
		g.Expect(svc).NotTo(BeNil())
		// It may be needed to populate the values from the cluster, machine and service network CIDRs
		noProxy := []string{
			".cluster.local",
			".svc",
			"localhost",
		}
		noProxy = append(noProxy, svc.Spec.ClusterIPs...)

		InstallOperator(t, g, ns)
		//g.Expect(KamelInstallWithID(t, ctx, operatorID, ns, "--operator-env-vars", fmt.Sprintf("HTTP_PROXY=http://%s", hostname), "--operator-env-vars", "NO_PROXY="+strings.Join(noProxy, ","))).To(Succeed())

		// Run the Integration
		name := RandomizedSuffixName("java")
		g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "--name", name).Execute()).To(Succeed())

		// Should not be able to build
		g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseError))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionKitAvailable), TestTimeoutShort).
			Should(Equal(corev1.ConditionFalse))
	})
}

func newHTTPDConfigMap(ns, hostname string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
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
				hostname, httpdTlsMountPath, corev1.TLSCertKey, httpdTlsMountPath, corev1.TLSPrivateKeyKey,
			),
		},
	}
}

func newHTTPDDeployment(ns, configName, secretName string) *appsv1.Deployment {
	// $ curl --proxy-cacert ca.crt --proxy https://proxy.http-proxy.svc:443 https://www.google.com
	// https://github.com/curl/curl/pull/1127
	return &appsv1.Deployment{
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
									MountPath: httpdTlsMountPath,
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
									SecretName: secretName,
								},
							},
						},
						{
							Name: "httpd-conf",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: configName,
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
}

func newHTTPDService(deployment *appsv1.Deployment) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: deployment.Namespace,
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
}
