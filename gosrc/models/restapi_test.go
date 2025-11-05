package models

import (
	"testing"

	"github.com/goccy/go-json"

	"github.com/stretchr/testify/require"
)

func verifyEncodeDecode[T any](t *testing.T, pre string, expected T, post string) {
	var actual T
	err := json.Unmarshal([]byte(pre), &actual)
	require.Nil(t, err)
	require.Equal(t, expected, actual)
	crushed, err := json.Marshal(actual)
	nice_crushed := string(crushed)
	require.Nil(t, err)
	require.JSONEq(t, nice_crushed, post, "expected\n%v", nice_crushed)
}

func TestResponsePostEvent(t *testing.T) {
	verifyEncodeDecode(t,
		`{"total_ok": 4, "total_failures": 6, "failures": []}`,
		ResponsePostEvent{TotalOk: 4, TotalFailures: 6, Failures: []ResponsePostEventFailure{}},
		`{"total_ok": 4, "total_failures": 6, "failures": []}`,
	)
	// test that ok is dropped on encode if empty
	verifyEncodeDecode(t,
		`{"total_ok": 4, "total_failures": 6, "failures": [], "ok": []}`,
		ResponsePostEvent{TotalOk: 4, TotalFailures: 6, Failures: []ResponsePostEventFailure{}, Ok: []interface{}{}},
		`{"total_ok": 4, "total_failures": 6, "failures": []}`,
	)
	verifyEncodeDecode(t,
		`{"total_ok": 4, "total_failures": 6, "failures": [], "ok": [{"smeg": true}]}`,
		ResponsePostEvent{TotalOk: 4, TotalFailures: 6, Failures: []ResponsePostEventFailure{}, Ok: []interface{}{map[string]interface{}{"smeg": true}}},
		`{"total_ok": 4, "total_failures": 6, "failures": [], "ok": [{"smeg": true}]}`,
	)
}
