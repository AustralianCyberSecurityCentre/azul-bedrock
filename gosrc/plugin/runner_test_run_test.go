package plugin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/events"
	"github.com/stretchr/testify/require"
)

func defaultRunTestOption() *RunTestOptions {
	return &RunTestOptions{
		ContentFileBytes:            []byte("Random test file"),
		DisableUncartingContentFile: true,
	}

}

func TestEmptyPlugin(t *testing.T) {
	pr := NewPluginRunner(NewDummyPlugin(nil))
	result := pr.RunTest(t, defaultRunTestOption(), "Benign text file.")
	result.AssertJobResultEqual(t, &TestJobResult{
		Status: "completed-empty",
		Events: []TestJobEvent{
			{},
		},
	})
}

func TestPluginAddFeatures(t *testing.T) {
	pr := NewPluginRunner(NewDummyPlugin(
		func(ctx context.Context, j *Job, inputUtils *PluginInputUtils) *PluginError {
			err := j.AddFeature("dummy", "value1")
			require.Nil(t, err)
			err = j.AddFeature("dummy", "value2")
			require.Nil(t, err)
			err = j.AddFeature("dummyInt", "10")
			require.Nil(t, err)
			return nil
		},
	))
	result := pr.RunTest(t, defaultRunTestOption(), "Benign text file.")

	result.AssertJobResultEqual(t, &TestJobResult{
		Status: "completed",
		Events: []TestJobEvent{
			{
				Features: map[string][]TestBinaryEntityFeature{
					"dummy": {
						{Value: "value1"},
						{Value: "value2"},
					},
					"dummyInt": {{Value: "10"}},
				},
			},
		},
	})
	// Ensure flipping the order of feature values doesn't cause a failure.
	result.AssertJobResultEqual(t, &TestJobResult{
		Status: "completed",
		Events: []TestJobEvent{
			{
				Features: map[string][]TestBinaryEntityFeature{
					"dummy": {
						{Value: "value2"},
						{Value: "value1"},
					},
					"dummyInt": {{Value: "10"}},
				},
			},
		},
	})

}

func TestPluginAddBadFeature(t *testing.T) {
	pr := NewPluginRunner(NewDummyPlugin(
		func(ctx context.Context, j *Job, inputUtils *PluginInputUtils) *PluginError {
			err := j.AddFeature("dummyInt", "abcdef")
			require.Nil(t, err)
			return nil
		},
	))
	result := pr.RunTest(t, defaultRunTestOption(), "Benign text file.")
	result.AssertJobResultEqual(t, &TestJobResult{
		Status:  "error-runner",
		Message: "Error occurred when attempting to process features. with error integer dummyInt must be a valid int value was abcdef and couldn't be parsed with error strconv.ParseInt: parsing \"abcdef\": invalid syntax",
	})
}

func TestPluginAddBadDateFeature(t *testing.T) {
	pr := NewPluginRunner(NewDummyPlugin(
		func(ctx context.Context, j *Job, inputUtils *PluginInputUtils) *PluginError {
			err := j.AddFeature("dummyDate", "abcdef")
			require.Nil(t, err)
			return nil
		},
	))
	result := pr.RunTest(t, defaultRunTestOption(), "Benign text file.")
	result.AssertJobResultEqual(t, &TestJobResult{
		Status:  "error-runner",
		Message: "Error occurred when attempting to process features. with error dateTime dummyDate must be a valid RFC3339 date value was abcdef and couldn't be parsed with error parsing time \"abcdef\" as \"2006-01-02T15:04:05Z07:00\": cannot parse \"abcdef\" as \"2006\"",
	})
}

func TestPluginAddChild(t *testing.T) {
	pr := NewPluginRunner(NewDummyPlugin(
		func(ctx context.Context, j *Job, inputUtils *PluginInputUtils) *PluginError {
			j.AddChildBytes([]byte("child file bytes"), map[string]string{"child": "extracted"})
			return nil
		},
	))
	result := pr.RunTest(t, defaultRunTestOption(), "Benign text file.")
	result.AssertJobResultEqual(t, &TestJobResult{
		Status: "completed-empty",
		Events: []TestJobEvent{
			{
				ChildrenSha256: []string{
					"129413404c6ecff77e7a32fbc886bec68dce30d605c06f534743d2218ad3f508",
				},
			},
			{
				AugmentedStreams: []ResultStream{
					{
						Relationship: map[string]string{
							"child": "extracted",
						},
						Label:  "content",
						Sha256: "129413404c6ecff77e7a32fbc886bec68dce30d605c06f534743d2218ad3f508",
						Size:   16,
					},
				},
			},
		},
	})
}

