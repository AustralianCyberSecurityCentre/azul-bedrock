package settings

/* Core Logger Settings for bedrock and all packages that import it.*/

import (
	"os"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

type StreamsAzure struct {
	Endpoint       string `koanf:"endpoint"`
	StorageAccount string `koanf:"storage_account"`
	Container      string `koanf:"container"`
	AccessKey      string `koanf:"access_key"`
}

type StreamsS3 struct {
	// S3 server address or empty to use local storage instead
	Endpoint string `koanf:"endpoint"`
	// Access key to auth against S3 bucket
	AccessKey string `koanf:"access_key"`
	// Secret key to auth against S3 bucket
	SecretKey string `koanf:"secret_key"`
	// Whether to utilise HTTPS for S3 transport
	Secure bool `koanf:"secure"`
	// S3 region or empty if unsupported by server
	Region string `koanf:"region"`
	// S3 bucket name to store to (will attempt to create if not exists)
	Bucket string `koanf:"bucket"`
}

type Streams struct {
	S3    StreamsS3    `koanf:"s3"`
	Azure StreamsAzure `koanf:"azure"`
}

type BedSettings struct {
	// logging level to render to stdout
	LogLevel string `koanf:"log_level"`
	// Render nice coloured log output (slower performance)
	LogPretty bool `koanf:"log_pretty"`
}

// Settings used purely for testing.
type BedTestSettings struct {
	Streams Streams `koanf:"streams"`
}

var Settings *BedSettings
var TestSettings *BedTestSettings
var Logger zerolog.Logger

var defaults BedSettings = BedSettings{
	LogLevel:  "INFO",
	LogPretty: true,
}

var testDefaults BedTestSettings = BedTestSettings{
	Streams: Streams{
		S3: StreamsS3{
			Bucket: "azul",
		},
		Azure: StreamsAzure{
			Container: "azul",
		},
	},
}

func ResetSettings() {
	Settings = ParseSettings(defaults, "BED", []mapstructure.DecodeHookFunc{})
	// Uses the same prefix as dispatcher to make setting variables during testing simple
	TestSettings = ParseSettings(testDefaults, "DP", []mapstructure.DecodeHookFunc{})
}

func RecreateLogger(level string) {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	// initialise logger with caller information
	internal := log.With().Caller().Logger()
	if Settings.LogPretty {
		// configure pretty print for log output (expensive)
		internal = internal.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	logmap := map[string]zerolog.Level{
		"TRACE": zerolog.TraceLevel,
		"DEBUG": zerolog.DebugLevel,
		"INFO":  zerolog.InfoLevel,
		"WARN":  zerolog.WarnLevel,
		"ERROR": zerolog.ErrorLevel,
	}
	internal = internal.Level(logmap[level])

	Logger = internal
	Logger.Info().Msg("created logger")
}

func init() {
	// load settings v2
	ResetSettings()
	RecreateLogger(Settings.LogLevel)
}
