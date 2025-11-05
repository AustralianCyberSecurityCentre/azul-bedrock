package settings

import (
	"os"
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/stretchr/testify/require"
)

type DummySettings struct {
	Dummy1 string             `koanf:"dummy1"`
	Dummy2 HumanReadableBytes `koanf:"dummy2"`
}

const PREFIX = "DUMMY"

func TestSettingParser(t *testing.T) {
	settingKey := PREFIX + ".dummy1"
	err := os.Unsetenv(settingKey)
	require.Nil(t, err)
	defaultValue := "defaultVal"
	result := ParseSettings(DummySettings{Dummy1: defaultValue}, PREFIX, []mapstructure.DecodeHookFunc{})
	require.Equal(t, defaultValue, result.Dummy1)

	newValue := "newValueForDummy1"
	os.Setenv(settingKey, newValue)
	defer os.Unsetenv(settingKey)
	result = ParseSettings(DummySettings{Dummy1: defaultValue}, PREFIX, []mapstructure.DecodeHookFunc{})
	require.Equal(t, newValue, result.Dummy1)
}

func TestSettingParserNoPrefix(t *testing.T) {
	settingKey := "dummy1"
	err := os.Unsetenv(settingKey)
	require.Nil(t, err)
	defaultValue := "defaultVal"
	result := ParseSettings(DummySettings{Dummy1: defaultValue}, "", []mapstructure.DecodeHookFunc{})
	require.Equal(t, defaultValue, result.Dummy1)

	newValue := "newValueForDummy1"
	os.Setenv(settingKey, newValue)
	defer os.Unsetenv(settingKey)
	result = ParseSettings(DummySettings{Dummy1: defaultValue}, "", []mapstructure.DecodeHookFunc{})
	require.Equal(t, newValue, result.Dummy1)
}

func TestCustomDecoder(t *testing.T) {
	settingKey := PREFIX + ".dummy2"
	err := os.Unsetenv(settingKey)
	require.Nil(t, err)
	defaultValue := "10Mi"
	result := ParseSettings(DummySettings{Dummy2: HumanToBytesFatal(defaultValue)}, PREFIX, []mapstructure.DecodeHookFunc{HumanReadableBytesHookFunc()})
	require.Equal(t, HumanToBytesFatal(defaultValue), result.Dummy2)

	newValue := "20Mi"
	os.Setenv(settingKey, newValue)
	defer os.Unsetenv(settingKey)
	result = ParseSettings(DummySettings{Dummy2: HumanToBytesFatal(defaultValue)}, PREFIX, []mapstructure.DecodeHookFunc{HumanReadableBytesHookFunc()})
	require.Equal(t, HumanToBytesFatal(newValue), result.Dummy2)
}

type NestedDummySettings struct {
	ChildDummy DummySettings `koanf:"inner_dummy"`
}

func TestReadWithNestedSettings(t *testing.T) {
	settingKey1 := PREFIX + ".inner_dummy.dummy1"
	settingKey2 := PREFIX + ".inner_dummy.dummy2"
	newValue1 := "newValueForDummy1"
	newValue2 := "20Gi"
	os.Setenv(settingKey1, newValue1)
	os.Setenv(settingKey2, newValue2)
	result := ParseSettings(NestedDummySettings{ChildDummy: DummySettings{Dummy2: HumanToBytesFatal("10Mi")}}, PREFIX, []mapstructure.DecodeHookFunc{HumanReadableBytesHookFunc()})
	require.Equal(t, newValue1, result.ChildDummy.Dummy1)
	require.Equal(t, HumanToBytesFatal(newValue2), result.ChildDummy.Dummy2)
}
