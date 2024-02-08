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
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	serving "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util/envvar"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	utilResource "github.com/apache/camel-k/v2/pkg/util/resource"
)

type mountTrait struct {
	BasePlatformTrait
	traitv1.MountTrait `property:",squash"`
}

func newMountTrait() Trait {
	return &mountTrait{
		// Must follow immediately the container trait
		BasePlatformTrait: NewBasePlatformTrait("mount", 1610),
	}
}

func (t *mountTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil || !e.IntegrationInRunningPhases() {
		return false, nil, nil
	}
	// Look for secrets which may have been created by service binding trait
	t.addServiceBindingSecret(e)
	// Look for implicit secrets which may be required by kamelets
	condition := t.addImplicitKameletsSecrets(e)

	// Validate resources and pvcs
	for _, c := range t.Configs {
		if !strings.HasPrefix(c, "configmap:") && !strings.HasPrefix(c, "secret:") {
			return false, nil, fmt.Errorf("unsupported config %s, must be a configmap or secret resource", c)
		}
	}
	for _, r := range t.Resources {
		if !strings.HasPrefix(r, "configmap:") && !strings.HasPrefix(r, "secret:") {
			return false, nil, fmt.Errorf("unsupported resource %s, must be a configmap or secret resource", r)
		}
	}

	// mount trait needs to be executed only when it has sources attached or any trait configuration
	return len(e.Integration.AllSources()) > 0 ||
		len(t.Configs) > 0 ||
		len(t.Resources) > 0 ||
		len(t.Volumes) > 0, condition, nil
}

func (t *mountTrait) Apply(e *Environment) error {
	container := e.GetIntegrationContainer()
	if container == nil {
		return fmt.Errorf("unable to find integration container: %s", e.Integration.Name)
	}

	var volumes *[]corev1.Volume
	visited := false

	// Deployment
	if err := e.Resources.VisitDeploymentE(func(deployment *appsv1.Deployment) error {
		volumes = &deployment.Spec.Template.Spec.Volumes
		visited = true
		return nil
	}); err != nil {
		return err
	}

	// Knative Service
	if err := e.Resources.VisitKnativeServiceE(func(service *serving.Service) error {
		volumes = &service.Spec.ConfigurationSpec.Template.Spec.Volumes
		visited = true
		return nil
	}); err != nil {
		return err
	}

	// CronJob
	if err := e.Resources.VisitCronJobE(func(cron *batchv1.CronJob) error {
		volumes = &cron.Spec.JobTemplate.Spec.Template.Spec.Volumes
		visited = true
		return nil
	}); err != nil {
		return err
	}

	if visited {
		// Volumes declared in the Integration resources
		camelPropertiesLocations := e.configureVolumesAndMounts(volumes, &container.VolumeMounts)
		// Volumes declared in the trait config/resource options
		props, propsAsEnv, err := t.configureVolumesAndMounts(e, volumes, &container.VolumeMounts)
		if err != nil {
			return err
		}
		// Properties file are implicitly read from configmaps/secrets
		if pointer.BoolDeref(t.ScanConfigsForProperties, true) {
			camelPropertiesLocations = append(camelPropertiesLocations, props...)
		}
		t.setConfigLocations(container, camelPropertiesLocations)
		// nolint: staticcheck
		// Use a configmap/secret as it was a properties file (using as a map of key/value pairs)
		if pointer.BoolDeref(t.ConfigsAsPropertyFiles, true) && propsAsEnv != nil {
			if err := t.setConfigPropertiesAsEnvVar(propsAsEnv, e, container, volumes); err != nil {
				return err
			}
		}
	}

	return nil
}

