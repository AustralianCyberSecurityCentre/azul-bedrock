// cart format interactions
// prototype
// does not support proper streaming as we expect the input stream to strictly be a single cart file
// this is because we seek around the file using the beginning and end offsets
package cart

import (
	"bytes"
	"compress/zlib"
	"crypto/rc4"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
)

// Format Overview
//
//	MANDATORY HEADER (Not compress, not encrypted.
//	4s     h         Q        16s         Q
//
// 'CART<VERSION><RESERVED><ARC4KEY><OPT_HEADER_LEN>'
//
// OPTIONAL_HEADER (OPT_HEADER_LEN bytes)
// RC4(<JSON_SERIALIZED_OPTIONAL_HEADER>)
//
// RC4(ZLIB(block encoded stream ))
//
// OPTIONAL_FOOTER_LEN (Q)
// <JSON_SERIALIZED_OPTIONAL_FOOTER>
//
//	MANDATORY FOOTER
//	4s      QQ           Q
//
// 'TRAC<RESERVED><OPT_FOOTER_LEN>'

const CART_MAGIC = "CART"

var CART_MAGIC_BYTES = [4]byte{'C', 'A', 'R', 'T'}
var TRAC_MAGIC_BYTES = [4]byte{'T', 'R', 'A', 'C'}
var DEFAULT_ARC4_KEY = [16]byte{0x03, 0x01, 0x04, 0x01, 0x05, 0x09, 0x02, 0x06, 0x03, 0x01, 0x04, 0x01, 0x05, 0x09, 0x02, 0x06}

const CART_MAJOR_VERSION = 1

type CartHeader struct {
	Cart            [4]byte
	Version         uint16
	Reserved        uint64
	Arc4key         [16]byte
	OptHeaderLength uint64
}
type CartFooter struct {
	Trac         [4]byte
	Reserved     [2]uint64
	OptFooterLen uint64
}

var debug bool = false

