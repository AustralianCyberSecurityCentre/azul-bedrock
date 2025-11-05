package settings

import (
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/go-viper/mapstructure/v2"
	kenv "github.com/knadh/koanf/providers/env"
	kstructs "github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

func ParseSettings[GenericSettings any](defaults GenericSettings, settingPrefix string, decodeHooks []mapstructure.DecodeHookFunc) *GenericSettings {
	// koanf instance. Use "." as the key path delimiter.
	var k = koanf.New(".")

	// load defaults first
	err := k.Load(kstructs.Provider(defaults, "koanf"), nil)
	if err != nil {
		log.Fatalf("defaults bad config load: %s", err.Error())
	}

	dotPrefix := ""
	// load environment 1 (with periods)
	if len(settingPrefix) > 0 {
		dotPrefix = fmt.Sprintf("%s.", settingPrefix)
	}
	f := kenv.Provider(dotPrefix, ".", func(s string) string {
		return strings.ReplaceAll(strings.ToLower(
			strings.TrimPrefix(s, dotPrefix)), ".", ".")
	})
	err = k.Load(f, nil)
	if err != nil {
		log.Fatalf("env bad config load 1: %s", err.Error())
	}

	// load environment 2 (with double underscores)
	doubleUnderscorePrefix := ""
	if len(settingPrefix) > 0 {
		doubleUnderscorePrefix = fmt.Sprintf("%s__", settingPrefix)
	}
	f = kenv.Provider(doubleUnderscorePrefix, "__", func(s string) string {
		return strings.ReplaceAll(strings.ToLower(
			strings.TrimPrefix(s, doubleUnderscorePrefix)), "__", "__")
	})
	err = k.Load(f, nil)
	if err != nil {
		log.Fatalf("env bad config load 2: %s", err.Error())
	}

	out := *new(GenericSettings)
	err = k.UnmarshalWithConf("", &out, koanf.UnmarshalConf{
		DecoderConfig: &mapstructure.DecoderConfig{
			DecodeHook: mapstructure.ComposeDecodeHookFunc(
				decodeHooks...,
			),
			Result:           &out,
			WeaklyTypedInput: true,
		},
	})
	if err != nil {
		log.Fatalf("bad config parse: %s", err.Error())
	}

	// check for invalid config (items that don't exist)
	// we check that the struct of values we just generated is not missing anything that was loaded into koanf
	var k2 = koanf.New(".")
	err = k2.Load(kstructs.Provider(out, "koanf"), nil)
	if err != nil {
		log.Fatalf("checker bad config load: %s", err.Error())
	}

	// check and print settings
	w := tabwriter.NewWriter(os.Stdout, 1, 1, 2, ' ', 0)
	fmt.Fprintf(w, "env-key\tenv-value\t\n")
	for key := range k.All() {
		if k2.Get(key) == nil {
			// Don't print unknown settings if we don't have a prefix.
			if settingPrefix == "" {
				continue
			}
			log.Printf("WARNING unknown config option supplied (check spelling): %v", dotPrefix+strings.ToUpper(key))
			continue
		}
		val := fmt.Sprintf("%v", k2.Get(key))
		// Hide secrets
		if strings.Contains(strings.ToLower(key), "secret") || strings.Contains(strings.ToLower(key), "access_key") || strings.Contains(strings.ToLower(key), "password") {
			val = "****"
		}
		// long lines don't print
		if len(val) > 100 {
			val = fmt.Sprintf("len(%d)", len(val))
		}
		fmt.Fprintf(w, "%s\t'%v'\t\n", key, val)
	}
	w.Flush()
	return &out
}
