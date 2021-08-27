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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"
)

func createLoggingTestEnv(t *testing.T, color bool, json bool, jsonPrettyPrint bool, logLevel string, logFormat string) *Environment {
	c, err := camel.DefaultCatalog()
	if err != nil {
		panic(err)
	}

	res := &Environment{
		Ctx:          context.TODO(),
		CamelCatalog: c,
		Catalog:      NewCatalog(nil),
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
				Traits: map[string]v1.TraitSpec{
					"logging": test.TraitSpecFromMap(t, map[string]interface{}{
						"color":             color,
						"format":            logFormat,
						"json":              json,
						"json-pretty-print": jsonPrettyPrint,
						"level":             logLevel,
					}),
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
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}

	return res
}

func createDefaultLoggingTestEnv(t *testing.T) *Environment {
	return createLoggingTestEnv(t, true, false, false, defaultLogLevel, "")
}

func NewLoggingTestCatalog() *Catalog {
	return NewCatalog(nil)
}

func TestEmptyLoggingTrait(t *testing.T) {
	env := createDefaultLoggingTestEnv(t)
	err := NewLoggingTestCatalog().apply(env)

	assert.Nil(t, err)
	assert.NotEmpty(t, env.ExecutedTraits)

	quarkusConsoleColor := false
	jsonFormat := false
	jsonPrettyPrint := false
	logLevelIsInfo := false
	logFormatIsNotDefault := false

	for _, e := range env.EnvVars {
		if e.Name == envVarQuarkusLogConsoleColor {
			if e.Value == "true" {
				quarkusConsoleColor = true
			}
		}

		if e.Name == envVarQuarkusLogConsoleJson {
			if e.Value == "true" {
				jsonFormat = true
			}
		}

		if e.Name == envVarQuarkusLogConsoleJsonPrettyPrint {
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
	env := createLoggingTestEnv(t, true, true, false, "TRACE", "%d{HH:mm:ss} %-5p (%t) %s%e%n")
	err := NewLoggingTestCatalog().apply(env)

	assert.Nil(t, err)
	assert.NotEmpty(t, env.ExecutedTraits)

	quarkusConsoleColor := false
	jsonFormat := true
	jsonPrettyPrint := false
	logLevelIsTrace := false
	logFormatIsNotDefault := false

	for _, e := range env.EnvVars {
		if e.Name == envVarQuarkusLogConsoleColor {
			if e.Value == "true" {
				quarkusConsoleColor = true
			}
		}

		if e.Name == envVarQuarkusLogConsoleJson {
			if e.Value == "true" {
				jsonFormat = true
			}
		}

		if e.Name == envVarQuarkusLogConsoleJsonPrettyPrint {
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
