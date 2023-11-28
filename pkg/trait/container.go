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
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"

	serving "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/digest"
	"github.com/apache/camel-k/v2/pkg/util/envvar"
	"github.com/apache/camel-k/v2/pkg/util/knative"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/openshift"
)

const (
	defaultContainerName     = "integration"
	defaultContainerPort     = 8080
	defaultContainerPortName = "http"
	defaultServicePort       = 80
	containerTraitID         = "container"
)

type containerTrait struct {
	BasePlatformTrait
	traitv1.ContainerTrait `property:",squash"`
}

func newContainerTrait() Trait {
	return &containerTrait{
		BasePlatformTrait: NewBasePlatformTrait(containerTraitID, 1600),
		ContainerTrait: traitv1.ContainerTrait{
			Port:            defaultContainerPort,
			ServicePort:     defaultServicePort,
			ServicePortName: defaultContainerPortName,
			Name:            defaultContainerName,
		},
	}
}

func (t *containerTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil {
		return false, nil, nil
	}

	if !e.IntegrationInPhase(v1.IntegrationPhaseInitialization) && !e.IntegrationInRunningPhases() {
		return false, nil, nil
	}

	knativeInstalled, _ := knative.IsInstalled(e.Client)
	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) && !knativeInstalled {
		hasKnativeEndpoint, err := containsEndpoint("knative", e, t.Client)
		if err != nil {
			return false, nil, err
		}

		if hasKnativeEndpoint {
			// fail fast the integration as there is no knative installed in the cluster
			t.L.ForIntegration(e.Integration).Infof("Integration %s/%s contains knative endpoint that cannot run, as knative is not installed in the cluster.", e.Integration.Namespace, e.Integration.Name)
			err := errors.New("integration cannot run, as knative is not installed in the cluster")
			return false, NewIntegrationCondition(
				v1.IntegrationConditionKnativeAvailable,
				corev1.ConditionFalse,
				v1.IntegrationConditionKnativeNotInstalledReason,
				err.Error(),
			), err
		}
	}

	if pointer.BoolDeref(t.Auto, true) {
		if t.Expose == nil {
			e := e.Resources.GetServiceForIntegration(e.Integration) != nil
			t.Expose = &e
		}
	}

	if !isValidPullPolicy(t.ImagePullPolicy) {
		return false, nil, fmt.Errorf("unsupported pull policy %s", t.ImagePullPolicy)
	}

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
	if t.Image != "" {
		if e.Integration.Spec.IntegrationKit != nil {
			return fmt.Errorf(
				"unsupported configuration: a container image has been set in conjunction with an IntegrationKit %v",
				e.Integration.Spec.IntegrationKit)
		}

		kitName := fmt.Sprintf("kit-%s", e.Integration.Name)
		kit := v1.NewIntegrationKit(e.Integration.Namespace, kitName)
		kit.Spec.Image = t.Image

		// Add some information for post-processing, this may need to be refactored
		// to a proper data structure
		kit.Labels = map[string]string{
			v1.IntegrationKitTypeLabel:            v1.IntegrationKitTypeExternal,
			kubernetes.CamelCreatorLabelKind:      v1.IntegrationKind,
			kubernetes.CamelCreatorLabelName:      e.Integration.Name,
			kubernetes.CamelCreatorLabelNamespace: e.Integration.Namespace,
			kubernetes.CamelCreatorLabelVersion:   e.Integration.ResourceVersion,
		}

		if v, ok := e.Integration.Annotations[v1.PlatformSelectorAnnotation]; ok {
			v1.SetAnnotation(&kit.ObjectMeta, v1.PlatformSelectorAnnotation, v)
		}
		operatorID := defaults.OperatorID()
		if operatorID != "" {
			kit.SetOperatorID(operatorID)
		}

		t.L.Infof("image %s", kit.Spec.Image)
		e.Resources.Add(kit)
		e.Integration.SetIntegrationKit(kit)
	}
	return nil
}

