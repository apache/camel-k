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
	"path/filepath"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	serving "knative.dev/serving/pkg/apis/serving/v1"
)

var (
	cmrMountPath = filepath.ToSlash(camel.ResourcesConfigmapsMountPath)
	scrMountPath = filepath.ToSlash(camel.ResourcesSecretsMountPath)
	// Deprecated.
	rdMountPath = filepath.ToSlash(camel.ResourcesDefaultMountPath)
)

func TestConfigureJvmTraitInRightPhasesDoesSucceed(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)

	configured, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.True(t, configured)
	assert.Nil(t, condition)
}

func TestConfigureJvmTraitInWrongIntegrationPhaseDoesNotSucceed(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)
	environment.Integration.Status.Phase = v1.IntegrationPhaseError

	configured, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.True(t, configured)
	assert.Nil(t, condition)
}

func TestConfigureJvmTraitInWrongIntegrationKitPhaseDoesNotSucceed(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)
	environment.IntegrationKit.Status.Phase = v1.IntegrationKitPhaseWaitingForPlatform

	configured, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.False(t, configured)
	assert.Nil(t, condition)
}

func TestConfigureJvmTraitInWrongJvmDisabled(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)
	trait.Enabled = ptr.To(false)

	expectedCondition := NewIntegrationCondition(
		"JVM",
		v1.IntegrationConditionTraitInfo,
		corev1.ConditionTrue,
		"TraitConfiguration",
		"explicitly disabled by the user; this configuration is deprecated and may be removed within next releases",
	)
	configured, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.False(t, configured)
	assert.NotNil(t, condition)
	assert.Equal(t, expectedCondition, condition)
}

func TestConfigureJvmTraitExecutableSelfManagedBuildContainer(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)
	environment.Integration.Spec.Traits.Container = &traitv1.ContainerTrait{
		Image: "my-image",
	}

	configured, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.False(t, configured)
	assert.Equal(t,
		"explicitly disabled by the platform: integration kit was not created via Camel K operator and the user did not provide the jar to execute",
		condition.message,
	)
}

func TestConfigureJvmTraitExecutableSelfManagedBuildContainerWithJar(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)
	environment.Integration.Spec.Traits.Container = &traitv1.ContainerTrait{
		Image: "my-image",
	}
	trait.Jar = "my-path/to/my-app.jar"

	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: defaultContainerName,
						},
					},
				},
			},
		},
	}

	environment.Resources.Add(&d)
	configured, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.True(t, configured)
	assert.Nil(t, condition)

	err = trait.Apply(environment)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"-cp",
		fmt.Sprintf("./resources:%s:%s:%s", rdMountPath, cmrMountPath, scrMountPath),
		"-jar", "my-path/to/my-app.jar",
	}, d.Spec.Template.Spec.Containers[0].Args)
}

func TestConfigureJvmTraitExecutableSelfManagedBuildContainerWithJarAndOptions(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)
	environment.Integration.Spec.Traits.Container = &traitv1.ContainerTrait{
		Image: "my-image",
	}
	trait.Jar = "my-path/to/my-app.jar"
	// Add some additional JVM configurations
	trait.Classpath = "deps/a.jar:deps/b.jar"
	trait.Options = []string{
		"-Xmx1234M",
		"-Dmy-prop=abc",
	}

	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: defaultContainerName,
						},
					},
				},
			},
		},
	}

	environment.Resources.Add(&d)
	configured, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.True(t, configured)
	assert.Nil(t, condition)

	err = trait.Apply(environment)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"-Xmx1234M", "-Dmy-prop=abc",
		"-cp", "./resources:/etc/camel/resources:/etc/camel/resources.d/_configmaps:/etc/camel/resources.d/_secrets:deps/a.jar:deps/b.jar",
		"-jar", "my-path/to/my-app.jar",
	}, d.Spec.Template.Spec.Containers[0].Args)
}

func TestConfigureJvmTraitWithJar(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)
	trait.Jar = "my-path/to/my-app.jar"

	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: defaultContainerName,
						},
					},
				},
			},
		},
	}

	environment.Resources.Add(&d)
	configured, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.True(t, configured)
	assert.Nil(t, condition)

	err = trait.Apply(environment)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"-cp",
		fmt.Sprintf("./resources:%s:%s:%s", rdMountPath, cmrMountPath, scrMountPath),
		"-jar", "my-path/to/my-app.jar",
	}, d.Spec.Template.Spec.Containers[0].Args)
}

