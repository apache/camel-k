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

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/openshift"
)

type pullSecretTrait struct {
	BaseTrait
	traitv1.PullSecretTrait `property:",squash"`
}

func newPullSecretTrait() Trait {
	return &pullSecretTrait{
		BaseTrait: NewBaseTrait("pull-secret", 1700),
	}
}

func (t *pullSecretTrait) Configure(e *Environment) (bool, error) {
	if !pointer.BoolDeref(t.Enabled, true) {
		return false, nil
	}

	if !e.IntegrationInRunningPhases() {
		return false, nil
	}

	if pointer.BoolDeref(t.Auto, true) {
		if t.SecretName == "" {
			secret := e.Platform.Status.Build.Registry.Secret
			if secret != "" {
				key := ctrl.ObjectKey{Namespace: e.Platform.Namespace, Name: secret}
				obj := corev1.Secret{}
				if err := t.Client.Get(e.Ctx, key, &obj); err != nil {
					return false, err
				}
				if obj.Type == corev1.SecretTypeDockerConfigJson {
					t.SecretName = secret
				}
			}
		}
		if t.ImagePullerDelegation == nil {
			var isOpenShift bool
			if t.Client != nil {
				var err error
				isOpenShift, err = openshift.IsOpenShift(t.Client)
				if err != nil {
					return false, err
				}
			}
			isOperatorGlobal := platform.IsCurrentOperatorGlobal()
			isKitExternal := e.Integration.GetIntegrationKitNamespace(e.Platform) != e.Integration.Namespace
			needsDelegation := isOpenShift && isOperatorGlobal && isKitExternal
			t.ImagePullerDelegation = &needsDelegation
		}
	}

	return t.SecretName != "" || pointer.BoolDeref(t.ImagePullerDelegation, false), nil
}

func (t *pullSecretTrait) Apply(e *Environment) error {
	if t.SecretName != "" {
		e.Resources.VisitPodSpec(func(p *corev1.PodSpec) {
			p.ImagePullSecrets = append(p.ImagePullSecrets, corev1.LocalObjectReference{
				Name: t.SecretName,
			})
		})
	}
	if pointer.BoolDeref(t.ImagePullerDelegation, false) {
		if err := t.delegateImagePuller(e); err != nil {
			return err
		}
	}

	return nil
}

func (t *pullSecretTrait) delegateImagePuller(e *Environment) error {
	// Applying the RoleBinding directly because it's a resource in the operator namespace
	// (different from the integration namespace when delegation is enabled).
	rb := t.newImagePullerRoleBinding(e)
	if err := kubernetes.ReplaceResource(e.Ctx, e.Client, rb); err != nil {
		return errors.Wrap(err, "error during the creation of the system:image-puller delegating role binding")
	}
	return nil
}

func (t *pullSecretTrait) newImagePullerRoleBinding(e *Environment) *rbacv1.RoleBinding {
	targetNamespace := e.Integration.GetIntegrationKitNamespace(e.Platform)
	var references []metav1.OwnerReference
	if e.Platform != nil && e.Platform.Namespace == targetNamespace {
		controller := true
		blockOwnerDeletion := true
		references = []metav1.OwnerReference{
			{
				APIVersion:         e.Platform.APIVersion,
				Kind:               e.Platform.Kind,
				Name:               e.Platform.Name,
				UID:                e.Platform.UID,
				Controller:         &controller,
				BlockOwnerDeletion: &blockOwnerDeletion,
			},
		}
	}
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
			Namespace:       targetNamespace,
			Name:            fmt.Sprintf("camel-k-puller-%s-%s", e.Integration.Namespace, serviceAccount),
			OwnerReferences: references,
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
