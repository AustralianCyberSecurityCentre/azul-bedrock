package events

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsStatusTypeCompleted(t *testing.T) {
	require.True(t, IsStatusTypeCompleted(StatusTypeCompleted))
	require.True(t, IsStatusTypeCompleted(StatusTypeCompletedEmpty))
	require.True(t, IsStatusTypeCompleted(StatusTypeCompletedWithErrors))
	require.False(t, IsStatusTypeCompleted(StatusTypeOptOut))
	require.False(t, IsStatusTypeCompleted(StatusTypeDequeued))
	require.False(t, IsStatusTypeCompleted(StatusTypeHeartbeat))
	require.False(t, IsStatusTypeCompleted(StatusTypeErrorOOM))
}

func TestIsStatusTypeError(t *testing.T) {
	require.False(t, IsStatusTypeError(StatusTypeCompleted))
	require.False(t, IsStatusTypeError(StatusTypeCompletedEmpty))
	require.False(t, IsStatusTypeError(StatusTypeCompletedWithErrors))
	require.False(t, IsStatusTypeError(StatusTypeOptOut))
	require.False(t, IsStatusTypeError(StatusTypeDequeued))
	require.False(t, IsStatusTypeError(StatusTypeHeartbeat))
	require.True(t, IsStatusTypeError(StatusTypeErrorException))
	require.True(t, IsStatusTypeError(StatusTypeErrorNetwork))
	require.True(t, IsStatusTypeError(StatusTypeErrorRunner))
	require.True(t, IsStatusTypeError(StatusTypeErrorInput))
	require.True(t, IsStatusTypeError(StatusTypeErrorOutput))
	require.True(t, IsStatusTypeError(StatusTypeErrorTimeout))
	require.True(t, IsStatusTypeError(StatusTypeErrorOOM))
}

func TestIsStatusTypeProcess(t *testing.T) {
	require.False(t, IsStatusTypeProcess(StatusTypeCompleted))
	require.False(t, IsStatusTypeProcess(StatusTypeCompletedEmpty))
	require.False(t, IsStatusTypeProcess(StatusTypeCompletedWithErrors))
	require.False(t, IsStatusTypeProcess(StatusTypeOptOut))
	require.True(t, IsStatusTypeProcess(StatusTypeDequeued))
	require.True(t, IsStatusTypeProcess(StatusTypeHeartbeat))
	require.False(t, IsStatusTypeProcess(StatusTypeErrorOOM))
}
