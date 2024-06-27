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

# list and remove dangling camel-k kit images

remove=0
verbose=0
while getopts "pv" opt; do
  case "${opt}" in
    v)
      verbose=1
      ;;
    p)
      remove=1
      ;;
    \?)
      ;;
  esac
done
shift $((OPTIND-1))

file_images=$(mktemp)
file_images_from_ints=$(mktemp)

trap "rm -f $file_images $file_images_from_ints" SIGINT SIGTERM ERR EXIT

IFS=''

# images from container registry
docker images |grep k-kit|awk '{print $1}'|sort|uniq > $file_images

if [ $verbose -eq 1 ]; then
    echo "> Images from container registry:"
    cat $file_images
    echo
fi

# images from integrations
kubectl get -A it -oyaml|grep 'image:'|sed 's/^\s*image: //g;s/@sha256.*//g'|sort|uniq > $file_images_from_ints
if [ $verbose -eq 1 ]; then
    echo "> Images of Camel K Integrations"
    cat $file_images_from_ints
    echo
fi

# use comm utility to show only the images with no associated integration
dangling=$(comm -3 $file_images $file_images_from_ints)
if [ -z $dangling ] ; then
    echo "> No dangling container images to prune."
else
    echo "> Images from container registry, eligible for pruning."
    echo $dangling
    if [ $remove -eq 1 ] ; then
        echo
        echo "> Delete Container Images"
        echo $dangling|while read imgaddr; do
            ns=$(echo $imgaddr|awk -F '/' '{print $2}');
            kit=$(echo $imgaddr|awk -F '/' '{print $3}'|sed 's/camel-k-//g');
            kubectl -n $ns delete ik/$kit
            imgid=$(docker images|grep $imgaddr|awk '{print $3}')
            docker rmi $imgid
        done
    fi
fi
