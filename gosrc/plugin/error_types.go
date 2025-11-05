package plugin

import (
	"errors"
	"fmt"
)

type PluginInnerError error

// Opt out error used to indicate an opt out occurred.
//
// nolint:staticcheck // ST1012 Not really an error so not following guidelines
var OptOutError PluginInnerError = errors.New("opt-out error")

// This is dedicated to errors that can only be resolved by the plugin author
var ErrorException PluginInnerError = errors.New("error-exception")

// Plugin could not communicate with some required service
var ErrorNetwork PluginInnerError = errors.New("error-network")

// Generic error in plugin harness
var ErrorRunner PluginInnerError = errors.New("error-runner")

// Error processing input entity (eg incorrect format, corrupted) - legacy "entity error"
var ErrorInput PluginInnerError = errors.New("error-input")

// Plugin returned something that couldn't be understood by the runner
var ErrorsOutput PluginInnerError = errors.New("error-output")

// Plugin exceeded its maximum execution time on a sample
var ErrorTimeout PluginInnerError = errors.New("error-timeout")

// Add the cause error for a plugin error to the end of the message
func (pe *PluginError) WithCausalError(err error) *PluginError {
	pe.message = fmt.Sprintf("%s with error %v", pe.message, err)
	return pe
}

// Create a plugin error that can be returned to indicate an opt-out has occurred, with the reason provided.
func NewPluginOptOut(message string) *PluginError {
	return &PluginError{
		innerError: OptOutError,
		errorTitle: "",
		message:    message,
	}
}

// Create a plugin error, the innerError must be specifically a PluginInnerError type not a generic error.
func NewPluginError(innerError PluginInnerError, errorTitle, message string) *PluginError {
	switch innerError {
	case OptOutError:
	case ErrorException:
	case ErrorNetwork:
	case ErrorRunner:
	case ErrorInput:
	case ErrorsOutput:
	case ErrorTimeout:
	default:
		Logger.Warn().Err(innerError).Msg("Warning the error provided to NewPluginError was the wrong type it should not be a generic error, it will be treated as an ErrorException.\n To add an error to the output message use the function PluginErrorOptionAddCauseError")
	}
	pe := &PluginError{
		innerError: innerError,
		errorTitle: errorTitle,
		message:    message,
	}
	return pe
}

type PluginError struct {
	innerError PluginInnerError
	errorTitle string
	message    string
}

func (e *PluginError) Error() string {
	return fmt.Sprintf("%s - %s - %s", e.errorTitle, e.innerError.Error(), e.message)
}

func (e *PluginError) GetInnerError() error {
	return e.innerError
}

func (e *PluginError) GetTitle() string {
	if e.innerError == OptOutError {
		return ""
	}
	return e.errorTitle
}

func (e *PluginError) GetMessage() string {
	return e.message
}
