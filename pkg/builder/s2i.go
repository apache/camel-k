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

package builder

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/json"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/kubernetes/customclient"
	"github.com/apache/camel-k/pkg/util/zip"
)

type s2iTask struct {
	c     client.Client
	build *v1.Build
	task  *v1.S2iTask
}

var _ Task = &s2iTask{}

func (t *s2iTask) Do(ctx context.Context) v1.BuildStatus {
	status := v1.BuildStatus{}

	bc := &buildv1.BuildConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: buildv1.GroupVersion.String(),
			Kind:       "BuildConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "camel-k-" + t.build.Name,
			Namespace: t.build.Namespace,
			Labels:    t.build.Labels,
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
						Name: "camel-k-" + t.build.Name + ":" + t.task.Tag,
					},
				},
			},
		},
	}

	err := t.c.Delete(ctx, bc)
	if err != nil && !apierrors.IsNotFound(err) {
		return status.Failed(errors.Wrap(err, "cannot delete build config"))
	}

	// Set the build controller as owner reference
	owner := t.getControllerReference()
	if owner == nil {
		// Default to the Build if no controller reference is present
		owner = t.build
	}

	if err := ctrlutil.SetOwnerReference(owner, bc, t.c.GetScheme()); err != nil {
		return status.Failed(errors.Wrapf(err, "cannot set owner reference on BuildConfig: %s", bc.Name))
	}

	err = t.c.Create(ctx, bc)
	if err != nil {
		return status.Failed(errors.Wrap(err, "cannot create build config"))
	}

	is := &imagev1.ImageStream{
		TypeMeta: metav1.TypeMeta{
			APIVersion: imagev1.GroupVersion.String(),
			Kind:       "ImageStream",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "camel-k-" + t.build.Name,
			Namespace: t.build.Namespace,
			Labels:    t.build.Labels,
		},
		Spec: imagev1.ImageStreamSpec{
			LookupPolicy: imagev1.ImageLookupPolicy{
				Local: true,
			},
		},
	}

	err = t.c.Delete(ctx, is)
	if err != nil && !apierrors.IsNotFound(err) {
		return status.Failed(errors.Wrap(err, "cannot delete image stream"))
	}

	if err := ctrlutil.SetOwnerReference(owner, is, t.c.GetScheme()); err != nil {
		return status.Failed(errors.Wrapf(err, "cannot set owner reference on ImageStream: %s", is.Name))
	}

	err = t.c.Create(ctx, is)
	if err != nil {
		return status.Failed(errors.Wrap(err, "cannot create image stream"))
	}

	tmpDir, err := ioutil.TempDir(os.TempDir(), t.build.Name+"-s2i-")
	if err != nil {
		return status.Failed(err)
	}
	archive := path.Join(tmpDir, "archive.zip")
	defer os.RemoveAll(tmpDir)

	err = zip.Directory(t.task.ContextDir, archive)
	if err != nil {
		return status.Failed(errors.Wrap(err, "cannot zip context directory"))
	}

	resource, err := ioutil.ReadFile(archive)
	if err != nil {
		return status.Failed(errors.Wrap(err, "cannot fully read zip file "+archive))
	}

	restClient, err := customclient.GetClientFor(t.c, "build.openshift.io", "v1")
	if err != nil {
		return status.Failed(err)
	}

	r := restClient.Post().
		Namespace(t.build.Namespace).
		Body(resource).
		Resource("buildconfigs").
		Name("camel-k-" + t.build.Name).
		SubResource("instantiatebinary").
		Do(ctx)

	if r.Error() != nil {
		return status.Failed(errors.Wrap(r.Error(), "cannot instantiate binary"))
	}

	data, err := r.Raw()
	if err != nil {
		return status.Failed(errors.Wrap(err, "no raw data retrieved"))
	}

	ocbuild := buildv1.Build{}
	err = json.Unmarshal(data, &ocbuild)
	if err != nil {
		return status.Failed(errors.Wrap(err, "cannot unmarshal instantiated binary response"))
	}

	// FIXME: Use context.WithTimeout
	err = kubernetes.WaitCondition(ctx, t.c, &ocbuild, func(obj interface{}) (bool, error) {
		if val, ok := obj.(*buildv1.Build); ok {
			if val.Status.Phase == buildv1.BuildPhaseComplete {
				if val.Status.Output.To != nil {
					status.Digest = val.Status.Output.To.ImageDigest
				}
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
		return status.Failed(err)
	}

	err = t.c.Get(ctx, ctrl.ObjectKeyFromObject(is), is)
	if err != nil {
		return status.Failed(err)
	}

	if is.Status.DockerImageRepository == "" {
		return status.Failed(errors.New("dockerImageRepository not available in ImageStream"))
	}

	status.Image = is.Status.DockerImageRepository + ":" + t.task.Tag

	return status
}

func (t *s2iTask) getControllerReference() metav1.Object {
	var owner metav1.Object
	for _, ref := range t.build.GetOwnerReferences() {
		if ref.Controller != nil && *ref.Controller {
			o := &unstructured.Unstructured{}
			o.SetNamespace(t.build.Namespace)
			o.SetName(ref.Name)
			o.SetUID(ref.UID)
			o.SetAPIVersion(ref.APIVersion)
			o.SetKind(ref.Kind)
			owner = o
			break
		}
	}
	return owner
}
