/*
Package settings controls reading configuration from environment and assigning defaults
*/
package plugin

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

var Logger zerolog.Logger

func RecreateLogger(level string) {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	// initialise logger with caller information
	internal := log.With().Caller().Logger()
	// configure pretty print for log output (expensive)
	internal = internal.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	logmap := map[string]zerolog.Level{
		"TRACE":   zerolog.TraceLevel,
		"DEBUG":   zerolog.DebugLevel,
		"INFO":    zerolog.InfoLevel,
		"WARN":    zerolog.WarnLevel,
		"WARNING": zerolog.WarnLevel,
		"ERROR":   zerolog.ErrorLevel,
	}
	level = strings.ToUpper(level)
	internal = internal.Level(logmap[level])

	Logger = internal
	Logger.Info().Msg("created logger")
}

func init() {
	// load settings v2
	pluginSettings := parsePluginSettings(NewDefaultPluginSettings())
	RecreateLogger(pluginSettings.RunnerLogLevel)
}
