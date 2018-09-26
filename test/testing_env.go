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

package test

import (
	"time"

	"github.com/apache/camel-k/pkg/install"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	// Initializes the kubernetes client to auto-detect the context
	kubernetes.InitKubeClient("")

	err := install.SetupClusterwideResources()
	if err != nil {
		panic(err)
	}

	err = install.Operator(getTargetNamespace())
	if err != nil {
		panic(err)
	}
}

func getTargetNamespace() string {
	ns, err := kubernetes.GetClientCurrentNamespace("")
	if err != nil {
		panic(err)
	}
	return ns
}

func createTimerToLogIntegrationCode() string {
	return `
import org.apache.camel.builder.RouteBuilder;

public class Routes extends RouteBuilder {

	@Override
    public void configure() throws Exception {
        from("timer:tick")
		  .to("log:info");
    }

}
`
}

func createDummyDeployment(name string, replicas *int32, labelKey string, labelValue string, command ...string) (*appsv1.Deployment, error) {
	deployment := getDummyDeployment(name, replicas, labelKey, labelValue, command...)
	gracePeriod := int64(0)
	err := sdk.Delete(&deployment, sdk.WithDeleteOptions(&metav1.DeleteOptions{GracePeriodSeconds: &gracePeriod}))
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	}
	for {
		list := v1.PodList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: v1.SchemeGroupVersion.String(),
			},
		}

		err := sdk.List(getTargetNamespace(), &list, sdk.WithListOptions(&metav1.ListOptions{
			LabelSelector: labelKey + "=" + labelValue,
		}))
		if err != nil {
			return nil, err
		}

		if len(list.Items) > 0 {
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}
	err = sdk.Create(&deployment)
	return &deployment, err
}

func getDummyDeployment(name string, replicas *int32, labelKey string, labelValue string, command ...string) appsv1.Deployment {
	return appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: getTargetNamespace(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					labelKey: labelValue,
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						labelKey: labelValue,
					},
				},
				Spec: getDummyPod(name, command...).Spec,
			},
		},
	}
}

func createDummyPod(name string, command ...string) (*v1.Pod, error) {
	pod := getDummyPod(name, command...)
	gracePeriod := int64(0)
	err := sdk.Delete(&pod, sdk.WithDeleteOptions(&metav1.DeleteOptions{GracePeriodSeconds: &gracePeriod}))
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	}
	for {
		err := sdk.Create(&pod)
		if err != nil && k8serrors.IsAlreadyExists(err) {
			time.Sleep(1 * time.Second)
		} else if err != nil {
			return nil, err
		} else {
			break
		}
	}
	return &pod, nil
}

func getDummyPod(name string, command ...string) v1.Pod {
	gracePeriod := int64(0)
	return v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: getTargetNamespace(),
			Name:      name,
		},
		Spec: v1.PodSpec{
			TerminationGracePeriodSeconds: &gracePeriod,
			Containers: []v1.Container{
				{
					Name:    name,
					Image:   "busybox",
					Command: command,
				},
			},
		},
	}
}
