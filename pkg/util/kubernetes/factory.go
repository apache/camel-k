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

package kubernetes

import (
	"fmt"
	"regexp"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	validTaintRegexp = regexp.MustCompile(`^([\w\/_\-\.]+)(=)?([\w_\-\.]+)?:(NoSchedule|NoExecute|PreferNoSchedule):?(\d*)?$`)
)

// ConfigMapAutogenLabel -- .
const ConfigMapAutogenLabel = "camel.apache.org/generated"

// ConfigMapOriginalFileNameLabel -- .
const ConfigMapOriginalFileNameLabel = "camel.apache.org/filename"

// ConfigMapTypeLabel -- .
const ConfigMapTypeLabel = "camel.apache.org/config.type"

// NewTolerations build an array of Tolerations from an array of string.
func NewTolerations(taints []string) ([]corev1.Toleration, error) {
	tolerations := make([]corev1.Toleration, 0)
	for _, t := range taints {
		if !validTaintRegexp.MatchString(t) {
			return nil, fmt.Errorf("could not match taint %v", t)
		}
		toleration := corev1.Toleration{}
		// Parse the regexp groups
		groups := validTaintRegexp.FindStringSubmatch(t)
		toleration.Key = groups[1]
		if groups[2] != "" {
			toleration.Operator = corev1.TolerationOpEqual
		} else {
			toleration.Operator = corev1.TolerationOpExists
		}
		if groups[3] != "" {
			toleration.Value = groups[3]
		}
		toleration.Effect = corev1.TaintEffect(groups[4])

		if groups[5] != "" {
			tolerationSeconds, err := strconv.ParseInt(groups[5], 10, 64)
			if err != nil {
				return nil, err
			}
			toleration.TolerationSeconds = &tolerationSeconds
		}
		tolerations = append(tolerations, toleration)
	}

	return tolerations, nil
}

// NewPersistentVolumeClaim will create a NewPersistentVolumeClaim based on a StorageClass.
func NewPersistentVolumeClaim(
	ns, name, storageClassName string, capacityStorage resource.Quantity, accessMode corev1.PersistentVolumeAccessMode) *corev1.PersistentVolumeClaim {
	pvc := corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: &storageClassName,
			AccessModes: []corev1.PersistentVolumeAccessMode{
				accessMode,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					"storage": capacityStorage,
				},
			},
		},
	}

	return &pvc
}

// ConfigureResource will set a resource on the specified resource list or returns an error.
func ConfigureResource(resourceQty string, list corev1.ResourceList, name corev1.ResourceName) (corev1.ResourceList, error) {
	if resourceQty != "" {
		v, err := resource.ParseQuantity(resourceQty)
		if err != nil {
			return list, err
		}
		list[name] = v
	}

	return list, nil
}
