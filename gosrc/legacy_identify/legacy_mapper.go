package identify

import (
	_ "embed"
	"strings"

	embeded_files "github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc"
	"gopkg.in/yaml.v3"
)

type IdMapping struct {
	Id        string
	Legacy    string
	Extension string
}

// Mapper for mapping legacy (virustotal file types) to new file types.
type VirusTotalAndLegacyMapper struct {
	Id_Mappings       []IdMapping
	LegacyToIdMapping map[string]IdMapping
}

func NewLegacyMapper() (*VirusTotalAndLegacyMapper, error) {
	// Legacy mapper allows you to map legacy types to new types.
	cfg := VirusTotalAndLegacyMapper{}
	err := yaml.Unmarshal(embeded_files.RawIdentifyConfig, &cfg)
	if err != nil {
		return nil, err
	}
	// Map all the Ids to their id types.
	cfg.LegacyToIdMapping = make(map[string]IdMapping)
	for _, id := range cfg.Id_Mappings {
		cfg.LegacyToIdMapping[strings.ToLower(id.Legacy)] = id
		cfg.LegacyToIdMapping[id.Legacy] = id
	}
	return &cfg, err
}

// Given the current file type is unknown or empty derive the file type from the legacy file type.
func (vtlm *VirusTotalAndLegacyMapper) FindFileType(currentFileType string, legacyFileType string) string {
	if len(currentFileType) > 0 && currentFileType != "unknown" {
		return currentFileType
	}
	if newFileType, ok := vtlm.LegacyToIdMapping[strings.ToLower(legacyFileType)]; ok {
		return newFileType.Id
	}
	return "unknown"
}
