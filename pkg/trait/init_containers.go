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
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	serving "knative.dev/serving/pkg/apis/serving/v1"

	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"k8s.io/utils/ptr"
)

const (
	initContainerTraitID    = "init-containers"
	initContainerTraitOrder = 1610
)

type containerTask struct {
	name      string
	image     string
	command   string
	isSidecar bool
	env       []corev1.EnvVar
}

type initContainersTrait struct {
	BaseTrait
	traitv1.InitContainersTrait `property:",squash"`

	tasks []containerTask
}

func newInitContainersTrait() Trait {
	return &initContainersTrait{
		BaseTrait: NewBaseTrait(initContainerTraitID, initContainerTraitOrder),
	}
}

func (t *initContainersTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil || !e.IntegrationInRunningPhases() {
		return false, nil, nil
	}

	if ok, err := t.parseTasks(); err != nil {
		return ok, nil, err
	}

	// Set the agent init container if any agent exists
	trait := e.Catalog.GetTrait(jvmTraitID)
	//nolint: nestif
	if trait != nil {
		jvm, ok := trait.(*jvmTrait)
		if ok && jvm.hasJavaAgents() {
			jvmAgents, err := jvm.parseJvmAgents()
			if err != nil {
				return false, nil, err
			}
			curlDownloadAgents := ""
			for _, agent := range jvmAgents {
				if curlDownloadAgents != "" {
					curlDownloadAgents += " && "
				}
				curlDownloadAgents += fmt.Sprintf("curl -o %s/%s.jar %s", defaultAgentDir, agent.name, agent.url)
			}
			agentDownloadTask := containerTask{
				name:    defaultAgentInitContainerName,
				image:   defaults.BaseImage(),
				command: fmt.Sprintf("/bin/bash -c \"%s\"", curlDownloadAgents),
			}
			t.tasks = append(t.tasks, agentDownloadTask)
		}
		// Set the CA cert truststore init container if configured
		if ok && jvm.hasCACerts() {
			var allCommands []string
			effectivePassword := jvm.getEffectiveTruststorePassword()

			if jvm.useSystemTruststore() {
				copyCmd := fmt.Sprintf("cp %s %s", jdkCacertsPath, jvm.getTrustStorePath())
				allCommands = append(allCommands, copyCmd)

				if jvm.hasCustomPassword() {
					changePassCmd := fmt.Sprintf(
						"keytool -storepasswd -keystore %s -storepass %s -new %s",
						jvm.getTrustStorePath(), jdkCacertsDefaultPassword, effectivePassword,
					)
					allCommands = append(allCommands, changePassCmd)
				}
			}

			certPaths := jvm.getAllCACertPaths()
			for i, certPath := range certPaths {
				var cmd string
				if jvm.hasCustomPassword() {
					cmd = fmt.Sprintf(
						"keytool -importcert -noprompt -alias custom-ca-%d -storepass:file %s -keystore %s -file %s",
						i, jvm.getCACertPasswordPath(), jvm.getTrustStorePath(), certPath,
					)
				} else {
					cmd = fmt.Sprintf(
						"keytool -importcert -noprompt -alias custom-ca-%d -storepass %s -keystore %s -file %s",
						i, jdkCacertsDefaultPassword, jvm.getTrustStorePath(), certPath,
					)
				}
				allCommands = append(allCommands, cmd)
			}

			fullCommand := strings.Join(allCommands, " && ")
			// Wrap in bash shell when there are multiple commands or shell features are used
			if len(allCommands) > 1 || jvm.useSystemTruststore() {
				fullCommand = fmt.Sprintf("/bin/bash -c \"%s\"", fullCommand)
			}
			caCertTask := containerTask{
				name:    "generate-truststore",
				image:   defaults.BaseImage(),
				command: fullCommand,
			}
			t.tasks = append(t.tasks, caCertTask)
		}
	}

	return len(t.tasks) > 0, nil, nil
}

func (t *initContainersTrait) Apply(e *Environment) error {
	var initContainers *[]corev1.Container

	if err := e.Resources.VisitDeploymentE(func(deployment *appsv1.Deployment) error {
		// Deployment
		initContainers = &deployment.Spec.Template.Spec.InitContainers

		return nil
	}); err != nil {
		return err
	} else if err := e.Resources.VisitKnativeServiceE(func(service *serving.Service) error {
		// Knative Service
		initContainers = &service.Spec.Template.Spec.InitContainers

		return nil
	}); err != nil {
		return err
	} else if err := e.Resources.VisitCronJobE(func(cron *batchv1.CronJob) error {
		// CronJob
		initContainers = &cron.Spec.JobTemplate.Spec.Template.Spec.InitContainers

		return nil
	}); err != nil {
		return err
	}

	t.configureContainers(initContainers)

	return nil
}

func (t *initContainersTrait) configureContainers(containers *[]corev1.Container) {
	if containers == nil {
		containers = &[]corev1.Container{}
	}
	for _, task := range t.tasks {
		initCont := corev1.Container{
			Name:    task.name,
			Image:   task.image,
			Command: splitContainerCommand(task.command),
			Env:     task.env,
		}
		if task.isSidecar {
			initCont.RestartPolicy = ptr.To(corev1.ContainerRestartPolicyAlways)
		}
		*containers = append(*containers, initCont)
	}
}

func (t *initContainersTrait) parseTasks() (bool, error) {
	if t.InitTasks == nil && t.SidecarTasks == nil {
		return false, nil
	}
	t.tasks = make([]containerTask, len(t.InitTasks)+len(t.SidecarTasks))
	i := 0
	for _, task := range t.InitTasks {
		split := strings.SplitN(task, ";", 3)
		if len(split) != 3 {
			return false, fmt.Errorf(`could not parse init container task "%s": format expected "name;container-image;command"`, task)
		}
		t.tasks[i] = containerTask{
			name:      split[0],
			image:     split[1],
			command:   split[2],
			isSidecar: false,
		}
		i++
	}
	for _, task := range t.SidecarTasks {
		split := strings.SplitN(task, ";", 3)
		if len(split) != 3 {
			return false, fmt.Errorf(`could not parse sidecar container task "%s": format expected "name;container-image;command"`, task)
		}
		t.tasks[i] = containerTask{
			name:      split[0],
			image:     split[1],
			command:   split[2],
			isSidecar: true,
		}
		i++
	}

	return true, nil
}
