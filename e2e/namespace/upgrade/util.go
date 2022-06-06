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

package common

import (
	"strings"
	"unsafe"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	olm "github.com/operator-framework/api/pkg/operators/v1alpha1"

	. "github.com/apache/camel-k/e2e/support"
	"github.com/apache/camel-k/pkg/util/log"
)

func clusterServiceVersion(conditions func(olm.ClusterServiceVersion) bool, ns string) func() *olm.ClusterServiceVersion {
	return func() *olm.ClusterServiceVersion {
		lst := olm.ClusterServiceVersionList{}
		if err := TestClient().List(TestContext, &lst, ctrl.InNamespace(ns)); err != nil {
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

func clusterServiceVersionPhase(conditions func(olm.ClusterServiceVersion) bool, ns string) func() olm.ClusterServiceVersionPhase {
	return func() olm.ClusterServiceVersionPhase {
		if csv := clusterServiceVersion(conditions, ns)(); csv != nil && unsafe.Sizeof(csv.Status) > 0 {
			return csv.Status.Phase
		}
		return ""
	}
}

func createOrUpdateCatalogSource(ns, name, image string) error {
	catalogSource := &olm.CatalogSource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
	}

	_, err := ctrlutil.CreateOrUpdate(TestContext, TestClient(), catalogSource, func() error {
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

func catalogSource(ns, name string) func() *olm.CatalogSource {
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
		if err := TestClient().Get(TestContext, ctrl.ObjectKeyFromObject(cs), cs); err != nil && errors.IsNotFound(err) {
			return nil
		} else if err != nil {
			log.Errorf(err, "Error while retrieving CatalogSource %s", name)
			return nil
		}
		return cs
	}
}

func catalogSourcePhase(ns, name string) func() string {
	return func() string {
		if source := catalogSource(ns, name)(); source != nil && source.Status.GRPCConnectionState != nil {
			return catalogSource(ns, name)().Status.GRPCConnectionState.LastObservedState
		}
		return ""
	}
}

func getSubscription(ns string) (*olm.Subscription, error) {
	lst := olm.SubscriptionList{}
	if err := TestClient().List(TestContext, &lst, ctrl.InNamespace(ns)); err != nil {
		return nil, err
	}
	for _, s := range lst.Items {
		if strings.Contains(s.Name, "camel-k") {
			return &s, nil
		}
	}
	return nil, nil
}
