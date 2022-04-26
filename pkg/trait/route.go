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
	"k8s.io/utils/pointer"

	routev1 "github.com/openshift/api/route/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

type routeTrait struct {
	BaseTrait
	traitv1.RouteTrait `property:",squash"`
	service            *corev1.Service
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
	if !pointer.BoolDeref(t.Enabled, true) {
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
