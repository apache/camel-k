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
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	serving "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/digest"
	"github.com/apache/camel-k/v2/pkg/util/envvar"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/openshift"
)

const (
	containerTraitID    = "container"
	containerTraitOrder = 1600

	defaultContainerName = "integration"
	defaultContainerPort = 8080
	defaultServicePort   = 80

	defaultContainerRunAsNonRoot             = false
	defaultContainerSeccompProfileType       = corev1.SeccompProfileTypeRuntimeDefault
	defaultContainerAllowPrivilegeEscalation = false
	defaultContainerCapabilitiesDrop         = "ALL"

	defaultContainerResourceCPU    = "125m"
	defaultContainerResourceMemory = "128Mi"
	defaultContainerLimitCPU       = "500m"
	defaultContainerLimitMemory    = "512Mi"
)

type containerTrait struct {
	BasePlatformTrait
	traitv1.ContainerTrait `property:",squash"`

	containerPorts []containerPort
}

// containerPort is supporting port parsing.
type containerPort struct {
	name     string
	port     int32
	protocol string
}

func newContainerTrait() Trait {
	return &containerTrait{
		BasePlatformTrait: NewBasePlatformTrait(containerTraitID, containerTraitOrder),
	}
}

func (t *containerTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil {
		return false, nil, nil
	}

	if !e.IntegrationInPhase(v1.IntegrationPhaseInitialization) && !e.IntegrationInRunningPhases() {
		return false, nil, nil
	}

	if ptr.Deref(t.Auto, true) {
		if t.Expose == nil {
			if e.Resources.GetServiceForIntegration(e.Integration) != nil {
				t.Expose = ptr.To(true)
			}
		}
	}

	if !isValidPullPolicy(t.ImagePullPolicy) {
		return false, nil, fmt.Errorf("unsupported pull policy %s", t.ImagePullPolicy)
	}
	containerPorts, err := t.parseContainerPorts()
	if err != nil {
		return false, nil, err
	}
	t.containerPorts = containerPorts

	return true, nil, nil
}

func isValidPullPolicy(policy corev1.PullPolicy) bool {
	return policy == "" || policy == corev1.PullAlways || policy == corev1.PullIfNotPresent || policy == corev1.PullNever
}

func (t *containerTrait) Apply(e *Environment) error {
	if err := t.configureImageIntegrationKit(e); err != nil {
		return err
	}
	return t.configureContainer(e)
}

func (t *containerTrait) configureImageIntegrationKit(e *Environment) error {
	if t.Image == "" {
		return nil
	}

	if e.Integration.Spec.IntegrationKit != nil {
		return fmt.Errorf(
			"unsupported configuration: a container image has been set in conjunction with an IntegrationKit %v",
			e.Integration.Spec.IntegrationKit)
	}

	e.Integration.Status.Image = t.Image

	return nil
}

func (t *containerTrait) configureContainer(e *Environment) error {
	if e.ApplicationProperties == nil {
		e.ApplicationProperties = make(map[string]string)
	}
	container := corev1.Container{
		Name:  t.getContainerName(),
		Image: e.Integration.Status.Image,
		Env:   make([]corev1.EnvVar, 0),
	}
	if t.ImagePullPolicy != "" {
		container.ImagePullPolicy = t.ImagePullPolicy
	}
	// combine Environment of integration with platform, kit, integration
	for _, env := range e.collectConfigurationPairs("env") {
		envvar.SetVal(&container.Env, env.Name, env.Value)
	}
	envvar.SetVal(&container.Env, digest.IntegrationDigestEnvVar, e.Integration.Status.Digest)
	envvar.SetVal(&container.Env, "CAMEL_K_CONF", filepath.Join(camel.BasePath, "application.properties"))
	envvar.SetVal(&container.Env, "CAMEL_K_CONF_D", camel.ConfDPath)

	var containers *[]corev1.Container
	visited := false
	knative := false
	// Deployment
	if err := e.Resources.VisitDeploymentE(func(deployment *appsv1.Deployment) error {
		for _, envVar := range e.EnvVars {
			envvar.SetVar(&container.Env, envVar)
		}
		containers = &deployment.Spec.Template.Spec.Containers
		visited = true
		return nil
	}); err != nil {
		return err
	}
	// Knative Service
	if err := e.Resources.VisitKnativeServiceE(func(service *serving.Service) error {
		for _, env := range e.EnvVars {
			switch {
			case env.ValueFrom == nil:
				envvar.SetVar(&container.Env, env)
			case env.ValueFrom.FieldRef != nil && env.ValueFrom.FieldRef.FieldPath == "metadata.namespace":
				envvar.SetVar(&container.Env, corev1.EnvVar{Name: env.Name, Value: e.Integration.Namespace})
			case env.ValueFrom.FieldRef != nil:
				t.L.Debugf("Skipping environment variable %s (fieldRef)", env.Name)
			case env.ValueFrom.ResourceFieldRef != nil:
				t.L.Debugf("Skipping environment variable %s (resourceFieldRef)", env.Name)
			default:
				envvar.SetVar(&container.Env, env)
			}
		}
		containers = &service.Spec.Template.Spec.Containers
		visited = true
		knative = true
		return nil
	}); err != nil {
		return err
	}
	// CronJob
	if err := e.Resources.VisitCronJobE(func(cron *batchv1.CronJob) error {
		for _, envVar := range e.EnvVars {
			envvar.SetVar(&container.Env, envVar)
		}
		containers = &cron.Spec.JobTemplate.Spec.Template.Spec.Containers
		visited = true
		return nil
	}); err != nil {
		return err
	}
	t.configureResources(&container)
	if !knative {
		// Knative does not like anybody touching the container ports
		t.configurePorts(&container)
	}
	if knative || ptr.Deref(t.Expose, false) {
		t.configureService(e, &container, knative)
	}
	t.configureCapabilities(e)
	err := t.setSecurityContext(e, &container)
	if err != nil {
		return err
	}
	if visited {
		*containers = append(*containers, container)
	}

	return nil
}

