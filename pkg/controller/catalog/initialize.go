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

package catalog

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
	"runtime"
	"strings"
	"time"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/builder"
	"github.com/apache/camel-k/v2/pkg/client"
	platformutil "github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/openshift"
	"github.com/apache/camel-k/v2/pkg/util/s2i"

	spectrum "github.com/container-tools/spectrum/pkg/builder"

	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// NewInitializeAction returns a action that initializes the catalog configuration when not provided by the user.
func NewInitializeAction() Action {
	return &initializeAction{}
}

type initializeAction struct {
	baseAction
}

func (action *initializeAction) Name() string {
	return "initialize"
}

func (action *initializeAction) CanHandle(catalog *v1.CamelCatalog) bool {
	return catalog.Status.Phase == v1.CamelCatalogPhaseNone
}

func (action *initializeAction) Handle(ctx context.Context, catalog *v1.CamelCatalog) (*v1.CamelCatalog, error) {
	action.L.Info("Initializing CamelCatalog")

	platform, err := platformutil.GetOrFindLocal(ctx, action.client, catalog.Namespace)

	if err != nil {
		return catalog, err
	}

	if platform.Status.Phase != v1.IntegrationPlatformPhaseReady {
		// Wait the platform to be ready
		return catalog, nil
	}

	if platform.Status.Build.PublishStrategy == v1.IntegrationPlatformBuildPublishStrategyS2I {
		return initializeS2i(ctx, action.client, platform, catalog)
	}
	// Default to spectrum
	// Make basic options for building image in the registry
	options, err := makeSpectrumOptions(ctx, action.client, platform.Namespace, platform.Status.Build.Registry)
	if err != nil {
		return catalog, err
	}
	return initializeSpectrum(options, platform, catalog)

}

func initializeSpectrum(options spectrum.Options, ip *v1.IntegrationPlatform, catalog *v1.CamelCatalog) (*v1.CamelCatalog, error) {
	target := catalog.DeepCopy()
	organization := ip.Status.Build.Registry.Organization
	if organization == "" {
		organization = catalog.Namespace
	}
	imageName := fmt.Sprintf(
		"%s/%s/camel-k-runtime-%s-builder:%s",
		ip.Status.Build.Registry.Address,
		organization,
		catalog.Spec.Runtime.Provider,
		strings.ToLower(catalog.Spec.Runtime.Version),
	)

	newStdR, newStdW, pipeErr := os.Pipe()
	defer util.CloseQuietly(newStdW)

	if pipeErr != nil {
		// In the unlikely case of an error, use stdout instead of aborting
		Log.Errorf(pipeErr, "Unable to remap I/O. Spectrum messages will be displayed on the stdout")
		newStdW = os.Stdout
	}
	go readSpectrumLogs(newStdR)

	// We use the future target image as a base just for the sake of pulling and verify it exists
	options.Base = imageName
	options.Stderr = newStdW
	options.Stdout = newStdW

	if !imageSnapshot(options.Base) && imageExistsSpectrum(options) {
		target.Status.Phase = v1.CamelCatalogPhaseReady
		target.Status.SetCondition(
			v1.CamelCatalogConditionReady,
			corev1.ConditionTrue,
			"Builder Image",
			"Container image exists on registry",
		)
		target.Status.Image = imageName

		return target, nil
	}

	// Now we properly set the base and the target image
	options.Base = catalog.Spec.GetQuarkusToolingImage()
	options.Target = imageName

	err := buildRuntimeBuilderWithTimeoutSpectrum(options, ip.Status.Build.GetBuildCatalogToolTimeout().Duration)

	if err != nil {
		target.Status.Phase = v1.CamelCatalogPhaseError
		target.Status.SetErrorCondition(
			v1.CamelCatalogConditionReady,
			"Builder Image",
			err,
		)
	} else {
		target.Status.Phase = v1.CamelCatalogPhaseReady
		target.Status.SetCondition(
			v1.CamelCatalogConditionReady,
			corev1.ConditionTrue,
			"Builder Image",
			"Container image successfully built",
		)
		target.Status.Image = imageName
	}

	return target, nil
}

