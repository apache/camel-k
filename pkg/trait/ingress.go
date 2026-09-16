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
	"errors"
	"fmt"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util/certmanager"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const (
	ingressTraitID    = "ingress"
	ingressTraitOrder = 2400

	defaultPath           = "/"
	defaultPathTypePrefix = networkingv1.PathTypePrefix
)

type ingressTrait struct {
	BaseTrait
	traitv1.IngressTrait `property:",squash"`

	certManagerAnnotationKey string
	certManagerIssuerName    string
}

func newIngressTrait() Trait {
	return &ingressTrait{
		BaseTrait: NewBaseTrait(ingressTraitID, ingressTraitOrder),
	}
}

// IsAllowedInProfile overrides default.
//
//nolint:staticcheck
func (t *ingressTrait) IsAllowedInProfile(profile v1.TraitProfile) bool {
	return profile.Equal(v1.TraitProfileKubernetes)
}

func (t *ingressTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil {
		return false, nil, nil
	}

	if !e.IntegrationInRunningPhases() {
		return false, nil, nil
	}

	if !ptr.Deref(t.Enabled, true) {
		return false, NewIntegrationCondition(
			"Ingress",
			v1.IntegrationConditionExposureAvailable,
			corev1.ConditionFalse,
			v1.IntegrationConditionIngressNotAvailableReason,
			"explicitly disabled",
		), nil
	}

	if ptr.Deref(t.Auto, true) {
		if e.Resources.GetUserServiceForIntegration(e.Integration) == nil {
			return false, nil, nil
		}
	}

	if t.TLSSecretName == "" && (len(t.TLSHosts) > 0 || t.Host != "") &&
		(t.TLSIssuerName != "" || ptr.Deref(t.TLSCertManagerAuto, false)) {
		annotationKey, issuerName, err := t.resolveCertManagerIssuer(e)
		if err != nil {
			return false, nil, err
		}
		t.certManagerAnnotationKey = annotationKey
		t.certManagerIssuerName = issuerName
	}

	//nolint:staticcheck
	if t.Path != "" {
		m := "The path parameter is deprecated and may be removed in a future release. Use the paths parameter instead."
		t.L.Info(m)
		condition := NewIntegrationCondition(
			"Ingress",
			v1.IntegrationConditionTraitInfo,
			corev1.ConditionTrue,
			TraitConfigurationReason,
			m,
		)

		return true, condition, nil
	}

	return true, nil, nil
}

func (t *ingressTrait) Apply(e *Environment) error {
	service := e.Resources.GetUserServiceForIntegration(e.Integration)
	if service == nil {
		return errors.New("cannot apply ingress trait: no target service")
	}

	ingress := networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: networkingv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        service.Name,
			Namespace:   service.Namespace,
			Annotations: t.Annotations,
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: t.Host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: t.getPaths(service),
						},
					},
				},
			},
		},
	}
	if t.IngressClassName != "" {
		ingress.Spec.IngressClassName = &t.IngressClassName
	}

	tlsHosts := t.TLSHosts
	secretName := t.TLSSecretName

	// The cert-manager annotation/issuer, if any, was already resolved in Configure().
	// This only consumes that decision; it never talks to the cluster itself. It may
	// fall back to t.Host so that auto-discovery also works for the common single-host
	// case, without changing the pre-existing manual TLS behavior (which requires
	// TLSHosts to be set explicitly and never considers t.Host).
	if secretName == "" {
		if len(tlsHosts) == 0 && t.Host != "" {
			tlsHosts = []string{t.Host}
		}

		if len(tlsHosts) > 0 && t.certManagerIssuerName != "" {
			if ingress.Annotations == nil {
				ingress.Annotations = map[string]string{}
			}
			ingress.Annotations[t.certManagerAnnotationKey] = t.certManagerIssuerName
			secretName = service.Name + "-tls"
		}
	}

	if len(tlsHosts) > 0 && secretName != "" {
		ingress.Spec.TLS = []networkingv1.IngressTLS{
			{
				Hosts:      tlsHosts,
				SecretName: secretName,
			},
		}
	}

	e.Resources.Add(&ingress)

	message := fmt.Sprintf("%s(%s) -> %s(%s)", ingress.Name, t.Host, service.Name, "http")

	e.Integration.Status.SetCondition(
		v1.IntegrationConditionExposureAvailable,
		corev1.ConditionTrue,
		v1.IntegrationConditionIngressAvailableReason,
		message,
	)

	return nil
}

