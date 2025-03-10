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

const JibMavenGoal = "jib:build"
const JibMavenToImageParam = "-Djib.to.image="
const JibMavenFromImageParam = "-Djib.from.image="
const JibMavenFromPlatforms = "-Djib.from.platforms="
const JibMavenBaseImageCache = "-Djib.baseImageCache="
const JibMavenInsecureRegistries = "-Djib.allowInsecureRegistries="
const JibDigestFile = "target/jib-image.digest"
const JibMavenPluginVersionDefault = "3.4.1"
const JibLayerFilterExtensionMavenVersionDefault = "0.3.0"

// See: https://github.com/GoogleContainerTools/jib/blob/master/jib-maven-plugin/README.md#using-docker-configuration-files
const JibRegistryConfigEnvVar = "DOCKER_CONFIG"

// The Jib profile configuration.
const XMLJibProfile = `
<profile>
  <id>jib</id>
  <activation>
    <activeByDefault>false</activeByDefault>
  </activation>
  <repositories></repositories>
  <pluginRepositories></pluginRepositories>
  <build>
    <plugins>
      <plugin>
        <groupId>com.google.cloud.tools</groupId>
        <artifactId>jib-maven-plugin</artifactId>
        <version>3.4.1</version>
        <executions></executions>
        <dependencies>
          <dependency>
            <groupId>com.google.cloud.tools</groupId>
            <artifactId>jib-layer-filter-extension-maven</artifactId>
            <version>0.3.0</version>
          </dependency>
        </dependencies>
        <configuration>
          <container>
            <entrypoint>INHERIT</entrypoint>
            <args>
              <arg>jshell</arg>
            </args>
          </container>
          <allowInsecureRegistries>true</allowInsecureRegistries>
          <extraDirectories>
            <paths>
              <path>
                <from>../context</from>
                <into>/deployments</into>
                <excludes></excludes>
              </path>
            </paths>
            <permissions>
              <permission>
                <file>/deployments/*</file>
                <mode>755</mode>
              </permission>
            </permissions>
          </extraDirectories>
          <pluginExtensions>
            <pluginExtension>
              <implementation>com.google.cloud.tools.jib.maven.extension.layerfilter.JibLayerFilterExtension</implementation>
              <configuration implementation="com.google.cloud.tools.jib.maven.extension.layerfilter.Configuration">
                <filters>
                  <Filter>
                    <glob>/app/**</glob>
                  </Filter>
                </filters>
              </configuration>
            </pluginExtension>
          </pluginExtensions>
        </configuration>
      </plugin>
    </plugins>
  </build>
</profile>
`
