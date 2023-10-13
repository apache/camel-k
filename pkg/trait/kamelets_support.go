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

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/cmd/source"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type kameletBundle struct {
	kamelets []*v1.Kamelet
}

func newKameletBundle() *kameletBundle {
	return &kameletBundle{
		kamelets: make([]*v1.Kamelet, 0),
	}
}

func (kb *kameletBundle) add(k *v1.Kamelet) {
	kb.kamelets = append(kb.kamelets, k)
}

// Split the contents of the Kamelets into one ore more configmap, making sure not to overpass the 1 MB limit.
func (kb *kameletBundle) toConfigmaps(itName, itNamespace string) ([]*corev1.ConfigMap, error) {
	configmaps := make([]*corev1.ConfigMap, 0)
	cmSize := 0
	cmID := 1
	cm := newBundleConfigmap(itName, itNamespace, cmID)
	for _, k := range kb.kamelets {
		serialized, err := kubernetes.ToYAMLNoManagedFields(k)
		if err != nil {
			return nil, err
		}
		// Add if it fits into a configmap, otherwise, create a new configmap
		if cmSize+len(serialized) > source.Megabyte {
			configmaps = append(configmaps, cm)
			// create a new configmap
			cmSize = 0
			cmID++
			cm = newBundleConfigmap(itName, itNamespace, cmID)
		}
		cm.Data[fmt.Sprintf("%s.kamelet.yaml", k.Name)] = string(serialized)
		cmSize += len(serialized)
	}
	// Add the last configmap
	configmaps = append(configmaps, cm)

	return configmaps, nil
}

func newBundleConfigmap(name, namespace string, id int) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("kamelets-bundle-%s-%03d", name, id),
			Namespace: namespace,
			Labels: map[string]string{
				v1.IntegrationLabel:           name,
				kubernetes.ConfigMapTypeLabel: "kamelets-bundle",
			},
			Annotations: map[string]string{
				kubernetes.ConfigMapAutogenLabel: "true",
			},
		},
		Data: map[string]string{},
	}
}
