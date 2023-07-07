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

FROM alpine:latest AS jibcli
ARG JIB_CLI_VERSION=0.12.0
ARG JIB_HOME="/opt/jib"
RUN apk --no-cache add ca-certificates \
    curl \
    unzip \
    && rm -rf /var/cache/apk/*
# replace xargs command not available in some base images
RUN set -xe \
    && curl -sSL -o jib.zip https://github.com/GoogleContainerTools/jib/releases/download/v${JIB_CLI_VERSION}-cli/jib-jre-${JIB_CLI_VERSION}.zip \
    && unzip jib.zip  \
    && rm jib.zip \
    && mv jib-${JIB_CLI_VERSION} ${JIB_HOME} \
    && sed -i "s/xargs -n1/tr \' \'  \'\\\n\'/g" ${JIB_HOME}/bin/jib

FROM eclipse-temurin:17-jdk as base

ARG MAVEN_DEFAULT_VERSION="3.8.6"
ARG MAVEN_HOME="/usr/share/maven"
ARG MAVEN_DIST_URL="https://archive.apache.org/dist/maven/maven-3/${MAVEN_DEFAULT_VERSION}/binaries/apache-maven-${MAVEN_DEFAULT_VERSION}-bin.zip"
ARG MVNW_DIR="/usr/share/maven/mvnw/"
ARG MVN_REPO="/etc/maven/m2"

ARG JIB_HOME="/opt/jib"

USER 0

# Maven configuration
RUN mkdir -p ${MAVEN_HOME}
RUN mkdir -p ${MVN_REPO}
COPY build/_maven_overlay/ /usr/share/maven/lib/
ADD build/_maven_overlay/logback.xml /usr/share/maven/conf/

# Prepare the maven wrapper required in the build Pod strategy
COPY build/mvnw/mvnw.tar ${MVNW_DIR}
RUN tar -xC ${MVNW_DIR} -f ${MVNW_DIR}mvnw.tar \
    && rm -v ${MVNW_DIR}mvnw.tar \
    && sed -i "s;distributionUrl=.*;distributionUrl=${MAVEN_DIST_URL};" ${MVNW_DIR}.mvn/wrapper/maven-wrapper.properties
# Used by mvnw to download maven dist into it
ENV MAVEN_USER_HOME="${MAVEN_HOME}"
# Install a default mvnw distribution at build time and prepare the config for formatting log
RUN ${MVNW_DIR}/mvnw --version | grep "Maven home:" | sed 's/Maven home: //' >> ${MVNW_DIR}default \
    && cp -r /usr/share/maven/lib/. $(cat ${MVNW_DIR}default)/lib \
    && rm $(cat ${MVNW_DIR}default)/lib/maven-slf4j-provider*
ENV MAVEN_OPTS="${MAVEN_OPTS} -Dlogback.configurationFile=${MAVEN_HOME}/conf/logback.xml"

COPY --from=jibcli ${JIB_HOME} ${JIB_HOME}

ADD build/_maven_output ${MVN_REPO}
ADD build/_kamelets /kamelets

RUN chgrp -R 0 ${MVN_REPO} \
    && chown -R 1001:0 ${MVN_REPO} \
    && chmod -R 775 ${MVN_REPO} \
    && chgrp -R 0 /kamelets \
    && chmod -R g=u /kamelets \
    && chgrp -R 0 ${MAVEN_HOME} \
    && chown -R 1001:0 ${MAVEN_HOME} \
    && chmod -R 775 ${MAVEN_HOME} \
    && chown -R 1001:0 ${JIB_HOME} \
    && chmod -R 775 ${JIB_HOME}

USER 1001:0

ADD build/_output/bin/kamel /usr/local/bin/kamel

FROM golang:1.18 as go

RUN go install github.com/go-delve/delve/cmd/dlv@latest

FROM base as debug

COPY --from=go /go/bin/dlv /usr/local/bin/dlv