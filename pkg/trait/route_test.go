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

package trait

import (
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"

	routev1 "github.com/openshift/api/route/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"
)

const (
	host = "my-host1"
	key  = `-----BEGIN PRIVATE KEY-----
MIICdwIBADANBgkqhkiG9w0BAQEFAASCAmEwggJdAgEAAoGBAKulUTZ8B1qccZ8c
DXRGSY08gW8KvLlcxxxGC4gZHNT3CBUF8n5R4KE30aZyYZ/rtsQZu05juZJxaJ0q
mbe75dlQ5d+Xc9BMXeQg/MpTZw5TAN7OIdGYYpFBe+1PLZ6wEfjkYrMqMUcfq2Lq
hTLdAbvBJnuRcYZLqmBeOQ8FTrKrAgMBAAECgYEAnkHRbEPU3/WISSQrP36iyCb2
S/SBZwKkzmvCrBxDWhPeDswp9c/2JY76rNWfLzy8iXgUG8WUzvHje61Qh3gmBcKe
bUaTGl4Vy8Ha1YBADo5RfRrdm0FE4tvgvu/TkqFqpBBZweu54285hk5zlG7n/D7Y
dnNXUpu5MlNb5x3gW0kCQQDUL//cwcXUxY/evaJP4jSe+ZwEQZo+zXRLiPUulBoV
aw28CVMuxdgwqAo1X1IKefPeUaf7RQu8gCKaRnpGuEuXAkEAzxZTfMmvmCUDIew4
5Gk6bK265XQWdhcgiq254lpBGOYmDj9yCE7yA+zmASQwMsXTdQOi1hOCEyrXuSJ5
c++EDQJAFh3WrnzoEPByuYXMmET8tSFRWMQ5vpgNqh3haHR5b4gUC2hxaiunCBNL
1RpVY9AoUiDywGcG/SPh93CnKB3niwJBAKP7AtsifZgVXtiizB4aMThTjVYaSZrz
D0Kg9DuHylpkDChmFu77TGrNUQgAVuYtfhb/bRblVa/F0hJ4eQHT3JUCQBVT68tb
OgRUk0aP9tC3021VN82X6+klowSQN8oBPX8+TfDWSUilp/+j24Hky+Z29Do7yR/R
qutnL92CvBlVLV4=
-----END PRIVATE KEY-----
`
	cert = `-----BEGIN CERTIFICATE-----
MIIBajCCARCgAwIBAgIUbYqrLSOSQHoxD8CwG6Bi2PJi9c8wCgYIKoZIzj0EAwIw
EzERMA8GA1UEAxMIc3dhcm0tY2EwHhcNMTcwNDI0MjE0MzAwWhcNMzcwNDE5MjE0
MzAwWjATMREwDwYDVQQDEwhzd2FybS1jYTBZMBMGByqGSM49AgEGCCqGSM49AwEH
A0IABJk/VyMPYdaqDXJb/VXh5n/1Yuv7iNrxV3Qb3l06XD46seovcDWs3IZNV1lf
3Skyr0ofcchipoiHkXBODojJydSjQjBAMA4GA1UdDwEB/wQEAwIBBjAPBgNVHRMB
Af8EBTADAQH/MB0GA1UdDgQWBBRUXxuRcnFjDfR/RIAUQab8ZV/n4jAKBggqhkjO
PQQDAgNIADBFAiAy+JTe6Uc3KyLCMiqGl2GyWGQqQDEcO3/YG36x7om65AIhAJvz
pxv6zFeVEkAEEkqIYi0omA9+CjanB/6Bz4n1uw8H
-----END CERTIFICATE-----
`
	caCert = `-----BEGIN CERTIFICATE-----
BLAajCCARCgAwIBAgIUbYqrLSOSQHoxD8CwG6Bi2PJi9c8wCgYIKoZIzj0EAwIw
EzERMA8GA1UEAxMIc3dhcm0tY2EwHhcNMTcwNDI0MjE0MzAwWhcNMzcwNDE5MjE0
MzAwWjATMREwDwYDVQQDEwhzd2FybS1jYTBZMBMGByqGSM49AgEGCCqGSM49AwEH
A0IABJk/VyMPYdaqDXJb/VXh5n/1Yuv7iNrxV3Qb3l06XD46seovcDWs3IZNV1lf
3Skyr0ofcchipoiHkXBODojJydSjQjBAMA4GA1UdDwEB/wQEAwIBBjAPBgNVHRMB
Af8EBTADAQH/MB0GA1UdDgQWBBRUXxuRcnFjDfR/RIAUQab8ZV/n4jAKBggqhkjO
PQQDAgNIADBFAiAy+JTe6Uc3KyLCMiqGl2GyWGQqQDEcO3/YG36x7om65AIhAJvz
pxv6zFeVEkAEEkqIYi0omA9+CjanB/6Bz4n1uw8H
-----END CERTIFICATE-----
`
	destinationCaCert = `-----BEGIN CERTIFICATE-----
FOOBARCCARCgAwIBAgIUbYqrLSOSQHoxD8CwG6Bi2PJi9c8wCgYIKoZIzj0EAwIw
EzERMA8GA1UEAxMIc3dhcm0tY2EwHhcNMTcwNDI0MjE0MzAwWhcNMzcwNDE5MjE0
MzAwWjATMREwDwYDVQQDEwhzd2FybS1jYTBZMBMGByqGSM49AgEGCCqGSM49AwEH
A0IABJk/VyMPYdaqDXJb/VXh5n/1Yuv7iNrxV3Qb3l06XD46seovcDWs3IZNV1lf
3Skyr0ofcchipoiHkXBODojJydSjQjBAMA4GA1UdDwEB/wQEAwIBBjAPBgNVHRMB
Af8EBTADAQH/MB0GA1UdDgQWBBRUXxuRcnFjDfR/RIAUQab8ZV/n4jAKBggqhkjO
PQQDAgNIADBFAiAy+JTe6Uc3KyLCMiqGl2GyWGQqQDEcO3/YG36x7om65AIhAJvz
pxv6zFeVEkAEEkqIYi0omA9+CjanB/6Bz4n1uw8H
-----END CERTIFICATE-----
`

	tlsKeySecretName        = "tls-test"
	tlsKeySecretOnlyKeyName = "tls.key"

	// Potential hardcoded credentials
	// #nosec G101
	tlsMultipleSecretsName = "tls-multiple-test"
	// #nosec G101
	tlsMultipleSecretsCert1Key = "cert1.crt"
	// #nosec G101
	tlsMultipleSecretsCert2Key = "cert2.crt"
	// #nosec G101
	tlsMultipleSecretsCert3Key = "cert3.crt"
)

