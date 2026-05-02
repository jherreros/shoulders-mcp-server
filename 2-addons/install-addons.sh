#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

CILIUM_VERSION="1.19.1"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SHOULDERS_PROFILE="${SHOULDERS_PROFILE:-medium}"

# Cilium
helm repo add cilium https://helm.cilium.io/
helm install cilium cilium/cilium --version ${CILIUM_VERSION} \
   --namespace kube-system \
   --set kubeProxyReplacement=true \
   --set image.pullPolicy=IfNotPresent \
   --set ipam.mode=kubernetes

cilium status --wait

# This script installs FluxCD.

if ! command -v flux &> /dev/null
then
    echo "Flux CLI not found. Installing..."
    curl -s https://fluxcd.io/install.sh | sudo bash
fi

if ! flux check --pre &> /dev/null
then
    echo "Flux pre-check failed. Please check your environment."
    exit 1
fi

if ! flux get kustomization flux-system &> /dev/null
then
    echo "Installing FluxCD..."
    cd "$SCRIPT_DIR"
    flux install
    kubectl apply -k "profiles/${SHOULDERS_PROFILE}/flux"
else
    echo "FluxCD already installed. Reconciling..."
    cd "$SCRIPT_DIR"
    kubectl apply -k "profiles/${SHOULDERS_PROFILE}/flux"
    flux reconcile source git flux-system
fi

echo "Waiting for Dex deployment..."
until kubectl -n dex get deploy dex >/dev/null 2>&1; do
    sleep 5
done

kubectl -n dex rollout status deploy/dex --timeout=10m

"$SCRIPT_DIR/../1-cluster/configure-apiserver-oidc.sh"
