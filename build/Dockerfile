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

FROM eclipse-temurin:17-jdk

ARG MAVEN_VERSION="3.8.6"
ARG MAVEN_HOME="/usr/share/maven"
ARG SHA="f790857f3b1f90ae8d16281f902c689e4f136ebe584aba45e4b1fa66c80cba826d3e0e52fdd04ed44b4c66f6d3fe3584a057c26dfcac544a60b301e6d0f91c26"
ARG BASE_URL="https://archive.apache.org/dist/maven/maven-3/${MAVEN_VERSION}/binaries"
ARG MVNW_DIR="/usr/share/maven/mvnw/"

USER 0

RUN mkdir -p ${MAVEN_HOME} \
    && curl -Lso /tmp/maven.tar.gz ${BASE_URL}/apache-maven-${MAVEN_VERSION}-bin.tar.gz \
    && echo "${SHA} /tmp/maven.tar.gz" | sha512sum -c - \
    && tar -xzC ${MAVEN_HOME} --strip-components=1 -f /tmp/maven.tar.gz \
    && rm -v /tmp/maven.tar.gz \
    && ln -s ${MAVEN_HOME}/bin/mvn /usr/bin/mvn \
    && rm ${MAVEN_HOME}/lib/maven-slf4j-provider*

ADD build/_maven_output /tmp/artifacts/m2
ADD build/_kamelets /kamelets
COPY build/_maven_overlay/ /usr/share/maven/lib/
ADD build/_maven_overlay/logback.xml /usr/share/maven/conf/

ENV MAVEN_OPTS="${MAVEN_OPTS} -Dlogback.configurationFile=${MAVEN_HOME}/conf/logback.xml"

# Prepare the maven wrapper required in the build Pod strategy
COPY build/mvnw/mvnw.tar ${MVNW_DIR}
RUN tar -xC ${MVNW_DIR} -f ${MVNW_DIR}mvnw.tar \
    && rm -v ${MVNW_DIR}mvnw.tar \
    && sed -i "s;distributionUrl=.*;distributionUrl=${BASE_URL}/apache-maven-${MAVEN_VERSION}-bin.zip;" ${MVNW_DIR}.mvn/wrapper/maven-wrapper.properties
# Used by mvnw to download maven dist into it
ENV MAVEN_USER_HOME="${MAVEN_HOME}"
# Let's install a default mvnw distribution at build time
RUN ${MVNW_DIR}/mvnw --version | grep "Maven home:" | sed 's/Maven home: //' >> ${MVNW_DIR}default
# TODO add maven overlay

RUN chgrp -R 0 /tmp/artifacts/m2 \
    && chmod -R g=u /tmp/artifacts/m2 \
    && chgrp -R 0 /kamelets \
    && chmod -R g=u /kamelets \
    && chgrp -R 0 ${MAVEN_HOME} \
    && chmod -R g=u ${MAVEN_HOME}

USER 1000

ADD build/_output/bin/kamel /usr/local/bin/kamel
