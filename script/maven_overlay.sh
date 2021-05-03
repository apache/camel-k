#!/bin/sh

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

mvn dependency:get -Dartifact=ch.qos.logback:logback-core:1.2.3 -Ddest=$1
mvn dependency:get -Dartifact=ch.qos.logback:logback-classic:1.2.3 -Ddest=$1
mvn dependency:get -Dartifact=net.logstash.logback:logstash-logback-encoder:4.11 -Ddest=$1
mvn dependency:get -Dartifact=com.fasterxml.jackson.core:jackson-annotations:2.9.10 -Ddest=$1
mvn dependency:get -Dartifact=com.fasterxml.jackson.core:jackson-core:2.9.10 -Ddest=$1
mvn dependency:get -Dartifact=com.fasterxml.jackson.core:jackson-databind:2.9.10.8 -Ddest=$1