#!/usr/bin/env bash
add_sidecar_registry ${TMPF}

# Install Camel K operator
wget https://github.com/apache/camel-k/releases/download/v1.12.0/camel-k-client-1.12.0-linux-64bit.tar.gz
tar -xvf camel-k-client-1.12.0-linux-64bit.tar.gz
./kamel install --registry localhost:5000 --registry-insecure --wait

# Add git-clone
add_task git-clone 0.7