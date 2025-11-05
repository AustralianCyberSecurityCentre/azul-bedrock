package settings

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testInput struct {
	inputString  string
	inputBytes   uint64
	outputString string
	outputBytes  uint64
}

func TestHumanToBytes(t *testing.T) {
	tables := []testInput{
		{inputString: "987B", inputBytes: uint64(987)},
		{inputString: "987", inputBytes: uint64(987), outputString: "987B"},
		{inputString: "0B", inputBytes: uint64(0)},
		{inputString: "0", inputBytes: uint64(0), outputString: "0B"},
		{inputString: "42B", inputBytes: uint64(42)},
		{inputString: "1KiB", inputBytes: uint64(1024)},
		{inputString: "1KiB", inputBytes: uint64(1026), outputBytes: uint64(1024)},
		{inputString: "59.6GiB", inputBytes: uint64(64 * 1000 * 1000 * 1000), outputBytes: uint64(63350767616)},
		{inputString: "64Gi", inputBytes: uint64(64 * 1024 * 1024 * 1024), outputString: "64GiB"},
		{inputString: "2048MiB", inputBytes: uint64(2048 * 1024 * 1024), outputString: "2GiB"},
		{inputString: "2TiB", inputBytes: uint64(2048 * 1024 * 1024 * 1024)},
	}
	for _, table := range tables {
		b, err := HumanToBytes(table.inputString)
		require.Nil(t, err, "Failed to convert: %v", err)
		// Use output bytes if set for special cases such as rounding.
		if table.outputBytes != 0 {
			assert.Equal(t, table.outputBytes, b, "HumanToBytes for Input: %s Expected: %d Got: %d", table.inputString, table.outputBytes, b)
		} else {
			assert.Equal(t, table.inputBytes, b, "HumanToBytes for Input: %s Expected: %d Got: %d", table.inputString, table.inputBytes, b)
		}
	}
	for _, table := range tables {
		s := BytesToHuman(table.inputBytes)
		// Use output string for special cases like rounding and differing units.
		if len(table.outputString) != 0 {
			assert.Equal(t, table.outputString, s, "BytesToHuman for Input: %d Expected: %s Got: %s", table.inputBytes, table.outputString, s)
		} else {
			assert.Equal(t, table.inputString, s, "BytesToHuman for Input: %d Expected: %s Got: %s", table.inputBytes, table.inputString, s)
		}

	}
}
