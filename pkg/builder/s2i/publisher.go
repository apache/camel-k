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
	"os"
	"path"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"

	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/kubernetes/customclient"
	"github.com/apache/camel-k/pkg/util/zip"
)

func publisher(ctx *builder.Context) error {
	bc := buildv1.BuildConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: buildv1.GroupVersion.String(),
			Kind:       "BuildConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "camel-k-" + ctx.Build.Name,
			Namespace: ctx.Namespace,
			Labels:    ctx.Build.Labels,
		},
		Spec: buildv1.BuildConfigSpec{
			CommonSpec: buildv1.CommonSpec{
				Source: buildv1.BuildSource{
					Type: buildv1.BuildSourceBinary,
				},
				Strategy: buildv1.BuildStrategy{
					DockerStrategy: &buildv1.DockerBuildStrategy{},
				},
				Output: buildv1.BuildOutput{
					To: &corev1.ObjectReference{
						Kind: "ImageStreamTag",
						Name: "camel-k-" + ctx.Build.Name + ":" + ctx.Build.Tag,
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
			APIVersion: imagev1.GroupVersion.String(),
			Kind:       "ImageStream",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "camel-k-" + ctx.Build.Name,
			Namespace: ctx.Namespace,
			Labels:    ctx.Build.Labels,
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

	archive := path.Join(ctx.Path, "archive.zip")
	err = zip.Directory(path.Join(ctx.Path, "context"), archive)
	if err != nil {
		return errors.Wrap(err, "cannot zip context directory")
	}

	resource, err := ioutil.ReadFile(archive)
	if err != nil {
		return errors.Wrap(err, "cannot fully read zip file "+archive)
	}

	defer os.RemoveAll(ctx.Path)

	restClient, err := customclient.GetClientFor(ctx.Client, "build.openshift.io", "v1")
	if err != nil {
		return err
	}

	result := restClient.Post().
		Namespace(ctx.Namespace).
		Body(resource).
		Resource("buildconfigs").
		Name("camel-k-" + ctx.Build.Name).
		SubResource("instantiatebinary").
		Do(ctx.C)

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
				if val.Status.Output.To != nil {
					ctx.Digest = val.Status.Output.To.ImageDigest
				}
				return true, nil
			} else if val.Status.Phase == buildv1.BuildPhaseCancelled ||
				val.Status.Phase == buildv1.BuildPhaseFailed ||
				val.Status.Phase == buildv1.BuildPhaseError {
				return false, errors.New("build failed")
			}
		}
		return false, nil
	}, ctx.Build.Timeout.Duration)

	if err != nil {
		return err
	}

	err = ctx.Client.Get(ctx.C, ctrl.ObjectKeyFromObject(&is), &is)
	if err != nil {
		return err
	}

	if is.Status.DockerImageRepository == "" {
		return errors.New("dockerImageRepository not available in ImageStream")
	}

	ctx.Image = is.Status.DockerImageRepository + ":" + ctx.Build.Tag

	return nil
}
