package testdata

import (
	"fmt"
	"log"
	"os"
	"path"
	"reflect"
	"runtime"
	"testing"

	"github.com/goccy/go-json"

	"github.com/stretchr/testify/require"
)

// dir of this go module, so tests can load files beneath it
var CartDir string
var Dir string

func GetBytes(filepath string) []byte {
	ret, err := os.ReadFile(path.Join(Dir, filepath))
	if err != nil {
		log.Fatalf("could not load test file %v: %v", filepath, err)
	}
	return ret
}

func DumpBytes(filepath string, data []byte) {
	err := os.WriteFile(path.Join(Dir, filepath), data, 0644)
	if err != nil {
		log.Fatalf("could not save test file %v: %v", filepath, err)
	}
}

func ExistsBytes(filepath string) bool {
	_, err := os.Stat(path.Join(Dir, filepath))
	return !os.IsNotExist(err)
}

func init() {
	_, filename, _, _ := runtime.Caller(0)
	Dir = path.Join(path.Dir(filename), "/../../testdata/")
	CartDir = path.Join(Dir, "/cart/")
}

// compare two structures by first dumping to json
// this removes problems with time, and other non-normalised data
func MarshalEqual(t *testing.T, in1, in2 interface{}) {
	// non pointer marshalling can prevent custom MarshalJSON from running properly
	if reflect.ValueOf(in1).Kind() != reflect.Ptr {
		panic(fmt.Errorf("provided in1 was not a pointer, was %s", reflect.TypeOf(in1)))
	}
	if reflect.ValueOf(in2).Kind() != reflect.Ptr {
		panic(fmt.Errorf("provided in2 was not a pointer, was %s", reflect.TypeOf(in2)))
	}
	raw1, err := json.Marshal(in1)
	require.Nil(t, err)
	raw2, err := json.Marshal(in2)
	require.Nil(t, err)
	require.JSONEq(t, string(raw1), string(raw2))
}
