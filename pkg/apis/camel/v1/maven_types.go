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

package v1

import (
	corev1 "k8s.io/api/core/v1"
)

// MavenSpec --
type MavenSpec struct {
	// The path of the local Maven repository.
	LocalRepository string `json:"localRepository,omitempty"`
	// The Maven properties.
	Properties map[string]string `json:"properties,omitempty"`
	// A reference to the ConfigMap or Secret key that contains
	// the Maven settings.
	Settings ValueSource `json:"settings,omitempty"`
	// The Secret name and key, containing the CA certificate(s) used to connect
	// to remote Maven repositories.
	// It can contain X.509 certificates, and PKCS#7 formatted certificate chains.
	// A JKS formatted keystore is automatically created to store the CA certificate(s),
	// and configured to be used as a trusted certificate(s) by the Maven commands.
	// Note that the root CA certificates are also imported into the created keystore.
	CASecret *corev1.SecretKeySelector `json:"caSecret,omitempty"`
	// The Maven build extensions.
	// See https://maven.apache.org/guides/mini/guide-using-extensions.html.
	Extension []MavenArtifact `json:"extension,omitempty"`
	// The CLI options that are appended to the list of arguments for Maven commands,
	// e.g., `-V,--no-transfer-progress,-Dstyle.color=never`.
	// See https://maven.apache.org/ref/3.8.4/maven-embedder/cli.html.
	CLIOptions []string `json:"cliOptions,omitempty"`
}

// Repository defines a Maven repository
type Repository struct {
	ID        string           `xml:"id" json:"id"`
	Name      string           `xml:"name,omitempty" json:"name,omitempty"`
	URL       string           `xml:"url" json:"url"`
	Snapshots RepositoryPolicy `xml:"snapshots,omitempty" json:"snapshots,omitempty"`
	Releases  RepositoryPolicy `xml:"releases,omitempty" json:"releases,omitempty"`
}

// RepositoryPolicy defines the policy associated to a Maven repository
type RepositoryPolicy struct {
	Enabled        bool   `xml:"enabled" json:"enabled"`
	UpdatePolicy   string `xml:"updatePolicy,omitempty" json:"updatePolicy,omitempty"`
	ChecksumPolicy string `xml:"checksumPolicy,omitempty" json:"checksumPolicy,omitempty"`
}

// MavenArtifact defines a Maven artifact
type MavenArtifact struct {
	GroupID    string `json:"groupId" yaml:"groupId" xml:"groupId"`
	ArtifactID string `json:"artifactId" yaml:"artifactId" xml:"artifactId"`
	Version    string `json:"version,omitempty" yaml:"version,omitempty" xml:"version,omitempty"`
}
