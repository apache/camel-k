#!/bin/bash

# this path is defined in Dockerfile
cd /usr/share/local/camel-deps

java -classpath quarkus-run.jar:lib/boot/*:lib/main/*:app/camel-deps.jar io.quarkus.bootstrap.runner.QuarkusEntryPoint $1
