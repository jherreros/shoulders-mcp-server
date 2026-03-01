#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
AUTH_CONFIG="$SCRIPT_DIR/authentication-config.yaml"
NODE_NAME="$(kind get nodes --name shoulders | grep control-plane | head -n1)"
AUTH_CONFIG_PATH="/etc/kubernetes/authn/authentication-config.yaml"
APISERVER_MANIFEST="/etc/kubernetes/manifests/kube-apiserver.yaml"
APISERVER_MANIFEST_BACKUP="/etc/kubernetes/kube-apiserver.yaml.pre-oidc"
DEFAULT_DEX_SERVICE_IP="10.96.0.24"
AUTH_CONFIG_B64="$(base64 < "$AUTH_CONFIG" | tr -d '\n')"

if command -v docker >/dev/null 2>&1; then
  CONTAINER_CLI="docker"
elif command -v podman >/dev/null 2>&1; then
  CONTAINER_CLI="podman"
else
  echo "Neither docker nor podman is available; cannot patch the kind control-plane container." >&2
  exit 1
fi

DEX_SERVICE_IP="$(kubectl -n dex get svc dex -o jsonpath='{.spec.clusterIP}' 2>/dev/null || true)"
if [[ -z "$DEX_SERVICE_IP" || "$DEX_SERVICE_IP" == "None" ]]; then
  DEX_SERVICE_IP="$DEFAULT_DEX_SERVICE_IP"
fi

PREVIOUS_APISERVER_START_TIME="$(kubectl -n kube-system get pod -l component=kube-apiserver -o jsonpath='{.items[0].status.startTime}' 2>/dev/null || true)"

echo "Writing kube-apiserver authentication config to ${NODE_NAME}..."

"$CONTAINER_CLI" exec -i "$NODE_NAME" sh -c "set -eu
mkdir -p /etc/kubernetes/authn
printf '%s' '${AUTH_CONFIG_B64}' | base64 -d > ${AUTH_CONFIG_PATH}

awk '!/dex\\.dex\\.svc\\.cluster\\.local|dex\\.127\\.0\\.0\\.1\\.sslip\\.io/' /etc/hosts > /etc/hosts.shoulders
printf '%s dex.dex.svc.cluster.local dex.dex.svc\n' '${DEX_SERVICE_IP}' >> /etc/hosts.shoulders
printf '%s dex.127.0.0.1.sslip.io\n' '${DEX_SERVICE_IP}' >> /etc/hosts.shoulders
cat /etc/hosts.shoulders > /etc/hosts
rm /etc/hosts.shoulders

cp ${APISERVER_MANIFEST} ${APISERVER_MANIFEST_BACKUP}
tmp=\$(mktemp)
cp ${APISERVER_MANIFEST} \"\$tmp\"

perl -0pi -e 's@\\n\\s*-\\s+--authentication-config=${AUTH_CONFIG_PATH//\//\\/}@@g; s@\\n\\s*- mountPath: /etc/kubernetes/authn\\n\\s*name: k8s-authn\\n\\s*readOnly: true@@g; s@\\n\\s*- hostPath:\\n\\s*path: /etc/kubernetes/authn\\n\\s*type: DirectoryOrCreate\\n\\s*name: k8s-authn@@g' \"\$tmp\"
perl -0pi -e 's@(\\n\\s*- kube-apiserver\\n)@\$1    - --authentication-config=${AUTH_CONFIG_PATH}\\n@' \"\$tmp\"
perl -0pi -e 's@(\\n\\s*- mountPath: /etc/kubernetes/pki\\n\\s*name: k8s-certs\\n\\s*readOnly: true\\n)@\$1    - mountPath: /etc/kubernetes/authn\\n      name: k8s-authn\\n      readOnly: true\\n@' \"\$tmp\"
perl -0pi -e 's@(\\n\\s*- hostPath:\\n\\s*path: /etc/kubernetes/pki\\n\\s*type: DirectoryOrCreate\\n\\s*name: k8s-certs\\n)@\$1  - hostPath:\\n      path: /etc/kubernetes/authn\\n      type: DirectoryOrCreate\\n    name: k8s-authn\\n@' \"\$tmp\"

mv \"\$tmp\" ${APISERVER_MANIFEST}
"

echo "Ensuring kube-apiserver uses the authentication config..."

echo "Waiting for kube-apiserver to restart..."
saw_unready=0
for _ in $(seq 1 60); do
  current_apiserver_start_time="$(kubectl -n kube-system get pod -l component=kube-apiserver -o jsonpath='{.items[0].status.startTime}' 2>/dev/null || true)"
  if kubectl get --raw=/readyz >/dev/null 2>&1; then
    if [[ "$saw_unready" -eq 1 ]] || [[ -n "$PREVIOUS_APISERVER_START_TIME" && -n "$current_apiserver_start_time" && "$current_apiserver_start_time" != "$PREVIOUS_APISERVER_START_TIME" ]]; then
      echo "kube-apiserver is ready"
      exit 0
    fi
  else
    saw_unready=1
  fi
  sleep 5
done

echo "kube-apiserver did not become ready after a confirmed restart, restoring previous manifest..." >&2
"$CONTAINER_CLI" exec "$NODE_NAME" sh -c "cp ${APISERVER_MANIFEST_BACKUP} ${APISERVER_MANIFEST}"

echo "Waiting for kube-apiserver to recover after rollback..."
saw_unready=0
ROLLBACK_PREVIOUS_APISERVER_START_TIME="$(kubectl -n kube-system get pod -l component=kube-apiserver -o jsonpath='{.items[0].status.startTime}' 2>/dev/null || true)"
for _ in $(seq 1 60); do
  current_apiserver_start_time="$(kubectl -n kube-system get pod -l component=kube-apiserver -o jsonpath='{.items[0].status.startTime}' 2>/dev/null || true)"
  if kubectl get --raw=/readyz >/dev/null 2>&1; then
    if [[ "$saw_unready" -eq 1 ]] || [[ -n "$ROLLBACK_PREVIOUS_APISERVER_START_TIME" && -n "$current_apiserver_start_time" && "$current_apiserver_start_time" != "$ROLLBACK_PREVIOUS_APISERVER_START_TIME" ]]; then
      echo "kube-apiserver recovered after rollback"
      exit 1
    fi
  else
    saw_unready=1
  fi
  sleep 5
done

echo "Timed out waiting for kube-apiserver readiness after enabling OIDC." >&2
exit 1