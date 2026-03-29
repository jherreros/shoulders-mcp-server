#!/bin/bash

set -o errexit

cd "$(dirname "$0")"

# Create a vind (vCluster-in-Docker) cluster named shoulders.
# Requires Docker and the vcluster CLI (v0.31.0+).
vcluster use driver docker
vcluster create shoulders --connect --chart-version 0.33.1 \
  --values vind-config.yaml