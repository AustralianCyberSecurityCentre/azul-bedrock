package identify

import (
	_ "embed"
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

const ZERO_BYTE_SHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

//go:embed identify_test.yaml
var raw_test_data []byte

func loadCartFile(t *testing.T, sha256 string) []byte {
	data, err := fileManager.DownloadFileBytes(sha256)
	if err != nil {
		t.Fatalf("FileManager could not download the hash %s with error %v", sha256, err)
	}
	return data
}

func dbgMismatch(t *testing.T, got, expected string, identified Identified) {
	t.Errorf("incorrect format, expected '%v' but got '%v'", expected, got)
	for _, magic := range identified.Magics {
		t.Errorf("magic: %v", magic)
	}
	for _, mime := range identified.Mimes {
		t.Errorf("mime: %v", mime)
	}
}

func testMatches(t *testing.T, sha256, exp_id, exp_legacy, exp_ext, exp_magic, exp_mime string) {
	var data []byte
	switch sha256 {
	case ZERO_BYTE_SHA256:
		data = []byte{}
	case "":
		t.Fatalf("Provided an empty sha256, other values were exp_id: %s, exp_legacy: %s, exp_ext: %s, exp_magic: %s, exp_mime: %s", exp_id, exp_legacy, exp_ext, exp_magic, exp_mime)
	default:
		data = loadCartFile(t, sha256)
	}

	uncartedTempFile, err := os.CreateTemp("", sha256)
	if err != nil {
		t.Fatalf("Failed to create the temporary file %s with error %s", sha256, err.Error())
	}
	defer uncartedTempFile.Close()
	defer os.Remove(uncartedTempFile.Name())
	if err != nil {
		t.Fatalf("Unable to open file with sha256 %s", sha256)
	}
	_, err = uncartedTempFile.Write(data)
	if err != nil {
		t.Fatalf("Unable to write bytes to the temp file for path %s", uncartedTempFile.Name())
	}

	id, err := cfg.Find(uncartedTempFile.Name())
	if err != nil {
		t.Fatalf("For file '%v' -%v-%v-%v error - %v", sha256, exp_id, exp_legacy, exp_ext, err)
	}
	if id.FileFormat != exp_id {
		dbgMismatch(t, id.FileFormat, exp_id, id)
	}
	if id.FileFormatLegacy != exp_legacy {
		dbgMismatch(t, id.FileFormatLegacy, exp_legacy, id)
	}
	if id.FileExtension != exp_ext {
		t.Errorf("bad extension: %v vs %v", id.FileExtension, exp_ext)
	}
	if exp_magic != "" && id.Magic != exp_magic {
		t.Errorf("bad magic: %v vs %v", id.Magic, exp_magic)
	}
	if exp_mime != "" && id.Mime != exp_mime {
		t.Errorf("bad mime: %v vs %v", id.Mime, exp_mime)
	}
}

type IdentifyTestData struct {
	Identify_Tests []struct {
		Sha256             string
		File_Format        string
		File_Format_Legacy string
		File_Extension     string
		Magic              string
		Mime               string
	}
}

func TestIdentifyAllFiles(t *testing.T) {
	test_data := IdentifyTestData{}
	err := yaml.Unmarshal(raw_test_data, &test_data)
	if err != nil {
		t.Fatalf("Failed to parse test data with error '%v'.", err)
	}

	for _, tc := range test_data.Identify_Tests {
		t.Logf("Testing file '%v'", tc.Sha256)
		testMatches(t, tc.Sha256, tc.File_Format, tc.File_Format_Legacy, tc.File_Extension, tc.Magic, tc.Mime)
	}
}
