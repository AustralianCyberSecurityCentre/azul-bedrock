package poststreams

// parameters available for getevents restapi endpoint on dispatcher
const (
	SkipIdentify   = "skip-identify"   // skips slow identification code - must also supply sha256
	ExpectedSha256 = "expected-sha256" // if supplied, this will be cross-checked with the server-side calculated sha256
)