func (t *containerTrait) configurePorts(container *corev1.Container) {
	for _, cp := range t.containerPorts {
		containerPort := corev1.ContainerPort{
			Name:          cp.name,
			ContainerPort: cp.port,
			Protocol:      corev1.Protocol(cp.protocol),
		}
		container.Ports = append(container.Ports, containerPort)
	}
}

func (t *containerTrait) parseContainerPorts() ([]containerPort, error) {
	containerPorts := make([]containerPort, 0, len(t.Ports))
	for _, port := range t.Ports {
		portSplit := strings.Split(port, ";")
		if len(portSplit) < 2 {
			return nil, fmt.Errorf("could not parse container port %s properly: expected format \"port-name;port-number[;port-protocol]\"", port)
		}
		portInt32, err := strconv.ParseInt(portSplit[1], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("could not parse container port number in %s properly: expected port-number as a number", port)
		}
		cp := containerPort{
			name: portSplit[0],
			port: int32(portInt32),
		}
		if len(portSplit) > 2 {
			cp.protocol = portSplit[2]
		} else {
			cp.protocol = "TCP"
		}
		containerPorts = append(containerPorts, cp)
	}

	return containerPorts, nil
}

func (t *containerTrait) configureService(e *Environment, container *corev1.Container, isKnative bool) {
	name := t.PortName
	if name == "" {
		name = e.determineDefaultContainerPortName()
	}
	containerPort := corev1.ContainerPort{
		Name:          name,
		ContainerPort: t.getPort(),
		Protocol:      corev1.ProtocolTCP,
	}
	if !isKnative {
		// The service is managed by Knative, so, we only take care of this when it's managed by us
		service := e.Resources.GetServiceForIntegration(e.Integration)
		if service != nil {
			servicePort := corev1.ServicePort{
				Name:       t.getServicePortName(),
				Port:       t.getServicePort(),
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromString(name),
			}
			e.Integration.Status.SetCondition(
				v1.IntegrationConditionServiceAvailable,
				corev1.ConditionTrue,
				v1.IntegrationConditionServiceAvailableReason,
				// service -> container
				fmt.Sprintf("%s(%s/%d) -> %s(%s/%d)",
					service.Name, servicePort.Name, servicePort.Port,
					container.Name, containerPort.Name, containerPort.ContainerPort),
			)
			service.Spec.Ports = append(service.Spec.Ports, servicePort)
			// Mark the service as a user service
			service.Labels["camel.apache.org/service.type"] = v1.ServiceTypeUser
		}
	}
	container.Ports = append(container.Ports, containerPort)
}