// resolveCertManagerIssuer determines which cert-manager Issuer or ClusterIssuer
// annotation to apply to the Ingress, if any. It returns an empty issuerName when
// no annotation should be applied (cert-manager auto-discovery is disabled, cert-manager
// is not installed, or no issuer is found). A forced TLSIssuerName is verified to exist
// and returns an error if it does not; auto-discovery degrades to a no-op instead.
func (t *ingressTrait) resolveCertManagerIssuer(e *Environment) (annotationKey, issuerName string, err error) {
	namespace := e.Integration.Namespace

	installed, err := certmanager.IsInstalled(t.Client)
	if err != nil {
		return "", "", err
	}

	if t.TLSIssuerName != "" {
		if !installed {
			return "", "", fmt.Errorf("cert-manager is not installed but tlsIssuerName %q was set", t.TLSIssuerName)
		}

		kind := t.TLSIssuerKind
		if kind == "" {
			kind = "ClusterIssuer"
		}

		switch kind {
		case "Issuer":
			exists, err := certmanager.GetIssuer(e.Ctx, t.Client, namespace, t.TLSIssuerName)
			if err != nil {
				return "", "", err
			}
			if !exists {
				return "", "", fmt.Errorf("issuer %q not found in namespace %q", t.TLSIssuerName, namespace)
			}

			return certmanager.AnnotationIssuer, t.TLSIssuerName, nil
		case "ClusterIssuer":
			exists, err := certmanager.GetClusterIssuer(e.Ctx, t.Client, t.TLSIssuerName)
			if err != nil {
				return "", "", err
			}
			if !exists {
				return "", "", fmt.Errorf("clusterissuer %q not found", t.TLSIssuerName)
			}

			return certmanager.AnnotationClusterIssuer, t.TLSIssuerName, nil
		default:
			return "", "", fmt.Errorf("invalid tlsIssuerKind %q: must be %q or %q", kind, "Issuer", "ClusterIssuer")
		}
	}

	if !ptr.Deref(t.TLSCertManagerAuto, false) || !installed {
		return "", "", nil
	}

	clusterIssuers, err := certmanager.ListClusterIssuers(e.Ctx, t.Client)
	if err != nil {
		return "", "", err
	}
	if len(clusterIssuers) > 0 {
		return certmanager.AnnotationClusterIssuer, clusterIssuers[0], nil
	}

	issuers, err := certmanager.ListIssuers(e.Ctx, t.Client, namespace)
	if err != nil {
		return "", "", err
	}
	if len(issuers) > 0 {
		return certmanager.AnnotationIssuer, issuers[0], nil
	}

	return "", "", nil
}

func (t *ingressTrait) getPaths(service *corev1.Service) []networkingv1.HTTPIngressPath {
	createIngressPath := func(path string) networkingv1.HTTPIngressPath {
		return networkingv1.HTTPIngressPath{
			Path:     path,
			PathType: t.getPathType(),
			Backend: networkingv1.IngressBackend{
				Service: &networkingv1.IngressServiceBackend{
					Name: service.Name,
					Port: networkingv1.ServiceBackendPort{
						Name: "http",
					},
				},
			},
		}
	}

	paths := []networkingv1.HTTPIngressPath{}
	//nolint:staticcheck
	if t.Path == "" && len(t.Paths) == 0 {
		paths = append(paths, createIngressPath(defaultPath))
	} else {
		if t.Path != "" {
			paths = append(paths, createIngressPath(t.Path))
		}
		for _, p := range t.Paths {
			paths = append(paths, createIngressPath(p))
		}
	}

	return paths
}

func (t *ingressTrait) getPathType() *networkingv1.PathType {
	if t.PathType == nil {
		return new(defaultPathTypePrefix)
	}

	return t.PathType
}
