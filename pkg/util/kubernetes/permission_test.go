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

package kubernetes

import (
	"context"
	"errors"
	"testing"

	authorizationv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/stretchr/testify/require"
)

func TestCheckServiceAccountPermissionByResourceName(t *testing.T) {
	client := fake.NewSimpleClientset()
	client.PrependReactor("create", "subjectaccessreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createAction, ok := action.(k8stesting.CreateAction)
		require.True(t, ok)
		review, ok := createAction.GetObject().(*authorizationv1.SubjectAccessReview)
		require.True(t, ok)
		require.Equal(t, "system:serviceaccount:source:integration", review.Spec.User)
		require.Equal(t, "camel.apache.org", review.Spec.ResourceAttributes.Group)
		require.Equal(t, "kamelets", review.Spec.ResourceAttributes.Resource)
		require.Equal(t, "target", review.Spec.ResourceAttributes.Namespace)
		require.Equal(t, "get", review.Spec.ResourceAttributes.Verb)

		review.Status.Allowed = review.Spec.ResourceAttributes.Name == "allowed-kamelet"

		return true, review, nil
	})

	allowed, err := CheckServiceAccountPermission(
		context.Background(), client, "system:serviceaccount:source:integration",
		"camel.apache.org", "kamelets", "target", "allowed-kamelet", "get",
	)
	require.NoError(t, err)
	require.True(t, allowed)

	allowed, err = CheckServiceAccountPermission(
		context.Background(), client, "system:serviceaccount:source:integration",
		"camel.apache.org", "kamelets", "target", "denied-kamelet", "get",
	)
	require.NoError(t, err)
	require.False(t, allowed)
}

func TestCheckServiceAccountPermissionForbidden(t *testing.T) {
	client := fake.NewSimpleClientset()
	client.PrependReactor("create", "subjectaccessreviews", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(
			schema.GroupResource{Group: "authorization.k8s.io", Resource: "subjectaccessreviews"},
			"",
			errors.New("forbidden"),
		)
	})

	allowed, err := CheckServiceAccountPermission(
		context.Background(), client, "system:serviceaccount:source:integration",
		"camel.apache.org", "kamelets", "target", "denied-kamelet", "get",
	)
	require.NoError(t, err)
	require.False(t, allowed)
}

func TestCheckServiceAccountPermissionError(t *testing.T) {
	expected := errors.New("subject access review failed")
	client := fake.NewSimpleClientset()
	client.PrependReactor("create", "subjectaccessreviews", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, expected
	})

	allowed, err := CheckServiceAccountPermission(
		context.Background(), client, "system:serviceaccount:source:integration",
		"camel.apache.org", "kamelets", "target", "denied-kamelet", "get",
	)
	require.ErrorIs(t, err, expected)
	require.False(t, allowed)
}
