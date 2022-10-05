#!/bin/bash

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
