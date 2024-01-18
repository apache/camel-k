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

set -e

display_usage() {

cat <<EOF
Create a new release branch, synchronizing all CI tasks and resources.

Usage: ./script/release-branch.sh

--help                    This help message
-d                        Dry run, do not push to GIT repo

Example: ./script/release-branch.sh
EOF

}

DRYRUN="false"
SEMVER="^([[:digit:]]+)\.([[:digit:]]+)\.([[:digit:]]+)(-SNAPSHOT)$"

main() {
  parse_args $@
  location=$(dirname $0)

  VERSION=$(make get-version)
  if ! [[ $VERSION =~ $SEMVER ]]; then
    echo "❗ POM version must match major.minor.patch(-SNAPSHOT) semantic version: $1"
    exit 1
  fi
  VERSION_FULL="${BASH_REMATCH[1]}.${BASH_REMATCH[2]}.${BASH_REMATCH[3]}"
  VERSION_MM="${BASH_REMATCH[1]}.${BASH_REMATCH[2]}"

  new_release_branch="release-$VERSION_MM.x"

  # Support nightly CI tasks
  # pick the oldest release assuming they all use the same strategy convention
  oldest_release_branch=$(yq '.jobs.auto-updates.strategy.matrix.ref-branch[] | select(. != "main")' $location/../.github/workflows/nightly-automatic-updates.yml | sort | head -1)
  echo "Swapping GH actions tasks from $oldest_release_branch to $new_release_branch"

  sed -i "s/$oldest_release_branch/$new_release_branch/g" $location/../.github/workflows/nightly-automatic-updates.yml
  # We're skipping from release branches because it takes too much resources
  #sed -i "s/$oldest_release_branch/$new_release_branch/g" $location/../.github/workflows/nightly-native-test.yml
  sed -i "s/$oldest_release_branch/$new_release_branch/g" $location/../.github/workflows/nightly-release.yml

  if [ $DRYRUN == "true" ]
  then
    echo "❗ dry-run mode on, won't push any change!"
  else
    git add --all
    git commit -m "chore: starting release branch for $new_release_branch" || true
    git push origin HEAD:$new_release_branch
    # We must push on main as well, as it contains the changes for CI workflows
    git push origin HEAD:main
    echo "🎉 Changes pushed correctly!"
  fi
}

parse_args(){
  while [ $# -gt 0 ]
  do
      arg="$1"
      case $arg in
        -h|--help)
          display_usage
          exit 0
          ;;
        -d)
          DRYRUN="true"
          ;;
        *)
          echo "❗ unknown argument: $1"
          display_usage
          exit 1
          ;;
      esac
      shift
  done
}

main $*
