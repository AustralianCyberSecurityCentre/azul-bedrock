// wrap libmagic for easier use in identify module
package identify

import (
	"strings"

	st "github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/settings"
	"github.com/rakyll/magicmime"
)

// including text formats is ~10x more expensive (uncomment for speed up) //| "magicmime.MAGIC_NO_CHECK_TEXT"
const DefaultMimeTypeFlags = magicmime.MAGIC_MIME_TYPE | magicmime.MAGIC_ERROR | magicmime.MAGIC_RAW | magicmime.MAGIC_CONTINUE
const DefaultMimeMagicFlags = magicmime.MAGIC_ERROR | magicmime.MAGIC_RAW | magicmime.MAGIC_CONTINUE

const MaxRegen = 100

type MagicWrap struct {
	mime       *magicmime.Decoder
	magic      *magicmime.Decoder
	magicCount int
	mimeCount  int
}

func NewMagicWrap() *MagicWrap {
	mw := MagicWrap{}
	mw.regen()
	return &mw
}

// regen regenerates the decoders, done to avoid a memory leak in the mime/magic library.
func (mw *MagicWrap) regen() {
	mw.magicCount = 0
	mw.mimeCount = 0
	mime, err := magicmime.NewDecoder(DefaultMimeTypeFlags)
	if err == nil {
		if mw.mime != nil {
			mw.mime.Close()
		}
		mw.mime = mime
	} else {
		st.Logger.Warn().Msg("identify failed to create a replacement mime.")
	}
	magic, err := magicmime.NewDecoder(DefaultMimeMagicFlags)
	if err == nil {
		if mw.magic != nil {
			mw.magic.Close()
		}
		mw.magic = magic
	} else {
		st.Logger.Warn().Msg("identify failed to create a replacement magic.")
	}
}

// split on newline and remove bizzare prefix dash from non-first lines
func fixLibmagicOutput(raw string) []string {
	rets := strings.Split(raw, "\n")
	for i := range rets {
		rets[i] = strings.TrimPrefix(rets[i], "- ")
		rets[i] = strings.TrimSpace(rets[i])
	}
	return rets
}

// Calculate the mime type from a buffer of bytes.
func (mw *MagicWrap) CalcMimesFromBytes(buf []byte) []string {
	mw.mimeCount += 1
	if mw.mimeCount > MaxRegen {
		// Regenerate mime periodically to avoid memory leak
		mw.regen()
	}
	raw, err := mw.mime.TypeByBuffer(buf)
	if err != nil {
		raw = "error - " + err.Error()
	}

	return fixLibmagicOutput(raw)
}

// Calculate the mime type from a file path.
func (mw *MagicWrap) CalcMimesFromPath(filePath string) []string {
	mw.mimeCount += 1
	if mw.mimeCount > MaxRegen {
		// Regenerate mime periodically to avoid memory leak
		mw.regen()
	}
	raw, err := mw.mime.TypeByFile(filePath)
	if err != nil {
		raw = "error - " + err.Error()
	}

	return fixLibmagicOutput(raw)
}

// Calculate the file magic from a buffer of bytes.
func (mw *MagicWrap) CalcMagicsFromBytes(buf []byte) []string {
	mw.magicCount += 1
	if mw.magicCount > MaxRegen {
		// Regenerate magic periodically to avoid memory leak
		mw.regen()
	}
	raw, err := mw.magic.TypeByBuffer(buf)
	if err != nil {
		raw = "error - " + err.Error()
	}

	return fixLibmagicOutput(raw)
}

// Calculate the file magic from a filepath.
func (mw *MagicWrap) CalcMagicsFromPath(filePath string) []string {
	mw.magicCount += 1
	if mw.magicCount > MaxRegen {
		// Regenerate magic periodically to avoid memory leak
		mw.regen()
	}
	raw, err := mw.magic.TypeByFile(filePath)
	if err != nil {
		raw = "error - " + err.Error()
	}
	return fixLibmagicOutput(raw)
}
