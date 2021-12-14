# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM quay.io/quarkus/ubi-quarkus-mandrel:21.3.0.0-Final-java11

ARG MAVEN_VERSION="3.8.4"
ARG MAVEN_HOME="/usr/share/maven"
ARG SHA="a9b2d825eacf2e771ed5d6b0e01398589ac1bfa4171f36154d1b5787879605507802f699da6f7cfc80732a5282fd31b28e4cd6052338cbef0fa1358b48a5e3c8"
ARG BASE_URL="https://archive.apache.org/dist/maven/maven-3/${MAVEN_VERSION}/binaries"

USER 0

RUN microdnf update && microdnf install procps

RUN temp=$(mktemp -d) \
&& curl -Lso "$temp"/mvnd.zip https://github.com/mvndaemon/mvnd/releases/download/0.7.1/mvnd-0.7.1-linux-amd64.zip \
&& unzip -d "$temp" "$temp"/mvnd.zip \
&& rm -v "$temp"/mvnd.zip \
&& mkdir -p "$MAVEN_HOME" \
&& mv "$temp"/*/* "$MAVEN_HOME" \
&& rmdir "$temp"/* "$temp" \
&& ln -s ${MAVEN_HOME}/bin/mvnd /usr/bin/mvnd

RUN mkdir /.m2
RUN chgrp -R 0 /.m2 \
&& chmod -R g=u /.m2

ADD build/_maven_output /tmp/artifacts/m2
ADD build/_kamelets /kamelets
#COPY build/_maven_overlay/ /usr/share/maven/lib/
#ADD build/logback.xml /usr/share/maven/conf/

#ENV MAVEN_OPTS="${MAVEN_OPTS} -Dlogback.configurationFile=/usr/share/maven/conf/logback.xml"

RUN chgrp -R 0 /tmp/artifacts/m2 \
&& chmod -R g=u /tmp/artifacts/m2 \
&& chgrp -R 0 /kamelets \
&& chmod -R g=u /kamelets

#USER 1000
ADD build/_output/bin/kamel /usr/local/bin/kamel
