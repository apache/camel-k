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

package trait

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	serving "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util/boolean"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/log"
	"github.com/apache/camel-k/v2/pkg/util/property"
	utilResource "github.com/apache/camel-k/v2/pkg/util/resource"
)

const (
	mountTraitID    = "mount"
	mountTraitOrder = 1620
)

type mountTrait struct {
	BasePlatformTrait
	traitv1.MountTrait `property:",squash"`
}

func newMountTrait() Trait {
	return &mountTrait{
		// Must follow immediately the container and init-containers trait
		BasePlatformTrait: NewBasePlatformTrait(mountTraitID, mountTraitOrder),
	}
}

func (t *mountTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil || !e.IntegrationInRunningPhases() {
		return false, nil, nil
	}

	// Validate resources and pvcs
	for _, c := range t.Configs {
		if !strings.HasPrefix(c, "configmap:") && !strings.HasPrefix(c, "secret:") {
			return false, nil, fmt.Errorf("unsupported config %s, must be a configmap or secret resource", c)
		}
	}
	for _, r := range t.Resources {
		if !strings.HasPrefix(r, "configmap:") && !strings.HasPrefix(r, "secret:") {
			return false, nil, fmt.Errorf("unsupported resource %s, must be a configmap or secret resource", r)
		}
	}

	return true, nil, nil
}

func (t *mountTrait) Apply(e *Environment) error {
	container := e.GetIntegrationContainer()
	if container == nil {
		return fmt.Errorf("unable to find integration container: %s", e.Integration.Name)
	}

	var volumes *[]corev1.Volume
	var initContainers *[]corev1.Container
	visited := false

	// Deployment
	if err := e.Resources.VisitDeploymentE(func(deployment *appsv1.Deployment) error {
		volumes = &deployment.Spec.Template.Spec.Volumes
		initContainers = &deployment.Spec.Template.Spec.InitContainers
		visited = true

		return nil
	}); err != nil {
		return err
	}

	// Knative Service
	if err := e.Resources.VisitKnativeServiceE(func(service *serving.Service) error {
		volumes = &service.Spec.Template.Spec.Volumes
		initContainers = &service.Spec.Template.Spec.InitContainers
		visited = true

		return nil
	}); err != nil {
		return err
	}

	// CronJob
	if err := e.Resources.VisitCronJobE(func(cron *batchv1.CronJob) error {
		volumes = &cron.Spec.JobTemplate.Spec.Template.Spec.Volumes
		initContainers = &cron.Spec.JobTemplate.Spec.Template.Spec.InitContainers
		visited = true

		return nil
	}); err != nil {
		return err
	}

	if visited {
		// Volumes declared in the trait config/resource options
		// as this func influences the application.properties
		// must be set as the first one to execute
		err := t.configureVolumesAndMounts(e, volumes, &container.VolumeMounts, initContainers)
		if err != nil {
			return err
		}
		// Here we configure the application.properties
		t.addSourcesProperties(e)
		if props, err := t.computeApplicationProperties(e); err != nil {
			return err
		} else if props != nil {
			e.Resources.Add(props)
		}
		// Volumes declared in the Integration resources (including the application.properties Configmap)
		t.configureCamelVolumesAndMounts(e, volumes, &container.VolumeMounts)
	}

	return nil
}

