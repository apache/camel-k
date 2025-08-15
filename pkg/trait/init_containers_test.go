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
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestParseInitContainerNoIntegration(t *testing.T) {
	environment := Environment{
		Resources: kubernetes.NewCollection(),
	}
	initCont := newInitContainersTrait()
	configured, condition, err := initCont.Configure(&environment)
	assert.False(t, configured, "Should not be configured, Integration is nil")
	assert.Nil(t, condition, "should be nil")
	assert.Nil(t, err, "should be nil")
}

func TestParseInitContainerShouldFail(t *testing.T) {
	environment := Environment{
		Resources: kubernetes.NewCollection(),
		Integration: &v1.Integration{
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseRunning,
			},
		},
	}
	initCont := initContainersTrait{
		InitContainersTrait: trait.InitContainersTrait{
			InitTasks: []string{
				"not a valid format",
			},
		},
	}
	configured, condition, err := initCont.Configure(&environment)
	assert.False(t, configured, "Should not be configured, there's an error")
	assert.Nil(t, condition)
	assert.NotNil(t, err)
	assert.Equal(t, `could not parse init container task "not a valid format": format expected "name;container-image;command"`, err.Error())
}

func TestParseInitContainerOK(t *testing.T) {
	environment := Environment{
		Resources: kubernetes.NewCollection(),
		Integration: &v1.Integration{
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseRunning,
			},
		},
	}
	initCont := initContainersTrait{
		InitContainersTrait: trait.InitContainersTrait{
			InitTasks: []string{
				"agent;my-agent-image:1.2.3;echo \"hello\"",
				"more thing;my-image;echo \"hello\" && something else",
			},
		},
	}
	configured, condition, err := initCont.Configure(&environment)
	assert.True(t, configured)
	assert.Nil(t, condition)
	require.Nil(t, err)
	require.Len(t, initCont.tasks, 2)
	assert.Equal(t, "agent", initCont.tasks[0].name)
	assert.Equal(t, "my-agent-image:1.2.3", initCont.tasks[0].image)
	assert.Equal(t, "echo \"hello\"", initCont.tasks[0].command)
	assert.Equal(t, "more thing", initCont.tasks[1].name)
	assert.Equal(t, "my-image", initCont.tasks[1].image)
	assert.Equal(t, "echo \"hello\" && something else", initCont.tasks[1].command)
}

func TestParseInitContainerDefault(t *testing.T) {
	environment := Environment{
		Resources: kubernetes.NewCollection(),
		Integration: &v1.Integration{
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseRunning,
			},
		},
	}
	initCont := initContainersTrait{}
	configured, condition, err := initCont.Configure(&environment)
	assert.False(t, configured)
	assert.Nil(t, condition)
	require.Nil(t, err)
}

func TestApplyInitContainerOK(t *testing.T) {
	environment := Environment{
		Resources: kubernetes.NewCollection(),
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-it",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseRunning,
			},
		},
	}
	environment.Resources.Add(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				v1.IntegrationLabel: "my-it",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{},
			},
		},
	})
	initCont := initContainersTrait{
		InitContainersTrait: trait.InitContainersTrait{
			InitTasks: []string{
				"agent;my-agent-image:1.2.3;echo hello",
			},
		},
	}
	configured, condition, err := initCont.Configure(&environment)
	assert.True(t, configured)
	assert.Nil(t, condition)
	require.Nil(t, err)
	err = initCont.Apply(&environment)
	require.Nil(t, err)
	deploy := environment.Resources.GetDeploymentForIntegration(environment.Integration)
	require.NotNil(t, deploy)

	require.Len(t, deploy.Spec.Template.Spec.InitContainers, 1)
	assert.Equal(t, "agent", deploy.Spec.Template.Spec.InitContainers[0].Name)
	assert.Equal(t, "my-agent-image:1.2.3", deploy.Spec.Template.Spec.InitContainers[0].Image)
	assert.Equal(t, []string{"echo", "hello"}, deploy.Spec.Template.Spec.InitContainers[0].Command)
	assert.NotEqual(t, ptr.To(corev1.ContainerRestartPolicyAlways), deploy.Spec.Template.Spec.InitContainers[0].RestartPolicy)
}

