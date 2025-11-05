# Go Runner

Go Runner is the golang version of azul-runner.
It is designed as a common framework for testing and running golang plugins.

It doesn't have all of the more complex features of `azul-runner` such as memory tracking (and automatic shutdown and reboot).
But it does have the core functionality and test framework that azul-runner has.

## Typical Plugin

There are two plugins that currently use go-runner these are `azul-plugin-goinfo` and `azul-plugin-entropy`.
They are good examples of the usage of golang plugins.

To create a Golang plugin the bedrock packages `bedrock/gosrc/plugin` and `bedrock/gosrc/plugin` must be imported.

Then create a `Plugin` that is compliant with the go interface.

```go
type Plugin interface {
    /*Getters used by the runner framework.*/
    GetName() string
    GetVersion() string
    GetDescription() string
    GetFeatures() []events.PluginEntityFeature
    GetDefaultSettings() *PluginSettings
    /*Core execute method used to execute events provided by the PluginRunner framework.
    Expected errors are opt-out, timeout and generic errors.
    */
    Execute(context context.Context, job *Job, inputUtils *PluginInputUtils) *PluginError
}
```

Once the `Plugin` has been created simply add the following to the bottom of the `main.go` file.

```go
func main() {
    pr := plugin.NewPluginRunner(&MyPlugin{})
    pr.Run()
}
```

## Plugin Interface

The functions `GetName`, `GetVersion`, `GetDescription` are static functions that simply return a string.
The name should be the name of the plugin without the `azul-plugin-` prefix.
The version is typically set as the date in the format `YYYY.MM.DD`.
The Description is a simple description of what the plugin does.

Implementing `GetFeatures` is typically done after you know what features your plugin will be emitting.
The list of `PluginEntitiyFeature` types should just have the name of the feature, a description and the type.

Legal types are definied by the FeatureType enum and include:

```go
type FeatureType string

const (
    FeatureInteger  FeatureType = "integer"
    FeatureFloat    FeatureType = "float"
    FeatureString   FeatureType = "string"
    FeatureBinary   FeatureType = "binary"
    FeatureDatetime FeatureType = "datetime"
    FeatureFilepath FeatureType = "filepath"
    FeatureUri      FeatureType = "uri"
)
```

Typically `GetDefaultSettings` will be similar to this:

```go
func (ep *MyPlugin) GetDefaultSettings() *plugin.PluginSettings {
defaultSettings := plugin.NewDefaultPluginSettings().WithContentFilterDataTypes([]string{
        // Windows exe
        "executable/windows/",
        // Non windows exe
        "executable/dll32",
        "executable/pe32",
        // Linux elf
        "executable/linux/elf64",
        "executable/linux/elf32",
        "executable/mach-o",
    })
}
```

Where the provided file types are the type of files that the plugin accepts.
There are other options you can have ontop of the default settings as well.

The execute method `Execute(context context.Context, job *Job, inputUtils *PluginInputUtils) *PluginError` simplest implementation
is simply to return nil and do nothing else.
Returning nil at the end is the expected behaviour and if no features have been added a COMPLETED_EMPTY event would be sent to dispatcher.

For details on how to add features and child binaries refer to the next section.

## Plugin Execute

Plugin execute has the following inputs:

- `context` - a context that will be cancelled when the plugin times out.
- `job` - details of the BinaryEvent from dispatcher ready for processing, also where output data is added.
- `inputUtils` - holds the plugin settings for use within the plugin and a logger to allow for log handling.

### Outputting Features

Plugins can output features, extracted files and augmented files.
Adding any of the above is done by calling functions on the `*Job` object passed into the `Execute` method.

To Add basic features the `AddFeature` command is used e.g:

```go
Execute(context context.Context, job *Job, inputUtils *PluginInputUtils) *PluginError{
    err = job.AddFeature("FeatureName", "FeatureValue")
    if err !=nil {
        return err
    }
    return nil
}
```

If the new feature has a `label`, `offset` or `size` associated with it, these can be added with the `AddFeatureWithExtra` method.

Note:

- `Label` - Label is extra information about a Value (e.g Feature `av_verdict`, Value `malicious`, Label `8/10` (where 8/10 is how confident the AV was the file is malicious))
- `Offset` - Offset of the feature relative to the start of a file. (e.g the feature value is the `.data` section of a PE and Offset indicates when the `.data` section starts)
- `Size` - Length in bytes of the feature. (e.g the feature value is the `.data` section of a PE and Size indicates how long the `.data` section is)

