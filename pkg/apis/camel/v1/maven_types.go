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
	"encoding/xml"

	corev1 "k8s.io/api/core/v1"
)

// MavenSpec --.
type MavenSpec struct {
	// The path of the local Maven repository.
	LocalRepository string `json:"localRepository,omitempty"`
	// The Maven properties.
	Properties map[string]string `json:"properties,omitempty"`
	// A reference to the ConfigMap or Secret key that contains
	// the Maven profile.
	Profiles []ValueSource `json:"profiles,omitempty"`
	// A reference to the ConfigMap or Secret key that contains
	// the Maven settings.
	Settings ValueSource `json:"settings,omitempty"`
	// A reference to the ConfigMap or Secret key that contains
	// the security of the Maven settings.
	SettingsSecurity ValueSource `json:"settingsSecurity,omitempty"`
	// The Secrets name and key, containing the CA certificate(s) used to connect
	// to remote Maven repositories.
	// It can contain X.509 certificates, and PKCS#7 formatted certificate chains.
	// A JKS formatted keystore is automatically created to store the CA certificate(s),
	// and configured to be used as a trusted certificate(s) by the Maven commands.
	// Note that the root CA certificates are also imported into the created keystore.
	CASecrets []corev1.SecretKeySelector `json:"caSecrets,omitempty"`
	// The Maven build extensions.
	// See https://maven.apache.org/guides/mini/guide-using-extensions.html.
	Extension []MavenArtifact `json:"extension,omitempty"`
	// The CLI options that are appended to the list of arguments for Maven commands,
	// e.g., `-V,--no-transfer-progress,-Dstyle.color=never`.
	// See https://maven.apache.org/ref/3.8.4/maven-embedder/cli.html.
	CLIOptions []string `json:"cliOptions,omitempty"`
}

// Repository defines a Maven repository.
type Repository struct {
	// identifies the repository
	ID string `json:"id" xml:"id"`
	// name of the repository
	Name string `json:"name,omitempty" xml:"name,omitempty"`
	// location of the repository
	URL string `json:"url" xml:"url"`
	// can use snapshot
	Snapshots RepositoryPolicy `json:"snapshots,omitempty" xml:"snapshots,omitempty"`
	// can use stable releases
	Releases RepositoryPolicy `json:"releases,omitempty" xml:"releases,omitempty"`
}

// RepositoryPolicy defines the policy associated to a Maven repository.
type RepositoryPolicy struct {
	// is the policy activated or not
	Enabled bool `json:"enabled" xml:"enabled"`
	// This element specifies how often updates should attempt to occur.
	// Maven will compare the local POM's timestamp (stored in a repository's maven-metadata file) to the remote.
	// The choices are: `always`, `daily` (default), `interval:X` (where X is an integer in minutes) or `never`
	UpdatePolicy string `json:"updatePolicy,omitempty" xml:"updatePolicy,omitempty"`
	// When Maven deploys files to the repository, it also deploys corresponding checksum files.
	// Your options are to `ignore`, `fail`, or `warn` on missing or incorrect checksums.
	ChecksumPolicy string `json:"checksumPolicy,omitempty" xml:"checksumPolicy,omitempty"`
}

// MavenArtifact defines a GAV (Group:Artifact:Type:Version:Classifier) Maven artifact.
type MavenArtifact struct {
	// Maven Group
	GroupID string `json:"groupId" xml:"groupId" yaml:"groupId"`
	// Maven Artifact
	ArtifactID string `json:"artifactId" xml:"artifactId" yaml:"artifactId"`
	// Maven Type
	Type string `json:"type,omitempty" xml:"type,omitempty" yaml:"type,omitempty"`
	// Maven Version
	Version string `json:"version,omitempty" xml:"version,omitempty" yaml:"version,omitempty"`
	// Maven Classifier
	Classifier string `json:"classifier,omitempty" xml:"classifier,omitempty" yaml:"classifier,omitempty"`
}

// Server see link:https://maven.apache.org/settings.html[Maven settings].
type Server struct {
	XMLName       xml.Name   `json:"-"                       xml:"server"`
	ID            string     `json:"id,omitempty"            xml:"id,omitempty"`
	Username      string     `json:"username,omitempty"      xml:"username,omitempty"`
	Password      string     `json:"password,omitempty"      xml:"password,omitempty"`
	Configuration Properties `json:"configuration,omitempty" xml:"configuration,omitempty"`
}

// StringOrProperties -- .
type StringOrProperties struct {
	Value      string     `json:"-"                    xml:",chardata"`
	Properties Properties `json:"properties,omitempty" xml:"properties,omitempty"`
}

// Properties -- .
type Properties map[string]string

// PluginProperties -- .
type PluginProperties map[string]StringOrProperties

// PluginConfiguration see link:https://maven.apache.org/settings.html[Maven settings].
type PluginConfiguration struct {
	Container               Container        `json:"container"               xml:"container"`
	AllowInsecureRegistries string           `json:"allowInsecureRegistries" xml:"allowInsecureRegistries"`
	ExtraDirectories        ExtraDirectories `json:"extraDirectories"        xml:"extraDirectories"`
	PluginExtensions        PluginExtensions `json:"pluginExtensions"        xml:"pluginExtensions"`
}

// Container -- .
type Container struct {
	Entrypoint string `json:"entrypoint" xml:"entrypoint"`
	Args       Args   `json:"args"       xml:"args"`
}

// Args -- .
type Args struct {
	Arg string `json:"arg" xml:"arg"`
}

// ExtraDirectories -- .
type ExtraDirectories struct {
	Paths       []Path       `json:"paths>path"                       xml:"paths>path"`
	Permissions []Permission `json:"permissions>permission,omitempty" xml:"permissions>permission,omitempty"`
}

// Path -- .
type Path struct {
	From     string   `json:"from"                       xml:"from"`
	Into     string   `json:"into"                       xml:"into"`
	Excludes []string `json:"excludes>exclude,omitempty" xml:"excludes>exclude,omitempty"`
}

// Permission -- .
type Permission struct {
	File string `json:"file" xml:"file"`
	Mode string `json:"mode" xml:"mode"`
}

// PluginExtensions -- .
type PluginExtensions struct {
	PluginExtension PluginExtension `json:"pluginExtension" xml:"pluginExtension"`
}

// PluginExtension -- .
type PluginExtension struct {
	Implementation string                       `json:"implementation" xml:"implementation"`
	Configuration  PluginExtensionConfiguration `json:"configuration"  xml:"configuration"`
}

// PluginExtensionConfiguration -- .
type PluginExtensionConfiguration struct {
	Filters        []Filter `json:"filters>Filter"  xml:"filters>Filter"`
	Implementation string   `json:"_implementation" xml:"implementation,attr"`
}

// Filter -- .
type Filter struct {
	Glob    string `json:"glob"              xml:"glob"`
	ToLayer string `json:"toLayer,omitempty" xml:"toLayer,omitempty"`
}
