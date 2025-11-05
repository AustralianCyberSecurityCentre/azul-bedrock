package events

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

func (b FeatureType) Str() string {
	return string(b)
}