// configureVolumesAndMounts is in charge to mount volumes and mounts coming from the trait configuration.
// icnts holds the InitContainers which also require to be mounted with the shared volumes.
func (t *mountTrait) configureVolumesAndMounts(
	e *Environment,
	vols *[]corev1.Volume,
	mnts *[]corev1.VolumeMount,
	icnts *[]corev1.Container,
) error {
	for _, c := range t.Configs {
		if conf, parseErr := utilResource.ParseConfig(c); parseErr == nil {
			// Let Camel parse these resources as properties
			destFilePath := t.mountResource(vols, mnts, conf)
			e.appendCloudPropertiesLocation(destFilePath)
		} else {
			return parseErr
		}
	}
	for _, r := range t.Resources {
		if res, parseErr := utilResource.ParseResource(r); parseErr == nil {
			t.mountResource(vols, mnts, res)
		} else {
			return parseErr
		}
	}
	for _, v := range t.Volumes {
		volume, volumeMount, parseErr := ParseAndCreateVolume(e, v)
		if parseErr != nil {
			return parseErr
		}
		*vols = append(*vols, *volume)
		*mnts = append(*mnts, *volumeMount)
		for i := range *icnts {
			(*icnts)[i].VolumeMounts = append((*icnts)[i].VolumeMounts, *volumeMount)
		}
	}
	for _, v := range t.EmptyDirs {
		volume, volumeMount, parseErr := ParseEmptyDirVolume(v)
		if parseErr != nil {
			return parseErr
		}
		*vols = append(*vols, *volume)
		*mnts = append(*mnts, *volumeMount)
		for i := range *icnts {
			(*icnts)[i].VolumeMounts = append((*icnts)[i].VolumeMounts, *volumeMount)
		}
	}
	// Mount the agent volume if any agent exists
	trait := e.Catalog.GetTrait(jvmTraitID)
	//nolint:nestif
	if trait != nil {
		jvm, ok := trait.(*jvmTrait)
		if ok && jvm.hasJavaAgents() {
			volume, volumeMount, parseErr := ParseEmptyDirVolume(fmt.Sprintf("%s:%s", defaultAgentVolume, defaultAgentDir))
			if parseErr != nil {
				return parseErr
			}
			*vols = append(*vols, *volume)
			*mnts = append(*mnts, *volumeMount)
			for i := range *icnts {
				(*icnts)[i].VolumeMounts = append((*icnts)[i].VolumeMounts, *volumeMount)
			}
		}
		// Mount CA cert volumes if configured
		if ok && jvm.hasCACert() {
			secretName, _, err := parseSecretRef(jvm.CACert)
			if err != nil {
				return err
			}
			mountPath := jvm.getCACertMountPath()

			// Secret volume for CA cert
			secretVolume := corev1.Volume{
				Name: caCertSecretVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: secretName,
					},
				},
			}
			*vols = append(*vols, secretVolume)

			// EmptyDir volume for truststore output
			trustStoreVolume := corev1.Volume{
				Name: caCertVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			}
			*vols = append(*vols, trustStoreVolume)

			// Mount truststore to main container
			*mnts = append(*mnts, corev1.VolumeMount{
				Name:      caCertVolumeName,
				MountPath: mountPath,
				ReadOnly:  true,
			})

			// Mount volumes to init containers for truststore generation
			for i := range *icnts {
				(*icnts)[i].VolumeMounts = append((*icnts)[i].VolumeMounts,
					corev1.VolumeMount{
						Name:      caCertSecretVolumeName,
						MountPath: "/etc/secrets/cacert",
						ReadOnly:  true,
					},
					corev1.VolumeMount{
						Name:      caCertVolumeName,
						MountPath: mountPath,
					},
				)
			}
		}
	}

	return nil
}

// configureCamelVolumesAndMounts is in charge to mount volumes and mounts coming from Camel configuration
// (ie, sources, properties, kamelets, etcetera).
func (t *mountTrait) configureCamelVolumesAndMounts(e *Environment, vols *[]corev1.Volume, mnts *[]corev1.VolumeMount) {
	// Sources index
	idx := 0
	// Configmap index (may differ as generated sources can have a different name)
	cmx := 0
	for _, s := range e.Integration.AllSources() {
		// We don't process routes embedded (native) or Kamelets
		if e.isEmbedded(s) || s.IsGeneratedFromKamelet() {
			continue
		}
		// Routes are copied under /etc/camel/sources and discovered by the runtime accordingly
		cmName := fmt.Sprintf("%s-source-%03d", e.Integration.Name, cmx)
		if s.ContentRef != "" {
			cmName = s.ContentRef
		}
		cmKey := "content"
		if s.ContentKey != "" {
			cmKey = s.ContentKey
		}
		resName := strings.TrimPrefix(s.Name, "/")
		refName := fmt.Sprintf("i-source-%03d", idx)
		resPath := filepath.Join(camel.SourcesMountPath, resName)
		vol := getVolume(refName, "configmap", cmName, cmKey, resName)
		mnt := getMount(refName, resPath, resName, true)

		*vols = append(*vols, *vol)
		*mnts = append(*mnts, *mnt)
		idx++
		if s.ContentRef == "" {
			cmx++
		}
	}
	// Resources (likely application properties or kamelets)
	if e.Resources != nil {
		e.Resources.VisitConfigMap(func(configMap *corev1.ConfigMap) {
			switch configMap.Labels[kubernetes.ConfigMapTypeLabel] {
			case CamelPropertiesType:
				// Camel properties
				propertiesType := configMap.Labels["camel.apache.org/properties.type"]
				resName := propertiesType + ".properties"

				var mountPath string
				switch propertiesType {
				case "application":
					mountPath = filepath.Join(camel.BasePath, resName)
				case "user":
					mountPath = filepath.Join(camel.ConfDPath, resName)
				}

				if propertiesType != "" {
					refName := propertiesType + "-properties"
					vol := getVolume(refName, "configmap", configMap.Name, "application.properties", resName)
					mnt := getMount(refName, mountPath, resName, true)

					*vols = append(*vols, *vol)
					*mnts = append(*mnts, *mnt)
				} else {
					log.WithValues("Function", "trait.configureVolumesAndMounts").Infof("Warning: could not determine camel properties type %s", propertiesType)
				}
			case KameletBundleType:
				// Kamelets bundle configmap
				kameletMountPoint := configMap.Annotations[kameletMountPointAnnotation]
				refName := KameletBundleType
				vol := getVolume(refName, "configmap", configMap.Name, "", "")
				mnt := getMount(refName, kameletMountPoint, "", true)

				*vols = append(*vols, *vol)
				*mnts = append(*mnts, *mnt)
			}
		})
	}
}

