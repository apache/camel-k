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
	"fmt"
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	routev1 "github.com/openshift/api/route/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

// The Route trait can be used to configure the creation of OpenShift routes for the integration.
//
// The certificate and key contents may be sourced either from the local filesystem or in a OpenShift `secret` object.
// The user may use the parameters ending in `-secret` (example: `tls-certificate-secret`) to reference a certificate stored in a `secret`.
// Parameters ending in `-secret` have higher priorities and in case the same route parameter is set, for example: `tls-key-secret` and `tls-key`,
// then `tls-key-secret` is used.
// The recommended approach to set the key and certificates is to use `secrets` to store their contents and use the
// following parameters to reference them: `tls-certificate-secret`, `tls-key-secret`, `tls-ca-certificate-secret`, `tls-destination-ca-certificate-secret`
// See the examples section at the end of this page to see the setup options.
//
// +camel-k:trait=route
// nolint: tagliatelle
type routeTrait struct {
	BaseTrait `property:",squash"`
	// To configure the host exposed by the route.
	Host string `property:"host" json:"host,omitempty"`
	// The TLS termination type, like `edge`, `passthrough` or `reencrypt`.
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSTermination string `property:"tls-termination" json:"tlsTermination,omitempty"`
	// The TLS certificate contents.
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSCertificate string `property:"tls-certificate" json:"tlsCertificate,omitempty"`
	// The secret name and key reference to the TLS certificate. The format is "secret-name[/key-name]", the value represents the secret name, if there is only one key in the secret it will be read, otherwise you can set a key name separated with a "/".
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSCertificateSecret string `property:"tls-certificate-secret" json:"tlsCertificateSecret,omitempty"`
	// The TLS certificate key contents.
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSKey string `property:"tls-key" json:"tlsKey,omitempty"`
	// The secret name and key reference to the TLS certificate key. The format is "secret-name[/key-name]", the value represents the secret name, if there is only one key in the secret it will be read, otherwise you can set a key name separated with a "/".
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSKeySecret string `property:"tls-key-secret" json:"tlsKeySecret,omitempty"`
	// The TLS CA certificate contents.
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSCACertificate string `property:"tls-ca-certificate" json:"tlsCACertificate,omitempty"`
	// The secret name and key reference to the TLS CA certificate. The format is "secret-name[/key-name]", the value represents the secret name, if there is only one key in the secret it will be read, otherwise you can set a key name separated with a "/".
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSCACertificateSecret string `property:"tls-ca-certificate-secret" json:"tlsCACertificateSecret,omitempty"`
	// The destination CA certificate provides the contents of the ca certificate of the final destination.  When using reencrypt
	// termination this file should be provided in order to have routers use it for health checks on the secure connection.
	// If this field is not specified, the router may provide its own destination CA and perform hostname validation using
	// the short service name (service.namespace.svc), which allows infrastructure generated certificates to automatically
	// verify.
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSDestinationCACertificate string `property:"tls-destination-ca-certificate" json:"tlsDestinationCACertificate,omitempty"`
	// The secret name and key reference to the destination CA certificate. The format is "secret-name[/key-name]", the value represents the secret name, if there is only one key in the secret it will be read, otherwise you can set a key name separated with a "/".
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSDestinationCACertificateSecret string `property:"tls-destination-ca-certificate-secret" json:"tlsDestinationCACertificateSecret,omitempty"`
	// To configure how to deal with insecure traffic, e.g. `Allow`, `Disable` or `Redirect` traffic.
	//
	// Refer to the OpenShift route documentation for additional information.
	TLSInsecureEdgeTerminationPolicy string `property:"tls-insecure-edge-termination-policy" json:"tlsInsecureEdgeTerminationPolicy,omitempty"`

	service *corev1.Service
}

func newRouteTrait() Trait {
	return &routeTrait{
		BaseTrait: NewBaseTrait("route", 2200),
	}
}

// IsAllowedInProfile overrides default.
func (t *routeTrait) IsAllowedInProfile(profile v1.TraitProfile) bool {
	return profile == v1.TraitProfileOpenShift
}

func (t *routeTrait) Configure(e *Environment) (bool, error) {
	if IsFalse(t.Enabled) {
		if e.Integration != nil {
			e.Integration.Status.SetCondition(
				v1.IntegrationConditionExposureAvailable,
				corev1.ConditionFalse,
				v1.IntegrationConditionRouteNotAvailableReason,
				"explicitly disabled",
			)
		}

		return false, nil
	}

	if !e.IntegrationInRunningPhases() {
		return false, nil
	}

	t.service = e.Resources.GetUserServiceForIntegration(e.Integration)
	if t.service == nil {
		if e.Integration != nil {
			e.Integration.Status.SetCondition(
				v1.IntegrationConditionExposureAvailable,
				corev1.ConditionFalse,
				v1.IntegrationConditionRouteNotAvailableReason,
				"no target service found",
			)
		}

		return false, nil
	}

	return true, nil
}

