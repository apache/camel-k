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
	"sort"
	"strconv"
	"strings"

	"github.com/apache/camel-k/pkg/util"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"

	serving "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/envvar"
)

const (
	defaultContainerName = "integration"
	defaultContainerPort = 8080
	defaultServicePort   = 80
	defaultProbePath     = "/health"
	containerTraitID     = "container"
)

// The Container trait can be used to configure properties of the container where the integration will run.
//
// It also provides configuration for Services associated to the container.
//
// +camel-k:trait=container
type containerTrait struct {
	BaseTrait `property:",squash"`

	Auto *bool `property:"auto"`

	// The minimum amount of CPU required.
	RequestCPU string `property:"request-cpu"`
	// The minimum amount of memory required.
	RequestMemory string `property:"request-memory"`
	// The maximum amount of CPU required.
	LimitCPU string `property:"limit-cpu"`
	// The maximum amount of memory required.
	LimitMemory string `property:"limit-memory"`

	// Can be used to enable/disable exposure via kubernetes Service.
	Expose *bool `property:"expose"`
	// To configure a different port exposed by the container (default `8080`).
	Port int `property:"port"`
	// To configure a different port name for the port exposed by the container (default `http`).
	PortName string `property:"port-name"`
	// To configure under which service port the container port is to be exposed (default `80`).
	ServicePort int `property:"service-port"`
	// To configure under which service port name the container port is to be exposed (default `http`).
	ServicePortName string `property:"service-port-name"`

	// The main container name. It's named `integration` by default.
	Name string `property:"name"`

	// ProbesEnabled enable/disable probes on the container (default `false`)
	ProbesEnabled bool `property:"probes-enabled"`
	// Path to access on the probe ( default `/health`). Note that this property is not supported
	// on quarkus runtime and setting it will result in the integration failing to start.
	ProbePath string `property:"probe-path"`
	// Number of seconds after the container has started before liveness probes are initiated.
	LivenessInitialDelay int32 `property:"liveness-initial-delay"`
	// Number of seconds after which the probe times out. Applies to the liveness probe.
	LivenessTimeout int32 `property:"liveness-timeout"`
	// How often to perform the probe. Applies to the liveness probe.
	LivenessPeriod int32 `property:"liveness-period"`
	// Minimum consecutive successes for the probe to be considered successful after having failed.
	// Applies to the liveness probe.
	LivenessSuccessThreshold int32 `property:"liveness-success-threshold"`
	// Minimum consecutive failures for the probe to be considered failed after having succeeded.
	// Applies to the liveness probe.
	LivenessFailureThreshold int32 `property:"liveness-failure-threshold"`
	// Number of seconds after the container has started before readiness probes are initiated.
	ReadinessInitialDelay int32 `property:"readiness-initial-delay"`
	// Number of seconds after which the probe times out. Applies to the readiness probe.
	ReadinessTimeout int32 `property:"readiness-timeout"`
	// How often to perform the probe. Applies to the readiness probe.
	ReadinessPeriod int32 `property:"readiness-period"`
	// Minimum consecutive successes for the probe to be considered successful after having failed.
	// Applies to the readiness probe.
	ReadinessSuccessThreshold int32 `property:"readiness-success-threshold"`
	// Minimum consecutive failures for the probe to be considered failed after having succeeded.
	// Applies to the readiness probe.
	ReadinessFailureThreshold int32 `property:"readiness-failure-threshold"`
}

func newContainerTrait() Trait {
	return &containerTrait{
		BaseTrait:       NewBaseTrait(containerTraitID, 1600),
		Port:            defaultContainerPort,
		PortName:        httpPortName,
		ServicePort:     defaultServicePort,
		ServicePortName: httpPortName,
		Name:            defaultContainerName,
		ProbesEnabled:   false,
		ProbePath:       defaultProbePath,
	}
}

func (t *containerTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	if !e.IntegrationInPhase(v1.IntegrationPhaseInitialization, v1.IntegrationPhaseDeploying, v1.IntegrationPhaseRunning) {
		return false, nil
	}

	if t.Auto == nil || *t.Auto {
		if t.Expose == nil {
			e := e.Resources.GetServiceForIntegration(e.Integration) != nil
			t.Expose = &e
		}
	}

	return true, nil
}

func (t *containerTrait) Apply(e *Environment) error {
	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		t.configureDependencies(e)
	}

	if e.IntegrationInPhase(v1.IntegrationPhaseDeploying, v1.IntegrationPhaseRunning) {
		return t.configureContainer(e)
	}

	return nil
}

// IsPlatformTrait overrides base class method
func (t *containerTrait) IsPlatformTrait() bool {
	return true
}

