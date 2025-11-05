package events

const (
	// Successfully completed
	StatusTypeCompleted string = "completed"
	// Successfully completed but no features or augmented streams were produced
	StatusTypeCompletedEmpty = "completed-empty"
	// Successfully completed but errors occurred which means the plugin might not have gotten all data.
	StatusTypeCompletedWithErrors string = "completed-with-errors"
	// Entity not suitable for this plugin (eg wrong size, type, ...)
	StatusTypeOptOut string = "opt-out"
	// Plugin heartbeat
	StatusTypeHeartbeat string = "heartbeat"
	// Event has been dequeued from kafka by dispatcher - not for use by plugins!
	StatusTypeDequeued string = "dequeued"
	// Plugin-specific code raised an unhandled exception
	StatusTypeErrorException string = "error-exception"
	// Plugin could not communicate with some required service
	StatusTypeErrorNetwork string = "error-network"
	// Generic error in plugin harness
	StatusTypeErrorRunner string = "error-runner"
	// Error processing input entity (eg incorrect format, corrupted) - legacy "entity error"
	StatusTypeErrorInput string = "error-input"
	// Plugin returned something that couldn't be understood by the runner
	StatusTypeErrorOutput string = "error-output"
	// Plugin exceeded its maximum execution time on a sample
	StatusTypeErrorTimeout string = "error-timeout"
	// Plugin execution was cancelled due to being out of memory
	StatusTypeErrorOOM string = "error-out-of-memory"
)

/*Check if the provided status is a completed type and if it is return true.*/
func IsStatusTypeCompleted(status string) bool {
	switch status {
	case StatusTypeCompleted:
		fallthrough
	case StatusTypeCompletedEmpty:
		fallthrough
	case StatusTypeCompletedWithErrors:
		return true
	default:
		return false
	}
}

/*Check if the provided status is a error type and if it is return true.*/
func IsStatusTypeError(status string) bool {
	switch status {
	case StatusTypeErrorException:
		fallthrough
	case StatusTypeErrorNetwork:
		fallthrough
	case StatusTypeErrorRunner:
		fallthrough
	case StatusTypeErrorInput:
		fallthrough
	case StatusTypeErrorOutput:
		fallthrough
	case StatusTypeErrorTimeout:
		fallthrough
	case StatusTypeErrorOOM:
		return true
	default:
		return false
	}
}

/*Check if the provided status is a progress type and if it is return true.*/
func IsStatusTypeProcess(status string) bool {
	switch status {
	case StatusTypeHeartbeat:
		fallthrough
	case StatusTypeDequeued:
		return true
	default:
		return false
	}
}