func TestConfigureJvmTraitWithJarAndConfigs(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)
	trait.Jar = "my-path/to/my-app.jar"
	// Add some additional JVM configurations
	trait.Classpath = "deps/a.jar:deps/b.jar"
	trait.Options = []string{
		"-Xmx1234M",
		"-Dmy-prop=abc",
	}

	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: defaultContainerName,
						},
					},
				},
			},
		},
	}

	environment.Resources.Add(&d)
	configured, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.True(t, configured)
	assert.Nil(t, condition)

	err = trait.Apply(environment)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"-Xmx1234M", "-Dmy-prop=abc",
		"-cp", "./resources:/etc/camel/resources:/etc/camel/resources.d/_configmaps:/etc/camel/resources.d/_secrets:deps/a.jar:deps/b.jar",
		"-jar", "my-path/to/my-app.jar",
	}, d.Spec.Template.Spec.Containers[0].Args)
}

func TestConfigureJvmTraitInWrongIntegrationKitPhaseExternal(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)
	environment.Integration.Spec.Traits.Container = &traitv1.ContainerTrait{
		Image: "my-image",
	}
	expectedCondition := NewIntegrationCondition(
		"JVM",
		v1.IntegrationConditionTraitInfo,
		corev1.ConditionTrue,
		"TraitConfiguration",
		"explicitly disabled by the platform: integration kit was not created via Camel K operator and the user did not provide the jar to execute",
	)
	configured, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.False(t, configured)
	assert.NotNil(t, condition)
	assert.Equal(t, expectedCondition, condition)
}

func TestApplyJvmTraitWithDeploymentResource(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)

	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: defaultContainerName,
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: "/mount/path",
								},
							},
						},
					},
				},
			},
		},
	}

	environment.Resources.Add(&d)
	configure, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.True(t, configure)
	assert.Nil(t, condition)
	err = trait.Apply(environment)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"-cp",
		fmt.Sprintf(
			"./resources:%s:%s:%s:/mount/path:dependencies/*",
			rdMountPath,
			cmrMountPath,
			scrMountPath,
		),
		"io.quarkus.bootstrap.runner.QuarkusEntryPoint",
	}, d.Spec.Template.Spec.Containers[0].Args)
}

func TestApplyJvmTraitWithKnativeResource(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)

	s := serving.Service{}
	s.Spec.ConfigurationSpec.Template = serving.RevisionTemplateSpec{}
	s.Spec.ConfigurationSpec.Template.Spec.Containers = []corev1.Container{
		{
			Name: defaultContainerName,
			VolumeMounts: []corev1.VolumeMount{
				{
					MountPath: "/mount/path",
				},
			},
		},
	}

	environment.Resources.Add(&s)
	configure, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.True(t, configure)
	assert.Nil(t, condition)
	err = trait.Apply(environment)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"-cp",
		fmt.Sprintf("./resources:%s:%s:%s:/mount/path:dependencies/*",
			rdMountPath, cmrMountPath, scrMountPath),
		"io.quarkus.bootstrap.runner.QuarkusEntryPoint",
	}, s.Spec.Template.Spec.Containers[0].Args)
}

func TestApplyJvmTraitWithDebugEnabled(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)
	trait.Debug = ptr.To(true)
	trait.DebugSuspend = ptr.To(true)

	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: defaultContainerName,
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: "/mount/path",
								},
							},
						},
					},
				},
			},
		},
	}

	environment.Resources.Add(&d)
	err := trait.Apply(environment)

	require.NoError(t, err)
	assert.Contains(t, d.Spec.Template.Spec.Containers[0].Args,
		"-agentlib:jdwp=transport=dt_socket,server=y,suspend=y,address=*:5005",
	)
}

func TestApplyJvmTraitWithExternalKitType(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypeExternal)

	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: defaultContainerName,
						},
					},
				},
			},
		},
	}

	environment.Resources.Add(&d)

	environment.Resources.Add(&d)
	configure, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.True(t, configure)
	assert.Nil(t, condition)
	err = trait.Apply(environment)
	require.NoError(t, err)

	assert.Equal(t, []string{
		"-cp",
		fmt.Sprintf("./resources:%s:%s:%s:dependencies/*", rdMountPath, cmrMountPath, scrMountPath),
		"io.quarkus.bootstrap.runner.QuarkusEntryPoint",
	}, d.Spec.Template.Spec.Containers[0].Args)
}

