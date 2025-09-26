package managers

import (
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// CopyKFJobFields copies the owned fields from one KFJob to another
// Returns true if the fields copied from don't match to.
func CopyKFJobFields(from, to *unstructured.Unstructured) bool {
	return CopyUnstructuredSpec(from, to, []string{"labels", "spec"})
}

// CopyUnstructuredSpec copies the owned fields from one unstructured to another
// Returns true if the fields copied from don't match to.
func CopyUnstructuredSpec(from, to *unstructured.Unstructured, fields []string) bool {
	requireUpdate := false

	for field := range fields {
		if CopyUnstructuredField(from, to, fields[field]) {
			requireUpdate = true
		}
	}
	return requireUpdate
}

// CopyUnstructuredField copies the owned fields from one unstructured to another
// Returns true if the fields copied from don't match to.
func CopyUnstructuredField(from, to *unstructured.Unstructured, field string) bool {
	fromSpec, found, err := unstructured.NestedMap(from.Object, field)
	if !found {
		return false
	}
	if err != nil {
		return false
	}

	toSpec, found, err := unstructured.NestedMap(to.Object, field)
	if !found || err != nil {
		unstructured.SetNestedMap(to.Object, fromSpec, field)
		return true
	}

	requiresUpdate := !reflect.DeepEqual(fromSpec, toSpec)
	if requiresUpdate {
		unstructured.SetNestedMap(to.Object, fromSpec, field)
	}
	return requiresUpdate
}
