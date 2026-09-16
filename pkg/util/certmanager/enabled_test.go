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

package certmanager

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	certmanagerv1 "github.com/apache/camel-k/v2/pkg/apis/duck/certmanager/v1"
	"github.com/apache/camel-k/v2/pkg/internal"
)

type errorMockReader struct {
	ctrl.Reader
	listErr error
	getErr  error
}

func (m *errorMockReader) List(ctx context.Context, list ctrl.ObjectList, opts ...ctrl.ListOption) error {
	if m.listErr != nil {
		return m.listErr
	}

	return m.Reader.List(ctx, list, opts...)
}

func (m *errorMockReader) Get(ctx context.Context, key ctrl.ObjectKey, obj ctrl.Object, opts ...ctrl.GetOption) error {
	if m.getErr != nil {
		return m.getErr
	}

	return m.Reader.Get(ctx, key, obj, opts...)
}

func newClusterIssuer(name string) *certmanagerv1.ClusterIssuer {
	return &certmanagerv1.ClusterIssuer{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
}

func newIssuer(namespace, name string) *certmanagerv1.Issuer {
	return &certmanagerv1.Issuer{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
	}
}

func TestListClusterIssuers(t *testing.T) {
	ci1 := newClusterIssuer("letsencrypt-staging")
	ci2 := newClusterIssuer("letsencrypt-prod")

	c, err := internal.NewFakeClient(ci1, ci2)
	require.NoError(t, err)

	names, err := ListClusterIssuers(context.Background(), c)
	require.NoError(t, err)
	assert.Equal(t, []string{"letsencrypt-prod", "letsencrypt-staging"}, names)
}

func TestGetClusterIssuer(t *testing.T) {
	ci := newClusterIssuer("my-cluster-issuer")

	c, err := internal.NewFakeClient(ci)
	require.NoError(t, err)

	exists, err := GetClusterIssuer(context.Background(), c, "my-cluster-issuer")
	require.NoError(t, err)
	assert.True(t, exists)

	notExists, err := GetClusterIssuer(context.Background(), c, "non-existent")
	require.NoError(t, err)
	assert.False(t, notExists)
}

func TestListIssuers(t *testing.T) {
	i1 := newIssuer("ns1", "issuer-b")
	i2 := newIssuer("ns1", "issuer-a")
	i3 := newIssuer("ns2", "issuer-other")

	c, err := internal.NewFakeClient(i1, i2, i3)
	require.NoError(t, err)

	names, err := ListIssuers(context.Background(), c, "ns1")
	require.NoError(t, err)
	assert.Equal(t, []string{"issuer-a", "issuer-b"}, names)

	namesOther, err := ListIssuers(context.Background(), c, "ns2")
	require.NoError(t, err)
	assert.Equal(t, []string{"issuer-other"}, namesOther)

	namesEmpty, err := ListIssuers(context.Background(), c, "ns-empty")
	require.NoError(t, err)
	assert.Empty(t, namesEmpty)
}

func TestGetIssuer(t *testing.T) {
	i := newIssuer("test-ns", "my-issuer")

	c, err := internal.NewFakeClient(i)
	require.NoError(t, err)

	exists, err := GetIssuer(context.Background(), c, "test-ns", "my-issuer")
	require.NoError(t, err)
	assert.True(t, exists)

	notExistsWrongNs, err := GetIssuer(context.Background(), c, "wrong-ns", "my-issuer")
	require.NoError(t, err)
	assert.False(t, notExistsWrongNs)

	notExistsWrongName, err := GetIssuer(context.Background(), c, "test-ns", "other-issuer")
	require.NoError(t, err)
	assert.False(t, notExistsWrongName)
}

func TestCRDsAbsentEntirely(t *testing.T) {
	baseClient, err := internal.NewFakeClient()
	require.NoError(t, err)

	noKindMatchErr := &meta.NoKindMatchError{
		GroupKind:        schema.GroupKind{Group: certmanagerv1.CertManagerGroup, Kind: "ClusterIssuerList"},
		SearchedVersions: []string{"v1"},
	}

	reader := &errorMockReader{
		Reader:  baseClient,
		listErr: noKindMatchErr,
		getErr:  noKindMatchErr,
	}

	// 1. ListClusterIssuers should return (nil, nil)
	ciList, err := ListClusterIssuers(context.Background(), reader)
	assert.NoError(t, err)
	assert.Nil(t, ciList)

	// 2. ListIssuers should return (nil, nil)
	iList, err := ListIssuers(context.Background(), reader, "my-ns")
	assert.NoError(t, err)
	assert.Nil(t, iList)

	// 3. GetClusterIssuer should return (false, nil)
	ciExists, err := GetClusterIssuer(context.Background(), reader, "my-ci")
	assert.NoError(t, err)
	assert.False(t, ciExists)

	// 4. GetIssuer should return (false, nil)
	iExists, err := GetIssuer(context.Background(), reader, "my-ns", "my-i")
	assert.NoError(t, err)
	assert.False(t, iExists)
}

func TestCRDsAbsentViaUnknownAPIError(t *testing.T) {
	baseClient, err := internal.NewFakeClient()
	require.NoError(t, err)

	unknownAPIErr := fmt.Errorf("no matches for kind \"ClusterIssuerList\" in version \"cert-manager.io/v1\"")

	reader := &errorMockReader{
		Reader:  baseClient,
		listErr: unknownAPIErr,
		getErr:  unknownAPIErr,
	}

	ciList, err := ListClusterIssuers(context.Background(), reader)
	assert.NoError(t, err)
	assert.Nil(t, ciList)

	ciExists, err := GetClusterIssuer(context.Background(), reader, "my-ci")
	assert.NoError(t, err)
	assert.False(t, ciExists)
}

func TestCRDsAbsentViaNotFoundError(t *testing.T) {
	baseClient, err := internal.NewFakeClient()
	require.NoError(t, err)

	notFoundErr := k8serrors.NewNotFound(
		schema.GroupResource{Group: certmanagerv1.CertManagerGroup, Resource: "clusterissuers"},
		"my-ci",
	)

	reader := &errorMockReader{
		Reader:  baseClient,
		listErr: notFoundErr,
		getErr:  notFoundErr,
	}

	ciList, err := ListClusterIssuers(context.Background(), reader)
	assert.NoError(t, err)
	assert.Nil(t, ciList)

	ciExists, err := GetClusterIssuer(context.Background(), reader, "my-ci")
	assert.NoError(t, err)
	assert.False(t, ciExists)
}

func TestGenericErrorPropagated(t *testing.T) {
	baseClient, err := internal.NewFakeClient()
	require.NoError(t, err)

	authErr := errors.New("connection refused: dial tcp 10.0.0.1:443")

	reader := &errorMockReader{
		Reader:  baseClient,
		listErr: authErr,
		getErr:  authErr,
	}

	// Any non-absence error MUST be returned and NOT swallowed
	ciList, err := ListClusterIssuers(context.Background(), reader)
	assert.ErrorIs(t, err, authErr)
	assert.Nil(t, ciList)

	iList, err := ListIssuers(context.Background(), reader, "my-ns")
	assert.ErrorIs(t, err, authErr)
	assert.Nil(t, iList)

	ciExists, err := GetClusterIssuer(context.Background(), reader, "my-ci")
	assert.ErrorIs(t, err, authErr)
	assert.False(t, ciExists)

	iExists, err := GetIssuer(context.Background(), reader, "my-ns", "my-i")
	assert.ErrorIs(t, err, authErr)
	assert.False(t, iExists)
}