func TestApplyJvmTraitWithClasspath(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)
	trait.Classpath = "/path/to/my-dep.jar:/path/to/another/dep.jar"
	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: defaultContainerName,
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: "/mount/path",
								},
							},
						},
					},
				},
			},
		},
	}

	environment.Resources.Add(&d)
	configure, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.True(t, configure)
	assert.Nil(t, condition)
	err = trait.Apply(environment)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"-cp",
		fmt.Sprintf("./resources:%s:%s:%s:/mount/path:%s:%s:dependencies/*",
			rdMountPath, cmrMountPath, scrMountPath,
			"/path/to/another/dep.jar", "/path/to/my-dep.jar"),
		"io.quarkus.bootstrap.runner.QuarkusEntryPoint",
	}, d.Spec.Template.Spec.Containers[0].Args)
}

func TestApplyJvmTraitWithClasspathAndExistingContainerCPArg(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)
	trait.Classpath = "/path/to/my-dep.jar:/path/to/another/dep.jar"
	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: defaultContainerName,
							Args: []string{
								"-cp",
								"my-precious-lib.jar",
							},
						},
					},
				},
			},
		},
	}

	environment.Resources.Add(&d)
	configure, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.True(t, configure)
	assert.Nil(t, condition)
	err = trait.Apply(environment)

	require.NoError(t, err)
	assert.Equal(t, []string{
		// WARN: we don't care if there are multiple classpath arguments
		// as the application will use the second one
		"-cp",
		"my-precious-lib.jar",
		"-cp",
		fmt.Sprintf("./resources:%s:%s:%s:%s:%s:dependencies/*:my-precious-lib.jar",
			rdMountPath, cmrMountPath, scrMountPath,
			"/path/to/another/dep.jar", "/path/to/my-dep.jar"),
		"io.quarkus.bootstrap.runner.QuarkusEntryPoint",
	}, d.Spec.Template.Spec.Containers[0].Args)
}

func TestApplyJvmTraitKitMissing(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)
	environment.Integration.Spec.Traits.Container = &traitv1.ContainerTrait{
		Image: "my-image",
	}
	err := trait.Apply(environment)

	require.Error(t, err)
	assert.Equal(t, "unable to find a container for my-it Integration", err.Error())
}

func TestApplyJvmTraitContainerResourceArgs(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)
	memoryLimit := make(corev1.ResourceList)
	memoryLimit, err := kubernetes.ConfigureResource("4Gi", memoryLimit, corev1.ResourceMemory)
	require.NoError(t, err)
	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: defaultContainerName,
							Resources: corev1.ResourceRequirements{
								Limits: memoryLimit,
							},
						},
					},
				},
			},
		},
	}
	environment.Resources.Add(&d)
	err = trait.Apply(environment)

	require.NoError(t, err)
	assert.Contains(t, d.Spec.Template.Spec.Containers[0].Args, "-Xmx2147M")

	// User specified Xmx option
	trait.Options = []string{"-Xmx1111M"}
	err = trait.Apply(environment)

	require.NoError(t, err)
	assert.Contains(t, d.Spec.Template.Spec.Containers[0].Args, "-Xmx1111M")
}

func TestApplyJvmTraitHttpProxyArgs(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)
	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: defaultContainerName,
							Env: []corev1.EnvVar{
								{
									Name:  "HTTP_PROXY",
									Value: "http://my-user:my-password@my-proxy:1234",
								},
								{
									Name:  "HTTPS_PROXY",
									Value: "https://my-secure-user:my-secure-password@my-secure-proxy:6789",
								},
								{
									Name:  "NO_PROXY",
									Value: "https://my-non-proxied-host,1.2.3.4",
								},
							},
						},
					},
				},
			},
		},
	}
	environment.Resources.Add(&d)
	err := trait.Apply(environment)

	require.NoError(t, err)
	assert.Contains(t, d.Spec.Template.Spec.Containers[0].Args, "-Dhttp.proxyHost=\"my-proxy\"")
	assert.Contains(t, d.Spec.Template.Spec.Containers[0].Args, "-Dhttp.proxyPort=\"1234\"")
	assert.Contains(t, d.Spec.Template.Spec.Containers[0].Args, "-Dhttp.proxyUser=\"my-user\"")
	assert.Contains(t, d.Spec.Template.Spec.Containers[0].Args, "-Dhttp.proxyPassword=\"my-password\"")
	assert.Contains(t, d.Spec.Template.Spec.Containers[0].Args, "-Dhttps.proxyHost=\"my-secure-proxy\"")
	assert.Contains(t, d.Spec.Template.Spec.Containers[0].Args, "-Dhttps.proxyPort=\"6789\"")
	assert.Contains(t, d.Spec.Template.Spec.Containers[0].Args, "-Dhttps.proxyUser=\"my-secure-user\"")
	assert.Contains(t, d.Spec.Template.Spec.Containers[0].Args, "-Dhttps.proxyPassword=\"my-secure-password\"")
	assert.Contains(t, d.Spec.Template.Spec.Containers[0].Args, "-Dhttp.nonProxyHosts=\"https://my-non-proxied-host|1.2.3.4\"")
}

