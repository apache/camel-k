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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/openshift"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// The Pull Secret trait sets a pull secret on the pod,
// to allow Kubernetes to retrieve the container image from an external registry.
//
// The pull secret can be specified manually or, in case you've configured authentication for an external container registry
// on the `IntegrationPlatform`, the same secret is used to pull images.
//
// It's enabled by default whenever you configure authentication for an external container registry,
// so it assumes that external registries are private.
//
// If your registry does not need authentication for pulling images, you can disable this trait.
//
// +camel-k:trait=pull-secret
type pullSecretTrait struct {
	BaseTrait `property:",squash"`
	// The pull secret name to set on the Pod. If left empty this is automatically taken from the `IntegrationPlatform` registry configuration.
	SecretName string `property:"secret-name" json:"secretName,omitempty"`
	// When using a global operator with a shared platform, this enables delegation of the `system:image-puller` cluster role on the operator namespace to the integration service account.
	ImagePullerDelegation *bool `property:"image-puller-delegation" json:"imagePullerDelegation,omitempty"`
	// Automatically configures the platform registry secret on the pod if it is of type `kubernetes.io/dockerconfigjson`.
	Auto *bool `property:"auto" json:"auto,omitempty"`
}

func newPullSecretTrait() Trait {
	return &pullSecretTrait{
		BaseTrait: NewBaseTrait("pull-secret", 1700),
	}
}

func (t *pullSecretTrait) Configure(e *Environment) (bool, error) {
	if util.IsFalse(t.Enabled) {
		return false, nil
	}

	if !e.IntegrationInPhase(v1.IntegrationPhaseDeploying, v1.IntegrationPhaseRunning) {
		return false, nil
	}

	if util.IsNilOrTrue(t.Auto) {
		if t.SecretName == "" {
			secret := e.Platform.Status.Build.Registry.Secret
			if secret != "" {
				key := client.ObjectKey{Namespace: e.Platform.Namespace, Name: secret}
				obj := corev1.Secret{}
				if err := t.Client.Get(t.Ctx, key, &obj); err != nil {
					return false, err
				}
				if obj.Type == corev1.SecretTypeDockerConfigJson {
					t.SecretName = secret
				}
			}
		}
		if t.ImagePullerDelegation == nil {
			var isOpenshift bool
			if t.Client != nil {
				var err error
				isOpenshift, err = openshift.IsOpenShift(t.Client)
				if err != nil {
					return false, err
				}
			}
			isOperatorGlobal := platform.IsCurrentOperatorGlobal()
			isKitExternal := e.Integration.GetIntegrationKitNamespace(e.Platform) != e.Integration.Namespace
			needsDelegation := isOpenshift && isOperatorGlobal && isKitExternal
			t.ImagePullerDelegation = &needsDelegation
		}
	}

	return t.SecretName != "" || util.IsTrue(t.ImagePullerDelegation), nil
}

func (t *pullSecretTrait) Apply(e *Environment) error {
	if t.SecretName != "" {
		e.Resources.VisitPodSpec(func(p *corev1.PodSpec) {
			p.ImagePullSecrets = append(p.ImagePullSecrets, corev1.LocalObjectReference{
				Name: t.SecretName,
			})
		})
	}
	if util.IsTrue(t.ImagePullerDelegation) {
		rb := t.newImagePullerRoleBinding(e)
		e.Resources.Add(rb)
	}

	return nil
}

func (t *pullSecretTrait) newImagePullerRoleBinding(e *Environment) *rbacv1.RoleBinding {
	serviceAccount := e.Integration.Spec.ServiceAccountName
	if serviceAccount == "" {
		serviceAccount = "default"
	}
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: rbacv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: e.Integration.GetIntegrationKitNamespace(e.Platform),
			Name:      fmt.Sprintf("camel-k-puller-%s", e.Integration.Namespace),
		},
		RoleRef: rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: "system:image-puller",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: e.Integration.Namespace,
				Name:      serviceAccount,
			},
		},
	}
}