func (t *routeTrait) Apply(e *Environment) error {
	servicePortName := defaultContainerPortName
	dt := e.Catalog.GetTrait(containerTraitID)
	if dt != nil {
		servicePortName = dt.(*containerTrait).ServicePortName
	}

	tlsConfig, err := t.getTLSConfig(e)
	if err != nil {
		return err
	}
	route := routev1.Route{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Route",
			APIVersion: routev1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.service.Name,
			Namespace: t.service.Namespace,
			Labels: map[string]string{
				v1.IntegrationLabel: e.Integration.Name,
			},
		},
		Spec: routev1.RouteSpec{
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString(servicePortName),
			},
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: t.service.Name,
			},
			Host: t.Host,
			TLS:  tlsConfig,
		},
	}

	e.Resources.Add(&route)

	var message string

	if t.Host == "" {
		message = fmt.Sprintf("%s -> %s(%s)",
			route.Name,
			route.Spec.To.Name,
			route.Spec.Port.TargetPort.String())
	} else {
		message = fmt.Sprintf("%s(%s) -> %s(%s)",
			route.Name,
			t.Host,
			route.Spec.To.Name,
			route.Spec.Port.TargetPort.String())
	}

	e.Integration.Status.SetCondition(
		v1.IntegrationConditionExposureAvailable,
		corev1.ConditionTrue,
		v1.IntegrationConditionRouteAvailableReason,
		message,
	)

	return nil
}

func (t *routeTrait) getTLSConfig(e *Environment) (*routev1.TLSConfig, error) {
	// a certificate is a multiline text, but to set it as value in a single line in CLI, the user must escape the new line character as \\n
	// but in the TLS configuration, the certificates should be a multiline string
	// then we need to replace the incoming escaped new lines \\n for a real new line \n
	key := strings.ReplaceAll(t.TLSKey, "\\n", "\n")
	certificate := strings.ReplaceAll(t.TLSCertificate, "\\n", "\n")
	CACertificate := strings.ReplaceAll(t.TLSCACertificate, "\\n", "\n")
	destinationCAcertificate := strings.ReplaceAll(t.TLSDestinationCACertificate, "\\n", "\n")
	var err error
	if t.TLSKeySecret != "" {
		key, err = t.readContentIfExists(e, t.TLSKeySecret)
		if err != nil {
			return nil, err
		}
	}
	if t.TLSCertificateSecret != "" {
		certificate, err = t.readContentIfExists(e, t.TLSCertificateSecret)
		if err != nil {
			return nil, err
		}
	}
	if t.TLSCACertificateSecret != "" {
		CACertificate, err = t.readContentIfExists(e, t.TLSCACertificateSecret)
		if err != nil {
			return nil, err
		}
	}
	if t.TLSDestinationCACertificateSecret != "" {
		destinationCAcertificate, err = t.readContentIfExists(e, t.TLSDestinationCACertificateSecret)
		if err != nil {
			return nil, err
		}
	}

	config := routev1.TLSConfig{
		Termination:                   routev1.TLSTerminationType(t.TLSTermination),
		Key:                           key,
		Certificate:                   certificate,
		CACertificate:                 CACertificate,
		DestinationCACertificate:      destinationCAcertificate,
		InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyType(t.TLSInsecureEdgeTerminationPolicy),
	}

	if reflect.DeepEqual(config, routev1.TLSConfig{}) {
		return nil, nil
	}

	return &config, nil
}

func (t *routeTrait) readContentIfExists(e *Environment, secretName string) (string, error) {
	key := ""
	strs := strings.Split(secretName, "/")
	if len(strs) > 1 {
		secretName = strs[0]
		key = strs[1]
	}

	secret := kubernetes.LookupSecret(e.Ctx, t.Client, t.service.Namespace, secretName)
	if secret == nil {
		return "", fmt.Errorf("%s secret not found in %s namespace, make sure to provide it before the Integration can run", secretName, t.service.Namespace)
	}
	if len(secret.Data) > 1 && len(key) == 0 {
		return "", fmt.Errorf("secret %s contains multiple data keys, but no key was provided", secretName)
	}
	if len(secret.Data) == 1 && len(key) == 0 {
		for _, value := range secret.Data {
			content := string(value)
			return content, nil
		}
	}
	if len(key) > 0 {
		content := string(secret.Data[key])
		if len(content) == 0 {
			return "", fmt.Errorf("could not find key %s in secret %s in namespace %s", key, secretName, t.service.Namespace)
		}
		return content, nil
	}
	return "", nil
}
