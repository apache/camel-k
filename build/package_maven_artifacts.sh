#!/bin/sh

location=$(dirname $0)
cd $location/../
./mvnw clean install -DskipTests -f runtime/pom.xml -s tmp/maven/settings.xml
