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
	"strings"

	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// ParseEmptyDirVolume will parse and return an empty-dir volume.
func ParseEmptyDirVolume(item string) (*corev1.Volume, *corev1.VolumeMount, error) {
	volumeParts := strings.Split(item, ":")

	if len(volumeParts) != 2 && len(volumeParts) != 3 {
		return nil, nil, fmt.Errorf("could not match emptyDir volume as %s", item)
	}

	refName := kubernetes.SanitizeLabel(volumeParts[0])
	sizeLimit := "500Mi"
	if len(volumeParts) == 3 {
		sizeLimit = volumeParts[2]
	}

	parsed, err := resource.ParseQuantity(sizeLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("could not parse sizeLimit from emptyDir volume: %s", volumeParts[2])
	}

	volume := &corev1.Volume{
		Name: refName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				SizeLimit: &parsed,
			},
		},
	}

	volumeMount := getMount(refName, volumeParts[1], "", false)

	return volume, volumeMount, nil
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
			return errors.New("could not find any default StorageClass")
		}
	}

	pvc := kubernetes.NewPersistentVolumeClaim(e.Integration.Namespace, volumeName, sc.Name, sizeQty, corev1.PersistentVolumeAccessMode(accessMode))
	if err := e.Client.Create(e.Ctx, pvc); err != nil {
		return err
	}

	return nil
}

// sanitizeVolumeName ensures the provided name is unique among the volumes.
// If `name` already exists, it appends -1, -2, etc. until a unique name is found.
func sanitizeVolumeName(name string, vols *[]corev1.Volume) string {
	name = kubernetes.SanitizeLabel(name)
	if !volumeExists(name, vols) {
		return name
	}

	suffix := 1
	for {
		candidate := fmt.Sprintf("%s-%d", name, suffix)
		if !volumeExists(candidate, vols) {
			return candidate
		}
		suffix++
	}
}

// Helper to check existence of a volume name in the list.
func volumeExists(name string, vols *[]corev1.Volume) bool {
	for _, v := range *vols {
		if v.Name == name {
			return true
		}
	}

	return false
}
