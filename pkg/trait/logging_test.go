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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/util/boolean"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/envvar"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

func createLoggingTestEnv(t *testing.T, color bool, json bool, jsonPrettyPrint bool, logLevel string, logFormat string) *Environment {
	t.Helper()

	client, _ := internal.NewFakeClient()
	c, err := camel.DefaultCatalog()
	if err != nil {
		panic(err)
	}

	res := &Environment{
		Ctx:          context.TODO(),
		CamelCatalog: c,
		Catalog:      NewCatalog(nil),
		Client:       client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileOpenShift,
				Traits: v1.Traits{
					Logging: &traitv1.LoggingTrait{
						Color:           ptr.To(color),
						Format:          logFormat,
						JSON:            ptr.To(json),
						JSONPrettyPrint: ptr.To(jsonPrettyPrint),
						Level:           logLevel,
					},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseBuildSubmitted,
			},
		},
		Platform: &v1.IntegrationPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
			},
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
				IntegrationPlatformSpec: v1.IntegrationPlatformSpec{
					Build: v1.IntegrationPlatformBuildSpec{
						RuntimeVersion: c.Runtime.Version,
					},
				},
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}

	return res
}

func createDefaultLoggingTestEnv(t *testing.T) *Environment {
	t.Helper()

	return createLoggingTestEnv(t, true, false, false, defaultLogLevel, "")
}

func NewLoggingTestCatalog() *Catalog {
	return NewCatalog(nil)
}

func TestEmptyLoggingTrait(t *testing.T) {
	env := createDefaultLoggingTestEnv(t)
	conditions, traits, err := NewLoggingTestCatalog().apply(env)

	require.NoError(t, err)
	assert.NotEmpty(t, traits)
	assert.NotEmpty(t, conditions)
	assert.NotEmpty(t, env.ExecutedTraits)

	assert.Equal(t, "INFO", env.ApplicationProperties["camel.k.logging.level"])
	assert.Equal(t, "", env.ApplicationProperties["camel.k.logging.format"])
	assert.Equal(t, boolean.FalseString, env.ApplicationProperties["camel.k.logging.json"])
	assert.Equal(t, "", env.ApplicationProperties["camel.k.logging.jsonPrettyPrint"])
	assert.Equal(t, boolean.TrueString, env.ApplicationProperties["camel.k.logging.color"])

	assert.Equal(t, "${camel.k.logging.level}", env.ApplicationProperties["quarkus.log.level"])
	assert.Equal(t, "", env.ApplicationProperties["quarkus.log.console.format"])
	assert.Equal(t, "${camel.k.logging.json}", env.ApplicationProperties["quarkus.log.console.json"])
	assert.Equal(t, "", env.ApplicationProperties["quarkus.log.console.json.pretty-print"])
	assert.Equal(t, "${camel.k.logging.color}", env.ApplicationProperties["quarkus.console.color"])
}

func TestJsonLoggingTrait(t *testing.T) {
	// When running, this log should look like "09:07:00 INFO  (main) Profile prod activated."
	env := createLoggingTestEnv(t, true, true, true, "TRACE", "%d{HH:mm:ss} %-5p (%t) %s%e%n")
	conditions, traits, err := NewLoggingTestCatalog().apply(env)

	require.NoError(t, err)
	assert.NotEmpty(t, traits)
	assert.NotEmpty(t, conditions)
	assert.NotEmpty(t, env.ExecutedTraits)

	assert.Equal(t, "TRACE", env.ApplicationProperties["camel.k.logging.level"])
	assert.Equal(t, "%d{HH:mm:ss} %-5p (%t) %s%e%n", env.ApplicationProperties["camel.k.logging.format"])
	assert.Equal(t, boolean.TrueString, env.ApplicationProperties["camel.k.logging.json"])
	assert.Equal(t, boolean.TrueString, env.ApplicationProperties["camel.k.logging.jsonPrettyPrint"])
	assert.Equal(t, "", env.ApplicationProperties["camel.k.logging.color"])

	assert.Equal(t, "${camel.k.logging.level}", env.ApplicationProperties["quarkus.log.level"])
	assert.Equal(t, "${camel.k.logging.format}", env.ApplicationProperties["quarkus.log.console.format"])
	assert.Equal(t, "${camel.k.logging.json}", env.ApplicationProperties["quarkus.log.console.json"])
	assert.Equal(t, "${camel.k.logging.jsonPrettyPrint}", env.ApplicationProperties["quarkus.log.console.json.pretty-print"])
	assert.Equal(t, "", env.ApplicationProperties["quarkus.console.color"])
}

func TestDefaultQuarkusLogging(t *testing.T) {
	env := createDefaultLoggingTestEnv(t)
	// Simulate an older catalog configuration which is missing the logging
	// capability
	env.CamelCatalog.Runtime.Capabilities["logging"] = v1.Capability{
		RuntimeProperties: nil,
	}
	env.EnvVars = []corev1.EnvVar{}
	conditions, traits, err := NewLoggingTestCatalog().apply(env)

	require.NoError(t, err)
	assert.NotEmpty(t, traits)
	assert.NotEmpty(t, conditions)
	assert.NotEmpty(t, env.ExecutedTraits)

	assert.Equal(t, &corev1.EnvVar{Name: "QUARKUS_LOG_LEVEL", Value: "INFO"}, envvar.Get(env.EnvVars, envVarQuarkusLogLevel))
	assert.Equal(t, &corev1.EnvVar{Name: "QUARKUS_LOG_CONSOLE_JSON", Value: "false"}, envvar.Get(env.EnvVars, envVarQuarkusLogConsoleJSON))
}
