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
	clientgocache "k8s.io/client-go/tools/cache"
	"knative.dev/serving/pkg/apis/serving"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

// ManageSyntheticIntegrations is the controller for synthetic Integrations. Consider that the lifecycle of the objects are driven
// by the way we are monitoring them. Since we're filtering by `camel.apache.org/integration` label in the cached clinet,
// you must consider an add, update or delete
// accordingly, ie, when the user label the resource, then it is considered as an add, when it removes the label, it is considered as a delete.
// We must filter only non managed objects in order to avoid to conflict with the reconciliation loop of managed objects (owned by an Integration).
func ManageSyntheticIntegrations(ctx context.Context, c client.Client, cache cache.Cache, reader ctrl.Reader) error {
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
						if it.Status.Phase == v1.IntegrationPhaseImportMissing {
							// Update with proper phase (reconciliation will take care)
							it.Status.Phase = v1.IntegrationPhaseNone
							if err = updateSyntheticIntegration(ctx, c, it); err != nil {
								log.Errorf(err, "Some error happened while updatinf a synthetic Integration %s", integrationName)
							}
						} else {
							log.Infof("Synthetic Integration %s is in phase %s. Skipping.", integrationName, it.Status.Phase)
						}
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
					// We must use a non caching client to understand if the object has been deleted from the cluster or only deleted from
					// the cache (ie, user removed the importing label)
					err := reader.Get(ctx, ctrl.ObjectKeyFromObject(ctrlObj), ctrlObj)
					if err != nil {
						if k8serrors.IsNotFound(err) {
							// Object removed from the cluster
							it, err := getSyntheticIntegration(ctx, c, ctrlObj.GetNamespace(), integrationName)
							if err != nil {
								log.Errorf(err, "Some error happened while loading a synthetic Integration %s", it.Name)
								return
							}
							// The resource from which we imported has been deleted, report in it status.
							// It may be a temporary situation, for example, if the deployment from which the Integration is imported
							// is being redeployed. For this reason we should keep the Integration instead of forcefully removing it.
							message := fmt.Sprintf(
								"import %s %s no longer available",
								it.Annotations[v1.IntegrationImportedKindLabel],
								it.Annotations[v1.IntegrationImportedNameLabel],
							)
							it.SetReadyConditionError(message)
							zero := int32(0)
							it.Status.Phase = v1.IntegrationPhaseImportMissing
							it.Status.Replicas = &zero
							if err = updateSyntheticIntegration(ctx, c, it); err != nil {
								log.Errorf(err, "Some error happened while updating a synthetic Integration %s", it.Name)
							}
							log.Infof("Updated synthetic Integration %s with status %s", it.GetName(), it.Status.Phase)
						} else {
							log.Errorf(err, "Some error happened while loading object %s from the cluster", ctrlObj.GetName())
							return
						}
					} else {
						// Importing label removed
						if err = deleteSyntheticIntegration(ctx, c, ctrlObj.GetNamespace(), integrationName); err != nil {
							log.Errorf(err, "Some error happened while deleting a synthetic Integration %s", integrationName)
						}
						log.Infof("Deleted synthetic Integration %s", integrationName)
					}
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

func updateSyntheticIntegration(ctx context.Context, c client.Client, it *v1.Integration) error {
	return c.Status().Update(ctx, it, ctrl.FieldOwner("camel-k-operator"))
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
	return nil, fmt.Errorf("unsupported %s object kind", obj)
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
	return &it
}

// getContainerNameFromDeployment returns the container name which is running the Camel application.
func (app *nonManagedCamelDeployment) getContainerNameFromDeployment() string {
	firstContainerName := ""
	for _, ct := range app.deploy.Spec.Template.Spec.Containers {
		// set as fallback if no container is named as the deployment
		if firstContainerName == "" {
			firstContainerName = app.deploy.Name
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
	return &it
}
