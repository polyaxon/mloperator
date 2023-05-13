#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

if [ -z "${GOPATH:-}" ]; then
    export GOPATH=$(go env GOPATH)
fi

VERSION="v1"

KNOWN_VIOLATION_EXCEPTIONS=hack/violation_exceptions.list
CURRENT_VIOLATION_EXCEPTIONS=hack/current_violation_exceptions.list

PROJECT_ROOT=$(cd $(dirname "$0")/.. ; pwd)
CODEGEN_PKG=${PROJECT_ROOT}/vendor/k8s.io/kube-openapi

# Generating OpenAPI specification
go run ${CODEGEN_PKG}/cmd/openapi-gen/openapi-gen.go \
    --input-dirs github.com/polyaxon/mloperator/api/${VERSION} \
    --output-package api/${VERSION}/ \
    --go-header-file hack/boilerplate.go.txt \
    --report-filename $CURRENT_VIOLATION_EXCEPTIONS \
    $@

test -f $CURRENT_VIOLATION_EXCEPTIONS || touch $CURRENT_VIOLATION_EXCEPTIONS

# The API rule fails if generated API rule violation report differs from the
# checked-in violation file, prints error message to request developer to
# fix either the API source code, or the known API rule violation file.
diff $CURRENT_VIOLATION_EXCEPTIONS $KNOWN_VIOLATION_EXCEPTIONS || \
    (echo -e "ERROR: \n\t API rule check failed. Reported violations in file $CURRENT_VIOLATION_EXCEPTIONS differ from known violations in file $KNOWN_VIOLATION_EXCEPTIONS. \n"; exit 1)

# Generating swagger file
go run cmd/openapi-gen/main.go 1.0 > api/${VERSION}/swagger.json
