package events

type StatusType string

const (
	// Successfully completed
	StatusTypeCompleted StatusType = "completed"
	// Successfully completed but no features or augmented streams were produced
	StatusTypeCompletedEmpty StatusType = "completed-empty"
	// Successfully completed but errors occurred which means the plugin might not have gotten all data.
	StatusTypeCompletedWithErrors StatusType = "completed-with-errors"
	// Entity not suitable for this plugin (eg wrong size, type, ...)
	StatusTypeOptOut StatusType = "opt-out"
	// Plugin heartbeat
	StatusTypeHeartbeat StatusType = "heartbeat"
	// Event has been dequeued from kafka by dispatcher - not for use by plugins!
	StatusTypeDequeued StatusType = "dequeued"
	// Download has been requested by a user or plugin.
	StatusTypeDownloadRequested StatusType = "download-requested"
	// Plugin-specific code raised an unhandled exception
	StatusTypeErrorException StatusType = "error-exception"
	// Plugin could not communicate with some required service
	StatusTypeErrorNetwork StatusType = "error-network"
	// Generic error in plugin harness
	StatusTypeErrorRunner StatusType = "error-runner"
	// Error processing input entity (eg incorrect format, corrupted) - legacy "entity error"
	StatusTypeErrorInput StatusType = "error-input"
	// Plugin returned something that couldn't be understood by the runner
	StatusTypeErrorOutput StatusType = "error-output"
	// Plugin exceeded its maximum execution time on a sample
	StatusTypeErrorTimeout StatusType = "error-timeout"
	// Plugin execution was cancelled due to being out of memory
	StatusTypeErrorOOM StatusType = "error-out-of-memory"
)

/*Check if the provided status is a completed type and if it is return true.*/
func IsStatusTypeCompleted(status StatusType) bool {
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
func IsStatusTypeError(status StatusType) bool {
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
func IsStatusTypeProcess(status StatusType) bool {
	switch status {
	case StatusTypeHeartbeat:
		fallthrough
	case StatusTypeDequeued:
		fallthrough
	case StatusTypeDownloadRequested:
		return true
	default:
		return false
	}
}
