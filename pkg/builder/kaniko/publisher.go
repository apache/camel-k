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

package kaniko

import (
	"io/ioutil"
	"path"
	"time"

	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/util/tar"

	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Publisher --
func Publisher(ctx *builder.Context) error {
	image := ctx.Request.Platform.Build.Registry + "/" + ctx.Namespace + "/camel-k-" + ctx.Request.Identifier.Name + ":" + ctx.Request.Identifier.Qualifier
	baseDir, _ := path.Split(ctx.Archive)
	contextDir := path.Join(baseDir, "context")
	if err := tar.Extract(ctx.Archive, contextDir); err != nil {
		return err
	}

	dockerFileContent := []byte(`
		FROM ` + ctx.Image + `
		ADD . /deployments
	`)

	err := ioutil.WriteFile(path.Join(contextDir, "Dockerfile"), dockerFileContent, 0777)
	if err != nil {
		return err
	}

	pod := v1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ctx.Namespace,
			Name:      "camel-k-" + ctx.Request.Identifier.Name,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "kaniko",
					Image: "gcr.io/kaniko-project/executor@sha256:f29393d9c8d40296e1692417089aa2023494bce9afd632acac7dd0aea763e5bc",
					Args: []string{
						"--dockerfile=Dockerfile",
						"--context=" + contextDir,
						"--destination=" + image,
						"--insecure",
					},
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      "camel-k-builder",
							MountPath: "/workspace",
						},
					},
				},
			},
			RestartPolicy: v1.RestartPolicyNever,
			Volumes: []v1.Volume{
				{
					Name: "camel-k-builder",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: "camel-k-builder",
							ReadOnly:  true,
						},
					},
				},
			},
		},
	}

	sdk.Delete(&pod)
	err = sdk.Create(&pod)
	if err != nil {
		return errors.Wrap(err, "cannot create kaniko builder pod")
	}

	err = kubernetes.WaitCondition(&pod, func(obj interface{}) (bool, error) {
		if val, ok := obj.(*v1.Pod); ok {
			if val.Status.Phase == v1.PodSucceeded {
				return true, nil
			} else if val.Status.Phase == v1.PodFailed {
				return false, errors.New("build failed")
			}
		}
		return false, nil
	}, 10*time.Minute)

	if err != nil {
		return err
	}

	ctx.Image = image
	return nil
}
