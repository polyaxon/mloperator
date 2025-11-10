package utils

import (
	apierrs "k8s.io/apimachinery/pkg/api/errors"
)

/*
IgnoreNotFound allows to ignore (not requeue) NotFound errors, since we'll get a
reconciliation request once the object exists, and requeuing in the meantime
won't help, and we can get them on deleted requests.
*/
func IgnoreNotFound(err error) error {
	if apierrs.IsNotFound(err) {
		return nil
	}
	return err
}

/* Polyaxon's default main container name */
const MainJobContainer = "polyaxon-main"