func (t *containerTrait) configureResources(container *corev1.Container) {
	requestsList := container.Resources.Requests
	limitsList := container.Resources.Limits
	var err error
	if requestsList == nil {
		requestsList = make(corev1.ResourceList)
	}
	if limitsList == nil {
		limitsList = make(corev1.ResourceList)
	}

	requestsList, err = kubernetes.ConfigureResource(t.getRequestCPU(), requestsList, corev1.ResourceCPU)
	if err != nil {
		t.L.Error(err, "unable to parse quantity", "request-cpu", t.getRequestCPU())
	}
	requestsList, err = kubernetes.ConfigureResource(t.getRequestMemory(), requestsList, corev1.ResourceMemory)
	if err != nil {
		t.L.Error(err, "unable to parse quantity", "request-memory", t.getRequestMemory())
	}
	limitsList, err = kubernetes.ConfigureResource(t.getLimitCPU(), limitsList, corev1.ResourceCPU)
	if err != nil {
		t.L.Error(err, "unable to parse quantity", "limit-cpu", t.getLimitCPU())
	}
	limitsList, err = kubernetes.ConfigureResource(t.getLimitMemory(), limitsList, corev1.ResourceMemory)
	if err != nil {
		t.L.Error(err, "unable to parse quantity", "limit-memory", t.getLimitMemory())
	}

	container.Resources.Requests = requestsList
	container.Resources.Limits = limitsList
}

func (t *containerTrait) configureCapabilities(e *Environment) {
	if util.StringSliceExists(e.Integration.Status.Capabilities, v1.CapabilityRest) {
		e.ApplicationProperties["camel.context.rest-configuration.component"] = "platform-http"
	}
}

func (t *containerTrait) setSecurityContext(e *Environment, container *corev1.Container) error {
	sc := corev1.SecurityContext{
		RunAsNonRoot: t.getRunAsNonRoot(),
		SeccompProfile: &corev1.SeccompProfile{
			Type: t.getSeccompProfileType(),
		},
		AllowPrivilegeEscalation: t.getAllowPrivilegeEscalation(),
		Capabilities:             &corev1.Capabilities{Drop: t.getCapabilitiesDrop(), Add: t.CapabilitiesAdd},
	}

	runAsUser, err := t.getUser(e)
	if err != nil {
		return err
	}

	t.RunAsUser = runAsUser

	sc.RunAsUser = t.RunAsUser
	container.SecurityContext = &sc

	return nil
}

func (t *containerTrait) getUser(e *Environment) (*int64, error) {
	if t.RunAsUser != nil {
		return t.RunAsUser, nil
	}

	// get security context UID from Openshift when non.configured by the user
	isOpenShift, err := openshift.IsOpenShift(e.Client)
	if err != nil {
		return nil, err
	}
	if !isOpenShift {
		return nil, nil
	}

	runAsUser, err := openshift.GetOpenshiftUser(e.Ctx, e.Client, e.Integration.Namespace)
	if err != nil {
		return nil, err
	}

	return runAsUser, nil
}

func (t *containerTrait) getPort() int32 {
	if t.Port == 0 {
		return defaultContainerPort
	}

	return t.Port
}

func (t *containerTrait) getServicePort() int32 {
	if t.ServicePort == 0 {
		return defaultServicePort
	}

	return t.ServicePort
}

func (t *containerTrait) getServicePortName() string {
	if t.ServicePortName == "" {
		return defaultContainerPortName
	}

	return t.ServicePortName
}

func (t *containerTrait) getContainerName() string {
	if t.Name == "" {
		return defaultContainerName
	}

	return t.Name
}

func (t *containerTrait) getRunAsNonRoot() *bool {
	if t.RunAsNonRoot == nil {
		return ptr.To(defaultContainerRunAsNonRoot)
	}

	return t.RunAsNonRoot
}

func (t *containerTrait) getSeccompProfileType() corev1.SeccompProfileType {
	if t.SeccompProfileType == "" {
		return defaultContainerSeccompProfileType
	}

	return t.SeccompProfileType
}

func (t *containerTrait) getAllowPrivilegeEscalation() *bool {
	if t.AllowPrivilegeEscalation == nil {
		return ptr.To(defaultContainerAllowPrivilegeEscalation)
	}

	return t.AllowPrivilegeEscalation
}

func (t *containerTrait) getCapabilitiesDrop() []corev1.Capability {
	if t.CapabilitiesDrop == nil {
		return []corev1.Capability{defaultContainerCapabilitiesDrop}
	}

	return t.CapabilitiesDrop
}

func (t *containerTrait) getRequestCPU() string {
	if t.RequestCPU == "" {
		return defaultContainerResourceCPU
	}

	return t.RequestCPU
}

func (t *containerTrait) getRequestMemory() string {
	if t.RequestMemory == "" {
		return defaultContainerResourceMemory
	}

	return t.RequestMemory
}

func (t *containerTrait) getLimitCPU() string {
	if t.LimitCPU == "" {
		return defaultContainerLimitCPU
	}

	return t.LimitCPU
}

func (t *containerTrait) getLimitMemory() string {
	if t.LimitMemory == "" {
		return defaultContainerLimitMemory
	}

	return t.LimitMemory
}
