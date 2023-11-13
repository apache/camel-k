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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/test"
)

func createLoggingTestEnv(t *testing.T, color bool, json bool, jsonPrettyPrint bool, logLevel string, logFormat string, logCategory map[string]string) *Environment {
	t.Helper()

	client, _ := test.NewFakeClient()
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
						Color:           pointer.Bool(color),
						Format:          logFormat,
						JSON:            pointer.Bool(json),
						JSONPrettyPrint: pointer.Bool(jsonPrettyPrint),
						Level:           logLevel,
						Category:        logCategory,
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

	return createLoggingTestEnv(t, true, false, false, defaultLogLevel, "", map[string]string{})
}

func NewLoggingTestCatalog() *Catalog {
	return NewCatalog(nil)
}

func TestEmptyLoggingTrait(t *testing.T) {
	env := createDefaultLoggingTestEnv(t)
	conditions, err := NewLoggingTestCatalog().apply(env)

	assert.Nil(t, err)
	assert.Empty(t, conditions)
	assert.NotEmpty(t, env.ExecutedTraits)

	quarkusConsoleColor := false
	jsonFormat := false
	jsonPrettyPrint := false
	logLevelIsInfo := false
	logFormatIsNotDefault := false

	for _, e := range env.EnvVars {
		if e.Name == envVarQuarkusConsoleColor {
			if e.Value == "true" {
				quarkusConsoleColor = true
			}
		}

		if e.Name == envVarQuarkusLogConsoleJSON {
			if e.Value == "true" {
				jsonFormat = true
			}
		}

		if e.Name == envVarQuarkusLogConsoleJSONPrettyPrint {
			if e.Value == "true" {
				jsonPrettyPrint = true
			}
		}

		if e.Name == envVarQuarkusLogLevel {
			if e.Value == "INFO" {
				logLevelIsInfo = true
			}
		}

		if e.Name == envVarQuarkusLogConsoleFormat {
			logFormatIsNotDefault = true
		}
	}

	assert.True(t, quarkusConsoleColor)
	assert.True(t, logLevelIsInfo)
	assert.False(t, jsonFormat)
	assert.False(t, jsonPrettyPrint)
	assert.False(t, logFormatIsNotDefault)
	assert.NotEmpty(t, env.ExecutedTraits)
}

func TestJsonLoggingTrait(t *testing.T) {
	// When running, this log should look like "09:07:00 INFO  (main) Profile prod activated."
	env := createLoggingTestEnv(t, true, true, false, "TRACE", "%d{HH:mm:ss} %-5p (%t) %s%e%n", map[string]string{})
	err := NewLoggingTestCatalog().apply(env)


	assert.Nil(t, err)
	assert.Empty(t, conditions)
	assert.NotEmpty(t, env.ExecutedTraits)

	quarkusConsoleColor := false
	jsonFormat := true
	jsonPrettyPrint := false
	logLevelIsTrace := false
	logFormatIsNotDefault := false

	for _, e := range env.EnvVars {
		if e.Name == envVarQuarkusConsoleColor {
			if e.Value == "true" {
				quarkusConsoleColor = true
			}
		}

		if e.Name == envVarQuarkusLogConsoleJSON {
			if e.Value == "true" {
				jsonFormat = true
			}
		}

		if e.Name == envVarQuarkusLogConsoleJSONPrettyPrint {
			if e.Value == "true" {
				jsonPrettyPrint = true
			}
		}

		if e.Name == envVarQuarkusLogLevel {
			if e.Value == "TRACE" {
				logLevelIsTrace = true
			}
		}

		if e.Name == envVarQuarkusLogConsoleFormat {
			if e.Value == "%d{HH:mm:ss} %-5p (%t) %s%e%n" {
				logFormatIsNotDefault = true
			}
		}
	}

	assert.False(t, quarkusConsoleColor)
	assert.True(t, jsonFormat)
	assert.False(t, jsonPrettyPrint)
	assert.True(t, logLevelIsTrace)
	assert.True(t, logFormatIsNotDefault)
	assert.NotEmpty(t, env.ExecutedTraits)
}

func TestSingleLoggingCategory(t *testing.T) {
	env := createLoggingTestEnv(t, true, true, false, "TRACE", "%d{HH:mm:ss} %-5p (%t) %s%e%n", map[string]string{})
	env.Integration.Spec.Traits = v1.Traits{
		Logging: &traitv1.LoggingTrait{
			Category: map[string]string{"org.test": "debug"},
		},
	}
	err := NewLoggingTestCatalog().apply(env)
	assert.Nil(t, err)

	testEnvVar := corev1.EnvVar{"QUARKUS_LOG_CATEGORY_ORG_TEST_LEVEL", "DEBUG", nil}
	assert.Contains(t, env.EnvVars, testEnvVar)
}

func TestLoggingCategories(t *testing.T) {
	env := createLoggingTestEnv(t, true, true, false, "TRACE", "%d{HH:mm:ss} %-5p (%t) %s%e%n", map[string]string{})
	env.Integration.Spec.Traits = v1.Traits{
		Logging: &traitv1.LoggingTrait{
			Category: map[string]string{"org.test": "debug", "org.jboss.resteasy": "debug"},
		},
	}
	err := NewLoggingTestCatalog().apply(env)
	assert.Nil(t, err)

	testEnvVars := []corev1.EnvVar{
		corev1.EnvVar{"QUARKUS_LOG_CATEGORY_ORG_TEST_LEVEL", "DEBUG", nil},
		corev1.EnvVar{"QUARKUS_LOG_CATEGORY_ORG_JBOSS_RESTEASY_LEVEL", "DEBUG", nil},
	}

	for _, v := range testEnvVars {
		assert.Contains(t, env.EnvVars, v)
	}

}