func createNominalJvmTest(kitType string) (*jvmTrait, *Environment) {
	catalog, _ := camel.DefaultCatalog()
	client, _ := internal.NewFakeClient()
	trait, _ := newJvmTrait().(*jvmTrait)
	trait.PrintCommand = ptr.To(false)
	trait.Client = client

	environment := &Environment{
		Catalog:      NewCatalog(nil),
		CamelCatalog: catalog,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "kit-namespace",
				Name:      "my-it",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1.SchemeGroupVersion.String(),
				Kind:       v1.IntegrationKitKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "kit-namespace",
				Name:      "kit-name",
				Labels: map[string]string{
					v1.IntegrationKitTypeLabel: kitType,
				},
			},
			Status: v1.IntegrationKitStatus{
				Artifacts: []v1.Artifact{
					{Target: "dependencies/my-dep.jar"},
				},
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Resources: kubernetes.NewCollection(),
	}

	return trait, environment
}

func TestApplyJvmTraitAgent(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)
	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: defaultContainerName,
						},
					},
				},
			},
		},
	}
	environment.Resources.Add(&d)

	trait.Agents = []string{"my-agent;url"}
	ok, cond, err := trait.Configure(environment)
	require.True(t, ok)
	require.Nil(t, cond)
	require.NoError(t, err)

	err = trait.Apply(environment)
	require.NoError(t, err)
	// The other args are coming by default
	assert.Len(t, d.Spec.Template.Spec.Containers[0].Args, 4)
	assert.Contains(t, d.Spec.Template.Spec.Containers[0].Args, "-javaagent:/agents/my-agent.jar")
}

func TestApplyJvmTraitAgents(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)
	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: defaultContainerName,
						},
					},
				},
			},
		},
	}
	environment.Resources.Add(&d)

	trait.Agents = []string{"my-agent;url", "another-agent;another-url;hello=world,my=test"}
	ok, cond, err := trait.Configure(environment)
	require.True(t, ok)
	require.Nil(t, cond)
	require.NoError(t, err)

	err = trait.Apply(environment)
	require.NoError(t, err)
	// The other args are coming by default
	assert.Len(t, d.Spec.Template.Spec.Containers[0].Args, 5)
	assert.Contains(t, d.Spec.Template.Spec.Containers[0].Args, "-javaagent:/agents/my-agent.jar")
	assert.Contains(t, d.Spec.Template.Spec.Containers[0].Args, "-javaagent:/agents/another-agent.jar=hello=world,my=test")
}

func TestApplyJvmTraitAgentFail(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)
	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: defaultContainerName,
						},
					},
				},
			},
		},
	}
	environment.Resources.Add(&d)

	trait.Agents = []string{"my-agent:url"}
	ok, cond, err := trait.Configure(environment)
	require.False(t, ok)
	require.Nil(t, cond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not parse JVM agent")
}

func TestApplyJvmTraitWithCACert(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)
	trait.CACertificates = []traitv1.CACertConfig{
		{CertPath: "/etc/camel/conf.d/_secrets/my-ca/ca.crt"},
	}
	trait.TruststorePasswordPath = "/etc/camel/conf.d/_secrets/truststore-pass/password"

	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: defaultContainerName,
						},
					},
				},
			},
		},
	}

	environment.Resources.Add(&d)
	configure, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.True(t, configure)
	assert.Nil(t, condition)

	err = trait.Apply(environment)
	require.NoError(t, err)

	assert.Contains(t, d.Spec.Template.Spec.Containers[0].Args, "-Djavax.net.ssl.trustStore=/etc/camel/conf.d/_truststore/truststore.jks")
	assert.Contains(t, d.Spec.Template.Spec.Containers[0].Args, "-Djavax.net.ssl.trustStorePassword=$(cat /etc/camel/conf.d/_secrets/truststore-pass/password)")
}

func TestApplyJvmTraitWithCustomCACertMountPath(t *testing.T) {
	trait, environment := createNominalJvmTest(v1.IntegrationKitTypePlatform)
	trait.CACertificates = []traitv1.CACertConfig{
		{CertPath: "/etc/camel/conf.d/_secrets/my-ca/ca.crt"},
	}
	trait.TruststorePasswordPath = "/etc/camel/conf.d/_secrets/truststore-pass/password"
	trait.CACertMountPath = "/custom/truststore/path"

	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: defaultContainerName,
						},
					},
				},
			},
		},
	}

	environment.Resources.Add(&d)
	configure, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.True(t, configure)
	assert.Nil(t, condition)

	err = trait.Apply(environment)
	require.NoError(t, err)

	assert.Contains(t, d.Spec.Template.Spec.Containers[0].Args, "-Djavax.net.ssl.trustStore=/custom/truststore/path/truststore.jks")
	assert.Contains(t, d.Spec.Template.Spec.Containers[0].Args, "-Djavax.net.ssl.trustStorePassword=$(cat /etc/camel/conf.d/_secrets/truststore-pass/password)")
}

