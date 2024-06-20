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
	"k8s.io/utils/pointer"

	serving "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util/boolean"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	utilResource "github.com/apache/camel-k/v2/pkg/util/resource"
)

const (
	mountTraitID    = "mount"
	mountTraitOrder = 1610
)

type mountTrait struct {
	BasePlatformTrait
	traitv1.MountTrait `property:",squash"`
}

func newMountTrait() Trait {
	return &mountTrait{
		// Must follow immediately the container trait
		BasePlatformTrait: NewBasePlatformTrait(mountTraitID, mountTraitOrder),
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

	return true, condition, nil
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
		e.configureVolumesAndMounts(volumes, &container.VolumeMounts)
		// Volumes declared in the trait config/resource options
		err := t.configureVolumesAndMounts(volumes, &container.VolumeMounts)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *mountTrait) configureVolumesAndMounts(vols *[]corev1.Volume, mnts *[]corev1.VolumeMount) error {
	for _, c := range t.Configs {
		if conf, parseErr := utilResource.ParseConfig(c); parseErr == nil {
			t.mountResource(vols, mnts, conf)
		} else {
			return parseErr
		}
	}
	for _, r := range t.Resources {
		if res, parseErr := utilResource.ParseResource(r); parseErr == nil {
			t.mountResource(vols, mnts, res)
		} else {
			return parseErr
		}
	}
	for _, v := range t.Volumes {
		if vol, parseErr := utilResource.ParseVolume(v); parseErr == nil {
			t.mountResource(vols, mnts, vol)
		} else {
			return parseErr
		}
	}

	return nil
}

func (t *mountTrait) mountResource(vols *[]corev1.Volume, mnts *[]corev1.VolumeMount, conf *utilResource.Config) {
	refName := kubernetes.SanitizeLabel(conf.Name())
	dstDir := conf.DestinationPath()
	dstFile := ""
	if conf.DestinationPath() != "" {
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
}

func (t *mountTrait) addServiceBindingSecret(e *Environment) {
	e.Resources.VisitSecret(func(secret *corev1.Secret) {
		if secret.Labels[serviceBindingLabel] == boolean.TrueString {
			t.Configs = append(t.Configs, "secret:"+secret.Name)
		}
	})
}

// Deprecated: to be removed in future releases.
func (t *mountTrait) addImplicitKameletsSecrets(e *Environment) *TraitCondition {
	featureUsed := false
	if trait := e.Catalog.GetTrait(kameletsTraitID); trait != nil {
		kamelets, ok := trait.(*kameletsTrait)
		if !ok {
			return NewIntegrationCondition(
				"Mount",
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
				"Mount",
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
			"Mount",
			v1.IntegrationConditionTraitInfo,
			corev1.ConditionTrue,
			traitConfigurationReason,
			"Implicit Kamelet labelling secrets are deprecated and may be removed in future releases. Make sure to use explicit mount.config secrets instead.",
		)
	}
	return nil
}
