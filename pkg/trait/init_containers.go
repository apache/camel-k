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
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

const (
	initContainerTraitID    = "init-containers"
	initContainerTraitOrder = 1610
)

type containerTask struct {
	name          string
	image         string
	command       string
	isSidecar     bool
	requestCPU    string
	requestMemory string
	limitCPU      string
	limitMemory   string
	env           []corev1.EnvVar
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
		if ok && jvm.hasCACerts() {
			if err := jvm.validateCACertConfig(); err != nil {
				return false, nil, err
			}

			var allCommands []string
			importPassPath := jvm.getTruststorePasswordPath()

			if jvm.hasBaseTruststore() {
				baseTruststore := jvm.getBaseTruststore()
				copyCmd := fmt.Sprintf("cp %s %s", baseTruststore.TruststorePath, jvm.getTrustStorePath())
				allCommands = append(allCommands, copyCmd)
				importPassPath = baseTruststore.PasswordPath
			}

			for i, entry := range jvm.getAllCACertEntries() {
				cmd := fmt.Sprintf(
					"keytool -importcert -noprompt -alias custom-ca-%d -storepass:file %s -keystore %s -file %s",
					i, importPassPath, jvm.getTrustStorePath(), entry.CertPath,
				)
				allCommands = append(allCommands, cmd)
			}

			if jvm.hasBaseTruststore() && jvm.TruststorePasswordPath != "" {
				storepasswdCmd := fmt.Sprintf(
					"keytool -storepasswd -keystore %s -storepass:file %s -new \"$(cat %s)\"",
					jvm.getTrustStorePath(), jvm.getBaseTruststore().PasswordPath, jvm.TruststorePasswordPath,
				)
				allCommands = append(allCommands, storepasswdCmd)
			}

			fullCommand := strings.Join(allCommands, " && ")
			if len(allCommands) > 1 || jvm.hasBaseTruststore() {
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
		if task.requestCPU != "" || task.requestMemory != "" || task.limitCPU != "" || task.limitMemory != "" {
			initCont.Resources = corev1.ResourceRequirements{
				Requests: corev1.ResourceList{},
				Limits:   corev1.ResourceList{},
			}
			if task.requestCPU != "" {
				initCont.Resources.Requests[corev1.ResourceCPU] = resource.MustParse(task.requestCPU)
			}
			if task.requestMemory != "" {
				initCont.Resources.Requests[corev1.ResourceMemory] = resource.MustParse(task.requestMemory)
			}
			if task.limitCPU != "" {
				initCont.Resources.Limits[corev1.ResourceCPU] = resource.MustParse(task.limitCPU)
			}
			if task.limitMemory != "" {
				initCont.Resources.Limits[corev1.ResourceMemory] = resource.MustParse(task.limitMemory)
			}
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
		parsed, err := parseSingleTask(task, false)
		if err != nil {
			return false, fmt.Errorf("could not parse init container task %q: %w", task, err)
		}
		t.tasks[i] = parsed
		i++
	}
	for _, task := range t.SidecarTasks {
		parsed, err := parseSingleTask(task, true)
		if err != nil {
			return false, fmt.Errorf("could not parse sidecar container task %q: %w", task, err)
		}
		t.tasks[i] = parsed
		i++
	}

	return true, nil
}

func parseSingleTask(task string, isSidecar bool) (containerTask, error) {
	segments := strings.Split(task, ";")

	var result containerTask
	result.isSidecar = isSidecar

	var commandParts []string

	for _, seg := range segments {
		if len(strings.TrimSpace(seg)) == 0 {
			continue
		}
		if strings.Contains(seg, "=") {
			kv := strings.SplitN(seg, "=", 2)
			key, value := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
			switch key {
			case "name":
				result.name = value
			case "image":
				result.image = value
			case "command":
				commandParts = append(commandParts, value)
			case "request-cpu":
				if _, err := resource.ParseQuantity(value); err != nil {
					return containerTask{}, fmt.Errorf("invalid request-cpu value %q: %w", value, err)
				}
				result.requestCPU = value
			case "request-memory":
				if _, err := resource.ParseQuantity(value); err != nil {
					return containerTask{}, fmt.Errorf("invalid request-memory value %q: %w", value, err)
				}
				result.requestMemory = value
			case "limit-cpu":
				if _, err := resource.ParseQuantity(value); err != nil {
					return containerTask{}, fmt.Errorf("invalid limit-cpu value %q: %w", value, err)
				}
				result.limitCPU = value
			case "limit-memory":
				if _, err := resource.ParseQuantity(value); err != nil {
					return containerTask{}, fmt.Errorf("invalid limit-memory value %q: %w", value, err)
				}
				result.limitMemory = value
			default:
				// Forward compatibility: unknown keys are appended to command
				commandParts = append(commandParts, seg)
			}
		} else {
			// Positional segment: fill first unset field
			if result.name == "" {
				result.name = seg
			} else if result.image == "" {
				result.image = seg
			} else {
				commandParts = append(commandParts, seg)
			}
		}
	}

	if len(commandParts) > 0 {
		result.command = strings.Join(commandParts, ";")
	}

	if result.name == "" || result.image == "" {
		return containerTask{}, fmt.Errorf("name and image are required (format: %q)", "name;image;command or name=...;image=...")
	}

	return result, nil
}
