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

if [[ "$OSTYPE" != "linux"* ]]; then
    echo "ERROR: This script is intended to run in linux. You may try to edit and tweak it to run in other OS."
    exit 1
fi

mvnCmd=$(which mvn)
central_repo=https://repo1.maven.org/maven2
remote_repo=https://repo1.maven.org/maven2
start_time=$(date -u +"%s")

while getopts ":v:m:r:d:s" opt; do
  case "${opt}" in
    m)
      mvnCmd="${OPTARG}"
      ;;
    r)
      remote_repo="${OPTARG}"
      ;;
    v)
      runtime_version="${OPTARG}"
      ;;
    d)
      offline_new_dir="${OPTARG}"
      ;;
    s)
      skip_https="-k"
      ;;
    *)
      ;;
  esac
done
shift $((OPTIND-1))

if [ -z ${runtime_version} ]; then
    echo "usage: $0 -v <Camel K Runtime version> [optional parameters]"
    echo "  -m </usr/share/bin/mvn>   - Path to mvn command"
    echo "  -r <http://my-repo.com>   - URL address of the maven repository manager"
    echo "  -d </var/tmp/offline-1.2> - Local directory to add the offline dependencies"
    echo "  -s                        - Skip Certificate validation"
    exit 1
fi

offline_dir=./_offline-${runtime_version}
if [ ! -z ${offline_new_dir} ]; then
    offline_dir=${offline_new_dir}
fi

offline_repo=${offline_dir}/repo

# the pom.xml is the one containing all the dependencies from the camel-catalog file
# it is used when running the go-offline and quarkus:go-offline goals to resolve all dependencies
pom=${offline_dir}/pom.xml
# the pom-min.xml is used to actually build the project
# it was noted that some transitive dependencies are not correctly resolve when using the go-offlilne plugin goals
# then this was required to resolve some transitive dependencies.
pom_min=${offline_dir}/pom-min.xml

