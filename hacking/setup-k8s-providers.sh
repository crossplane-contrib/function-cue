#!/bin/bash

set -euo pipefail

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "${SCRIPT_DIR}/.env"

XP_PROVIDER_K8S_VERSION=${XP_PROVIDER_K8S_VERSION:-v0.11.4}

echo "apply k8s provider"
${RUN_KUBECTL} apply -f - <<EOF
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-kubernetes
spec:
  package: xpkg.upbound.io/crossplane-contrib/provider-kubernetes:${XP_PROVIDER_K8S_VERSION}
  runtimeConfigRef:
    name: deploy-config-k8s
EOF

echo "apply k8s provider deployment config"
${RUN_KUBECTL} apply -f - <<EOF
apiVersion: pkg.crossplane.io/v1beta1
kind: DeploymentRuntimeConfig
metadata:
  name: deploy-config-k8s
spec:
  serviceAccountTemplate:
    metadata:
      name: crossplane-provider-kubernetes
EOF

echo "Waiting for the crossplane k8s provider to be ready. This could take some minutes..."
${RUN_KUBECTL} wait --timeout=300s --for=condition=Healthy providers.pkg.crossplane.io provider-kubernetes

echo "set credentials for k8s provider to be the injected identity"
${RUN_KUBECTL} apply -f - <<EOF
apiVersion: kubernetes.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: InjectedIdentity
EOF

echo "setting up cluster role bindings for the k8s provider"
${RUN_KUBECTL} create clusterrolebinding provider-kubernetes-admin-binding \
 --clusterrole cluster-admin --serviceaccount="${XP_CROSSPLANE_NS}:crossplane-provider-kubernetes" \
 --dry-run=client -o yaml \
 | ${RUN_KUBECTL} apply -f -