// Returns the list of mount paths.
func (t *mountTrait) configureVolumesAndMounts(e *Environment, vols *[]corev1.Volume, mnts *[]corev1.VolumeMount) ([]string, []corev1.EnvVar, error) {
	var camelProperties []string
	var propsAsEnv []corev1.EnvVar
	for _, c := range t.Configs {
		if conf, parseErr := utilResource.ParseConfig(c); parseErr == nil {
			cp, pe := t.mountResource(e, vols, mnts, conf)
			camelProperties = append(camelProperties, cp...)
			propsAsEnv = append(propsAsEnv, pe...)
		} else {
			return nil, nil, parseErr
		}
	}
	for _, r := range t.Resources {
		if res, parseErr := utilResource.ParseResource(r); parseErr == nil {
			t.mountResource(e, vols, mnts, res)
		} else {
			return nil, nil, parseErr
		}
	}
	for _, v := range t.Volumes {
		if vol, parseErr := utilResource.ParseVolume(v); parseErr == nil {
			t.mountResource(e, vols, mnts, vol)
		} else {
			return nil, nil, parseErr
		}
	}

	return camelProperties, propsAsEnv, nil
}

// Returns the single resource file path or the list of all the files mounted.
func (t *mountTrait) mountResource(e *Environment, vols *[]corev1.Volume, mnts *[]corev1.VolumeMount, conf *utilResource.Config) ([]string, []corev1.EnvVar) {
	var paths []string
	var propsAsEnv []corev1.EnvVar
	refName := kubernetes.SanitizeLabel(conf.Name())
	dstDir := conf.DestinationPath()
	dstFile := ""
	if dstDir != "" {
		if conf.Key() != "" {
			dstFile = filepath.Base(conf.DestinationPath())
		} else {
			dstFile = conf.Key()
		}
	}
	vol := getVolume(refName, string(conf.StorageType()), conf.Name(), conf.Key(), dstFile)
	mntPath := getMountPoint(conf.Name(), dstDir, string(conf.StorageType()), string(conf.ContentType()))
	readOnly := true
	if conf.StorageType() == utilResource.StorageTypePVC {
		readOnly = false
	}
	mnt := getMount(refName, mntPath, dstFile, readOnly)

	*vols = append(*vols, *vol)
	*mnts = append(*mnts, *mnt)

	// User specified location file (only properties file)
	if dstDir != "" {
		if strings.HasSuffix(dstDir, ".properties") {
			return []string{mntPath}, nil
		}
		return nil, nil
	}

	// We only process this for text configuration .properties files, never for resources
	if conf.ContentType() == utilResource.ContentTypeText {
		// the user asked to store the entire resource without specifying any filter
		// we need to list all the resources belonging to the resource
		if conf.StorageType() == utilResource.StorageTypeConfigmap {
			cm := kubernetes.LookupConfigmap(e.Ctx, e.Client, e.Integration.Namespace, conf.Name())
			if cm != nil {
				for k := range cm.Data {
					if strings.HasSuffix(k, ".properties") {
						paths = append(paths, fmt.Sprintf("%s/%s", mntPath, k))
					} else {
						// Deprecated: use explicit configuration instead
						envName := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(k, "-", "_"), ".", "_"))
						t.L.Infof(`Deprecation notice: the operator is adding the environment variable %s which will take runtime value from configmap.
						This feature may disappear in future releases, make sure to use properties file in you configmap instead.`, envName)
						propsAsEnv = append(propsAsEnv, corev1.EnvVar{
							Name: envName,
							ValueFrom: &corev1.EnvVarSource{
								ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: cm.Name,
									},
									Key: k,
								},
							},
						})
					}
				}
			}
		} else if conf.StorageType() == utilResource.StorageTypeSecret {
			sec := kubernetes.LookupSecret(e.Ctx, e.Client, e.Integration.Namespace, conf.Name())
			if sec != nil {
				for k := range sec.Data {
					if strings.HasSuffix(k, ".properties") {
						paths = append(paths, fmt.Sprintf("%s/%s", mntPath, k))
					} else {
						// Deprecated: use explicit configuration instead
						envName := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(k, "-", "_"), ".", "_"))
						t.L.Infof(`Deprecation notice: the operator is adding the environment variable %s which will take runtime value from secret.
						This feature may disappear in future releases, make sure to use properties file in you secret instead.`, envName)
						propsAsEnv = append(propsAsEnv, corev1.EnvVar{
							Name: envName,
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: sec.Name,
									},
									Key: k,
								},
							},
						})
					}
				}
			}
		}
	}

	return paths, propsAsEnv
}