/** Uncart file to bytes array, requires memory equal to uncompressed filesize.  */
func UncartBytes(path string) ([]byte, error) {
	ios, err := Uncart(path)
	if err != nil {
		return nil, err
	}
	defer ios.Close()
	var b bytes.Buffer
	_, err = b.ReadFrom(ios)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

/** Return streaming decoder for cart file (ensure you close the cart after use). */
func Uncart(path string) (io.ReadCloser, error) {
	var header CartHeader

	// simple uncart
	dataIn, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer dataIn.Close()
	// verify header
	err = binary.Read(dataIn, binary.LittleEndian, &header)
	if err != nil {
		return nil, err
	}
	if debug {
		log.Printf("header: %v", header)
	}
	if string(header.Cart[:]) != CART_MAGIC {
		return nil, errors.New("invalid header")
	}

	if header.OptHeaderLength >= 1024 {
		return nil, errors.New("optional header too large")
	}
	optional_header_enc := make([]byte, header.OptHeaderLength)
	optional_header_dec := make([]byte, header.OptHeaderLength)

	// read optional header
	// could probably just skip over it
	_, err = dataIn.Read(optional_header_enc)
	if err != nil {
		log.Printf("Failed un-carting when reading optional header.")
		return nil, err
	}
	cipher, err := rc4.NewCipher(header.Arc4key[:])
	if err != nil {
		return nil, err
	}
	cipher.XORKeyStream(optional_header_dec, optional_header_enc)
	if debug {
		log.Printf("decoded optional header: %v", string(optional_header_dec))
	}

	// memorise position of data block
	dataPos, err := dataIn.Seek(0, 1)
	if err != nil {
		return nil, err
	}

	// verify footer
	var footer CartFooter

	// skip over content and the optional footer and seek to the mandatory footer
	_, err = dataIn.Seek(-4-16-8, 2) // end of file
	if err != nil {
		return nil, err
	}

	// verify footer
	err = binary.Read(dataIn, binary.LittleEndian, &footer)
	if err != nil {
		return nil, err
	}
	if debug {
		log.Printf("footer: %v", footer)
	}
	if string(footer.Trac[:]) != "TRAC" {
		return nil, fmt.Errorf("invalid footer '%s'", string(footer.Trac[:]))
	}

	// Ignoring the optional footer here by subtracting its length from the mandatory footer

	// create streaming decoder for cart data portion
	dataIn.Close()
	dataPassInToCart, err := os.Open(path)
	if err != nil {
		log.Print("Failed to re-open file before passing it off to cart")
	}
	_, err = dataPassInToCart.Seek(dataPos, 0)
	if err != nil {
		log.Printf("Failed un-carting when creating streaming decoder.")
		dataPassInToCart.Close()
		return nil, err
	}
	cart, err := newCartReader(dataPassInToCart, header.Arc4key[:])
	if err != nil {
		return nil, err
	}
	return cart, nil
}

/** Need to make RC4Reader streamable so it can be an input to lzib which only works with streams. */
type RC4Reader struct {
	istream io.ReadCloser
	cipher  *rc4.Cipher
	ichunk  [10240]byte
}

func newRC4(istream io.ReadCloser, key []byte) (*RC4Reader, error) {
	var s RC4Reader
	var err error
	s.istream = istream
	s.cipher, err = rc4.NewCipher(key)
	if err != nil {
		return &s, err
	}
	return &s, nil
}

func (s *RC4Reader) Read(p []byte) (int, error) {
	trylen := len(p)
	if trylen > 10240 {
		trylen = 10240
	}
	readlen, err := s.istream.Read(s.ichunk[:trylen])
	s.cipher.XORKeyStream(p, s.ichunk[:readlen])
	return readlen, err
}

func (s *RC4Reader) Close() error {
	// deprecated
	// s.cipher.Reset()
	s.istream.Close()
	return nil
}

/* Stream handler for carting to a stream. */
type CartReader struct {
	rc4  *RC4Reader
	zlib io.ReadCloser
}

func newCartReader(istream io.ReadCloser, key []byte) (*CartReader, error) {
	var cart CartReader
	var err error
	rc4, err := newRC4(istream, key)
	if err != nil {
		return nil, err
	}
	// add some extra methods to resolve byte reader issue with zlib
	cart.rc4 = rc4

	cart.zlib, err = zlib.NewReader(cart.rc4)
	if err != nil {
		return nil, err
	}

	return &cart, nil
}

func (s *CartReader) Read(p []byte) (n int, err error) {
	return s.zlib.Read(p)
}

func (s *CartReader) Close() error {
	s.rc4.Close()
	s.zlib.Close()
	return nil
}

/*RC4 for creating a cart file.*/
// Write RC4 content to an output stream.
type RC4Writer struct {
	ostream io.Writer
	cipher  *rc4.Cipher
}

func newRc4Writer(ostream io.Writer, key []byte) (*RC4Writer, error) {
	cipher, err := rc4.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return &RC4Writer{
		ostream: ostream,
		cipher:  cipher,
	}, nil

}
func (s *RC4Writer) Write(p []byte) (int, error) {
	dest := make([]byte, len(p))
	s.cipher.XORKeyStream(dest, p)
	return s.ostream.Write(dest)
}

/*Cart a file and return a reader for the carted file.*/
func PackCart(istream io.Reader, ostream io.Writer) error {
	mandatoryHeader := CartHeader{
		Cart:            CART_MAGIC_BYTES,
		Version:         CART_MAJOR_VERSION,
		Reserved:        0,
		Arc4key:         DEFAULT_ARC4_KEY,
		OptHeaderLength: 0,
	}
	err := binary.Write(ostream, binary.LittleEndian, &mandatoryHeader)
	if err != nil {
		return fmt.Errorf("failed to write header for Cart with error: %v", err.Error())
	}

	rc4Writer, err := newRc4Writer(ostream, DEFAULT_ARC4_KEY[:])
	if err != nil {
		return fmt.Errorf("failed to start ARC4 encoder for Cart with error: %v", err.Error())
	}

	zlibWriter := zlib.NewWriter(rc4Writer)
	_, err = io.Copy(zlibWriter, istream)
	if err != nil {
		return fmt.Errorf("failed to pack (zlib/arc4) Cart file with error: %v", err.Error())
	}
	err = zlibWriter.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush Zlib compressor for Cart with error: %v", err.Error())
	}
	// Close zlib writer to write the zlib footer before the cart footer.
	zlibWriter.Close()

	mandatoryFooter := CartFooter{
		Trac:         TRAC_MAGIC_BYTES,
		Reserved:     [2]uint64{},
		OptFooterLen: 0,
	}
	err = binary.Write(ostream, binary.LittleEndian, &mandatoryFooter)
	if err != nil {
		return fmt.Errorf("failed to write footer for Cart with error: %v", err.Error())
	}
	return nil
}

/*Shortcut to writing a file to another location as a cart.*/
func PackCartFile(filePath string, destPath string) error {
	inFileRef, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("could not open source raw file '%s', with error %v", filePath, err.Error())
	}
	outFileRef, err := os.Open(destPath)
	if err != nil {
		return fmt.Errorf("could not open the destination cart file location '%s', with error %v", destPath, err.Error())
	}
	return PackCart(inFileRef, outFileRef)
}
