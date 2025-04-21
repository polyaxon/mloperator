package utils

import "time"

const (
	// DefaultBackOff is the max backoff period, exported for the e2e test
	DefaultBackOff = 10 * time.Second
	// MaxBackOff is the max backoff period, exported for the e2e test
	MaxBackOff = 360 * time.Second
	// DefaultRestartPolicy for Jobs
	DefaultRestartPolicy = "Never"
	// ZeroValue
	ZeroValue = 0
	// DefaultNumReplicas
	DefaultNumReplicas = 1
)

/*
GetTTL adjusts backoff in way that Polyaxon has time to finalize the operation
*/
func GetTTL(ttlSecondsAfterFinished *int32) *int32 {
	jobTtlSecondsAfterFinished := ttlSecondsAfterFinished
	if ttlSecondsAfterFinished == nil {
		defaultTtl := int32(ZeroValue)
		jobTtlSecondsAfterFinished = &defaultTtl
	}
	return jobTtlSecondsAfterFinished
}

/*
GetBackoffLimit utils function to handle default case
*/
func GetBackoffLimit(backoffLimit *int32) *int32 {
	jobBackoffLimit := backoffLimit
	if backoffLimit == nil {
		defaultBackoffLimit := int32(ZeroValue)
		jobBackoffLimit = &defaultBackoffLimit
	}
	return jobBackoffLimit
}

/*
GetNumReplicas utils function to handle default case
*/
func GetNumReplicas(numReplicas *int32, defaultValues ...int32) *int32 {
	opReplicas := numReplicas
	var defaultNumReplicas int32

	if len(defaultValues) > 0 {
		defaultNumReplicas = defaultValues[0]
	} else {
		defaultNumReplicas = DefaultNumReplicas
	}

	if numReplicas == nil {
		opReplicas = &defaultNumReplicas
	}

	return opReplicas
}