func (t *containerTrait) configureContainer(e *Environment) error {
	if e.ApplicationProperties == nil {
		e.ApplicationProperties = make(map[string]string)
	}

	container := corev1.Container{
		Name:  t.Name,
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

	e.addSourcesProperties()
	if props, err := e.computeApplicationProperties(); err != nil {
		return err
	} else if props != nil {
		e.Resources.Add(props)
	}

	t.configureResources(&container)
	if pointer.BoolDeref(t.Expose, false) {
		t.configureService(e, &container)
	}
	t.configureCapabilities(e)

	t.configureSecurityContext(e, &container)

	var containers *[]corev1.Container
	visited := false

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
				t.L.Infof("Skipping environment variable %s (fieldRef)", env.Name)
			case env.ValueFrom.ResourceFieldRef != nil:
				t.L.Infof("Skipping environment variable %s (resourceFieldRef)", env.Name)
			default:
				envvar.SetVar(&container.Env, env)
			}
		}

		containers = &service.Spec.ConfigurationSpec.Template.Spec.Containers
		visited = true
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

	if visited {
		*containers = append(*containers, container)
	}

	return nil
}

func (t *containerTrait) configureService(e *Environment, container *corev1.Container) {
	service := e.Resources.GetServiceForIntegration(e.Integration)
	if service == nil {
		return
	}

	name := t.PortName
	if name == "" {
		name = defaultContainerPortName
	}

	containerPort := corev1.ContainerPort{
		Name:          name,
		ContainerPort: int32(t.Port),
		Protocol:      corev1.ProtocolTCP,
	}

	servicePort := corev1.ServicePort{
		Name:       t.ServicePortName,
		Port:       int32(t.ServicePort),
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

	container.Ports = append(container.Ports, containerPort)
	service.Spec.Ports = append(service.Spec.Ports, servicePort)

	// Mark the service as a user service
	service.Labels["camel.apache.org/service.type"] = v1.ServiceTypeUser
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

	requestsList, err = kubernetes.ConfigureResource(t.RequestCPU, requestsList, corev1.ResourceCPU)
	if err != nil {
		t.L.Error(err, "unable to parse quantity", "request-cpu", t.RequestCPU)
	}
	requestsList, err = kubernetes.ConfigureResource(t.RequestMemory, requestsList, corev1.ResourceMemory)
	if err != nil {
		t.L.Error(err, "unable to parse quantity", "request-memory", t.RequestMemory)
	}
	limitsList, err = kubernetes.ConfigureResource(t.LimitCPU, limitsList, corev1.ResourceCPU)
	if err != nil {
		t.L.Error(err, "unable to parse quantity", "limit-cpu", t.LimitCPU)
	}
	limitsList, err = kubernetes.ConfigureResource(t.LimitMemory, limitsList, corev1.ResourceMemory)
	if err != nil {
		t.L.Error(err, "unable to parse quantity", "limit-memory", t.LimitMemory)
	}

	container.Resources.Requests = requestsList
	container.Resources.Limits = limitsList
}

func (t *containerTrait) configureCapabilities(e *Environment) {
	if util.StringSliceExists(e.Integration.Status.Capabilities, v1.CapabilityRest) {
		e.ApplicationProperties["camel.context.rest-configuration.component"] = "platform-http"
	}
}

func (t *containerTrait) configureSecurityContext(e *Environment, container *corev1.Container) {
	// get security context from security context constraint configuration in namespace
	isOpenShift, _ := openshift.IsOpenShift(e.Client)
	if isOpenShift {
		securityContext, _ := openshift.GetOpenshiftSecurityContextRestricted(e.Ctx, e.Client, e.Integration.Namespace)
		if securityContext != nil {
			container.SecurityContext = securityContext
		}
	}
}

// It's a user provided image if it does not match the naming convention used by Camel K Integration Kits.
func (t *containerTrait) hasUserProvidedImage() bool {
	return t.Image != "" && !strings.Contains(t.Image, "camel-k-kit-")
}
