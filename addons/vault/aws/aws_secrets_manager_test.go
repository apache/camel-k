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

package aws

import (
	"testing"

	"github.com/apache/camel-k/v2/pkg/util/test"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/trait"
	"github.com/apache/camel-k/v2/pkg/util/camel"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestAwsSecretsManagerTraitApply(t *testing.T) {
	e := createEnvironment(t, camel.QuarkusCatalog)
	aws := NewAwsSecretsManagerTrait()
	secrets, _ := aws.(*awsSecretsManagerTrait)
	secrets.Enabled = pointer.Bool(true)
	secrets.UseDefaultCredentialsProvider = pointer.Bool(false)
	secrets.Region = "eu-west-1"
	secrets.AccessKey = "access-key"
	secrets.SecretKey = "secret-key"
	ok, err := secrets.Configure(e)
	assert.Nil(t, err)
	assert.True(t, ok)

	err = secrets.Apply(e)
	assert.Nil(t, err)

	assert.Empty(t, e.ApplicationProperties["quarkus.jaeger.enabled"])
	assert.Equal(t, "eu-west-1", e.ApplicationProperties["camel.vault.aws.region"])
	assert.Equal(t, "access-key", e.ApplicationProperties["camel.vault.aws.accessKey"])
	assert.Equal(t, "secret-key", e.ApplicationProperties["camel.vault.aws.secretKey"])
	assert.Equal(t, "false", e.ApplicationProperties["camel.vault.aws.defaultCredentialsProvider"])
}

func TestAwsSecretsManagerTraitNoDefaultCreds(t *testing.T) {
	e := createEnvironment(t, camel.QuarkusCatalog)
	aws := NewAwsSecretsManagerTrait()
	secrets, _ := aws.(*awsSecretsManagerTrait)
	secrets.Enabled = pointer.Bool(true)
	secrets.Region = "eu-west-1"
	secrets.AccessKey = "access-key"
	secrets.SecretKey = "secret-key"
	ok, err := secrets.Configure(e)
	assert.Nil(t, err)
	assert.True(t, ok)

	err = secrets.Apply(e)
	assert.Nil(t, err)

	assert.Empty(t, e.ApplicationProperties["quarkus.jaeger.enabled"])
	assert.Equal(t, "eu-west-1", e.ApplicationProperties["camel.vault.aws.region"])
	assert.Equal(t, "access-key", e.ApplicationProperties["camel.vault.aws.accessKey"])
	assert.Equal(t, "secret-key", e.ApplicationProperties["camel.vault.aws.secretKey"])
	assert.Equal(t, "false", e.ApplicationProperties["camel.vault.aws.defaultCredentialsProvider"])
}

func TestAwsSecretsManagerTraitWithSecrets(t *testing.T) {
	e := createEnvironment(t, camel.QuarkusCatalog, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "my-secret1",
		},
		Data: map[string][]byte{
			"aws-secret-key": []byte("my-secret-key"),
		},
	}, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "my-secret2",
		},
		Data: map[string][]byte{
			"aws-access-key": []byte("my-access-key"),
		},
	})

	aws := NewAwsSecretsManagerTrait()
	secrets, _ := aws.(*awsSecretsManagerTrait)
	secrets.Enabled = pointer.Bool(true)
	secrets.Region = "eu-west-1"
	secrets.AccessKey = "secret:my-secret2/aws-access-key"
	secrets.SecretKey = "secret:my-secret1/aws-secret-key"
	ok, err := secrets.Configure(e)
	assert.Nil(t, err)
	assert.True(t, ok)

	err = secrets.Apply(e)
	assert.Nil(t, err)

	assert.Empty(t, e.ApplicationProperties["quarkus.jaeger.enabled"])
	assert.Equal(t, "eu-west-1", e.ApplicationProperties["camel.vault.aws.region"])
	assert.Equal(t, "my-access-key", e.ApplicationProperties["camel.vault.aws.accessKey"])
	assert.Equal(t, "my-secret-key", e.ApplicationProperties["camel.vault.aws.secretKey"])
	assert.Equal(t, "false", e.ApplicationProperties["camel.vault.aws.defaultCredentialsProvider"])
}

func TestAwsSecretsManagerTraitWithConfigMap(t *testing.T) {
	e := createEnvironment(t, camel.QuarkusCatalog, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "my-configmap1",
		},
		Data: map[string]string{
			"aws-secret-key": "my-secret-key",
		},
	}, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "my-configmap2",
		},
		Data: map[string]string{
			"aws-access-key": "my-access-key",
		},
	})

	aws := NewAwsSecretsManagerTrait()
	secrets, _ := aws.(*awsSecretsManagerTrait)
	secrets.Enabled = pointer.Bool(true)
	secrets.Region = "eu-west-1"
	secrets.AccessKey = "configmap:my-configmap2/aws-access-key"
	secrets.SecretKey = "configmap:my-configmap1/aws-secret-key"
	ok, err := secrets.Configure(e)
	assert.Nil(t, err)
	assert.True(t, ok)

	err = secrets.Apply(e)
	assert.Nil(t, err)

	assert.Empty(t, e.ApplicationProperties["quarkus.jaeger.enabled"])
	assert.Equal(t, "eu-west-1", e.ApplicationProperties["camel.vault.aws.region"])
	assert.Equal(t, "my-access-key", e.ApplicationProperties["camel.vault.aws.accessKey"])
	assert.Equal(t, "my-secret-key", e.ApplicationProperties["camel.vault.aws.secretKey"])
	assert.Equal(t, "false", e.ApplicationProperties["camel.vault.aws.defaultCredentialsProvider"])
}

func createEnvironment(t *testing.T, catalogGen func() (*camel.RuntimeCatalog, error), objects ...runtime.Object) *trait.Environment {
	t.Helper()

	catalog, err := catalogGen()
	client, _ := test.NewFakeClient(objects...)
	assert.Nil(t, err)

	e := trait.Environment{
		CamelCatalog:          catalog,
		ApplicationProperties: make(map[string]string),
		Client:                client,
	}

	it := v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "test",
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseDeploying,
		},
	}
	platform := v1.IntegrationPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "test",
		},
	}
	e.Integration = &it
	e.Platform = &platform
	return &e
}