func createTestRouteEnvironment(t *testing.T, name string) *Environment {
	t.Helper()

	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)
	client, _ := test.NewFakeClient(
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test-ns",
				Name:      tlsKeySecretName,
			},
			Data: map[string][]byte{
				tlsKeySecretOnlyKeyName: []byte(key),
			},
		},
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test-ns",
				Name:      tlsMultipleSecretsName,
			},
			Data: map[string][]byte{
				tlsMultipleSecretsCert1Key: []byte(cert),
				tlsMultipleSecretsCert2Key: []byte(caCert),
				tlsMultipleSecretsCert3Key: []byte(destinationCaCert),
			},
		},
	)
	res := &Environment{
		CamelCatalog: catalog,
		Catalog:      NewCatalog(client),
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "test-ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyS2I,
					Registry:        v1.RegistrySpec{Address: "registry"},
				},
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources: kubernetes.NewCollection(
			&corev1.Service{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Service",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: "test-ns",
					Labels: map[string]string{
						v1.IntegrationLabel:             name,
						"camel.apache.org/service.type": v1.ServiceTypeUser,
					},
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{},
					Selector: map[string]string{
						v1.IntegrationLabel: name,
					},
				},
			},
		),
	}
	res.Platform.ResyncStatusFullConfig()
	return res
}