func TestGetAllCACertEntries(t *testing.T) {
	trait, _ := createNominalJvmTest(v1.IntegrationKitTypePlatform)

	trait.CACertificates = []traitv1.CACertConfig{
		{CertPath: "/path/to/ca1.crt"},
		{CertPath: "/path/to/ca2.crt"},
	}
	trait.CACert = ""
	trait.CACertPassword = ""
	entries := trait.getAllCACertEntries()
	assert.Len(t, entries, 2)
	assert.Equal(t, "/path/to/ca1.crt", entries[0].CertPath)
	assert.Equal(t, "/path/to/ca2.crt", entries[1].CertPath)

	trait.CACertificates = nil
	trait.CACert = "/path/to/legacy.crt"
	trait.CACertPassword = "/path/to/legacy-pass"
	entries = trait.getAllCACertEntries()
	assert.Len(t, entries, 1)
	assert.Equal(t, "/path/to/legacy.crt", entries[0].CertPath)

	trait.CACertificates = []traitv1.CACertConfig{
		{CertPath: "/path/to/ca1.crt"},
	}
	trait.CACert = "/path/to/ca2.crt"
	trait.CACertPassword = "/path/to/pass2"
	entries = trait.getAllCACertEntries()
	assert.Len(t, entries, 2)

	trait.CACertificates = nil
	trait.CACert = ""
	trait.CACertPassword = ""
	entries = trait.getAllCACertEntries()
	assert.Len(t, entries, 0)
}

func TestHasCACerts(t *testing.T) {
	trait, _ := createNominalJvmTest(v1.IntegrationKitTypePlatform)

	trait.CACertificates = nil
	trait.CACert = ""
	assert.False(t, trait.hasCACerts())

	trait.CACertificates = []traitv1.CACertConfig{
		{CertPath: "/path/to/ca1.crt"},
	}
	trait.CACert = ""
	assert.True(t, trait.hasCACerts())

	trait.CACertificates = nil
	trait.CACert = "/path/to/legacy.crt"
	assert.True(t, trait.hasCACerts())

	trait.CACertificates = []traitv1.CACertConfig{
		{CertPath: "/path/to/ca1.crt"},
	}
	trait.CACert = "/path/to/ca2.crt"
	assert.True(t, trait.hasCACerts())
}

func TestHasBaseTruststore(t *testing.T) {
	trait, _ := createNominalJvmTest(v1.IntegrationKitTypePlatform)

	trait.BaseTruststore = nil
	assert.False(t, trait.hasBaseTruststore())

	trait.BaseTruststore = &traitv1.BaseTruststore{}
	assert.False(t, trait.hasBaseTruststore())

	trait.BaseTruststore = &traitv1.BaseTruststore{
		TruststorePath: "/path/to/cacerts",
	}
	assert.False(t, trait.hasBaseTruststore())

	trait.BaseTruststore = &traitv1.BaseTruststore{
		PasswordPath: "/path/to/pass",
	}
	assert.False(t, trait.hasBaseTruststore())

	trait.BaseTruststore = &traitv1.BaseTruststore{
		TruststorePath: "/opt/java/openjdk/lib/security/cacerts",
		PasswordPath:   "/path/to/pass",
	}
	assert.True(t, trait.hasBaseTruststore())
}

