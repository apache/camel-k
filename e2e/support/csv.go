//go:build integration
// +build integration

// To enable compilation of this file in Goland, go to "Settings -> Go -> Vendoring & Build Tags -> Custom Tags" and add "integration"

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

package support

import (
	"fmt"
	"strings"
	"testing"
	"time"
	"unsafe"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	olm "github.com/operator-framework/api/pkg/operators/v1alpha1"

	"github.com/apache/camel-k/v2/pkg/util/log"
)

func ClusterServiceVersion(t *testing.T, conditions func(olm.ClusterServiceVersion) bool, ns string) func() *olm.ClusterServiceVersion {
	return func() *olm.ClusterServiceVersion {
		lst := olm.ClusterServiceVersionList{}
		if err := TestClient(t).List(TestContext, &lst, ctrl.InNamespace(ns)); err != nil {
			panic(err)
		}
		for _, s := range lst.Items {
			if strings.Contains(s.Name, "camel-k") && conditions(s) {
				return &s
			}
		}
		return nil
	}
}

func ClusterServiceVersionPhase(t *testing.T, conditions func(olm.ClusterServiceVersion) bool, ns string) func() olm.ClusterServiceVersionPhase {
	return func() olm.ClusterServiceVersionPhase {
		if csv := ClusterServiceVersion(t, conditions, ns)(); csv != nil && unsafe.Sizeof(csv.Status) > 0 {
			return csv.Status.Phase
		}
		return ""
	}
}

func CreateOrUpdateCatalogSource(t *testing.T, ns, name, image string) error {
	catalogSource := &olm.CatalogSource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
	}

	_, err := ctrlutil.CreateOrUpdate(TestContext, TestClient(t), catalogSource, func() error {
		catalogSource.Spec = olm.CatalogSourceSpec{
			Image:       image,
			SourceType:  "grpc",
			DisplayName: "OLM upgrade test Catalog",
			Publisher:   "grpc",
		}
		return nil
	})

	return err
}

func CatalogSource(t *testing.T, ns, name string) func() *olm.CatalogSource {
	return func() *olm.CatalogSource {
		cs := &olm.CatalogSource{
			TypeMeta: metav1.TypeMeta{
				Kind:       "CatalogSource",
				APIVersion: olm.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      name,
			},
		}
		if err := TestClient(t).Get(TestContext, ctrl.ObjectKeyFromObject(cs), cs); err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			log.Errorf(err, "Error while retrieving CatalogSource %s", name)
			return nil
		}
		return cs
	}
}

func CatalogSourcePhase(t *testing.T, ns, name string) func() string {
	return func() string {
		if source := CatalogSource(t, ns, name)(); source != nil && source.Status.GRPCConnectionState != nil {
			return CatalogSource(t, ns, name)().Status.GRPCConnectionState.LastObservedState
		}
		return ""
	}
}

func CatalogSourcePod(t *testing.T, ns, csName string) func() *corev1.Pod {
	return func() *corev1.Pod {
		podList, err := TestClient(t).CoreV1().Pods(ns).List(TestContext, metav1.ListOptions{})
		if err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			panic(err)
		}

		if len(podList.Items) == 0 {
			return nil
		}

		for _, pod := range podList.Items {
			if strings.HasPrefix(pod.Name, csName) {
				return &pod
			}
		}

		return nil
	}
}

func CatalogSourcePodRunning(t *testing.T, ns, csName string) error {
	podFunc := CatalogSourcePod(t, ns, csName)

	for i := 1; i < 5; i++ {
		csPod := podFunc()
		if csPod != nil && csPod.Status.Phase == "Running" {
			return nil // Pod good to go
		}

		if i == 2 {
			fmt.Println("Catalog Source Pod still not ready so delete & allow it to be redeployed ...")
			if err := TestClient(t).Delete(TestContext, csPod); err != nil {
				return err
			}
		}

		fmt.Println("Catalog Source Pod not ready so waiting for 2 minutes ...")
		time.Sleep(2 * time.Minute)
	}

	return fmt.Errorf("Catalog Source Pod failed to reach a 'running' state")
}

func GetSubscription(t *testing.T, ns string) (*olm.Subscription, error) {
	lst := olm.SubscriptionList{}
	if err := TestClient(t).List(TestContext, &lst, ctrl.InNamespace(ns)); err != nil {
		return nil, err
	}
	for _, s := range lst.Items {
		if strings.Contains(s.Name, "camel-k") {
			return &s, nil
		}
	}
	return nil, nil
}
