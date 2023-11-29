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

package integration

import (
	"context"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/util/log"
	"github.com/apache/camel-k/v2/pkg/util/patch"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// nonManagedCamelAppEnqueueRequestsFromMapFunc represent the function to discover the Integration which has to be woke up: it creates a synthetic
// Integration if the Integration does not exist. This is used to import external Camel applications.
func nonManagedCamelAppEnqueueRequestsFromMapFunc(ctx context.Context, c client.Client, adp NonManagedCamelApplicationAdapter) []reconcile.Request {
	if adp.GetIntegrationName() == "" {
		return []reconcile.Request{}
	}
	it := v1.NewIntegration(adp.GetIntegrationNameSpace(), adp.GetIntegrationName())
	err := c.Get(ctx, ctrl.ObjectKeyFromObject(&it), &it)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// We must perform this check to make sure the resource is not being deleted.
			// In such case it makes no sense to create an Integration after it.
			err := c.Get(ctx, ctrl.ObjectKeyFromObject(adp.GetAppObj()), adp.GetAppObj())
			if err != nil {
				if k8serrors.IsNotFound(err) {
					return []reconcile.Request{}
				}
				log.Errorf(err, "Some error happened while trying to get %s %s resource", adp.GetName(), adp.GetKind())
			}
			createSyntheticIntegration(&it, adp)
			target, err := patch.ApplyPatch(&it)
			if err == nil {
				err = c.Patch(ctx, target, ctrl.Apply, ctrl.ForceOwnership, ctrl.FieldOwner("camel-k-operator"))
				if err != nil {
					log.Errorf(err, "Some error happened while creating a synthetic Integration after %s %s resource", adp.GetName(), adp.GetKind())
					return []reconcile.Request{}
				}
				log.Infof(
					"Created a synthetic Integration %s after %s %s",
					it.GetName(),
					adp.GetName(),
					adp.GetKind(),
				)
				return []reconcile.Request{
					{
						NamespacedName: types.NamespacedName{
							Namespace: it.Namespace,
							Name:      it.Name,
						},
					},
				}
			}
			if err != nil {
				log.Infof("Could not create Integration %s: %s", adp.GetIntegrationName(), err.Error())
				return []reconcile.Request{}
			}
		}
		log.Errorf(err, "Could not get Integration %s", it.GetName())
		return []reconcile.Request{}
	}

	return []reconcile.Request{
		{
			NamespacedName: types.NamespacedName{
				Namespace: it.Namespace,
				Name:      it.Name,
			},
		},
	}
}

// createSyntheticIntegration set all required values for a synthetic Integration.
func createSyntheticIntegration(it *v1.Integration, adp NonManagedCamelApplicationAdapter) {
	// We need to create a synthetic Integration
	it.SetAnnotations(map[string]string{
		v1.IntegrationImportedNameLabel: adp.GetName(),
		v1.IntegrationImportedKindLabel: adp.GetKind(),
		v1.IntegrationSyntheticLabel:    "true",
	})
	it.Spec = v1.IntegrationSpec{
		Traits: adp.GetTraits(),
	}
}

// NonManagedCamelApplicationAdapter represents a Camel application built and deployed outside the operator lifecycle.
type NonManagedCamelApplicationAdapter interface {
	// GetName returns the name of the Camel application.
	GetName() string
	// GetKind returns the kind of the Camel application (ie, Deployment, Cronjob, ...).
	GetKind() string
	// GetTraits in used to retrieve the trait configuration.
	GetTraits() v1.Traits
	// GetIntegrationName return the name of the Integration which has to be imported.
	GetIntegrationName() string
	// GetIntegrationNameSpace return the namespace of the Integration which has to be imported.
	GetIntegrationNameSpace() string
	// GetAppObj return the object from which we're importing.
	GetAppObj() ctrl.Object
}

// NonManagedCamelDeployment represents a regular Camel application built and deployed outside the operator lifecycle.
type NonManagedCamelDeployment struct {
	deploy *appsv1.Deployment
}

// GetName returns the name of the Camel application.
func (app *NonManagedCamelDeployment) GetName() string {
	return app.deploy.GetName()
}

// GetKind returns the kind of the Camel application (ie, Deployment, Cronjob, ...).
func (app *NonManagedCamelDeployment) GetKind() string {
	return "Deployment"
}

// GetTraits in used to retrieve the trait configuration.
func (app *NonManagedCamelDeployment) GetTraits() v1.Traits {
	return v1.Traits{
		Container: &trait.ContainerTrait{
			Name: app.getContainerNameFromDeployment(),
		},
	}
}

// GetAppObj return the object from which we're importing.
func (app *NonManagedCamelDeployment) GetAppObj() ctrl.Object {
	return app.deploy
}

// GetIntegrationName return the name of the Integration which has to be imported.
func (app *NonManagedCamelDeployment) GetIntegrationName() string {
	return app.deploy.Labels[v1.IntegrationLabel]
}

// GetIntegrationNameSpace return the namespace of the Integration which has to be imported.
func (app *NonManagedCamelDeployment) GetIntegrationNameSpace() string {
	return app.deploy.Namespace
}

// getContainerNameFromDeployment returns the container name which is running the Camel application.
func (app *NonManagedCamelDeployment) getContainerNameFromDeployment() string {
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

// GetName returns the name of the Camel application.
func (app *NonManagedCamelCronjob) GetName() string {
	return app.cron.GetName()
}

// GetKind returns the kind of the Camel application (ie, Deployment, Cronjob, ...).
func (app *NonManagedCamelCronjob) GetKind() string {
	return "CronJob"
}

// GetTraits in used to retrieve the trait configuration.
func (app *NonManagedCamelCronjob) GetTraits() v1.Traits {
	return v1.Traits{}
}

// GetIntegrationName return the name of the Integration which has to be imported.
func (app *NonManagedCamelCronjob) GetIntegrationName() string {
	return app.cron.Labels[v1.IntegrationLabel]
}

// GetIntegrationNameSpace return the namespace of the Integration which has to be imported.
func (app *NonManagedCamelCronjob) GetIntegrationNameSpace() string {
	return app.cron.Namespace
}

// GetAppObj return the object from which we're importing.
func (app *NonManagedCamelCronjob) GetAppObj() ctrl.Object {
	return app.cron
}

// NonManagedCamelKnativeService represents a Knative Service based Camel application built and deployed outside the operator lifecycle.
type NonManagedCamelKnativeService struct {
	ksvc *servingv1.Service
}

// GetName returns the name of the Camel application.
func (app *NonManagedCamelKnativeService) GetName() string {
	return app.ksvc.GetName()
}

// GetKind returns the kind of the Camel application (ie, Deployment, Cronjob, ...).
func (app *NonManagedCamelKnativeService) GetKind() string {
	return "KnativeService"
}

// GetTraits in used to retrieve the trait configuration.
func (app *NonManagedCamelKnativeService) GetTraits() v1.Traits {
	return v1.Traits{}
}

// GetIntegrationName return the name of the Integration which has to be imported.
func (app *NonManagedCamelKnativeService) GetIntegrationName() string {
	return app.ksvc.Labels[v1.IntegrationLabel]
}

// GetIntegrationNameSpace return the namespace of the Integration which has to be imported.
func (app *NonManagedCamelKnativeService) GetIntegrationNameSpace() string {
	return app.ksvc.Namespace
}

// GetAppObj return the object from which we're importing.
func (app *NonManagedCamelKnativeService) GetAppObj() ctrl.Object {
	return app.ksvc
}
