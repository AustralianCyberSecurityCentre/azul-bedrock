package store

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestS3MemeProviderStandardTests(t *testing.T) {
	s3 := NewStoreMem()
	StoreImplementationBaseTests(t, s3)
	StoreImplementationListBaseTests(t, s3)
}

func TestS3MemProvider(t *testing.T) {
	s3 := NewStoreMem()
	var has bool
	var err error
	var data []byte

	// not exist
	has, err = s3.Exists("", "", "a")
	require.Nil(t, err)
	require.Equal(t, has, false)
	has, err = s3.Exists("", "", "b")
	require.Nil(t, err)
	require.Equal(t, has, false)
	has, err = s3.Exists("", "", "c")
	require.Nil(t, err)
	require.Equal(t, has, false)

	// put
	inData := []byte("apple")
	reader := bytes.NewReader(inData)
	readCloser := io.NopCloser(reader)
	err = s3.Put("", "", "a", readCloser, int64(len(inData)))
	require.Nil(t, err)

	inData = []byte("banana")
	reader = bytes.NewReader(inData)
	readCloser = io.NopCloser(reader)
	err = s3.Put("", "", "b", readCloser, int64(len(inData)))
	require.Nil(t, err)

	// exist
	has, err = s3.Exists("", "", "a")
	require.Nil(t, err)
	require.Equal(t, has, true)
	has, err = s3.Exists("", "", "b")
	require.Nil(t, err)
	require.Equal(t, has, true)
	has, err = s3.Exists("", "", "c")
	require.Nil(t, err)
	require.Equal(t, has, false)

	// fetch
	// Apple
	dataSlice, err := s3.Fetch("", "", "a")
	require.Nil(t, err)
	data, err = io.ReadAll(dataSlice.DataReader)
	require.Nil(t, err)
	require.Equal(t, string(data), "apple")
	// Banana
	dataSlice, err = s3.Fetch("", "", "b")
	require.Nil(t, err)
	data, err = io.ReadAll(dataSlice.DataReader)
	require.Nil(t, err)
	require.Equal(t, string(data), "banana")
	// error case
	_, err = s3.Fetch("", "", "c")
	require.NotNil(t, err)

	// delete
	ok, err := s3.Delete("", "", "a")
	require.Nil(t, err)
	require.Equal(t, ok, true)
	ok, err = s3.Delete("", "", "c")
	require.NotNil(t, err)
	require.Equal(t, ok, false)

	// put in unsorted order
	for i := range 10 {
		txt := fmt.Sprintf("%d", i)
		inData := []byte("z" + txt)
		reader := bytes.NewReader(inData)
		readCloser := io.NopCloser(reader)
		err = s3.Put("root", "pathway", "z"+txt, readCloser, int64(len(inData)))
		require.Nil(t, err)

		inData = []byte("n" + txt)
		reader = bytes.NewReader(inData)
		readCloser = io.NopCloser(reader)
		err = s3.Put("root", "pathway", "n"+txt, readCloser, int64(len(inData)))
		require.Nil(t, err)
	}

	// list all
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	ch := s3.List(ctx, "", "")
	require.NotNil(t, ch)
	listed := []string{}
	for obj := range ch {
		listed = append(listed, createIdPath(obj.Source, obj.Label, obj.Id))
	}
	require.Equal(t, 21, len(listed))

	// list start after
	ch = s3.List(ctx, "", "root/pathway/n9")
	require.NotNil(t, ch)
	listed = []string{}
	for obj := range ch {
		listed = append(listed, createIdPath(obj.Source, obj.Label, obj.Id))
	}
	require.Equal(t, 10, len(listed))

	// list prefix
	ch = s3.List(ctx, "root/pathway/n", "")
	require.NotNil(t, ch)
	listed = []string{}
	for obj := range ch {
		listed = append(listed, createIdPath(obj.Source, obj.Label, obj.Id))
	}
	require.Equal(t, 10, len(listed))

}