// nolint: maintidx // TODO: refactor the code
func initializeS2i(ctx context.Context, c client.Client, ip *v1.IntegrationPlatform, catalog *v1.CamelCatalog) (*v1.CamelCatalog, error) {
	target := catalog.DeepCopy()
	// No registry in s2i
	imageName := fmt.Sprintf(
		"camel-k-runtime-%s-builder",
		catalog.Spec.Runtime.Provider,
	)
	imageTag := strings.ToLower(catalog.Spec.Runtime.Version)

	uidStr := getS2iUserID(ctx, c, ip, catalog)

	// Dockerfile
	dockerfile := `
		FROM ` + catalog.Spec.GetQuarkusToolingImage() + `
		USER ` + uidStr + `:0
		ADD --chown=` + uidStr + `:0 /usr/local/bin/kamel /usr/local/bin/kamel
		ADD --chown=` + uidStr + `:0 /usr/share/maven/mvnw/ /usr/share/maven/mvnw/
	`
	if imageSnapshot(imageName + ":" + imageTag) {
		dockerfile = dockerfile + `
		ADD --chown=` + uidStr + `:0 ` + defaults.LocalRepository + ` ` + defaults.LocalRepository + `
	`
	}

	owner := catalogReference(catalog)

	// BuildConfig
	bc := &buildv1.BuildConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: buildv1.GroupVersion.String(),
			Kind:       "BuildConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      imageName,
			Namespace: ip.Namespace,
			Labels: map[string]string{
				kubernetes.CamelCreatorLabelKind:      v1.CamelCatalogKind,
				kubernetes.CamelCreatorLabelName:      catalog.Name,
				kubernetes.CamelCreatorLabelNamespace: catalog.Namespace,
				kubernetes.CamelCreatorLabelVersion:   catalog.ResourceVersion,
				"camel.apache.org/runtime.version":    catalog.Spec.Runtime.Version,
				"camel.apache.org/runtime.provider":   string(catalog.Spec.Runtime.Provider),
			},
		},
		Spec: buildv1.BuildConfigSpec{
			CommonSpec: buildv1.CommonSpec{
				Source: buildv1.BuildSource{
					Type:       buildv1.BuildSourceBinary,
					Dockerfile: &dockerfile,
				},
				Strategy: buildv1.BuildStrategy{
					DockerStrategy: &buildv1.DockerBuildStrategy{},
				},
				Output: buildv1.BuildOutput{
					To: &corev1.ObjectReference{
						Kind: "ImageStreamTag",
						Name: imageName + ":" + imageTag,
					},
				},
			},
		},
	}

	// ImageStream
	is := &imagev1.ImageStream{
		TypeMeta: metav1.TypeMeta{
			APIVersion: imagev1.GroupVersion.String(),
			Kind:       "ImageStream",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      bc.Name,
			Namespace: bc.Namespace,
			Labels: map[string]string{
				kubernetes.CamelCreatorLabelKind:      v1.CamelCatalogKind,
				kubernetes.CamelCreatorLabelName:      catalog.Name,
				kubernetes.CamelCreatorLabelNamespace: catalog.Namespace,
				kubernetes.CamelCreatorLabelVersion:   catalog.ResourceVersion,
				"camel.apache.org/runtime.provider":   string(catalog.Spec.Runtime.Provider),
			},
		},
		Spec: imagev1.ImageStreamSpec{
			LookupPolicy: imagev1.ImageLookupPolicy{
				Local: true,
			},
		},
	}

	if !imageSnapshot(imageName+":"+imageTag) && imageExistsS2i(ctx, c, is) {
		target.Status.Phase = v1.CamelCatalogPhaseReady
		target.Status.SetCondition(
			v1.CamelCatalogConditionReady,
			corev1.ConditionTrue,
			"Builder Image",
			"Container image exists on registry (later)",
		)
		target.Status.Image = imageName
		return target, nil
	}

	err := s2i.BuildConfig(ctx, c, bc, owner)
	if err != nil {
		target.Status.Phase = v1.CamelCatalogPhaseError
		target.Status.SetErrorCondition(
			v1.CamelCatalogConditionReady,
			"Builder Image",
			err,
		)
		return target, err
	}

	err = s2i.ImageStream(ctx, c, is, owner)
	if err != nil {
		target.Status.Phase = v1.CamelCatalogPhaseError
		target.Status.SetErrorCondition(
			v1.CamelCatalogConditionReady,
			"Builder Image",
			err,
		)
		return target, err
	}

	err = util.WithTempDir(imageName+"-s2i-", func(tmpDir string) error {
		archive := filepath.Join(tmpDir, "archive.tar.gz")

		archiveFile, err := os.Create(archive)
		if err != nil {
			return fmt.Errorf("cannot create tar archive: %w", err)
		}

		directories := []string{
			"/usr/local/bin/kamel:/usr/local/bin/kamel",
			"/usr/share/maven/mvnw/:/usr/share/maven/mvnw/",
		}
		if imageSnapshot(imageName + ":" + imageTag) {
			directories = append(directories, defaults.LocalRepository+":"+defaults.LocalRepository)
		}

		err = tarEntries(archiveFile, directories...)
		if err != nil {
			return fmt.Errorf("cannot tar path entry: %w", err)
		}

		f, err := util.Open(archive)
		if err != nil {
			return err
		}

		restClient, err := apiutil.RESTClientForGVK(
			schema.GroupVersionKind{Group: "build.openshift.io", Version: "v1"}, false,
			c.GetConfig(), serializer.NewCodecFactory(c.GetScheme()))
		if err != nil {
			return err
		}

		r := restClient.Post().
			Namespace(bc.Namespace).
			Body(bufio.NewReader(f)).
			Resource("buildconfigs").
			Name(bc.Name).
			SubResource("instantiatebinary").
			Do(ctx)

		if r.Error() != nil {
			return fmt.Errorf("cannot instantiate binary: %w", err)
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

		err = s2i.WaitForS2iBuildCompletion(ctx, c, &s2iBuild)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				// nolint: contextcheck
				if err := s2i.CancelBuild(context.Background(), c, &s2iBuild); err != nil {
					return fmt.Errorf("cannot cancel s2i Build: %s/%s", s2iBuild.Namespace, s2iBuild.Name)
				}
			}
			return err
		}
		if s2iBuild.Status.Output.To != nil {
			Log.Infof("Camel K builder container image %s:%s@%s created", imageName, imageTag, s2iBuild.Status.Output.To.ImageDigest)
		}

		err = c.Get(ctx, ctrl.ObjectKeyFromObject(is), is)
		if err != nil {
			return err
		}

		if is.Status.DockerImageRepository == "" {
			return errors.New("dockerImageRepository not available in ImageStream")
		}

		target.Status.Phase = v1.CamelCatalogPhaseReady
		target.Status.SetCondition(
			v1.CamelCatalogConditionReady,
			corev1.ConditionTrue,
			"Builder Image",
			"Container image successfully built",
		)
		target.Status.Image = is.Status.DockerImageRepository + ":" + imageTag

		return f.Close()
	})

	if err != nil {
		target.Status.Phase = v1.CamelCatalogPhaseError
		target.Status.SetErrorCondition(
			v1.CamelCatalogConditionReady,
			"Builder Image",
			err,
		)
		return target, err
	}

	return target, nil
}

