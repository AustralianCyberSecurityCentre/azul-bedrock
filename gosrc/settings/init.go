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

type S3TransportSettings struct {
	DialTimeoutSeconds           int  `koanf:"dial_timeout_seconds"`
	DialKeepAliveSeconds         int  `koanf:"dial_keep_alive_seconds"`
	MaxIdleConnections           int  `koanf:"max_idle_connections"`
	MaxIdleConnectionsPerHost    int  `koanf:"max_idle_connections_per_host"`
	IdleHeaderTimeoutSeconds     int  `koanf:"idle_header_timeout_seconds"`
	IdleConnectionTimeoutSeconds int  `koanf:"idle_connection_timeout_seconds"`
	TlsHandshakeTimeoutSeconds   int  `koanf:"tlsh_handshake_timeout_seconds"`
	DisableCompression           bool `koanf:"disable_compression"`
}

type S3ListSettings struct {
	// Use the v1Api
	UseV1Api bool `koanf:"use_v1_api"`
	// Max keys per batch of keys retrieved by the list API. (must be greater than 0 to have an effect)
	MaxKeys int `koanf:"max_keys"`
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
	// Secret key for aes encryption, only required if aes-256 encryption is required.
	// Length of the key must be exactly 24 characters long or 0.
	// When set AES encryption is enabled.
	AesKey string `koanf:"aes_key"`
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
	// Advanced settings
	S3TransportSettings S3TransportSettings `koanf:"transport"`
	S3ListSettings      S3ListSettings      `koanf:"s3_list"`
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
	S3TransportSettings: S3TransportSettings{
		DialTimeoutSeconds:           30,
		DialKeepAliveSeconds:         30,
		MaxIdleConnections:           256,
		MaxIdleConnectionsPerHost:    16,
		IdleHeaderTimeoutSeconds:     60,
		IdleConnectionTimeoutSeconds: 60,
		TlsHandshakeTimeoutSeconds:   10,
		DisableCompression:           true,
	},
	S3ListSettings: S3ListSettings{
		UseV1Api: false,
		MaxKeys:  -1,
	},
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
