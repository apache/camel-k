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

// Package jib contains utilities for jib strategy builds.
package jib

import (
	"context"
	"encoding/xml"
	"fmt"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/maven"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const JibMavenGoal = "jib:build"
const JibMavenToImageParam = "-Djib.to.image="
const JibMavenFromImageParam = "-Djib.from.image="
const JibMavenInsecureRegistries = "-Djib.allowInsecureRegistries="
const JibDigestFile = "target/jib-image.digest"

type JibBuild struct {
	Plugins []maven.Plugin `xml:"plugins>plugin,omitempty"`
}

type JibProfile struct {
	XMLName xml.Name
	ID      string   `xml:"id"`
	Build   JibBuild `xml:"build,omitempty"`
}

// Create a Configmap containing the default jib profile.
func CreateProfileConfigmap(ctx context.Context, c client.Client, kit *v1.IntegrationKit) error {
	profile, err := jibMavenProfile()
	if err != nil {
		return fmt.Errorf("error generating default maven jib profile: %w. ", err)
	}

	annotations := util.CopyMap(kit.Annotations)
	controller := true
	blockOwnerDeletion := true
	jibProfileConfigMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        kit.Name + "-publish-jib-profile",
			Namespace:   kit.Namespace,
			Annotations: annotations,
			Labels: map[string]string{
				v1.IntegrationKitLabel: kit.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         kit.APIVersion,
					Kind:               kit.Kind,
					Name:               kit.Name,
					UID:                kit.UID,
					Controller:         &controller,
					BlockOwnerDeletion: &blockOwnerDeletion,
				}},
		},
		Data: map[string]string{
			"profile.xml": profile,
		},
	}

	err = c.Create(ctx, jibProfileConfigMap)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating the configmap containing the default maven jib profile: %s: %w. ", kit.Name+"-publish-jib-profile", err)
	}
	return nil
}

func jibMavenProfile() (string, error) {
	jibPlugin := maven.Plugin{
		GroupID:    "com.google.cloud.tools",
		ArtifactID: "jib-maven-plugin",
		Version:    "3.3.2",
		Dependencies: []maven.Dependency{
			{
				GroupID:    "com.google.cloud.tools",
				ArtifactID: "jib-layer-filter-extension-maven",
				Version:    "0.3.0",
			},
		},
		Configuration: v1.PluginConfiguration{
			Container: v1.Container{
				Entrypoint: "INHERIT",
				Args: v1.Args{
					Arg: "jshell",
				},
			},
			AllowInsecureRegistries: "true",
			ExtraDirectories: v1.ExtraDirectories{
				Paths: []v1.Path{
					{
						From: "../context",
						Into: "/deployments",
					},
				},
				Permissions: []v1.Permission{
					{
						File: "/deployments/*",
						Mode: "544",
					},
				},
			},
			PluginExtensions: v1.PluginExtensions{
				PluginExtension: v1.PluginExtension{
					Implementation: "com.google.cloud.tools.jib.maven.extension.layerfilter.JibLayerFilterExtension",
					Configuration: v1.PluginExtensionConfiguration{
						Implementation: "com.google.cloud.tools.jib.maven.extension.layerfilter.Configuration",
						Filters: []v1.Filter{
							{
								Glob: "/app/**",
							},
						},
					},
				},
			},
		},
	}

	jibMavenPluginProfile := JibProfile{
		XMLName: xml.Name{Local: "profile"},
		ID:      "jib",
		Build: JibBuild{
			Plugins: []maven.Plugin{jibPlugin},
		},
	}
	content, err := util.EncodeXMLWithoutHeader(jibMavenPluginProfile)
	if err != nil {
		return "", err
	}
	return string(content), nil

}