func TestRoute_Default(t *testing.T) {
	name := xid.New().String()
	environment := createTestRouteEnvironment(t, name)
	traitsCatalog := environment.Catalog

	err := traitsCatalog.apply(environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("container"))
	assert.NotNil(t, environment.GetTrait("route"))

	route := environment.Resources.GetRoute(func(r *routev1.Route) bool {
		return r.ObjectMeta.Name == name
	})

	assert.NotNil(t, route)
	assert.Nil(t, route.Spec.TLS)
	assert.NotNil(t, route.Spec.Port)
	assert.Equal(t, defaultContainerPortName, route.Spec.Port.TargetPort.StrVal)
}

func TestRoute_Disabled(t *testing.T) {
	name := xid.New().String()
	environment := createTestRouteEnvironment(t, name)
	environment.Integration.Spec.Traits = v1.Traits{
		Route: &traitv1.RouteTrait{
			Trait: traitv1.Trait{
				Enabled: pointer.Bool(false),
			},
		},
	}

	traitsCatalog := environment.Catalog
	err := traitsCatalog.apply(environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.Nil(t, environment.GetTrait("route"))

	route := environment.Resources.GetRoute(func(r *routev1.Route) bool {
		return r.ObjectMeta.Name == name
	})

	assert.Nil(t, route)
}

func TestRoute_Configure_IntegrationKitOnly(t *testing.T) {
	name := xid.New().String()
	environment := createTestRouteEnvironment(t, name)
	environment.Integration = nil

	routeTrait, _ := newRouteTrait().(*routeTrait)
	enabled := false
	routeTrait.Enabled = &enabled

	result, err := routeTrait.Configure(environment)
	assert.False(t, result)
	assert.Nil(t, err)
}

func TestRoute_Host(t *testing.T) {
	name := xid.New().String()
	environment := createTestRouteEnvironment(t, name)
	traitsCatalog := environment.Catalog

	environment.Integration.Spec.Traits = v1.Traits{
		Route: &traitv1.RouteTrait{
			Host: host,
		},
	}

	err := traitsCatalog.apply(environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("route"))

	route := environment.Resources.GetRoute(func(r *routev1.Route) bool {
		return r.ObjectMeta.Name == name
	})

	assert.NotNil(t, route)
	assert.Equal(t, host, route.Spec.Host)
	assert.Nil(t, route.Spec.TLS)
}

func TestRoute_TLS_From_Secret_reencrypt(t *testing.T) {
	name := xid.New().String()
	environment := createTestRouteEnvironment(t, name)
	traitsCatalog := environment.Catalog

	environment.Integration.Spec.Traits = v1.Traits{
		Route: &traitv1.RouteTrait{
			TLSTermination:                    string(routev1.TLSTerminationReencrypt),
			Host:                              host,
			TLSKeySecret:                      tlsKeySecretName,
			TLSCertificateSecret:              tlsMultipleSecretsName + "/" + tlsMultipleSecretsCert1Key,
			TLSCACertificateSecret:            tlsMultipleSecretsName + "/" + tlsMultipleSecretsCert2Key,
			TLSDestinationCACertificateSecret: tlsMultipleSecretsName + "/" + tlsMultipleSecretsCert3Key,
		},
	}
	err := traitsCatalog.apply(environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("route"))

	route := environment.Resources.GetRoute(func(r *routev1.Route) bool {
		return r.ObjectMeta.Name == name
	})

	assert.NotNil(t, route)
	assert.NotNil(t, route.Spec.TLS)
	assert.Equal(t, routev1.TLSTerminationReencrypt, route.Spec.TLS.Termination)
	assert.Equal(t, key, route.Spec.TLS.Key)
	assert.Equal(t, host, route.Spec.Host)
	assert.Equal(t, cert, route.Spec.TLS.Certificate)
	assert.Equal(t, caCert, route.Spec.TLS.CACertificate)
	assert.Equal(t, destinationCaCert, route.Spec.TLS.DestinationCACertificate)
}

func TestRoute_TLS_wrong_secret(t *testing.T) {
	name := xid.New().String()
	environment := createTestRouteEnvironment(t, name)
	traitsCatalog := environment.Catalog

	environment.Integration.Spec.Traits = v1.Traits{
		Route: &traitv1.RouteTrait{
			TLSTermination:                    string(routev1.TLSTerminationReencrypt),
			Host:                              host,
			TLSKeySecret:                      "foo",
			TLSCertificateSecret:              "bar",
			TLSCACertificateSecret:            "test",
			TLSDestinationCACertificateSecret: "404",
		},
	}
	err := traitsCatalog.apply(environment)

	// there must be errors as the trait has wrong configuration
	assert.NotNil(t, err)
	assert.Nil(t, environment.GetTrait("route"))

	route := environment.Resources.GetRoute(func(r *routev1.Route) bool {
		return r.ObjectMeta.Name == name
	})

	// route trait is expected to not be created
	assert.Nil(t, route)
}

func TestRoute_TLS_secret_wrong_key(t *testing.T) {
	name := xid.New().String()
	environment := createTestRouteEnvironment(t, name)
	traitsCatalog := environment.Catalog

	environment.Integration.Spec.Traits = v1.Traits{
		Route: &traitv1.RouteTrait{
			TLSTermination:         string(routev1.TLSTerminationReencrypt),
			Host:                   host,
			TLSKeySecret:           tlsKeySecretName,
			TLSCertificateSecret:   tlsMultipleSecretsName + "/" + tlsMultipleSecretsCert1Key,
			TLSCACertificateSecret: tlsMultipleSecretsName + "/foo",
		},
	}
	err := traitsCatalog.apply(environment)

	// there must be errors as the trait has wrong configuration
	assert.NotNil(t, err)
	assert.Nil(t, environment.GetTrait("route"))

	route := environment.Resources.GetRoute(func(r *routev1.Route) bool {
		return r.ObjectMeta.Name == name
	})

	// route trait is expected to not be created
	assert.Nil(t, route)
}

func TestRoute_TLS_secret_missing_key(t *testing.T) {
	name := xid.New().String()
	environment := createTestRouteEnvironment(t, name)
	traitsCatalog := environment.Catalog

	environment.Integration.Spec.Traits = v1.Traits{
		Route: &traitv1.RouteTrait{
			TLSTermination:         string(routev1.TLSTerminationReencrypt),
			Host:                   host,
			TLSKeySecret:           tlsKeySecretName,
			TLSCertificateSecret:   tlsMultipleSecretsName + "/" + tlsMultipleSecretsCert1Key,
			TLSCACertificateSecret: tlsMultipleSecretsName,
		},
	}
	err := traitsCatalog.apply(environment)

	// there must be errors as the trait has wrong configuration
	assert.NotNil(t, err)
	assert.Nil(t, environment.GetTrait("route"))

	route := environment.Resources.GetRoute(func(r *routev1.Route) bool {
		return r.ObjectMeta.Name == name
	})

	// route trait is expected to not be created
	assert.Nil(t, route)
}

func TestRoute_TLS_reencrypt(t *testing.T) {
	name := xid.New().String()
	environment := createTestRouteEnvironment(t, name)
	traitsCatalog := environment.Catalog

	environment.Integration.Spec.Traits = v1.Traits{
		Route: &traitv1.RouteTrait{
			TLSTermination:              string(routev1.TLSTerminationReencrypt),
			Host:                        host,
			TLSKey:                      key,
			TLSCertificate:              cert,
			TLSCACertificate:            caCert,
			TLSDestinationCACertificate: destinationCaCert,
		},
	}
	err := traitsCatalog.apply(environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("route"))

	route := environment.Resources.GetRoute(func(r *routev1.Route) bool {
		return r.ObjectMeta.Name == name
	})

	assert.NotNil(t, route)
	assert.NotNil(t, route.Spec.TLS)
	assert.Equal(t, routev1.TLSTerminationReencrypt, route.Spec.TLS.Termination)
	assert.Equal(t, key, route.Spec.TLS.Key)
	assert.Equal(t, host, route.Spec.Host)
	assert.Equal(t, cert, route.Spec.TLS.Certificate)
	assert.Equal(t, caCert, route.Spec.TLS.CACertificate)
	assert.Equal(t, destinationCaCert, route.Spec.TLS.DestinationCACertificate)
}

func TestRoute_TLS_edge(t *testing.T) {
	name := xid.New().String()
	environment := createTestRouteEnvironment(t, name)
	traitsCatalog := environment.Catalog

	environment.Integration.Spec.Traits = v1.Traits{
		Route: &traitv1.RouteTrait{
			TLSTermination:   string(routev1.TLSTerminationEdge),
			Host:             host,
			TLSKey:           key,
			TLSCertificate:   cert,
			TLSCACertificate: caCert,
		},
	}
	err := traitsCatalog.apply(environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("route"))

	route := environment.Resources.GetRoute(func(r *routev1.Route) bool {
		return r.ObjectMeta.Name == name
	})

	assert.NotNil(t, route)
	assert.NotNil(t, route.Spec.TLS)
	assert.Equal(t, routev1.TLSTerminationEdge, route.Spec.TLS.Termination)
	assert.Equal(t, key, route.Spec.TLS.Key)
	assert.Equal(t, host, route.Spec.Host)
	assert.Equal(t, cert, route.Spec.TLS.Certificate)
	assert.Equal(t, caCert, route.Spec.TLS.CACertificate)
	assert.Empty(t, route.Spec.TLS.DestinationCACertificate)
}

func TestRoute_TLS_passthrough(t *testing.T) {
	name := xid.New().String()
	environment := createTestRouteEnvironment(t, name)
	traitsCatalog := environment.Catalog

	environment.Integration.Spec.Traits = v1.Traits{
		Route: &traitv1.RouteTrait{
			TLSTermination:                   string(routev1.TLSTerminationPassthrough),
			Host:                             host,
			TLSInsecureEdgeTerminationPolicy: string(routev1.InsecureEdgeTerminationPolicyAllow),
		},
	}
	err := traitsCatalog.apply(environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("route"))

	route := environment.Resources.GetRoute(func(r *routev1.Route) bool {
		return r.ObjectMeta.Name == name
	})

	assert.NotNil(t, route)
	assert.NotNil(t, route.Spec.TLS)
	assert.Equal(t, routev1.TLSTerminationPassthrough, route.Spec.TLS.Termination)
	assert.Equal(t, host, route.Spec.Host)
	assert.Equal(t, routev1.InsecureEdgeTerminationPolicyAllow, route.Spec.TLS.InsecureEdgeTerminationPolicy)
	assert.Empty(t, route.Spec.TLS.Certificate)
	assert.Empty(t, route.Spec.TLS.CACertificate)
	assert.Empty(t, route.Spec.TLS.DestinationCACertificate)
}

func TestRoute_WithCustomServicePort(t *testing.T) {
	name := xid.New().String()
	environment := createTestRouteEnvironment(t, name)
	environment.Integration.Spec.Traits = v1.Traits{
		Container: &traitv1.ContainerTrait{
			ServicePortName: "my-port",
		},
	}

	traitsCatalog := environment.Catalog
	err := traitsCatalog.apply(environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("container"))
	assert.NotNil(t, environment.GetTrait("route"))

	route := environment.Resources.GetRoute(func(r *routev1.Route) bool {
		return r.ObjectMeta.Name == name
	})

	assert.NotNil(t, route)
	assert.NotNil(t, route.Spec.Port)

	trait := environment.Integration.Spec.Traits.Container
	assert.Equal(
		t,
		trait.ServicePortName,
		route.Spec.Port.TargetPort.StrVal,
	)
}
