/*
Copyright 2018-2023 Polyaxon, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/validation/spec"

	// _ "k8s.io/kube-openapi/cmd/openapi-gen"
	// _ "k8s.io/code-generator"
	// _ "k8s.io/code-generator/cmd/go-to-protobuf/protoc-gen-gogo"
	ctrl "sigs.k8s.io/controller-runtime"

	operationv1 "github.com/polyaxon/mloperator/api/v1"
)

func main() {
	log := ctrl.Log.WithName("openapi-gen")
	if len(os.Args) <= 1 {
		log.Info("Supply a version", "No version found")
		os.Exit(1)
	}
	version := os.Args[1]
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	var oAPIDefs = map[string]common.OpenAPIDefinition{}
	defs := spec.Definitions{}
	refCallback := func(path string) spec.Ref {
		return spec.MustCreateRef("#/definitions/" + common.EscapeJsonPointer(swaggify(path)))
	}
	for k, v := range operationv1.GetOpenAPIDefinitions(refCallback) {
		oAPIDefs[k] = v
	}

	swagger := spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Swagger:     "2.0",
			Definitions: defs,
			Paths:       &spec.Paths{Paths: map[string]spec.PathItem{}},
			Info: &spec.Info{
				InfoProps: spec.InfoProps{
					Title:       "Polyaxon Operator",
					Description: "Polyaxon SDK for Operator specification",
					Version:     version,
				},
			},
		},
	}
	jsonBytes, err := json.MarshalIndent(swagger, "", "  ")
	if err != nil {
		log.Error(err, "Error marshal")
	}
	fmt.Println(string(jsonBytes))
}

func swaggify(name string) string {
	name = strings.Replace(name, "github.com/polyaxon/mloperator/api/", "", -1)
	name = strings.Replace(name, "k8s.io/api/core/", "", -1)
	name = strings.Replace(name, "k8s.io/apimachinery/pkg/apis/meta/", "", -1)
	name = strings.Replace(name, "/", ".", -1)
	return name
}
