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

package maven

import (
	"encoding/xml"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util"
)

// DefaultMavenRepositories is a comma separated list of default maven repositories
// This variable can be overridden at build time
var DefaultMavenRepositories = "https://repo.maven.apache.org/maven2@id=central"

func NewSettings() Settings {
	return Settings{
		XMLName:           xml.Name{Local: "settings"},
		XMLNs:             "http://maven.apache.org/SETTINGS/1.0.0",
		XMLNsXsi:          "http://www.w3.org/2001/XMLSchema-instance",
		XsiSchemaLocation: "http://maven.apache.org/SETTINGS/1.0.0 https://maven.apache.org/xsd/settings-1.0.0.xsd",
	}
}

func NewDefaultSettings(repositories []v1.Repository, mirrors []Mirror) Settings {
	settings := NewSettings()

	var additionalRepos []v1.Repository
	for _, defaultRepo := range defaultMavenRepositories() {
		if !containsRepo(repositories, defaultRepo.ID) {
			additionalRepos = append(additionalRepos, defaultRepo)
		}
	}
	if len(additionalRepos) > 0 {
		repositories = append(additionalRepos, repositories...)
	}

	settings.Profiles = []Profile{
		{
			ID: "maven-settings",
			Activation: Activation{
				ActiveByDefault: true,
			},
			Repositories:       repositories,
			PluginRepositories: repositories,
		},
	}

	settings.Mirrors = mirrors

	return settings
}

func SettingsConfigMap(namespace string, name string, settings Settings) (*corev1.ConfigMap, error) {
	data, err := util.EncodeXML(settings)
	if err != nil {
		return nil, err
	}

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-maven-settings",
			Namespace: namespace,
			Labels: map[string]string{
				"app": "camel-k",
			},
		},
		Data: map[string]string{
			"settings.xml": string(data),
		},
	}

	return cm, nil
}

func defaultMavenRepositories() (repos []v1.Repository) {
	for _, repoDesc := range strings.Split(DefaultMavenRepositories, ",") {
		repos = append(repos, NewRepository(repoDesc))
	}
	return
}

func containsRepo(repositories []v1.Repository, id string) bool {
	for _, r := range repositories {
		if r.ID == id {
			return true
		}
	}
	return false
}
