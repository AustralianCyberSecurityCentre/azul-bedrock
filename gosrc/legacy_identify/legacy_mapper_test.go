package identify

import (
	_ "embed"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapLegacyToNewTypes(t *testing.T) {
	legacyMapper, err := NewLegacyMapper()
	require.Nil(t, err)
	// Various options to end up with android
	require.Equal(t, legacyMapper.FindFileType("", "android"), "android/apk")
	require.Equal(t, legacyMapper.FindFileType("", "ANDROID"), "android/apk")
	require.Equal(t, legacyMapper.FindFileType("android/apk", "html"), "android/apk")

	// html code
	require.Equal(t, legacyMapper.FindFileType("", "HTML"), "code/html/component")
	require.Equal(t, legacyMapper.FindFileType("", "HtMl"), "code/html/component")
	require.Equal(t, legacyMapper.FindFileType("unknown", "HtMl"), "code/html/component")
	require.Equal(t, legacyMapper.FindFileType("code/html/component", ""), "code/html/component")

	// Garbage in garbage out as expected
	require.Equal(t, legacyMapper.FindFileType("blarghh", "HtMl"), "blarghh")
}