func (t *containerTrait) configureDependencies(e *Environment) {
	if !t.ProbesEnabled {
		return
	}

	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		if capability, ok := e.CamelCatalog.Runtime.Capabilities[v1.CapabilityHealth]; ok {
			for _, dependency := range capability.Dependencies {
				util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, fmt.Sprintf("mvn:%s/%s", dependency.GroupID, dependency.ArtifactID))
			}

			// sort the dependencies to get always the same list if they don't change
			sort.Strings(e.Integration.Status.Dependencies)
		}
	}
}

// nolint:gocyclo
func (t *containerTrait) configureContainer(e *Environment) error {
	if e.ApplicationProperties == nil {
		e.ApplicationProperties = make(map[string]string)
	}

	container := corev1.Container{
		Name:  t.Name,
		Image: e.Integration.Status.Image,
		Env:   make([]corev1.EnvVar, 0),
	}

	// combine Environment of integration with platform, kit, integration
	for key, value := range e.CollectConfigurationPairs("env") {
		envvar.SetVal(&container.Env, key, value)
	}

	envvar.SetVal(&container.Env, "CAMEL_K_DIGEST", e.Integration.Status.Digest)
	envvar.SetVal(&container.Env, "CAMEL_K_ROUTES", strings.Join(e.ComputeSourcesURI(), ","))
	envvar.SetVal(&container.Env, "CAMEL_K_CONF", "/etc/camel/conf/application.properties")
	envvar.SetVal(&container.Env, "CAMEL_K_CONF_D", "/etc/camel/conf.d")

	t.configureResources(e, &container)

	if t.Expose != nil && *t.Expose {
		t.configureService(e, &container)
	}
	if err := t.configureCapabilities(e); err != nil {
		return err
	}

	//
	// Deployment
	//
	if err := e.Resources.VisitDeploymentE(func(deployment *appsv1.Deployment) error {
		if t.ProbesEnabled && t.PortName == httpPortName {
			if err := t.configureProbes(e, &container, t.Port, t.ProbePath); err != nil {
				return err
			}
		}

		for _, envVar := range e.EnvVars {
			envvar.SetVar(&container.Env, envVar)
		}
		if props := e.ComputeApplicationProperties(); props != nil {
			e.Resources.Add(props)
		}

		e.ConfigureVolumesAndMounts(
			&deployment.Spec.Template.Spec.Volumes,
			&container.VolumeMounts,
		)

		deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, container)

		return nil
	}); err != nil {
		return err
	}

	//
	// Knative Service
	//
	if err := e.Resources.VisitKnativeServiceE(func(service *serving.Service) error {
		if t.ProbesEnabled && t.PortName == httpPortName {
			// don't set the port on Knative service as it is not allowed.
			if err := t.configureProbes(e, &container, 0, t.ProbePath); err != nil {
				return err
			}
		}

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
		if props := e.ComputeApplicationProperties(); props != nil {
			e.Resources.Add(props)
		}

		e.ConfigureVolumesAndMounts(
			&service.Spec.ConfigurationSpec.Template.Spec.Volumes,
			&container.VolumeMounts,
		)

		service.Spec.ConfigurationSpec.Template.Spec.Containers = append(service.Spec.ConfigurationSpec.Template.Spec.Containers, container)

		return nil
	}); err != nil {
		return err
	}

	//
	// CronJob
	//
	if err := e.Resources.VisitCronJobE(func(cron *v1beta1.CronJob) error {
		if t.ProbesEnabled && t.PortName == httpPortName {
			if err := t.configureProbes(e, &container, t.Port, t.ProbePath); err != nil {
				return err
			}
		}

		for _, envVar := range e.EnvVars {
			envvar.SetVar(&container.Env, envVar)
		}
		if props := e.ComputeApplicationProperties(); props != nil {
			e.Resources.Add(props)
		}

		e.ConfigureVolumesAndMounts(
			&cron.Spec.JobTemplate.Spec.Template.Spec.Volumes,
			&container.VolumeMounts,
		)

		cron.Spec.JobTemplate.Spec.Template.Spec.Containers = append(cron.Spec.JobTemplate.Spec.Template.Spec.Containers, container)

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (t *containerTrait) configureService(e *Environment, container *corev1.Container) {
	service := e.Resources.GetServiceForIntegration(e.Integration)
	if service == nil {
		return
	}

	containerPort := corev1.ContainerPort{
		Name:          t.PortName,
		ContainerPort: int32(t.Port),
		Protocol:      corev1.ProtocolTCP,
	}

	servicePort := corev1.ServicePort{
		Name:       t.ServicePortName,
		Port:       int32(t.ServicePort),
		Protocol:   corev1.ProtocolTCP,
		TargetPort: intstr.FromString(t.PortName),
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

func (t *containerTrait) configureResources(_ *Environment, container *corev1.Container) {
	//
	// Requests
	//
	if container.Resources.Requests == nil {
		container.Resources.Requests = make(corev1.ResourceList)
	}

	if t.RequestCPU != "" {
		v, err := resource.ParseQuantity(t.RequestCPU)
		if err != nil {
			t.L.Error(err, "unable to parse quantity", "request-cpu", t.RequestCPU)
		} else {
			container.Resources.Requests[corev1.ResourceCPU] = v
		}
	}
	if t.RequestMemory != "" {
		v, err := resource.ParseQuantity(t.RequestMemory)
		if err != nil {
			t.L.Error(err, "unable to parse quantity", "request-memory", t.RequestMemory)
		} else {
			container.Resources.Requests[corev1.ResourceMemory] = v
		}
	}

	//
	// Limits
	//
	if container.Resources.Limits == nil {
		container.Resources.Limits = make(corev1.ResourceList)
	}

	if t.LimitCPU != "" {
		v, err := resource.ParseQuantity(t.LimitCPU)
		if err != nil {
			t.L.Error(err, "unable to parse quantity", "limit-cpu", t.LimitCPU)
		} else {
			container.Resources.Limits[corev1.ResourceCPU] = v
		}
	}
	if t.LimitMemory != "" {
		v, err := resource.ParseQuantity(t.LimitMemory)
		if err != nil {
			t.L.Error(err, "unable to parse quantity", "limit-memory", t.LimitMemory)
		} else {
			container.Resources.Limits[corev1.ResourceMemory] = v
		}
	}
}

func (t *containerTrait) configureHTTP(e *Environment) error {
	switch e.CamelCatalog.Runtime.Provider {
	case v1.RuntimeProviderMain:
		e.ApplicationProperties["customizer.platform-http.enabled"] = True
		e.ApplicationProperties["customizer.platform-http.bind-port"] = strconv.Itoa(t.Port)
	case v1.RuntimeProviderQuarkus:
		// Quarkus does not offer a runtime option to change http listening ports
		return nil
	default:
		return fmt.Errorf("unsupported runtime: %s", e.CamelCatalog.Runtime.Provider)
	}

	return nil

}

func (t *containerTrait) configureCapabilities(e *Environment) error {
	requiresHTTP := false

	if util.StringSliceExists(e.Integration.Status.Capabilities, v1.CapabilityRest) {
		e.ApplicationProperties["camel.context.rest-configuration.component"] = "platform-http"
		requiresHTTP = true
	}

	if util.StringSliceExists(e.Integration.Status.Capabilities, v1.CapabilityPlatformHttp) {
		requiresHTTP = true
	}

	if requiresHTTP {
		return t.configureHTTP(e)
	}

	return nil
}

func (t *containerTrait) configureProbes(e *Environment, container *corev1.Container, port int, path string) error {
	if err := t.configureHTTP(e); err != nil {
		return nil
	}

	switch e.CamelCatalog.Runtime.Provider {
	case v1.RuntimeProviderMain:
		e.ApplicationProperties["customizer.health.enabled"] = True
		e.ApplicationProperties["customizer.health.path"] = path
	case v1.RuntimeProviderQuarkus:
		// Quarkus does not offer a runtime option to change the path of the health endpoint but there
		// is a build time property:
		//
		//     quarkus.smallrye-health.root-path
		//
		// so failing in case user tries to change the path.
		//
		// NOTE: we could probably be more opinionated and make the path an internal detail.
		if path != defaultProbePath {
			return fmt.Errorf("health check root path can't be changed at runtimme on Quarkus")
		}
	default:
		return fmt.Errorf("unsupported runtime: %s", e.CamelCatalog.Runtime.Provider)
	}

	container.LivenessProbe = t.newLivenessProbe(port, path)
	container.ReadinessProbe = t.newReadinessProbe(port, path)

	return nil
}

func (t *containerTrait) newLivenessProbe(port int, path string) *corev1.Probe {
	action := corev1.HTTPGetAction{}
	action.Path = path

	if port > 0 {
		action.Port = intstr.FromInt(port)
	}

	p := corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &action,
		},
	}

	p.InitialDelaySeconds = t.LivenessInitialDelay
	p.TimeoutSeconds = t.LivenessTimeout
	p.PeriodSeconds = t.LivenessPeriod
	p.SuccessThreshold = t.LivenessSuccessThreshold
	p.FailureThreshold = t.LivenessFailureThreshold

	return &p
}

func (t *containerTrait) newReadinessProbe(port int, path string) *corev1.Probe {
	p := corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Port: intstr.FromInt(port),
				Path: path,
			},
		},
	}

	p.InitialDelaySeconds = t.ReadinessInitialDelay
	p.TimeoutSeconds = t.ReadinessTimeout
	p.PeriodSeconds = t.ReadinessPeriod
	p.SuccessThreshold = t.ReadinessSuccessThreshold
	p.FailureThreshold = t.ReadinessFailureThreshold

	return &p
}