// mountResource add the resource to volumes and mounts and return the final path where the resource is mounted.
func (t *mountTrait) mountResource(vols *[]corev1.Volume, mnts *[]corev1.VolumeMount, conf *utilResource.Config) string {
	refName := sanitizeVolumeName(conf.Name(), vols)
	dstDir := conf.DestinationPath()
	dstFile := ""
	if conf.DestinationPath() != "" {
		if conf.Key() != "" {
			dstFile = filepath.Base(conf.DestinationPath())
		} else {
			dstFile = conf.Key()
		}
	}
	vol := getVolume(refName, string(conf.StorageType()), refName, conf.Key(), dstFile)
	mntPath := getMountPoint(refName, dstDir, string(conf.StorageType()), string(conf.ContentType()))
	readOnly := (conf.StorageType() != utilResource.StorageTypePVC)

	mnt := getMount(refName, mntPath, dstFile, readOnly)

	*vols = append(*vols, *vol)
	*mnts = append(*mnts, *mnt)

	return mnt.MountPath
}

// computeApplicationProperties is in charge to configure the configmap containing Camel application.properties.
func (t *mountTrait) computeApplicationProperties(e *Environment) (*corev1.ConfigMap, error) {
	// application properties
	applicationProperties, err := property.EncodePropertyFile(e.ApplicationProperties)
	if err != nil {
		return nil, fmt.Errorf("could not compute application properties: %w", err)
	}

	if applicationProperties != "" {
		return &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      e.Integration.Name + "-application-properties",
				Namespace: e.Integration.Namespace,
				Labels: map[string]string{
					v1.IntegrationLabel:                e.Integration.Name,
					"camel.apache.org/properties.type": "application",
					kubernetes.ConfigMapTypeLabel:      CamelPropertiesType,
				},
			},
			Data: map[string]string{
				"application.properties": applicationProperties,
			},
		}, nil
	}

	return nil, nil
}

// addSourcesProperties is in charge to add the sources in the application.properties required by Camel K Runtime.
//
//nolint:nestif
func (t *mountTrait) addSourcesProperties(e *Environment) {
	if e.ApplicationProperties == nil {
		e.ApplicationProperties = make(map[string]string)
	}
	if e.CamelCatalog.GetRuntimeProvider() == v1.RuntimeProviderPlainQuarkus {
		sourceLocationEnabled := false
		for _, s := range e.Integration.AllSources() {
			// We don't process routes embedded (native) or Kamelets
			if e.isEmbedded(s) || s.IsGeneratedFromKamelet() {
				continue
			}
			sourceLocationEnabled = true

			break
		}
		if sourceLocationEnabled {
			e.ApplicationProperties["camel.main.source-location-enabled"] = boolean.TrueString
			e.ApplicationProperties["camel.main.routes-include-pattern"] = fmt.Sprintf("file:%s/**", camel.SourcesMountPath)
		}
	} else {
		idx := 0
		for _, s := range e.Integration.AllSources() {
			// We don't process routes embedded (native) or Kamelets
			if e.isEmbedded(s) || s.IsGeneratedFromKamelet() {
				continue
			}
			srcName := strings.TrimPrefix(filepath.ToSlash(s.Name), "/")
			src := "file:" + path.Join(filepath.ToSlash(camel.SourcesMountPath), srcName)
			e.ApplicationProperties[fmt.Sprintf("camel.k.sources[%d].location", idx)] = src

			simpleName := srcName
			if strings.Contains(srcName, ".") {
				simpleName = srcName[0:strings.Index(srcName, ".")]
			}
			e.ApplicationProperties[fmt.Sprintf("camel.k.sources[%d].name", idx)] = simpleName

			for pid, p := range s.PropertyNames {
				e.ApplicationProperties[fmt.Sprintf("camel.k.sources[%d].property-names[%d]", idx, pid)] = p
			}

			if s.Type != "" {
				e.ApplicationProperties[fmt.Sprintf("camel.k.sources[%d].type", idx)] = string(s.Type)
			}
			if s.InferLanguage() != "" {
				e.ApplicationProperties[fmt.Sprintf("camel.k.sources[%d].language", idx)] = string(s.InferLanguage())
			}
			if s.Loader != "" {
				e.ApplicationProperties[fmt.Sprintf("camel.k.sources[%d].loader", idx)] = s.Loader
			}
			if s.Compression {
				e.ApplicationProperties[fmt.Sprintf("camel.k.sources[%d].compressed", idx)] = boolean.TrueString
			}

			idx++
		}
	}
}