func TestApplyInitContainerSidecarOK(t *testing.T) {
	environment := Environment{
		Resources: kubernetes.NewCollection(),
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-it",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseRunning,
			},
		},
	}
	environment.Resources.Add(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				v1.IntegrationLabel: "my-it",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{},
			},
		},
	})
	initCont := initContainersTrait{
		InitContainersTrait: trait.InitContainersTrait{
			SidecarTasks: []string{
				"agent;my-agent-image:1.2.3;echo hello",
			},
		},
	}
	configured, condition, err := initCont.Configure(&environment)
	assert.True(t, configured)
	assert.Nil(t, condition)
	require.Nil(t, err)
	err = initCont.Apply(&environment)
	require.Nil(t, err)
	deploy := environment.Resources.GetDeploymentForIntegration(environment.Integration)
	require.NotNil(t, deploy)

	require.Len(t, deploy.Spec.Template.Spec.InitContainers, 1)
	assert.Equal(t, "agent", deploy.Spec.Template.Spec.InitContainers[0].Name)
	assert.Equal(t, "my-agent-image:1.2.3", deploy.Spec.Template.Spec.InitContainers[0].Image)
	assert.Equal(t, []string{"echo", "hello"}, deploy.Spec.Template.Spec.InitContainers[0].Command)
	assert.Equal(t, ptr.To(corev1.ContainerRestartPolicyAlways), deploy.Spec.Template.Spec.InitContainers[0].RestartPolicy)
}

func TestApplyInitContainerAndSidecarOK(t *testing.T) {
	environment := Environment{
		Resources: kubernetes.NewCollection(),
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-it",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseRunning,
			},
		},
	}
	environment.Resources.Add(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				v1.IntegrationLabel: "my-it",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{},
			},
		},
	})
	initCont := initContainersTrait{
		InitContainersTrait: trait.InitContainersTrait{
			SidecarTasks: []string{
				"sidecar;my-agent-image:1.2.3;echo sidecar",
			},
			InitTasks: []string{
				"init;my-agent-image:3.2.1;echo init",
			},
		},
	}
	configured, condition, err := initCont.Configure(&environment)
	assert.True(t, configured)
	assert.Nil(t, condition)
	require.Nil(t, err)
	err = initCont.Apply(&environment)
	require.Nil(t, err)
	deploy := environment.Resources.GetDeploymentForIntegration(environment.Integration)
	require.NotNil(t, deploy)

	require.Len(t, deploy.Spec.Template.Spec.InitContainers, 2)
	assert.Contains(t, deploy.Spec.Template.Spec.InitContainers, corev1.Container{
		Name:          "sidecar",
		Image:         "my-agent-image:1.2.3",
		Command:       []string{"echo", "sidecar"},
		RestartPolicy: ptr.To(corev1.ContainerRestartPolicyAlways),
	})
	assert.Contains(t, deploy.Spec.Template.Spec.InitContainers, corev1.Container{
		Name:          "init",
		Image:         "my-agent-image:3.2.1",
		Command:       []string{"echo", "init"},
		RestartPolicy: nil,
	})
}

func TestParseTasks(t *testing.T) {
	trait := &initContainersTrait{
		InitContainersTrait: trait.InitContainersTrait{
			InitTasks:    []string{"name;image;init-command"},
			SidecarTasks: []string{"name;image;sidecar-command;done"},
		},
	}

	trait.parseTasks()
	assert.Contains(t, trait.tasks, containerTask{
		name:      "name",
		image:     "image",
		command:   "init-command",
		isSidecar: false,
	})
	assert.Contains(t, trait.tasks, containerTask{
		name:      "name",
		image:     "image",
		command:   "sidecar-command;done",
		isSidecar: true,
	})
}
