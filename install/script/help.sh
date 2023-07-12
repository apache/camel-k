#!/bin/bash

# ---------------------------------------------------------------------------
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
# ---------------------------------------------------------------------------


awk 'BEGIN {
      printf "\nUsage: make \033[31m<PARAM1=val1 PARAM2=val2>\033[0m \033[36m<target>\033[0m\n"
      printf "\nAvailable targets are:\n"
    }
    /^#@/ { printf "\033[36m%-15s\033[0m", $2; subdesc=0; next }
    /^#===/ { printf "%-14s \033[32m%s\033[0m\n", " ", substr($0, 5); subdesc=1; next }
    /^#==/ { printf "\033[0m%s\033[0m\n\n", substr($0, 4); next }
    /^#\*\*/ { printf "%-14s \033[31m%s\033[0m\n", " ", substr($0, 4); next }
    /^#\*/ && (subdesc == 1) { printf "\n"; next }
    /^#\-\-\-/ { printf "\n"; next }' ${1}
