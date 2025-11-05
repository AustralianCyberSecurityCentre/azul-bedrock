package embeded_files

import (
	_ "embed"
)

//go:embed identify.yaml
var RawIdentifyConfig []byte
