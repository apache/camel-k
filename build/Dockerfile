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

FROM quay.io/quarkus/ubi-quarkus-native-image:21.1.0-java11
#FROM quay.io/quarkus/ubi-quarkus-mandrel:21.2.0.0-Final-java11

ARG MAVEN_VERSION="3.6.3"
ARG SHA="c35a1803a6e70a126e80b2b3ae33eed961f83ed74d18fcd16909b2d44d7dada3203f1ffe726c17ef8dcca2dcaa9fca676987befeadc9b9f759967a8cb77181c0"
ARG BASE_URL="https://apache.osuosl.org/maven/maven-3/${MAVEN_VERSION}/binaries"

USER 0

RUN mkdir -p /usr/share/maven \
    && curl -Lso /tmp/maven.tar.gz ${BASE_URL}/apache-maven-${MAVEN_VERSION}-bin.tar.gz \
    && echo "${SHA} /tmp/maven.tar.gz" | sha512sum -c - \
    && tar -xzC /usr/share/maven --strip-components=1 -f /tmp/maven.tar.gz \
    && rm -v /tmp/maven.tar.gz \
    && ln -s /usr/share/maven/bin/mvn /usr/bin/mvn \
    && rm /usr/share/maven/lib/maven-slf4j-provider*

ADD build/_maven_output /tmp/artifacts/m2
ADD build/_kamelets /kamelets
COPY build/_maven_overlay/ /usr/share/maven/lib/
ADD build/logback.xml /usr/share/maven/conf/

ENV MAVEN_OPTS="${MAVEN_OPTS} -Dlogback.configurationFile=/usr/share/maven/conf/logback.xml"

RUN chgrp -R 0 /tmp/artifacts/m2 \
    && chmod -R g=u /tmp/artifacts/m2 \
    && chgrp -R 0 /kamelets \
    && chmod -R g=u /kamelets

USER 1000

ADD build/_output/bin/kamel /usr/local/bin/kamel