// Configure the list of location which the runtime will look for application.properties files.
func (t *mountTrait) setConfigLocations(container *corev1.Container, configPaths []string) {
	if configPaths != nil {
		envvar.SetVar(&container.Env, corev1.EnvVar{
			Name:  "QUARKUS_CONFIG_LOCATIONS",
			Value: strings.Join(configPaths, ","),
		})
	}
}

// Deprecated: use explicit configmap/secret properties file instead.
// Configure a list of environment variable which will be transformed as camel properties at runtime.
func (t *mountTrait) setConfigPropertiesAsEnvVar(propsAsEnv []corev1.EnvVar, e *Environment, container *corev1.Container, vols *[]corev1.Volume) error {
	additionalProperties := ""
	for _, env := range propsAsEnv {
		envvar.SetVar(&container.Env, env)
		var property string
		if env.ValueFrom.ConfigMapKeyRef != nil {
			property = env.ValueFrom.ConfigMapKeyRef.Key
		} else if env.ValueFrom.SecretKeyRef != nil {
			property = env.ValueFrom.SecretKeyRef.Key
		}
		if property != "" {
			// at runtime it will take the value from the env var
			additionalProperties += fmt.Sprintf("%s=\n", property)
		}
	}
	// We need to hack a little bit more because of
	// https://smallrye.io/smallrye-config/Main/config/environment-variables/#environment-variables
	// Basically, we must include into the user or application properties the name of the key
	// in order to recognize the variable `CAMEL_TIMER_SOURCE` as `camel.timer-source` instead of `camel.timer.source`
	// We must create a file expected to be mounted on $PWD/config/application.properties with the list of variables.
	if additionalProperties != "" {
		cm := corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      e.Integration.Name + "-imp-props",
				Namespace: e.Integration.Namespace,
				Labels: map[string]string{
					v1.IntegrationLabel:                e.Integration.Name,
					"camel.apache.org/properties.type": "application",
					kubernetes.ConfigMapTypeLabel:      "camel-properties",
				},
			},
			Data: map[string]string{
				"application.properties": additionalProperties,
			},
		}
		e.Resources.Add(&cm)
		// We need also to mount this on the container as we've done previously with the other configmaps
		c, err := utilResource.ParseConfig(
			fmt.Sprintf(
				"configmap:%s/%s@%s",
				e.Integration.Name+"-imp-props",
				"application.properties",
				"/deployments/config/application.properties",
			),
		)
		if err != nil {
			return err
		}
		t.mountResource(e, vols, &container.VolumeMounts, c)
	}
	return nil
}

func (t *mountTrait) addServiceBindingSecret(e *Environment) {
	e.Resources.VisitSecret(func(secret *corev1.Secret) {
		switch secret.Labels[serviceBindingLabel] {
		case "true":
			t.Configs = append(t.Configs, "secret:"+secret.Name)
		}
	})
}

// Deprecated: to be removed in future releases.
// nolint: staticcheck
func (t *mountTrait) addImplicitKameletsSecrets(e *Environment) *TraitCondition {
	featureUsed := false
	if trait := e.Catalog.GetTrait(kameletsTraitID); trait != nil {
		kamelets, ok := trait.(*kameletsTrait)
		if !ok {
			return NewIntegrationCondition(
				v1.IntegrationConditionTraitInfo,
				corev1.ConditionTrue,
				traitConfigurationReason,
				"Unexpected error happened while casting to kamelets trait",
			)
		}
		if !pointer.BoolDeref(t.ScanKameletsImplicitLabelSecrets, true) {
			return nil
		}
		implicitKameletSecrets, err := kamelets.listConfigurationSecrets(e)
		if err != nil {
			return NewIntegrationCondition(
				v1.IntegrationConditionTraitInfo,
				corev1.ConditionTrue,
				traitConfigurationReason,
				err.Error(),
			)
		}
		for _, secret := range implicitKameletSecrets {
			featureUsed = true
			t.Configs = append(t.Configs, "secret:"+secret)
		}
	}

	if featureUsed {
		return NewIntegrationCondition(
			v1.IntegrationConditionTraitInfo,
			corev1.ConditionTrue,
			traitConfigurationReason,
			"Implicit Kamelet labelling secrets are deprecated and may be removed in future releases. Make sure to use explicit mount.config secrets instead.",
		)
	}
	return nil
}
