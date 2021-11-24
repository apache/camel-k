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
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/apache/camel-k/pkg/util"

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
	"github.com/apache/camel-k/pkg/util/log"
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
	archive := path.Join(tmpDir, "archive.tar.gz")
	defer os.RemoveAll(tmpDir)

	contextDir := t.task.ContextDir
	if contextDir == "" {
		// Use the working directory.
		// This is useful when the task is executed in-container,
		// so that its WorkingDir can be used to share state and
		// coordinate with other tasks.
		pwd, err := os.Getwd()
		if err != nil {
			return status.Failed(err)
		}
		contextDir = path.Join(pwd, ContextDir)
	}

	archiveFile, err := os.Create(archive)
	if err != nil {
		return status.Failed(errors.Wrap(err, "cannot create tar archive"))
	}

	err = tarDir(contextDir, archiveFile)
	if err != nil {
		return status.Failed(errors.Wrap(err, "cannot tar context directory"))
	}

	resource, err := util.ReadFile(archive)
	if err != nil {
		return status.Failed(errors.Wrap(err, "cannot read tar file "+archive))
	}

	restClient, err := kubernetes.GetClientFor(t.c, "build.openshift.io", "v1")
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

	s2iBuild := buildv1.Build{}
	err = json.Unmarshal(data, &s2iBuild)
	if err != nil {
		return status.Failed(errors.Wrap(err, "cannot unmarshal instantiated binary response"))
	}

	err = t.waitForS2iBuildCompletion(ctx, t.c, &s2iBuild)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			// nolint: contextcheck
			if err := t.cancelBuild(context.Background(), &s2iBuild); err != nil {
				log.Errorf(err, "cannot cancel s2i Build: %s/%s", s2iBuild.Namespace, s2iBuild.Name)
			}
		}
		return status.Failed(err)
	}
	if s2iBuild.Status.Output.To != nil {
		status.Digest = s2iBuild.Status.Output.To.ImageDigest
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

func (t *s2iTask) waitForS2iBuildCompletion(ctx context.Context, c client.Client, build *buildv1.Build) error {
	key := ctrl.ObjectKeyFromObject(build)
	for {
		select {

		case <-ctx.Done():
			return ctx.Err()

		case <-time.After(1 * time.Second):
			err := c.Get(ctx, key, build)
			if err != nil {
				if apierrors.IsNotFound(err) {
					continue
				}
				return err
			}

			if build.Status.Phase == buildv1.BuildPhaseComplete {
				return nil
			} else if build.Status.Phase == buildv1.BuildPhaseCancelled ||
				build.Status.Phase == buildv1.BuildPhaseFailed ||
				build.Status.Phase == buildv1.BuildPhaseError {
				return errors.New("build failed")
			}
		}
	}
}

func (t *s2iTask) cancelBuild(ctx context.Context, build *buildv1.Build) error {
	target := build.DeepCopy()
	target.Status.Cancelled = true
	if err := t.c.Patch(ctx, target, ctrl.MergeFrom(build)); err != nil {
		return err
	}
	*build = *target
	return nil
}

func tarDir(src string, writers ...io.Writer) error {
	// ensure the src actually exists before trying to tar it
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("unable to tar files: %w", err)
	}

	mw := io.MultiWriter(writers...)

	gzw := gzip.NewWriter(mw)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	return filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !fi.Mode().IsRegular() {
			return nil
		}

		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

		// update the name to correctly reflect the desired destination when un-taring
		header.Name = strings.TrimPrefix(strings.ReplaceAll(file, src, ""), string(filepath.Separator))

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		f, err := util.Open(file)
		if err != nil {
			return err
		}

		if _, err := io.Copy(tw, f); err != nil {
			return err
		}

		// manually close here after each file operation; deferring would cause each file close
		// to wait until all operations have completed.
		return f.Close()
	})
}