############# SETUP MAVEN
# get the maven version used by camel-k-operator
camelk_mvn_ver=$(curl -s https://raw.githubusercontent.com/apache/camel-k/release-2.3.x/build/Dockerfile|grep MAVEN_DEFAULT_VERSION= |cut -d\" -f2)
# get the maven version set by the user from the parameters
mvn_ver=$($mvnCmd --version |grep "Apache Maven"|awk '{print $3}')
# the maven version executing the task MUST be exactly the same versin as set by the camel-k-operator
if [ "${camelk_mvn_ver}" != "${mvn_ver}" ]; then
    # if the maven version is different, download the correct maven version
    url="https://archive.apache.org/dist/maven/maven-3/${camelk_mvn_ver}/binaries/apache-maven-${camelk_mvn_ver}-bin.tar.gz"
    echo "WARNING: Wrong Maven version \"${mvn_ver}\", it must be the same as in camel-k operator: \"${camelk_mvn_ver}\""
    echo "         This script will attempt to download it from: ${url}"
    install_dir=`mktemp -d --suffix _maven`
    curl -fsSL ${url} | tar zx --strip-components=1 -C ${install_dir}
    trap "{ rm -r "${install_dir}" ; exit 255; }" SIGINT SIGTERM ERR EXIT
    mvnCmd=${install_dir}/bin/mvn
fi

mkdir -p ${offline_repo}
# ignore file not found
rm -f ${pom} 2> /dev/null

$mvnCmd --version | grep "Apache Maven"

# setup the maven settings in case there is a custom maven repository url
if [[ "${central_repo}" != "${remote_repo}" ]]; then
    curl -sfSL https://raw.githubusercontent.com/apache/camel-k/release-2.3.x/script/maven-settings-offline-template.xml -o ${offline_dir}/maven-settings-offline-template.xml
    sed "s,_local-maven-proxy_,${remote_repo},g;/<mirrors>/,/<\/mirrors>/d" ${offline_dir}/maven-settings-offline-template.xml > ${offline_dir}/custom-maven-settings.xml

fi

## END SETUP MAVEN

############# SETUP CAMEL CATALOG
echo "INFO: downloading catalog for Camel K Runtime ${runtime_version}"
url=${remote_repo}/org/apache/camel/k/camel-k-catalog/${runtime_version}/camel-k-catalog-${runtime_version}-catalog.yaml

if [ -z "${skip_https}" ]; then
  # validate if there are certificate issues connecting with curl
  cert_problem=$(curl --no-progress-meter -o /dev/null -LI -w '%{http_code}' ${url} 2>&1|grep -i 'SSL certificate problem'|head -1)
  if [ ! -z "${cert_problem}" ]; then
      echo "ERROR: There is a problem to connect to the maven repository: ${cert_problem}"
      echo "You can set the parameter -s to skip certificate validation."
      exit 1
  fi
fi

response_code=$(curl --no-progress-meter -o /dev/null --silent -LI ${skip_https} -w '%{http_code}' ${url} 2>&1)
if [ 200 != ${response_code} ]; then
    echo "ERROR: Camel K Runtime version ${runtime_version} catalog doesn't exist at ${url}"
    exit 1
fi
catalog="${offline_dir}/camel-catalog-${runtime_version}.yaml"
curl -sfSL ${skip_https} ${url} -o ${catalog}
## END SETUP CAMEL CATALOG

############# SETUP POM PROJECT
ckr_version=$(yq .spec.runtime.version ${catalog})
cq_version=$(yq '.spec.runtime.metadata."camel-quarkus.version"' $catalog)
quarkus_version=$(yq '.spec.runtime.metadata."quarkus.version"' $catalog)
jibVersion=$(curl -s https://raw.githubusercontent.com/apache/camel-k/release-2.3.x/pkg/util/jib/configuration.go|grep 'const JibMavenPluginVersionDefault'|cut -d\" -f2)
jibLayerFilterVersion=$(curl -s https://raw.githubusercontent.com/apache/camel-k/release-2.3.x/pkg/util/jib/configuration.go|grep 'const JibLayerFilterExtensionMavenVersionDefault'|cut -d\" -f2)

echo "INFO: configuring offline dependencies for Camel K Runtime $ckr_version, Camel Quarkus $cq_version and Quarkus Platform version $quarkus_version"
echo "INFO: preparing a base project to download maven dependencies..."

cat <<EOF > ${pom}
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>
    <groupId>org.apache.camel.k.integration</groupId>
    <artifactId>camel-k-integration-offline</artifactId>
    <version>0.1</version>
    <properties>
        <project.build.sourceEncoding>UTF-8</project.build.sourceEncoding>
        <quarkus.package.jar.type>fast-jar</quarkus.package.jar.type>
        <maven.compiler.release>17</maven.compiler.release>
        <maven.compiler.source>17</maven.compiler.source>
        <maven.compiler.target>17</maven.compiler.target>
    </properties>
    <dependencyManagement>
        <dependencies>
            <dependency>
                <groupId>org.apache.camel.k</groupId>
                <artifactId>camel-k-runtime-bom</artifactId>
                <version>$runtime_version</version>
                <type>pom</type>
                <scope>import</scope>
            </dependency>
        </dependencies>
    </dependencyManagement>
    <dependencies>
EOF

# collect all artifacts from the camel-catalog and add them to the pom file
sed 's/- //g' $catalog | grep "groupId\|artifactId" | paste -d " "  - - |awk '{print $2":"$4}'|sort|uniq|while read line; do
#     echo $line;
    g=$(echo $line|cut -d: -f1);
    a=$(echo $line|cut -d: -f2);

    # there is no opentracing extension in CEQ, but it was present at the time camel-catalog, skipping it.
    if [[ $a == *opentracing ]] || [[ $a == *camel-quarkus-beanio ]]; then
        continue;
    fi

    # the jolokia agent must set the classifier
    if [[ $a == "jolokia-agent-jvm" ]]; then
      cat <<EOF >> ${pom};
      <dependency>
          <groupId>$g</groupId>
          <artifactId>$a</artifactId>
          <classifier>javaagent</classifier>
      </dependency>
EOF
        continue;
    fi

    cat <<EOF >> ${pom};
      <dependency>
          <groupId>$g</groupId>
          <artifactId>$a</artifactId>
      </dependency>
EOF

done

# tweak the jib dependency to retrieve the correct dependencies
cat <<EOF >> ${pom}
  </dependencies>

  <build>
    <plugins>
      <plugin>
        <groupId>io.quarkus</groupId>
        <artifactId>quarkus-maven-plugin</artifactId>
        <version>${quarkus_version}</version>
        <executions>
        </executions>
      </plugin>
      <plugin>
        <groupId>com.google.cloud.tools</groupId>
        <artifactId>jib-maven-plugin</artifactId>
        <version>${jibVersion}</version>
        <executions>
        </executions>
        <dependencies>
            <dependency>
                <groupId>com.google.cloud.tools</groupId>
                <artifactId>jib-layer-filter-extension-maven</artifactId>
                <version>${jibLayerFilterVersion}</version>
            </dependency>
        </dependencies>
      </plugin>
    </plugins>
  </build>
</project>

EOF

# project with minimum pom dependencies, only to run the mvn package to resolve the dependencies
# this is necessary since the quarkus-maven-plugin resolves some transitive dependencies not resolved by quarkus:go-offline

cat <<EOF > ${pom_min}
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>
    <groupId>org.apache.camel.k.integration</groupId>
    <artifactId>camel-k-integration-offline-min</artifactId>
    <version>0.1</version>
    <properties>
        <project.build.sourceEncoding>UTF-8</project.build.sourceEncoding>
        <quarkus.package.jar.type>fast-jar</quarkus.package.jar.type>
        <maven.compiler.release>17</maven.compiler.release>
        <maven.compiler.source>17</maven.compiler.source>
        <maven.compiler.target>17</maven.compiler.target>
    </properties>
    <dependencyManagement>
        <dependencies>
            <dependency>
                <groupId>org.apache.camel.k</groupId>
                <artifactId>camel-k-runtime-bom</artifactId>
                <version>$runtime_version</version>
                <type>pom</type>
                <scope>import</scope>
            </dependency>
        </dependencies>
    </dependencyManagement>
    <dependencies>
EOF

sed 's/- //g' $catalog | grep "groupId\|artifactId" | paste -d " "  - - |awk '{print $2":"$4}'|sort|uniq|while read line; do
#     echo $line;
    g=$(echo $line|cut -d: -f1);
    a=$(echo $line|cut -d: -f2);

    # only adds these dependencies
    if [[ $g == org.apache.camel.k ]] || [[ $a == *timer ]] || [[ $a == *log ]] || [[ $a == *knative ]] || [[ $a == *-core ]] || [[ $a == *-http ]] || [[ $a == *dsl ]]; then
      cat <<EOF >> ${pom_min};
        <dependency>
            <groupId>$g</groupId>
            <artifactId>$a</artifactId>
        </dependency>
EOF
    fi

done

# tweak the jib dependency to retrieve the correct dependencies
cat <<EOF >> ${pom_min}
    </dependencies>

  <build>
    <plugins>
      <plugin>
        <groupId>io.quarkus</groupId>
        <artifactId>quarkus-maven-plugin</artifactId>
        <version>${quarkus_version}</version>
        <executions>
          <execution>
            <id>build-integration</id>
            <goals>
                <goal>build</goal>
            </goals>
            <configuration>
              <properties>
                <quarkus.camel.routes-discovery.enabled>false</quarkus.camel.routes-discovery.enabled>
                <quarkus.banner.enabled>false</quarkus.banner.enabled>
                <quarkus.camel.servlet.url-patterns>/*</quarkus.camel.servlet.url-patterns>
                <quarkus.hibernate-orm.enabled>false</quarkus.hibernate-orm.enabled>
              </properties>
            </configuration>
          </execution>
        </executions>
      </plugin>
    </plugins>
  </build>
</project>

EOF

# add a single route to compile
mkdir -p ${offline_dir}/src/main/java/foo
cat <<EOF > ${offline_dir}/src/main/java/foo/Foo.java
package foo;

import java.lang.Exception;
import java.lang.Override;
import org.apache.camel.builder.RouteBuilder;

public class Foo extends RouteBuilder {

  @Override
  public void configure() throws Exception {
    from("timer:java?period=200000")
        .to("log:info");
  }
}

EOF
############# END SETUP POM PROJECT

# resolve and download artifacts in parallel
perf_params="-Dmaven.artifact.threads=6 -T 6 -Daether.dependencyCollector.impl=bf"
silent="-ntp -Dsilent=true"
mvn_skip_ssl=""
if [ ! -z "${skip_https}" ]; then
    mvn_skip_ssl="-Dmaven.wagon.http.ssl.insecure=true"
fi
settings_param=""
if [[ "${central_repo}" != "${remote_repo}" ]]; then
    settings_param="-s ${offline_dir}/custom-maven-settings.xml"
fi

$mvnCmd ${perf_params} ${silent} ${mvn_skip_ssl} ${settings_param} -Dmaven.repo.local=$offline_repo dependency:go-offline quarkus:go-offline -f ${pom}
$mvnCmd ${perf_params} ${silent} ${mvn_skip_ssl} ${settings_param} -Dmaven.repo.local=$offline_repo package -f ${pom_min}

# remove _remote.repositories as they interfere with the original repo resolver when running in the camel-k-operator pod
find $offline_repo -type f -name _remote.repositories -delete

# we can bundle into a single archive now
offline_file=${offline_dir}/camel-k-runtime-$runtime_version-maven-offline.tar.gz
echo "INFO: building ${offline_file} archive"
tar -czf ${offline_file} -C $offline_repo .

# not removig the cached dependencies, since if any failure occurs while executing the script, it can run again and continue the operation.
# echo "INFO: deleting cached dependencies..."
# rm -rf $offline_repo

echo "Success: your bundled set of offline dependencies is available in ${offline_file} file."
echo "The maven artifacts are in $offline_repo taking space, you may want to remove it later."
end_time=$(date -u +"%s")
elapsed=$(($end_time-$start_time))
echo "Elapsed Time: "$(date -u -d "@${elapsed}" +%T)