```go
job.AddFeatureWithExtra("FeatureName", "FeatureValue", &AddFeatureOptions{Label: "LabelValue", Offset: 123, Size: 9000})
```

#### Adding Info

Info is Json data that can be added to a plugin result only once per plugin.
Attempting to set info more than once will override the previous value.

New kinds of Info require changes to the Azul UI to render them appropriately.

Info is used for storing large amounts of data that can't be stored in a feature.
There are a few main uses of Info including storing full file Entropy information for a file.
Cape Sandbox results and Yara match information.

```go
info, normalErr := json.Marshal(&map[string]string{"info": "infoValues"})
if err != nil {
    // Returning a plugin error if json marshalling fails
    return plugin.NewPluginError(plugin.ErrorException, "Info Parsing Error", fmt.Sprintf("Failed to parse info with error %v", err))
}
// Adding the Info
job.rootEvent.AddInfo(info)
```

#### Adding Files

There are two kinds of files to add to a result `child` and `augmented`.

Child files are files that are derived from the original file that are standalone content.
E.g a file extracted from a zip file, a PE hidden after 1000 null bytes.

```go
// Add Raw bytes as child
childEvent := job.AddChildBytes([]byte("abcdef child binary!"), map[string]string{"child": "extracted"})
// Add Child by path
childEvent, err := job.AddChild("/tmp/path/to/file", map[string]string{"child": "extracted"})
// Handle any errors
if err != nil {
    // Returning a plugin error if reading the child file failed.
    return plugin.NewPluginError(plugin.ErrorException, "Failed to Read Child Error", fmt.Sprintf("Failed to read the child file with error %v", err))
}
// After adding a child the returned event is effectively another job and can have features, children and augmented streams added to it.
childEvent.AddFeature("FeatureName", "FeatureValue")
```

Augmented files are files that are modified version of the existing file that make understanding the original document easier. But aren't standalone documents that should be analysed by plugins.
E.g a safe PNG derived from a malicious image, decompiled code.

```go
job.AddAugmentedBytes([]byte("Augmented stream!"), events.DataLabelTest)
err := job.AddAugmented("/tmp/path/to/file"), events.DataLabelTest)
// Handle any errors
if err != nil {
    // Returning a plugin error if reading the augmented file failed.
    return plugin.NewPluginError(plugin.ErrorException, "Failed to Read Augmented Error", fmt.Sprintf("Failed to read the augmented file with error %v", err))
}
```

### Outputting Errors

If a plugin fails to run for any reason and needs to exit as either and Opt-Out or error state a non-nil value needs to be returned from the Execute method.

To Opt-Out return the following error type:

```go
return plugin.NewPluginOptOut("Opt-out reason (file is not a PE and cannot be analysed!)")
```

To return an error use this method (noting that ErrorException is not the only available error type.)

```go
return plugin.NewPluginError(plugin.ErrorException, "Error title", "Error details")
```

## Plugin Testing

Once a plugin is outputting features it needs to be tested to verify that given a known input file
appropriate features and files are outputted.

To make this simple go-runner comes with it's own test methods.
The method used is `RunTest` and then the result of the test can be compared to an expected `TestJobResult` with the method `AssertJobResultEqual`.

```go
func TestDummyTest(t *testing.T) {
    // Create an instance of the plugin to test.
    pr := plugin.NewPluginRunner(&MyPlugin{})
    // Run the exectue method and get the associated test result.
    // RunTestOptions can be used to download a file or source a local file as required.
    result := pr.RunTest(t, &plugin.RunTestOptions{DownloadSha256: "<sha256>"}, "Description of the provided file")
    // Verify the result is equal to what was expected.
    result.AssertJobResultEqual(t, &TestJobResult{
        Status:  "error-runner",
        Message: "Error occurred when attempting to process features. with error ...",
    })
}
```

Check the logged output for any failures as the failure message will contain the actual `TestJobResult` if the expected value didn't match.

This allows you to take the actual value and paste it into the expected result to set the appropriate value.
Always verify that the actual output is what is expected and there isn't a bug with the plugin.