func TestGetTruststorePasswordPath(t *testing.T) {
	trait, _ := createNominalJvmTest(v1.IntegrationKitTypePlatform)

	trait.TruststorePasswordPath = ""
	trait.BaseTruststore = nil
	trait.CACert = ""
	trait.CACertPassword = ""
	assert.Equal(t, "", trait.getTruststorePasswordPath())

	trait.TruststorePasswordPath = "/path/to/user-provided-pass"
	trait.BaseTruststore = &traitv1.BaseTruststore{
		TruststorePath: "/path/to/cacerts",
		PasswordPath:   "/path/to/base-pass",
	}
	trait.CACert = "/path/to/legacy.crt"
	trait.CACertPassword = "/path/to/legacy-pass"
	assert.Equal(t, "/path/to/user-provided-pass", trait.getTruststorePasswordPath())

	trait.TruststorePasswordPath = "/path/to/user-provided-pass"
	trait.BaseTruststore = nil
	trait.CACert = "/path/to/legacy.crt"
	trait.CACertPassword = "/path/to/legacy-pass"
	assert.Equal(t, "/path/to/user-provided-pass", trait.getTruststorePasswordPath())

	trait.TruststorePasswordPath = ""
	trait.BaseTruststore = nil
	trait.CACert = "/path/to/legacy.crt"
	trait.CACertPassword = "/path/to/legacy-pass"
	assert.Equal(t, "/path/to/legacy-pass", trait.getTruststorePasswordPath())

	trait.TruststorePasswordPath = "/path/to/user-provided-pass"
	trait.BaseTruststore = &traitv1.BaseTruststore{
		TruststorePath: "/path/to/cacerts",
	}
	trait.CACert = ""
	trait.CACertPassword = ""
	assert.Equal(t, "/path/to/user-provided-pass", trait.getTruststorePasswordPath())

	trait.TruststorePasswordPath = ""
	trait.BaseTruststore = &traitv1.BaseTruststore{
		TruststorePath: "/path/to/cacerts",
	}
	trait.CACert = "/path/to/legacy.crt"
	trait.CACertPassword = "/path/to/legacy-pass"
	assert.Equal(t, "/path/to/legacy-pass", trait.getTruststorePasswordPath())

	trait.TruststorePasswordPath = ""
	trait.BaseTruststore = &traitv1.BaseTruststore{
		TruststorePath: "/path/to/cacerts",
	}
	trait.CACert = ""
	trait.CACertPassword = ""
	assert.Equal(t, "", trait.getTruststorePasswordPath())
}

func TestValidateCACertConfig(t *testing.T) {
	trait, _ := createNominalJvmTest(v1.IntegrationKitTypePlatform)

	trait.CACertificates = nil
	trait.CACert = ""
	trait.CACertPassword = ""
	trait.BaseTruststore = nil
	trait.TruststorePasswordPath = ""
	assert.NoError(t, trait.validateCACertConfig())

	trait.CACertificates = []traitv1.CACertConfig{
		{CertPath: "/path/to/ca.crt"},
	}
	trait.TruststorePasswordPath = "/path/to/truststore-pass"
	assert.NoError(t, trait.validateCACertConfig())

	trait.CACertificates = []traitv1.CACertConfig{
		{CertPath: "/path/to/ca.crt"},
	}
	trait.TruststorePasswordPath = ""
	trait.BaseTruststore = nil
	assert.Error(t, trait.validateCACertConfig())
	assert.Contains(t, trait.validateCACertConfig().Error(), "truststore-password-path is required")

	trait.CACertificates = []traitv1.CACertConfig{
		{CertPath: "/path/to/ca.crt"},
	}
	trait.TruststorePasswordPath = ""
	trait.BaseTruststore = &traitv1.BaseTruststore{
		TruststorePath: "/path/to/cacerts",
		PasswordPath:   "/path/to/base-pass",
	}
	assert.NoError(t, trait.validateCACertConfig())

	trait.CACertificates = []traitv1.CACertConfig{
		{CertPath: "/path/to/ca.crt"},
	}
	trait.TruststorePasswordPath = "/path/to/truststore-pass"
	trait.BaseTruststore = nil
	assert.NoError(t, trait.validateCACertConfig())

	trait.CACertificates = []traitv1.CACertConfig{
		{CertPath: "/path/to/ca.crt"},
	}
	trait.TruststorePasswordPath = ""
	trait.BaseTruststore = nil
	trait.CACert = "/path/to/legacy.crt"
	trait.CACertPassword = "/path/to/legacy-pass"
	assert.Error(t, trait.validateCACertConfig())
	assert.Contains(t, trait.validateCACertConfig().Error(), "truststore-password-path is required")

	trait.CACertificates = nil
	trait.CACert = "/path/to/ca.crt"
	trait.CACertPassword = ""
	trait.TruststorePasswordPath = ""
	assert.Error(t, trait.validateCACertConfig())

	trait.CACert = "/path/to/ca.crt"
	trait.CACertPassword = "/path/to/pass"
	assert.NoError(t, trait.validateCACertConfig())

	trait.CACert = ""
	trait.CACertPassword = ""
	trait.BaseTruststore = &traitv1.BaseTruststore{
		TruststorePath: "/path/to/cacerts",
	}
	assert.Error(t, trait.validateCACertConfig())

	trait.BaseTruststore = &traitv1.BaseTruststore{
		TruststorePath: "/path/to/cacerts",
		PasswordPath:   "/path/to/pass",
	}
	assert.NoError(t, trait.validateCACertConfig())
}

