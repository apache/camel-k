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
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/log"
	"github.com/apache/camel-k/v2/pkg/util/s2i"
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
		return status.Failed(fmt.Errorf("cannot delete build config: %w", err))
	}

	// Set the build controller as owner reference
	owner := t.getControllerReference()
	if owner == nil {
		// Default to the Build if no controller reference is present
		owner = t.build
	}

	if err := ctrlutil.SetOwnerReference(owner, bc, t.c.GetScheme()); err != nil {
		return status.Failed(fmt.Errorf("cannot set owner reference on BuildConfig: %s: %w", bc.Name, err))
	}

	err = t.c.Create(ctx, bc)
	if err != nil {
		return status.Failed(fmt.Errorf("cannot create build config: %w", err))
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
		return status.Failed(fmt.Errorf("cannot delete image stream: %w", err))
	}

	if err := ctrlutil.SetOwnerReference(owner, is, t.c.GetScheme()); err != nil {
		return status.Failed(fmt.Errorf("cannot set owner reference on ImageStream: %s: %w", is.Name, err))
	}

	err = t.c.Create(ctx, is)
	if err != nil {
		return status.Failed(fmt.Errorf("cannot create image stream: %w", err))
	}

	err = util.WithTempDir(t.build.Name+"-s2i-", func(tmpDir string) error {
		archive := filepath.Join(tmpDir, "archive.tar.gz")

		contextDir := t.task.ContextDir
		if contextDir == "" {
			// Use the working directory.
			// This is useful when the task is executed in-container,
			// so that its WorkingDir can be used to share state and
			// coordinate with other tasks.
			pwd, err := os.Getwd()
			if err != nil {
				return err
			}
			contextDir = filepath.Join(pwd, ContextDir)
		}

		archiveFile, err := os.Create(archive)
		if err != nil {
			return fmt.Errorf("cannot create tar archive: %w", err)
		}

		err = tarDir(contextDir, archiveFile)
		if err != nil {
			return fmt.Errorf("cannot tar context directory: %w", err)
		}

		f, err := util.Open(archive)
		if err != nil {
			return err
		}

		restClient, err := apiutil.RESTClientForGVK(
			schema.GroupVersionKind{Group: "build.openshift.io", Version: "v1"}, false,
			t.c.GetConfig(), serializer.NewCodecFactory(t.c.GetScheme()))
		if err != nil {
			return err
		}

		r := restClient.Post().
			Namespace(t.build.Namespace).
			Body(bufio.NewReader(f)).
			Resource("buildconfigs").
			Name("camel-k-" + t.build.Name).
			SubResource("instantiatebinary").
			Do(ctx)

		if r.Error() != nil {
			return fmt.Errorf("cannot instantiate binary: %w", r.Error())
		}

		data, err := r.Raw()
		if err != nil {
			return fmt.Errorf("no raw data retrieved: %w", err)
		}

		s2iBuild := buildv1.Build{}
		err = json.Unmarshal(data, &s2iBuild)
		if err != nil {
			return fmt.Errorf("cannot unmarshal instantiated binary response: %w", err)
		}

		err = s2i.WaitForS2iBuildCompletion(ctx, t.c, &s2iBuild)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				// nolint: contextcheck
				if err := s2i.CancelBuild(context.Background(), t.c, &s2iBuild); err != nil {
					log.Errorf(err, "cannot cancel s2i Build: %s/%s", s2iBuild.Namespace, s2iBuild.Name)
				}
			}
			return err
		}
		if s2iBuild.Status.Output.To != nil {
			status.Digest = s2iBuild.Status.Output.To.ImageDigest
		}

		err = t.c.Get(ctx, ctrl.ObjectKeyFromObject(is), is)
		if err != nil {
			return err
		}

		if is.Status.DockerImageRepository == "" {
			return errors.New("dockerImageRepository not available in ImageStream")
		}

		status.Image = is.Status.DockerImageRepository + ":" + t.task.Tag

		return f.Close()
	})

	if err != nil {
		return status.Failed(err)
	}

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

func tarDir(src string, writers ...io.Writer) error {
	// ensure the src actually exists before trying to tar it
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("unable to tar files: %w", err)
	}

	mw := io.MultiWriter(writers...)

	gzw := gzip.NewWriter(mw)
	defer util.CloseQuietly(gzw)

	tw := tar.NewWriter(gzw)
	defer util.CloseQuietly(tw)

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