func TestPluginAddAugmented(t *testing.T) {
	pr := NewPluginRunner(NewDummyPlugin(
		func(ctx context.Context, j *Job, inputUtils *PluginInputUtils) *PluginError {
			j.AddAugmentedBytes([]byte("augmented file bytes"), events.DataLabelTest)
			return nil
		},
	))
	result := pr.RunTest(t, defaultRunTestOption(), "Benign text file.")
	result.AssertJobResultEqual(t, &TestJobResult{
		Status: "completed",
		Events: []TestJobEvent{
			{
				AugmentedStreams: []ResultStream{
					{
						Label:  "test",
						Sha256: "f7c863453018acd13d9b7fb50189a07c9fbca8333dbf206ac681f51dc2efac5a",
						Size:   20,
					},
				},
			},
		},
	})
}

func TestPluginAddInfo(t *testing.T) {
	pr := NewPluginRunner(NewDummyPlugin(
		func(ctx context.Context, j *Job, inputUtils *PluginInputUtils) *PluginError {
			marshalledinfo, err := json.Marshal(&map[string]string{"info": "infoValues"})
			require.Nil(t, err)
			j.AddInfo(marshalledinfo)
			return nil
		},
	))
	result := pr.RunTest(t, defaultRunTestOption(), "Benign text file.")
	result.AssertJobResultEqual(t, &TestJobResult{
		Status: "completed",
		Events: []TestJobEvent{
			{
				Info: "{\"info\":\"infoValues\"}",
			},
		},
	})
}

func TestPluginAddMixed(t *testing.T) {
	pr := NewPluginRunner(NewDummyPlugin(
		func(ctx context.Context, j *Job, inputUtils *PluginInputUtils) *PluginError {
			pluginError := j.AddFeature("dummy", "hello!")
			require.Nil(t, pluginError)
			pluginError = j.AddFeature("dummyInt", "123")
			require.Nil(t, pluginError)
			pluginError = j.AddFeature("dummyDate", "2025-05-15T15:30:05Z")
			require.Nil(t, pluginError)
			j.AddChildBytes([]byte("child file bytes"), map[string]string{"child": "extracted"})
			j.AddAugmentedBytes([]byte("augmented file bytes"), events.DataLabelTest)
			marshalledInfo, err := json.Marshal(&map[string]string{"info": "infoValues"})
			require.Nil(t, err)
			j.AddInfo(marshalledInfo)

			return nil
		},
	))
	result := pr.RunTest(t, defaultRunTestOption(), "Benign text file.")
	result.AssertJobResultEqual(t, &TestJobResult{
		Status: "completed",
		Events: []TestJobEvent{
			{
				ChildrenSha256: []string{
					"129413404c6ecff77e7a32fbc886bec68dce30d605c06f534743d2218ad3f508",
				},
				Features: map[string][]TestBinaryEntityFeature{
					"dummy":     {{Value: "hello!"}},
					"dummyDate": {{Value: "2025-05-15T15:30:05Z"}},
					"dummyInt":  {{Value: "123"}},
				},
				Info: "{\"info\":\"infoValues\"}",
				AugmentedStreams: []ResultStream{
					{
						Label:  "test",
						Sha256: "f7c863453018acd13d9b7fb50189a07c9fbca8333dbf206ac681f51dc2efac5a",
						Size:   20,
					},
				},
			},
			{
				AugmentedStreams: []ResultStream{
					{
						Relationship: map[string]string{
							"child": "extracted",
						},
						Label:  "content",
						Sha256: "129413404c6ecff77e7a32fbc886bec68dce30d605c06f534743d2218ad3f508",
						Size:   16,
					},
				},
			},
		},
	})

}