func TestAllCACertScenarios(t *testing.T) {
	t.Run("Scenario1_NothingConfigured", func(t *testing.T) {
		trait, _ := createNominalJvmTest(v1.IntegrationKitTypePlatform)
		trait.CACertificates = nil
		trait.TruststorePasswordPath = ""
		trait.BaseTruststore = nil
		trait.CACert = ""
		trait.CACertPassword = ""

		assert.False(t, trait.hasCACerts())
		assert.Equal(t, "", trait.getTruststorePasswordPath())
	})

	t.Run("Scenario2_CACertificatesNoPassword", func(t *testing.T) {
		trait, _ := createNominalJvmTest(v1.IntegrationKitTypePlatform)
		trait.CACertificates = []traitv1.CACertConfig{
			{CertPath: "/path/to/cert"},
		}
		trait.TruststorePasswordPath = ""
		trait.BaseTruststore = nil
		trait.CACert = ""
		trait.CACertPassword = ""

		assert.True(t, trait.hasCACerts())
		assert.Equal(t, "", trait.getTruststorePasswordPath())
		err := trait.validateCACertConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "truststore-password-path is required")
	})

	t.Run("Scenario3_CACertificatesWithTruststorePassword", func(t *testing.T) {
		trait, _ := createNominalJvmTest(v1.IntegrationKitTypePlatform)
		trait.CACertificates = []traitv1.CACertConfig{
			{CertPath: "/path/to/cert"},
		}
		trait.TruststorePasswordPath = "/path/to/ts-pass"
		trait.BaseTruststore = nil
		trait.CACert = ""
		trait.CACertPassword = ""

		assert.True(t, trait.hasCACerts())
		assert.Equal(t, "/path/to/ts-pass", trait.getTruststorePasswordPath())
		assert.NoError(t, trait.validateCACertConfig())
	})

	t.Run("Scenario4_CACertificatesWithBaseTruststore", func(t *testing.T) {
		trait, _ := createNominalJvmTest(v1.IntegrationKitTypePlatform)
		trait.CACertificates = []traitv1.CACertConfig{
			{CertPath: "/path/to/cert"},
		}
		trait.TruststorePasswordPath = ""
		trait.BaseTruststore = &traitv1.BaseTruststore{
			TruststorePath: "/path/to/cacerts",
			PasswordPath:   "/path/to/base-pass",
		}
		trait.CACert = ""
		trait.CACertPassword = ""

		assert.True(t, trait.hasCACerts())
		assert.True(t, trait.hasBaseTruststore())
		assert.Equal(t, "/path/to/base-pass", trait.getTruststorePasswordPath())
		assert.NoError(t, trait.validateCACertConfig())
	})

	t.Run("Scenario5_AllPasswordSourcesTruststorePasswordWins", func(t *testing.T) {
		trait, _ := createNominalJvmTest(v1.IntegrationKitTypePlatform)
		trait.CACertificates = []traitv1.CACertConfig{
			{CertPath: "/path/to/cert"},
		}
		trait.TruststorePasswordPath = "/path/to/ts-pass"
		trait.BaseTruststore = &traitv1.BaseTruststore{
			TruststorePath: "/path/to/cacerts",
			PasswordPath:   "/path/to/base-pass",
		}
		trait.CACert = ""
		trait.CACertPassword = ""

		assert.True(t, trait.hasCACerts())
		assert.True(t, trait.hasBaseTruststore())
		assert.Equal(t, "/path/to/ts-pass", trait.getTruststorePasswordPath())
		assert.NoError(t, trait.validateCACertConfig())
	})

	t.Run("Scenario6_CACertificatesWithLegacyNoExplicitPassword", func(t *testing.T) {
		trait, _ := createNominalJvmTest(v1.IntegrationKitTypePlatform)
		trait.CACertificates = []traitv1.CACertConfig{
			{CertPath: "/path/to/cert"},
		}
		trait.TruststorePasswordPath = ""
		trait.BaseTruststore = nil
		trait.CACert = "/path/to/legacy-cert"
		trait.CACertPassword = "/path/to/legacy-pass"

		assert.True(t, trait.hasCACerts())
		assert.Equal(t, "/path/to/legacy-pass", trait.getTruststorePasswordPath())
		err := trait.validateCACertConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "truststore-password-path is required")
	})

	t.Run("Scenario7_LegacyOnly", func(t *testing.T) {
		trait, _ := createNominalJvmTest(v1.IntegrationKitTypePlatform)
		trait.CACertificates = nil
		trait.TruststorePasswordPath = ""
		trait.BaseTruststore = nil
		trait.CACert = "/path/to/legacy-cert"
		trait.CACertPassword = "/path/to/legacy-pass"

		assert.True(t, trait.hasCACerts())
		assert.Equal(t, "/path/to/legacy-pass", trait.getTruststorePasswordPath())
		assert.NoError(t, trait.validateCACertConfig())
	})

	t.Run("Scenario8_LegacyWithoutPassword", func(t *testing.T) {
		trait, _ := createNominalJvmTest(v1.IntegrationKitTypePlatform)
		trait.CACertificates = nil
		trait.TruststorePasswordPath = ""
		trait.BaseTruststore = nil
		trait.CACert = "/path/to/legacy-cert"
		trait.CACertPassword = ""

		assert.True(t, trait.hasCACerts())
		assert.Equal(t, "", trait.getTruststorePasswordPath())
		err := trait.validateCACertConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ca-cert-password is required")
	})

	t.Run("Scenario9_BaseTruststoreOnlyNoCerts", func(t *testing.T) {
		trait, _ := createNominalJvmTest(v1.IntegrationKitTypePlatform)
		trait.CACertificates = nil
		trait.TruststorePasswordPath = ""
		trait.BaseTruststore = &traitv1.BaseTruststore{
			TruststorePath: "/path/to/cacerts",
			PasswordPath:   "/path/to/base-pass",
		}
		trait.CACert = ""
		trait.CACertPassword = ""

		assert.False(t, trait.hasCACerts())
		assert.True(t, trait.hasBaseTruststore())
		assert.Equal(t, "/path/to/base-pass", trait.getTruststorePasswordPath())
		assert.NoError(t, trait.validateCACertConfig())
	})

	t.Run("Scenario10_IncompleteBaseTruststoreWithTsPassword", func(t *testing.T) {
		trait, _ := createNominalJvmTest(v1.IntegrationKitTypePlatform)
		trait.CACertificates = []traitv1.CACertConfig{
			{CertPath: "/path/to/cert"},
		}
		trait.TruststorePasswordPath = "/path/to/ts-pass"
		trait.BaseTruststore = &traitv1.BaseTruststore{
			TruststorePath: "/path/to/cacerts",
			PasswordPath:   "",
		}
		trait.CACert = ""
		trait.CACertPassword = ""

		assert.True(t, trait.hasCACerts())
		assert.False(t, trait.hasBaseTruststore())
		assert.Equal(t, "/path/to/ts-pass", trait.getTruststorePasswordPath())

		err := trait.validateCACertConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "base-truststore: both truststore-path and password-path are required")
	})

	t.Run("Scenario11_IncompleteBaseTruststoreNoTsPassword", func(t *testing.T) {
		trait, _ := createNominalJvmTest(v1.IntegrationKitTypePlatform)
		trait.CACertificates = []traitv1.CACertConfig{
			{CertPath: "/path/to/cert"},
		}
		trait.TruststorePasswordPath = ""
		trait.BaseTruststore = &traitv1.BaseTruststore{
			TruststorePath: "/path/to/cacerts",
			PasswordPath:   "",
		}
		trait.CACert = ""
		trait.CACertPassword = ""

		assert.True(t, trait.hasCACerts())
		assert.False(t, trait.hasBaseTruststore())
		assert.Equal(t, "", trait.getTruststorePasswordPath())
		err := trait.validateCACertConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "base-truststore: both truststore-path and password-path are required")
	})

	t.Run("Scenario12_EmptyCACertificatesArray", func(t *testing.T) {
		trait, _ := createNominalJvmTest(v1.IntegrationKitTypePlatform)
		trait.CACertificates = []traitv1.CACertConfig{}
		trait.TruststorePasswordPath = ""
		trait.BaseTruststore = nil
		trait.CACert = ""
		trait.CACertPassword = ""

		assert.False(t, trait.hasCACerts())
		assert.Equal(t, "", trait.getTruststorePasswordPath())
		assert.NoError(t, trait.validateCACertConfig())
	})

	t.Run("Scenario13_CACertificatesEntryMissingCertPath", func(t *testing.T) {
		trait, _ := createNominalJvmTest(v1.IntegrationKitTypePlatform)
		trait.CACertificates = []traitv1.CACertConfig{
			{CertPath: ""},
		}
		trait.TruststorePasswordPath = "/path/to/ts-pass"
		trait.BaseTruststore = nil
		trait.CACert = ""
		trait.CACertPassword = ""

		assert.True(t, trait.hasCACerts())
		err := trait.validateCACertConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cert-path is required")
	})
}
