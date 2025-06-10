#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
# set -x

# SCRIPT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
SCRIPT_ROOT="/home/gsanti/myLiqo/foreign_cluster_connector"
CODEGEN_PKG="/home/gsanti/myLiqo/code-generator"

# Optional: set your module path here
MODULE_PATH="github.com/Gabbe64/foreign_cluster_connector"

# Source the kube codegen helpers
source "${CODEGEN_PKG}/kube_codegen.sh"

# Generate clientset, listers, and informers for your APIs
kube::codegen::gen_client \
  "${SCRIPT_ROOT}" \
  --output-dir "${SCRIPT_ROOT}/pkg/client" \
  --output-pkg "${MODULE_PATH}/pkg/client" \
  --boilerplate "${SCRIPT_ROOT}/hack/boilerplate.go.txt" \
  --with-watch
