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

package synthetic

import (
	"context"
	"fmt"
	"reflect"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/log"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientgocache "k8s.io/client-go/tools/cache"
	"knative.dev/serving/pkg/apis/serving"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	controller         = true
	blockOwnerDeletion = true
)

// ManageSyntheticIntegrations is the controller for synthetic Integrations. Consider that the lifecycle of the objects are driven
// by the way we are monitoring them. Since we're filtering by `camel.apache.org/integration` label in the cached client,
// you must consider an add, update or delete
// accordingly, ie, when the user label the resource, then it is considered as an add, when it removes the label, it is considered as a delete.
// We must filter only non managed objects in order to avoid to conflict with the reconciliation loop of managed objects (owned by an Integration).
func ManageSyntheticIntegrations(ctx context.Context, c client.Client, cache cache.Cache) error {
	informers, err := getInformers(ctx, c, cache)
	if err != nil {
		return err
	}
	for _, informer := range informers {
		_, err := informer.AddEventHandler(clientgocache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				ctrlObj, ok := obj.(ctrl.Object)
				if !ok {
					log.Error(fmt.Errorf("type assertion failed: %v", obj), "Failed to retrieve Object on add event")
					return
				}
				if !isManagedObject(ctrlObj) {
					integrationName := ctrlObj.GetLabels()[v1.IntegrationLabel]
					it, err := getSyntheticIntegration(ctx, c, ctrlObj.GetNamespace(), integrationName)
					if err != nil {
						if k8serrors.IsNotFound(err) {
							adapter, err := nonManagedCamelApplicationFactory(ctrlObj)
							if err != nil {
								log.Errorf(err, "Some error happened while creating a Camel application adapter for %s", integrationName)
							}
							if err = createSyntheticIntegration(ctx, c, adapter.Integration()); err != nil {
								log.Errorf(err, "Some error happened while creating a synthetic Integration %s", integrationName)
							}
							log.Infof("Created a synthetic Integration %s after %s resource object", it.GetName(), ctrlObj.GetName())
						} else {
							log.Errorf(err, "Some error happened while loading a synthetic Integration %s", integrationName)
						}
					} else {
						log.Infof("Synthetic Integration %s is in phase %s. Skipping.", integrationName, it.Status.Phase)
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				ctrlObj, ok := obj.(ctrl.Object)
				if !ok {
					log.Error(fmt.Errorf("type assertion failed: %v", obj), "Failed to retrieve Object on delete event")
					return
				}
				if !isManagedObject(ctrlObj) {
					integrationName := ctrlObj.GetLabels()[v1.IntegrationLabel]
					// Importing label removed
					if err = deleteSyntheticIntegration(ctx, c, ctrlObj.GetNamespace(), integrationName); err != nil {
						log.Errorf(err, "Some error happened while deleting a synthetic Integration %s", integrationName)
					}
					log.Infof("Deleted synthetic Integration %s", integrationName)
				}
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func getInformers(ctx context.Context, cl client.Client, c cache.Cache) ([]cache.Informer, error) {
	deploy, err := c.GetInformer(ctx, &appsv1.Deployment{})
	if err != nil {
		return nil, err
	}
	informers := []cache.Informer{deploy}
	// Watch for the CronJob conditionally
	if ok, err := kubernetes.IsAPIResourceInstalled(cl, batchv1.SchemeGroupVersion.String(), reflect.TypeOf(batchv1.CronJob{}).Name()); ok && err == nil {
		cron, err := c.GetInformer(ctx, &batchv1.CronJob{})
		if err != nil {
			return nil, err
		}
		informers = append(informers, cron)
	}
	// Watch for the Knative Services conditionally
	if ok, err := kubernetes.IsAPIResourceInstalled(cl, servingv1.SchemeGroupVersion.String(), reflect.TypeOf(servingv1.Service{}).Name()); ok && err == nil {
		if ok, err := kubernetes.CheckPermission(ctx, cl, serving.GroupName, "services", platform.GetOperatorWatchNamespace(), "", "watch"); ok && err == nil {
			ksvc, err := c.GetInformer(ctx, &servingv1.Service{})
			if err != nil {
				return nil, err
			}
			informers = append(informers, ksvc)
		}
	}

	return informers, nil
}

func getSyntheticIntegration(ctx context.Context, c client.Client, namespace, name string) (*v1.Integration, error) {
	it := v1.NewIntegration(namespace, name)
	err := c.Get(ctx, ctrl.ObjectKeyFromObject(&it), &it)
	return &it, err
}

func createSyntheticIntegration(ctx context.Context, c client.Client, it *v1.Integration) error {
	return c.Create(ctx, it, ctrl.FieldOwner("camel-k-operator"))
}

func deleteSyntheticIntegration(ctx context.Context, c client.Client, namespace, name string) error {
	// As the Integration label was removed, we don't know which is the Synthetic integration to remove
	it := v1.NewIntegration(namespace, name)
	return c.Delete(ctx, &it)
}

// isManagedObject returns true if the object is managed by an Integration.
func isManagedObject(obj ctrl.Object) bool {
	for _, mr := range obj.GetOwnerReferences() {
		if mr.APIVersion == "camel.apache.org/v1" &&
			mr.Kind == "Integration" {
			return true
		}
	}
	return false
}

// nonManagedCamelApplicationAdapter represents a Camel application built and deployed outside the operator lifecycle.
type nonManagedCamelApplicationAdapter interface {
	// Integration return an Integration resource fed by the Camel application adapter.
	Integration() *v1.Integration
}

func nonManagedCamelApplicationFactory(obj ctrl.Object) (nonManagedCamelApplicationAdapter, error) {
	deploy, ok := obj.(*appsv1.Deployment)
	if ok {
		return &nonManagedCamelDeployment{deploy: deploy}, nil
	}
	cronjob, ok := obj.(*batchv1.CronJob)
	if ok {
		return &NonManagedCamelCronjob{cron: cronjob}, nil
	}
	ksvc, ok := obj.(*servingv1.Service)
	if ok {
		return &NonManagedCamelKnativeService{ksvc: ksvc}, nil
	}
	return nil, fmt.Errorf("unsupported %s object kind", obj.GetName())
}

// NonManagedCamelDeployment represents a regular Camel application built and deployed outside the operator lifecycle.
type nonManagedCamelDeployment struct {
	deploy *appsv1.Deployment
}

// Integration return an Integration resource fed by the Camel application adapter.
func (app *nonManagedCamelDeployment) Integration() *v1.Integration {
	it := v1.NewIntegration(app.deploy.Namespace, app.deploy.Labels[v1.IntegrationLabel])
	it.SetAnnotations(map[string]string{
		v1.IntegrationImportedNameLabel: app.deploy.Name,
		v1.IntegrationImportedKindLabel: "Deployment",
		v1.IntegrationSyntheticLabel:    "true",
	})
	it.Spec = v1.IntegrationSpec{
		Traits: v1.Traits{
			Container: &trait.ContainerTrait{
				Name: app.getContainerNameFromDeployment(),
			},
		},
	}
	references := []metav1.OwnerReference{
		{
			APIVersion:         "apps/v1",
			Kind:               "Deployment",
			Name:               app.deploy.Name,
			UID:                app.deploy.UID,
			Controller:         &controller,
			BlockOwnerDeletion: &blockOwnerDeletion,
		},
	}
	it.SetOwnerReferences(references)
	return &it
}

// getContainerNameFromDeployment returns the container name which is running the Camel application.
func (app *nonManagedCamelDeployment) getContainerNameFromDeployment() string {
	firstContainerName := ""
	for _, ct := range app.deploy.Spec.Template.Spec.Containers {
		// set as fallback if no container is named as the deployment
		if firstContainerName == "" {
			firstContainerName = ct.Name
		}
		if ct.Name == app.deploy.Name {
			return app.deploy.Name
		}
	}
	return firstContainerName
}

// NonManagedCamelCronjob represents a cron Camel application built and deployed outside the operator lifecycle.
type NonManagedCamelCronjob struct {
	cron *batchv1.CronJob
}

// Integration return an Integration resource fed by the Camel application adapter.
func (app *NonManagedCamelCronjob) Integration() *v1.Integration {
	it := v1.NewIntegration(app.cron.Namespace, app.cron.Labels[v1.IntegrationLabel])
	it.SetAnnotations(map[string]string{
		v1.IntegrationImportedNameLabel: app.cron.Name,
		v1.IntegrationImportedKindLabel: "CronJob",
		v1.IntegrationSyntheticLabel:    "true",
	})
	it.Spec = v1.IntegrationSpec{
		Traits: v1.Traits{},
	}
	references := []metav1.OwnerReference{
		{
			APIVersion:         "batch/v1",
			Kind:               "CronJob",
			Name:               app.cron.Name,
			UID:                app.cron.UID,
			Controller:         &controller,
			BlockOwnerDeletion: &blockOwnerDeletion,
		},
	}
	it.SetOwnerReferences(references)
	return &it
}

// NonManagedCamelKnativeService represents a Knative Service based Camel application built and deployed outside the operator lifecycle.
type NonManagedCamelKnativeService struct {
	ksvc *servingv1.Service
}

// Integration return an Integration resource fed by the Camel application adapter.
func (app *NonManagedCamelKnativeService) Integration() *v1.Integration {
	it := v1.NewIntegration(app.ksvc.Namespace, app.ksvc.Labels[v1.IntegrationLabel])
	it.SetAnnotations(map[string]string{
		v1.IntegrationImportedNameLabel: app.ksvc.Name,
		v1.IntegrationImportedKindLabel: "KnativeService",
		v1.IntegrationSyntheticLabel:    "true",
	})
	it.Spec = v1.IntegrationSpec{
		Traits: v1.Traits{},
	}
	references := []metav1.OwnerReference{
		{
			APIVersion:         servingv1.SchemeGroupVersion.String(),
			Kind:               "Service",
			Name:               app.ksvc.Name,
			UID:                app.ksvc.UID,
			Controller:         &controller,
			BlockOwnerDeletion: &blockOwnerDeletion,
		},
	}
	it.SetOwnerReferences(references)
	return &it
}