func imageExistsSpectrum(options spectrum.Options) bool {
	Log.Infof("Checking if Camel K builder container %s already exists...", options.Base)
	ctrImg, err := spectrum.Pull(options)
	// Ignore the error-indent-flow as we save the need to declare a dependency explicitly
	// nolint: revive
	if ctrImg != nil && err == nil {
		if hash, err := ctrImg.Digest(); err != nil {
			Log.Errorf(err, "Cannot calculate digest")
			return false
		} else {
			Log.Infof("Found Camel K builder container with digest %s", hash.String())
			return true
		}
	}

	Log.Infof("Couldn't pull image due to %s", err.Error())
	return false
}

func imageExistsS2i(ctx context.Context, c client.Client, is *imagev1.ImageStream) bool {
	Log.Infof("Checking if Camel K builder container %s already exists...", is.Name)
	key := ctrl.ObjectKey{
		Namespace: is.Namespace,
		Name:      is.Name,
	}

	err := c.Get(ctx, key, is)

	if err != nil {
		if !k8serrors.IsNotFound(err) {
			Log.Infof("Couldn't pull image due to %s", err.Error())
		}
		Log.Info("Could not find Camel K builder container")
		return false
	}
	Log.Info("Found Camel K builder container ")
	return true
}

func imageSnapshot(imageName string) bool {
	return strings.HasSuffix(imageName, "snapshot")
}

