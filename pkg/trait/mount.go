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
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	serving "knative.dev/serving/pkg/apis/serving/v1"

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

	return true, nil, nil
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
		err := t.configureVolumesAndMounts(e, volumes, &container.VolumeMounts)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *mountTrait) configureVolumesAndMounts(e *Environment, vols *[]corev1.Volume, mnts *[]corev1.VolumeMount) error {
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
		volume, volumeMount, parseErr := ParseAndCreateVolume(e, v)
		if parseErr != nil {
			return parseErr
		}
		*vols = append(*vols, *volume)
		*mnts = append(*mnts, *volumeMount)
	}
	for _, v := range t.EmptyDirs {
		if vol, parseErr := utilResource.ParseEmptyDirVolume(v); parseErr == nil {
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
	if conf.StorageType() == utilResource.StorageTypePVC ||
		conf.StorageType() == utilResource.StorageTypeEmptyDir {
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

// ParseAndCreateVolume will parse a volume configuration. If the volume does not exist it tries to create one based on the storage
// class configuration provided or default.
// item is expected to be as: name:path/to/mount<:size:accessMode<:storageClassName>>.
func ParseAndCreateVolume(e *Environment, item string) (*corev1.Volume, *corev1.VolumeMount, error) {
	volumeParts := strings.Split(item, ":")
	volumeName := volumeParts[0]
	pvc, err := kubernetes.LookupPersistentVolumeClaim(e.Ctx, e.Client, e.Integration.Namespace, volumeName)
	if err != nil {
		return nil, nil, err
	}
	var volume *corev1.Volume
	if pvc == nil {
		if len(volumeParts) == 2 {
			return nil, nil, fmt.Errorf("volume %s does not exist. "+
				"Make sure to provide one or configure a dynamic PVC as trait volume configuration pvcName:path/to/mount:size:accessMode<:storageClassName>",
				volumeName,
			)
		}
		if err = createPVC(e, volumeParts); err != nil {
			return nil, nil, err
		}
	}

	volume = &corev1.Volume{
		Name: kubernetes.SanitizeLabel(volumeName),
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: volumeName,
			},
		},
	}

	volumeMount := getMount(volumeName, volumeParts[1], "", false)
	return volume, volumeMount, nil
}

// createPVC is in charge to create a PersistentVolumeClaim based on the configuration provided. Or it fail within the intent.
// volumeParts is expected to be as: name, path/to/mount, size, accessMode, <storageClassName>.
func createPVC(e *Environment, volumeParts []string) error {
	if len(volumeParts) < 4 || len(volumeParts) > 5 {
		return fmt.Errorf(
			"volume mount syntax error, must be name:path/to/mount:size:accessMode<:storageClassName> was %s",
			strings.Join(volumeParts, ":"),
		)
	}
	volumeName := volumeParts[0]
	size := volumeParts[2]
	accessMode := volumeParts[3]
	sizeQty, err := resource.ParseQuantity(size)
	if err != nil {
		return fmt.Errorf("could not parse size %s, %s", size, err.Error())
	}

	var sc *storagev1.StorageClass
	//nolint: nestif
	if len(volumeParts) == 5 {
		scName := volumeParts[4]
		sc, err = kubernetes.LookupStorageClass(e.Ctx, e.Client, e.Integration.Namespace, scName)
		if err != nil {
			return fmt.Errorf("error looking up for StorageClass %s, %w", scName, err)
		}
		if sc == nil {
			return fmt.Errorf("could not find any %s StorageClass", scName)
		}
	} else {
		sc, err = kubernetes.LookupDefaultStorageClass(e.Ctx, e.Client)
		if err != nil {
			return fmt.Errorf("error looking up for default StorageClass, %w", err)
		}
		if sc == nil {
			return fmt.Errorf("could not find any default StorageClass")
		}
	}

	pvc := kubernetes.NewPersistentVolumeClaim(e.Integration.Namespace, volumeName, sc.Name, sizeQty, corev1.PersistentVolumeAccessMode(accessMode))
	if err := e.Client.Create(e.Ctx, pvc); err != nil {
		return err
	}

	return nil
}
