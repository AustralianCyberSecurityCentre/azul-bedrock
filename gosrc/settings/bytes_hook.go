package settings

import (
	"errors"
	"fmt"
	"log"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-viper/mapstructure/v2"
)

type HumanReadableBytes uint64

type conv struct {
	unit   int
	pow    int
	format string
}

// units maps common size unit suffixes to a multiplier, power and unit type
// not using init as need to ensure available before api init()
var units map[string]conv = map[string]conv{
	// for simplicity
	"":  {1, 1, "any"},
	"B": {1, 1, "any"},
	// SI
	"kB": {1000, 1, "si"},
	"MB": {1000, 2, "si"},
	"GB": {1000, 3, "si"},
	"TB": {1000, 4, "si"},
	"PB": {1000, 5, "si"},
	"EB": {1000, 6, "si"},
	// shorthand
	"K": {1000, 1, "si"},
	"M": {1000, 2, "si"},
	"G": {1000, 3, "si"},
	"T": {1000, 4, "si"},
	"P": {1000, 5, "si"},
	"E": {1000, 6, "si"},
	// IEC
	"KiB": {1024, 1, "iec"},
	"MiB": {1024, 2, "iec"},
	"GiB": {1024, 3, "iec"},
	"TiB": {1024, 4, "iec"},
	"PiB": {1024, 5, "iec"},
	"EiB": {1024, 6, "iec"},
	//shorthand
	"Ki": {1024, 1, "iec"},
	"Mi": {1024, 2, "iec"},
	"Gi": {1024, 3, "iec"},
	"Ti": {1024, 4, "iec"},
	"Pi": {1024, 5, "iec"},
	"Ei": {1024, 6, "iec"},
}

// HumanToBytes takes a human readable size string and returns in bytes.
// Both IEC and SI byte suffixes are supported.
func HumanToBytes(s string) (uint64, error) {
	pos := len(s)
	for {
		if pos <= 0 {
			return 0, errors.New("invalid size string")
		}

		if s[pos-1] >= '0' && s[pos-1] <= '9' {
			break
		}
		pos--
	}
	num, err := strconv.ParseFloat(s[0:pos], 64)
	if err != nil {
		return 0, errors.New("invalid numeric in string")
	}
	suffix := s[pos:]
	c, found := units[suffix]
	if !found {
		return 0, errors.New("invalid suffix on string")
	}
	return uint64(num) * uint64(math.Pow(float64(c.unit), float64(c.pow))), nil
}

// HumanToBytesF same as HumanToBytes but cannot return error and diff return type.
// To initialise default values in settings.
func HumanToBytesFatal(s string) HumanReadableBytes {
	ret, err := HumanToBytes(s)
	if err != nil {
		log.Fatalf("could not convert %s to bytes: %s", s, err.Error())
	}
	return HumanReadableBytes(ret)
}

// BytesToHuman formats a size of bytes into human readable format.
// IEC (pow of 2) are always used when converting.
func BytesToHuman(b uint64) string {
	// Default to IEC
	unit := 1
	pow := 0
	num := float64(b)
	for {
		if num >= 1024 {
			num /= 1024
			unit = 1024
			pow++
		} else {
			break
		}
	}
	suffix := "B"
	for k, v := range units {
		if v.unit == unit && v.pow == pow && v.format == "iec" && len(k) == 3 {
			suffix = k
			break
		}
	}
	// couldn't find matching pow??
	// just return orig input
	if suffix == "B" && pow > 1 {
		num = float64(b)
	}
	s := fmt.Sprintf("%.1f", num)
	// strip non-significant decimals.. is there a way to do this with printf?
	s = strings.TrimSuffix(s, ".0")
	return s + suffix
}

// convert strings to human readable bytes
func HumanReadableBytesHookFunc() mapstructure.DecodeHookFuncType {
	return func(
		f reflect.Type,
		t reflect.Type,
		data any,
	) (any, error) {
		// Check that the data is string
		if f.Kind() != reflect.String {
			return data, nil
		}

		// Check that the target type is our custom type
		if t != reflect.TypeOf(HumanReadableBytes(1)) {
			return data, nil
		}

		// Return the parsed value
		parsed, err := HumanToBytes(data.(string))
		if err != nil {
			log.Fatalf("config item was not valid human readable bytes string: %s", err.Error())
		}
		return HumanReadableBytes(parsed), nil
	}
}