func buildRuntimeBuilderWithTimeoutSpectrum(options spectrum.Options, timeout time.Duration) error {
	// Backward compatibility with IP which had not a timeout field
	if timeout == 0 {
		return buildRuntimeBuilderImageSpectrum(options)
	}
	result := make(chan error, 1)
	go func() {
		result <- buildRuntimeBuilderImageSpectrum(options)
	}()
	select {
	case <-time.After(timeout):
		return fmt.Errorf("build timeout: %s", timeout.String())
	case result := <-result:
		return result
	}
}

// This func will take care to dynamically build an image that will contain the tools required
// by the catalog build plus kamel binary and a maven wrapper required for the build.
func buildRuntimeBuilderImageSpectrum(options spectrum.Options) error {
	if options.Base == "" {
		return fmt.Errorf("missing base image, likely catalog is not compatible with this Camel K version")
	}
	Log.Infof("Making up Camel K builder container %s", options.Target)

	if jobs := runtime.GOMAXPROCS(0); jobs > 1 {
		options.Jobs = jobs
	}

	directories := []string{
		"/usr/local/bin/kamel:/usr/local/bin/",
		"/usr/share/maven/mvnw/:/usr/share/maven/mvnw/",
	}
	if imageSnapshot(options.Target) {
		directories = append(directories, defaults.LocalRepository+":"+defaults.LocalRepository)
	}

	_, err := spectrum.Build(options, directories...)
	if err != nil {
		return err
	}

	return nil
}

func readSpectrumLogs(newStdOut io.Reader) {
	scanner := bufio.NewScanner(newStdOut)

	for scanner.Scan() {
		line := scanner.Text()
		Log.Infof(line)
	}
}

func makeSpectrumOptions(ctx context.Context, c client.Client, platformNamespace string, registry v1.RegistrySpec) (spectrum.Options, error) {
	options := spectrum.Options{}
	var err error
	registryConfigDir := ""
	if registry.Secret != "" {
		registryConfigDir, err = builder.MountSecret(ctx, c, platformNamespace, registry.Secret)
		if err != nil {
			return options, err
		}
	}
	options.PullInsecure = registry.Insecure
	options.PushInsecure = registry.Insecure
	options.PullConfigDir = registryConfigDir
	options.PushConfigDir = registryConfigDir
	options.Recursive = true

	return options, nil
}

// Add entries (files or folders) into tar with the possibility to change its path.
func tarEntries(writer io.Writer, files ...string) error {

	gzw := gzip.NewWriter(writer)
	defer util.CloseQuietly(gzw)

	tw := tar.NewWriter(gzw)
	defer util.CloseQuietly(tw)

	// Iterate over files and and add them to the tar archive
	for _, fileDetail := range files {
		fileSource := strings.Split(fileDetail, ":")[0]
		fileTarget := strings.Split(fileDetail, ":")[1]
		// ensure the src actually exists before trying to tar it
		if _, err := os.Stat(fileSource); err != nil {
			return fmt.Errorf("unable to tar files: %w", err)
		}

		if err := filepath.Walk(fileSource, func(file string, fi os.FileInfo, err error) error {
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
			header.Name = strings.TrimPrefix(strings.ReplaceAll(file, fileSource, fileTarget), string(filepath.Separator))

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

			return f.Close()
		}); err != nil {
			return fmt.Errorf("unable to tar: %w", err)
		}

	}
	return nil
}

func catalogReference(catalog *v1.CamelCatalog) *unstructured.Unstructured {
	owner := &unstructured.Unstructured{}
	owner.SetName(catalog.Name)
	owner.SetUID(catalog.UID)
	owner.SetAPIVersion(catalog.APIVersion)
	owner.SetKind(catalog.Kind)
	return owner
}

// get user id from security context constraint configuration in namespace if present.
func getS2iUserID(ctx context.Context, c client.Client, ip *v1.IntegrationPlatform, catalog *v1.CamelCatalog) string {
	ugfidStr := "1001"
	if ip.Status.Cluster == v1.IntegrationPlatformClusterOpenShift {
		uidStr, err := openshift.GetOpenshiftUser(ctx, c, catalog.GetNamespace())
		if err != nil {
			Log.Error(err, "Unable to retieve an Openshift user and group Ids.")
		}
		return uidStr
	}
	return ugfidStr
}
