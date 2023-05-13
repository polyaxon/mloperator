#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

KUBE_ROOT=$(dirname "${BASH_SOURCE}")/..
CODEGEN_PKG=vendor/k8s.io/code-generator
if [ -z "${GOPATH:-}" ]; then
    export GOPATH=$(go env GOPATH)
fi
vendor/k8s.io/code-generator/generate-groups.sh all "github.com/polyaxon/mloperator/pkg/client" "github.com/polyaxon/mloperator/api" core:v1  --go-header-file hack/boilerplate.go.txt
