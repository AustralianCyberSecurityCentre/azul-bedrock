/*
Enum for labeling data streams.
*/
package events

// NOTE - when updating these enums ensure you update the python equivalent in models_network.py
// Also update the IsDataLabelValid function.

type DatastreamLabel string

const (
	// Metadata from assemblyline.
	DataLabelAssemblyline DatastreamLabel = "assemblyline"
	// Full json formatted cape report.
	DataLabelCapeReport DatastreamLabel = "cape_report"
	// Content is the default for a binary that is being examined by Azul.
	DataLabelContent DatastreamLabel = "content"
	// A C# call tree holding all the functions a C# application can make.
	DataLabelCsCallTree DatastreamLabel = "cs_call_tree"
	// Decompiled C# content.
	DataLabelDecompiledCs DatastreamLabel = "decompiled_cs"
	// Decompiled C content.
	DataLabelDecompiledC DatastreamLabel = "decompiled_c"
	// Deobfuscated Javascript content.
	DataLabelDeobJs DatastreamLabel = "deob_js"
	// Words extracted from a file that can potentially be used for extracting related zip files.
	DataLabelPasswordDictionary DatastreamLabel = "password_dictionary"
	// Network PCAP capture data.
	DataLabelPcap DatastreamLabel = "pcap"
	//  Report about a binary found from an external source.
	DataLabelReport DatastreamLabel = "report"
	// A safe version of an image with all metadata stripped out, ensuring it can't execute.
	DataLabelSafePng DatastreamLabel = "safe_png"
	// A screenshot of actions taken by a malware sandbox (e.g cape).
	DataLabelScreenshot DatastreamLabel = "screenshot"
	// Test stream label type just used for testing stream files.
	DataLabelTest DatastreamLabel = "test"
	// Plain text file that provides a large amount of information about a binary.
	DataLabelText DatastreamLabel = "text"
)

func (b DatastreamLabel) Str() string {
	return string(b)
}

/*Return true if the provided data label is valid.*/
func IsDataLabelValid(dataLabel DatastreamLabel) bool {
	switch dataLabel {
	case DataLabelAssemblyline:
		fallthrough
	case DataLabelCapeReport:
		fallthrough
	case DataLabelContent:
		fallthrough
	case DataLabelCsCallTree:
		fallthrough
	case DataLabelDecompiledCs:
		fallthrough
	case DataLabelDecompiledC:
		fallthrough
	case DataLabelDeobJs:
		fallthrough
	case DataLabelPasswordDictionary:
		fallthrough
	case DataLabelPcap:
		fallthrough
	case DataLabelReport:
		fallthrough
	case DataLabelSafePng:
		fallthrough
	case DataLabelScreenshot:
		fallthrough
	case DataLabelTest:
		fallthrough
	case DataLabelText:
		return true
	default:
		return false
	}
}
