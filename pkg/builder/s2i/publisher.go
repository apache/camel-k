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

package s2i

import (
	"io/ioutil"
	"time"

	"k8s.io/apimachinery/pkg/util/json"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/camel-k/pkg/builder"

	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/kubernetes/customclient"
	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pkg/errors"
)

// Publisher --
func Publisher(ctx *builder.Context) error {
	bc := buildv1.BuildConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: buildv1.SchemeGroupVersion.String(),
			Kind:       "BuildConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "camel-k-" + ctx.Request.Meta.Name,
			Namespace: ctx.Namespace,
		},
		Spec: buildv1.BuildConfigSpec{
			CommonSpec: buildv1.CommonSpec{
				Source: buildv1.BuildSource{
					Type: buildv1.BuildSourceBinary,
				},
				Strategy: buildv1.BuildStrategy{
					SourceStrategy: &buildv1.SourceBuildStrategy{
						From: v1.ObjectReference{
							Kind: "DockerImage",
							Name: ctx.Image,
						},
					},
				},
				Output: buildv1.BuildOutput{
					To: &v1.ObjectReference{
						Kind: "ImageStreamTag",
						Name: "camel-k-" + ctx.Request.Meta.Name + ":" + ctx.Request.Meta.ResourceVersion,
					},
				},
			},
		},
	}

	err := ctx.Client.Delete(ctx.C, &bc)
	if err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrap(err, "cannot delete build config")
	}

	err = ctx.Client.Create(ctx.C, &bc)
	if err != nil {
		return errors.Wrap(err, "cannot create build config")
	}

	is := imagev1.ImageStream{
		TypeMeta: metav1.TypeMeta{
			APIVersion: imagev1.SchemeGroupVersion.String(),
			Kind:       "ImageStream",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "camel-k-" + ctx.Request.Meta.Name,
			Namespace: ctx.Namespace,
		},
		Spec: imagev1.ImageStreamSpec{
			LookupPolicy: imagev1.ImageLookupPolicy{
				Local: true,
			},
		},
	}

	err = ctx.Client.Delete(ctx.C, &is)
	if err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrap(err, "cannot delete image stream")
	}

	err = ctx.Client.Create(ctx.C, &is)
	if err != nil {
		return errors.Wrap(err, "cannot create image stream")
	}

	resource, err := ioutil.ReadFile(ctx.Archive)
	if err != nil {
		return errors.Wrap(err, "cannot fully read tar file "+ctx.Archive)
	}

	restClient, err := customclient.GetClientFor(ctx.Client, "build.openshift.io", "v1")
	if err != nil {
		return err
	}

	result := restClient.Post().
		Namespace(ctx.Namespace).
		Body(resource).
		Resource("buildconfigs").
		Name("camel-k-" + ctx.Request.Meta.Name).
		SubResource("instantiatebinary").
		Do()

	if result.Error() != nil {
		return errors.Wrap(result.Error(), "cannot instantiate binary")
	}

	data, err := result.Raw()
	if err != nil {
		return errors.Wrap(err, "no raw data retrieved")
	}

	ocbuild := buildv1.Build{}
	err = json.Unmarshal(data, &ocbuild)
	if err != nil {
		return errors.Wrap(err, "cannot unmarshal instantiated binary response")
	}

	err = kubernetes.WaitCondition(ctx.C, ctx.Client, &ocbuild, func(obj interface{}) (bool, error) {
		if val, ok := obj.(*buildv1.Build); ok {
			if val.Status.Phase == buildv1.BuildPhaseComplete {
				return true, nil
			} else if val.Status.Phase == buildv1.BuildPhaseCancelled ||
				val.Status.Phase == buildv1.BuildPhaseFailed ||
				val.Status.Phase == buildv1.BuildPhaseError {
				return false, errors.New("build failed")
			}
		}
		return false, nil
	}, 5*time.Minute)
	if err != nil {
		return err
	}

	key, err := k8sclient.ObjectKeyFromObject(&is)
	if err != nil {
		return err
	}
	err = ctx.Client.Get(ctx.C, key, &is)
	if err != nil {
		return err
	}

	if is.Status.DockerImageRepository == "" {
		return errors.New("dockerImageRepository not available in ImageStream")
	}

	ctx.Image = is.Status.DockerImageRepository + ":" + ctx.Request.Meta.ResourceVersion

	return nil
}
