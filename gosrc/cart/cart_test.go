package cart

import (
	"bytes"
	"os"
	"path"
	"testing"

	testdata "github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/testdata"
	"github.com/stretchr/testify/assert"
)

func TestTextLong(t *testing.T) {
	raw, err := UncartBytes(path.Join(testdata.CartDir, "text-long.txt.cart"))
	if err != nil {
		t.Fatalf("%v", err)
	}

	if len(raw) != 23055 {
		t.Fatalf("unexpected length %v", len(raw))
	}
}

func TestTextShort(t *testing.T) {
	raw, err := UncartBytes(path.Join(testdata.CartDir, "text.txt.cart"))
	if err != nil {
		t.Fatalf("%v", err)
	}

	if len(raw) != 29 {
		t.Fatalf("unexpected length %v", len(raw))
	}

	if string(raw) != "This is some test text data.\n" {
		t.Fatalf("bad content '%v'", string(raw))
	}
}

func TestPackAndUnpackCart(t *testing.T) {
	outputStream := bytes.NewBuffer([]byte{})
	inputBytes := []byte("A random binary file\nWhy not something simple!")
	inStream := bytes.NewBuffer(inputBytes)
	err := PackCart(inStream, outputStream)
	if err != nil {
		t.Fatalf("%v", err)
	}
	outputStream.Bytes()
	tempFile, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())

	// Used to collect a sample cart to be uncarted with the python uncart.
	// f, _ := os.Create("./sample.cart")
	//f.Write(outputStream.Bytes())

	_, err = tempFile.Write(outputStream.Bytes())
	if err != nil {
		t.Fatalf("%v", err)
	}
	roundTripBytes, err := UncartBytes(tempFile.Name())
	if err != nil {
		t.Fatalf("%v", err)
	}

	assert.Equal(t, inputBytes, roundTripBytes)
}
