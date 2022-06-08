#!/bin/bash

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

location=$(dirname $0)
rootdir=$location/../

denylist=("zz_generated" "zz_desc_generated" "vendor" "./.mvn/wrapper" "./docs/" "./.idea" "./build/" "./deploy/traits.yaml" "./pom.xml")

cd $rootdir
go build ./cmd/util/license-check/

check_licenses() {
	files=$1
	header=$2

	set +e
    failed=0
    find . -type f -name "$files" -print0 | while IFS= read -r -d '' file; do
        check=true
        for b in ${denylist[*]}; do
        	if [[ "$file" == *"$b"* ]]; then
        	  #echo "skip $file"
        	  check=false
        	fi
        done
    	if [ "$check" = true ]; then
    		#echo "exec $file"
    		./license-check "$file" $header
    		if [ $? -ne 0 ]; then
    	  		failed=1
    		fi
    	fi
    done
    set -e

    if [ $failed -ne 0 ]; then
      exit 1
    fi
}


check_licenses *.go ./script/headers/default.txt
check_licenses *.groovy ./script/headers/default-jvm.txt
check_licenses *.java ./script/headers/default-jvm.txt
check_licenses *.kts ./script/headers/default-jvm.txt
check_licenses *.js ./script/headers/js.txt
check_licenses *.xml ./script/headers/xml.txt
check_licenses *.yaml ./script/headers/yaml.txt
check_licenses *.yml ./script/headers/yaml.txt
check_licenses *.sh ./script/headers/sh.txt
