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
	"context"
	"errors"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestDeployerRollsBackResourcesWhenApplyFailsAfterPersistence(t *testing.T) {
	deployer, environment := createNominalDeployerTest()
	environment.Ctx = context.Background()
	environment.Integration.Namespace = "namespace"
	environment.Integration.Generation = 2

	previousDeployment := getIntegrationDeployment(environment.Integration)
	previousDeployment.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("Deployment"))
	previousDeployment.Labels[v1.IntegrationGenerationLabel] = "1"
	previousDeployment.Spec.Template.Spec.Containers[0].Image = "old-image"

	desiredDeployment := previousDeployment.DeepCopy()
	desiredDeployment.Labels[v1.IntegrationGenerationLabel] = "2"
	desiredDeployment.Spec.Template.Spec.Containers[0].Image = "new-image"

	newConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "new-config",
			Namespace: "namespace",
		},
		Data: map[string]string{"config": "new"},
	}
	newConfigMap.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))

	apiClient, err := internal.NewFakeClient(previousDeployment)
	require.NoError(t, err)

	environment.Client = apiClient
	environment.Resources = kubernetes.NewCollection(newConfigMap, desiredDeployment)

	injectedFailure := false
	apiClient.(*internal.FakeClient).Intercept(&interceptor.Funcs{
		Update: func(
			ctx context.Context,
			next ctrl.WithWatch,
			obj ctrl.Object,
			opts ...ctrl.UpdateOption,
		) error {
			if obj.GetName() == desiredDeployment.Name && !injectedFailure {
				injectedFailure = true

				if err := next.Update(ctx, obj, opts...); err != nil {
					return err
				}

				return errors.New("injected failure after persistence")
			}

			return next.Update(ctx, obj, opts...)
		},
	})

	require.NoError(t, deployer.Apply(environment))

	gcTrait, ok := newGCTrait().(*gcTrait)
	require.True(t, ok)
	require.NoError(t, gcTrait.Apply(environment))
	require.Len(t, environment.PostActions, 2)

	// Execute deployment followed by the actual GC post-action.
	// A fatal deployment failure must prevent GC from deleting generation 1.
	err = executePostActions(environment)

	require.Error(t, err)
	require.Truef(t, injectedFailure, "deployer error: %v", err)
	assert.Contains(t, err.Error(), "previous generation has been preserved")

	var fatalErr *fatalPostActionError
	require.ErrorAs(t, err, &fatalErr)

	actualDeployment := getIntegrationDeployment(environment.Integration)
	require.NoError(t, apiClient.Get(
		environment.Ctx,
		ctrl.ObjectKeyFromObject(previousDeployment),
		actualDeployment,
	))
	assert.Equal(t, "old-image", actualDeployment.Spec.Template.Spec.Containers[0].Image)
	assert.Equal(t, "1", actualDeployment.Labels[v1.IntegrationGenerationLabel])

	actualConfigMap := &corev1.ConfigMap{}
	err = apiClient.Get(
		environment.Ctx,
		ctrl.ObjectKeyFromObject(newConfigMap),
		actualConfigMap,
	)
	require.True(t, apierrors.IsNotFound(err))
}

func TestFatalPostActionStopsSubsequentActions(t *testing.T) {
	environment := &Environment{}
	garbageCollectionCalled := false

	environment.PostActions = []func(*Environment) error{
		func(*Environment) error {
			return newFatalPostActionError(errors.New("deployment failed"))
		},
		func(*Environment) error {
			garbageCollectionCalled = true
			return nil
		},
	}

	err := executePostActions(environment)

	require.Error(t, err)
	assert.False(t, garbageCollectionCalled)
}

func TestNonFatalPostActionErrorDoesNotStopSubsequentActions(t *testing.T) {
	environment := &Environment{}
	subsequentActionCalled := false

	environment.PostActions = []func(*Environment) error{
		func(*Environment) error {
			return errors.New("non-fatal post-action failure")
		},
		func(*Environment) error {
			subsequentActionCalled = true
			return nil
		},
	}

	err := executePostActions(environment)

	require.Error(t, err)
	assert.True(t, subsequentActionCalled)
}
