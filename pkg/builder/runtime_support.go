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

import "path/filepath"

// QuarkusRuntimeNativeAdapter is used to get the proper Quarkus native configuration which may be different
// in Camel Quarkus version. It is known that before Camel Quarkus 3.5 there was no support to native-source,
// and using this interface will adapt the configuration to build natively according the previous configuration.
type QuarkusRuntimeNativeAdapter interface {
	// The commands used to build a native application
	BuildCommands() string
	// The directory where to execute the command
	Directory() string
	// The directory where to expect the native compiled artifact
	TargetDirectory(ctxPath, runner string) string
	// The parameter to use for the maven project
	NativeMavenProperty() string
}

// NativeSourcesAdapter used for Camel Quarkus runtime >= 3.5.0.
type NativeSourcesAdapter struct {
}

// BuildCommands -- .
func (n *NativeSourcesAdapter) BuildCommands() string {
	return "cd " + n.Directory() + " && echo NativeImage version is $(native-image --version) && echo GraalVM expected version is $(cat graalvm.version) && echo WARN: Make sure they are compatible, otherwise the native compilation may results in error && native-image $(cat native-image.args)"
}

// Directory -- .
func (n *NativeSourcesAdapter) Directory() string {
	return filepath.Join("maven", "target", "native-sources")
}

// TargetDirectory -- .
func (n *NativeSourcesAdapter) TargetDirectory(ctxPath, runner string) string {
	return filepath.Join(ctxPath, "maven", "target", "native-sources", runner)
}

// NativeMavenProperty -- .
func (n *NativeSourcesAdapter) NativeMavenProperty() string {
	return "native-sources"
}

// NativeAdapter used for Camel Quarkus runtime < 3.5.0.
type NativeAdapter struct {
}

// BuildCommands -- .
func (n *NativeAdapter) BuildCommands() string {
	return "cd " + n.Directory() + " && ./mvnw package -Dquarkus.package.type=native --global-settings settings.xml"
}

// Directory -- .
func (n *NativeAdapter) Directory() string {
	return "maven"
}

// TargetDirectory -- .
func (n *NativeAdapter) TargetDirectory(ctxPath, runner string) string {
	return filepath.Join(ctxPath, "maven", "target", runner)
}

// NativeMavenProperty -- .
func (n *NativeAdapter) NativeMavenProperty() string {
	// Empty on purpose. The parameter will be provided later by the command (see BuildCommands()).
	return ""
}

// QuarkusRuntimeSupport is used to get the proper native configuration based on the Camel Quarkus version.
func QuarkusRuntimeSupport(version string) QuarkusRuntimeNativeAdapter {
	if version < "3.5.0" {
		return &NativeAdapter{}
	}
	return &NativeSourcesAdapter{}
}
