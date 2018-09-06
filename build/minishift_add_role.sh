#!/bin/sh

user=$(oc whoami)

oc login -u system:admin
oc policy add-role-to-user --role-namespace=$(oc project -q) camel-k $user
oc login -u $user
