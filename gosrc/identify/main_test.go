package identify

import (
	"log"
	"os"
	"testing"

	testutils "github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/testutils"
)

var cfg *Identify
var fileManager *testutils.FileManager

func TestMain(m *testing.M) {
	var err error
	fileManager, err = testutils.NewFileManager()
	if err != nil {
		log.Fatalf("Failed to setup file manager with error %v", err)
	}
	cfg, err = NewIdentify()

	// Force the use of dosIdent on all pe32 files to ensure it works.
	for i, rule := range cfg.Refine_Rules {
		if rule.Function_Name == "dos_ident" {
			cfg.Refine_Rules[i].Trigger_On = append(rule.Trigger_On, "executable/windows/pe32")
		}
	}

	if err != nil {
		log.Fatalf("Identify failed to init %v", err)
	}
	os.Exit(m.Run())
}
