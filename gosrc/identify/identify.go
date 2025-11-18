package identify

import (
	"archive/zip"
	"bufio"
	"bytes"
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"

	embeded_files "github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc"
	st "github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/settings"
	"github.com/hillu/go-yara/v4"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

// Max number of seconds to allow yara to scan a file for.
const MAX_YARA_SCAN_TIME_SECONDS = 10

var re_standard_ascii = regexp.MustCompile("[[:^ascii:]]")

//go:embed trusted_mime.yaml
var raw_trusted_mimes []byte

//go:embed yara_rules.yar
var raw_yara_rules []byte

var ErrMismatch = errors.New("failed to match id")

const POINTS_STRONG = 15
const POINTS_WEAK = 1
const MIN_POINTS = 15

// Max size of the start of the file to buffer for identification (32kB)
const MAX_INDICATOR_BUFFERED_BYTES_SIZE = 32000

/* return minimum of two numbers*/
func min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

type Identified struct {
	FileFormat       string
	FileFormatLegacy string
	FileExtension    string
	Mimes            []string
	Magics           []string
	Mime             string
	Magic            string
}

type RefinedRule struct {
	Function_Name      string
	Trigger_On         []string
	Run_On_func_Output bool
}

type IdMapping struct {
	Id        string
	Legacy    string
	Extension string
}

type refineFunction func([]byte, string, string, string, string, string) string

type Identify struct {
	Version         uint32
	mw              *MagicWrap
	Id_Mappings     []IdMapping
	Id_Mappings_Map map[string]IdMapping

	Yara_Scanner *yara.Scanner

	Rules []struct {
		Id        string
		Legacy    string
		Extension string
		Magic     string
		MagicC    *regexp.Regexp
		Mime      string
		MimeC     *regexp.Regexp
	}

	Refine_Rules []RefinedRule
	Indicators   []struct {
		Id         string
		Trigger_On []string
		RE_Strong  []string
		RE_StrongC []*regexp.Regexp
		RE_Weak    []string
		RE_WeakC   []*regexp.Regexp
	}
	Trusted_Mimes map[string]string

	RefineRulesMapping map[string]refineFunction
}

func NewIdentify() (*Identify, error) {
	cfg := Identify{
		mw: NewMagicWrap(),
	}
	err := yaml.Unmarshal(embeded_files.RawIdentifyConfig, &cfg)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(raw_trusted_mimes, &cfg)
	if err != nil {
		return nil, err
	}

	// Map all the Ids to their id types.
	cfg.Id_Mappings_Map = make(map[string]IdMapping)
	for _, id := range cfg.Id_Mappings {
		cfg.Id_Mappings_Map[id.Id] = id
	}

	// compile regexes
	for x := range cfg.Rules {
		cfg.Rules[x].MagicC = regexp.MustCompile(strings.ToLower(cfg.Rules[x].Magic))
		cfg.Rules[x].MimeC = regexp.MustCompile(strings.ToLower(cfg.Rules[x].Mime))
	}
	for x := range cfg.Indicators {
		for _, rawre := range cfg.Indicators[x].RE_Strong {
			cfg.Indicators[x].RE_StrongC = append(cfg.Indicators[x].RE_StrongC, regexp.MustCompile(rawre))
		}
		for _, rawre := range cfg.Indicators[x].RE_Weak {
			cfg.Indicators[x].RE_WeakC = append(cfg.Indicators[x].RE_WeakC, regexp.MustCompile(rawre))
		}
	}

	// Compile Yara
	compiler, err := yara.NewCompiler()
	if err != nil {
		st.Logger.Fatal().Err(err).Msg("Couldn't build a yara compiler skipping yara identification.")
	}
	_ = compiler.DefineVariable("mime", "default-mime")
	_ = compiler.DefineVariable("magic", "default-magic")
	_ = compiler.DefineVariable("type", "default-type")

	err = compiler.AddString(string(raw_yara_rules[:]), "default")
	if err != nil {
		for _, errs := range compiler.Errors {
			st.Logger.Error().Msgf("Failed to compile yara rule '%s' with error %s", errs.Rule, errs.Text)
		}
		st.Logger.Fatal().Err(err).Msg("Yara compiler experienced error can't identify.")
	}
	rules, err := compiler.GetRules()
	if err != nil {
		st.Logger.Fatal().Err(err).Msg("Yara compiler couldn't compile rules can't identify.")
	}
	scanner, err := yara.NewScanner(rules)
	if err != nil {
		st.Logger.Fatal().Err(err).Msg("Yara Scanner - Couldn't create a scanner can't identify.")
	}
	cfg.Yara_Scanner = scanner
	cfg.Yara_Scanner.SetTimeout(30 * time.Second)
	cfg.Yara_Scanner.SetFlags(yara.ScanFlagsFastMode)

	// Return config.
	cfg.RefineRulesMapping = map[string]refineFunction{
		"yara_ident": cfg.yaraIdent,
		"zip_ident":  zipIdent,
		"dos_ident":  dosIdent,
		"cart_ident": cartIdent,
		"pdf_ident":  pdfIdent,
	}
	return &cfg, nil
}

/* Apply identification rules, returns the first match found */
func (cfg *Identify) applyRules(magics, mimes []string) (string, string, string) {
	for _, elem := range cfg.Rules {
		if len(elem.Magic) > 0 {
			for _, magic := range magics {
				lc := strings.ToLower(magic)
				if elem.MagicC.MatchString(lc) {
					return elem.Id, elem.Legacy, elem.Extension
				}
			}
		}
		if len(elem.Mime) > 0 {
			for _, mime := range mimes {
				lc := strings.ToLower(mime)
				if elem.MimeC.MatchString(lc) {
					return elem.Id, elem.Legacy, elem.Extension
				}
			}
		}
	}
	return "", "", ""
}

/* Apply identification indicators, uses a point system to guess file type. */
func (cfg *Identify) applyIndicators(id string, bufferedContent []byte) string {
	ret_id := id

	content_length := len(bufferedContent)
	max_read_head := min(content_length, MAX_INDICATOR_BUFFERED_BYTES_SIZE)

	best_score := 0
	best_id := ""
	for _, indicator := range cfg.Indicators {
		tally := 0
		if !slices.Contains(indicator.Trigger_On, id) {
			// if not a candidate, do not inspect
			continue
		}
		for _, re := range indicator.RE_StrongC {
			tally += len(re.FindAll(bufferedContent[:max_read_head], 100)) * POINTS_STRONG
		}
		for _, re := range indicator.RE_WeakC {
			tally += len(re.FindAll(bufferedContent[:max_read_head], 100)) * POINTS_WEAK
		}
		if best_score < tally {
			best_id = indicator.Id
			best_score = tally
		}

	}

	if best_score >= MIN_POINTS {
		ret_id = best_id
	}

	return ret_id
}

/** Identify returns the standardised file type descriptor for the given input. */
func (cfg *Identify) Find(contentPath string) (Identified, error) {
	rawContent, err := os.Open(contentPath)
	if err != nil {
		return Identified{
			Mime:             "inode/x-empty",
			Magic:            "empty",
			FileFormatLegacy: "Data",
			FileFormat:       "unknown",
			FileExtension:    "",
		}, err
	}
	defer rawContent.Close()

	b := make([]byte, MAX_INDICATOR_BUFFERED_BYTES_SIZE)
	bytesRead, err := rawContent.Read(b)

	if err != nil || bytesRead <= 0 {
		return Identified{
			Mime:             "inode/x-empty",
			Magic:            "empty",
			FileFormatLegacy: "Data",
			FileFormat:       "unknown",
			FileExtension:    "",
		}, nil
	}
	bufferedContent := b[:bytesRead]

	magics := cfg.mw.CalcMagicsFromPath(contentPath)
	mimes := cfg.mw.CalcMimesFromPath(contentPath)
	// identify based on magic
	fileTypeAlWithOverrides, overrideIdLegacy, overrideExtension := cfg.applyRules(magics, mimes)
	fileTypeAl := fileTypeAlWithOverrides
	// Find fist good mime type and use it.
	bestMime := ""
	for _, mime := range mimes {
		if mime != "" {
			bestMime = mime
			break
		}
	}

	// Try to identify off mime if magic couldn't find the type
	if fileTypeAl == "unknown" || fileTypeAl == "text/plain" {
		for _, mime := range mimes {
			mime = dotDump(mime)
			newAlType, success := cfg.Trusted_Mimes[mime]
			if success {
				fileTypeAl = newAlType
				break
			}
		}
	}

	// refined identity based on content
	fileTypeAl = cfg.applyIndicators(fileTypeAl, bufferedContent)

	fnRan := false
	for _, refFn := range cfg.Refine_Rules {
		// Skip all functions that don't run on other functions output once a function has been run.
		if fnRan && !refFn.Run_On_func_Output {
			continue
		}

		// If the filetype is listed in the function trigger values run it.
		for _, trigger_on_val := range refFn.Trigger_On {
			if fileTypeAl == trigger_on_val {
				fnRan = true
				fileTypeAl = cfg.RefineRulesMapping[refFn.Function_Name](bufferedContent, contentPath, magics[0], bestMime, fileTypeAl, fileTypeAl)
				break
			}
		}
	}

	legacy := ""
	extension := ""
	// Map to extensions and legacy.
	if fileTypeAlWithOverrides == fileTypeAl && overrideIdLegacy != "" {
		legacy = overrideIdLegacy
		extension = overrideExtension
	} else {
		foundType, success := cfg.Id_Mappings_Map[fileTypeAl]
		if !success {
			fmt.Println("PRINTING MAPPINGS")
			for _, val := range cfg.Id_Mappings_Map {
				fmt.Println(val)
			}
			fmt.Printf("Error: the assemblyline type '%v'  doesn't have a mapping and should", fileTypeAl)
			panic(fmt.Sprintf("Error: the assemblyline type '%v'  doesn't have a mapping and should", fileTypeAl))
		}

		legacy = foundType.Legacy
		extension = foundType.Extension
	}

	// identify based on filepath
	// i.e. perhaps for code language identification
	ret := Identified{
		// drop non ascii characters from primary magic/mime
		Mime:             re_standard_ascii.ReplaceAllLiteralString(mimes[0], "."),
		Magic:            re_standard_ascii.ReplaceAllLiteralString(magics[0], "."),
		Mimes:            mimes,
		Magics:           magics,
		FileFormat:       fileTypeAl,
		FileFormatLegacy: legacy,
		FileExtension:    extension,
	}

	if fileTypeAl == "" {
		return ret, ErrMismatch
	}
	return ret, nil
}

func dotDump(inStr string) string {
	// Replace all non-ascii characters with '.' (required for assemblyline trusted mime types).
	var sb strings.Builder
	for _, chr := range inStr {
		if chr > 31 || chr < unicode.MaxASCII {
			sb.WriteRune(chr)
		} else {
			sb.WriteRune('.')
		}
	}

	return sb.String()
}

func zipIdent(bufStartOfFile []byte, contentPath string, magic string, mime string, currentType string, fallback string) string {
	/// Extract filenames of a zipfile and attempt to identify a file type.
	fileHandle, err := os.Open(contentPath)
	if err != nil {
		st.Logger.Warn().Msg("failed to open file for zip identification.")
		return fallback
	}
	defer fileHandle.Close()
	info, err := fileHandle.Stat()
	if err != nil {
		st.Logger.Warn().Msg("failed to get size information for for zip identification.")
		return fallback
	}
	zipReader, err := zip.NewReader(fileHandle, info.Size())
	// Probably not a zip directory.
	if err != nil {
		return fallback
	}
	// Get contents of directory.
	var file_list []string
	for _, f := range zipReader.File {
		file_list = append(file_list, f.Name)
	}
	// Determine if the files in the zip means it's a special kind of archive.
	tot_files, tot_class, tot_jar := 0, 0, 0
	is_ipa := false
	is_jar := false
	is_word := false
	is_excel := false
	is_ppt := false
	doc_props := false
	doc_rels := false
	doc_types := false
	android_manifest := false
	android_dex := false
	nuspec := false
	psmdcp := false

	for _, file_name := range file_list {
		if strings.HasPrefix(file_name, "META-INF/") {
			is_jar = true
		} else if file_name == "AndroidManifest.xml" {
			android_manifest = true
		} else if file_name == "classes.dex" {
			android_dex = true
		} else if strings.HasPrefix(file_name, "Payload/") && strings.HasSuffix(file_name, ".app/Info.plist") {
			is_ipa = true
		} else if strings.HasSuffix(file_name, ".nuspec") {
			nuspec = true
		} else if strings.HasPrefix(file_name, "package/services/metadata/core-properties/") && strings.HasSuffix(file_name, ".psmdcp") {
			psmdcp = true
		} else if strings.HasSuffix(file_name, ".class") {
			tot_class += 1
		} else if strings.HasSuffix(file_name, ".jar") {
			tot_jar += 1
		} else if strings.HasPrefix(file_name, "word/") {
			is_word = true
		} else if strings.HasPrefix(file_name, "xl/") {
			is_excel = true
		} else if strings.HasPrefix(file_name, "ppt/") {
			is_ppt = true
		} else if strings.HasPrefix(file_name, "docProps/") {
			doc_props = true
		} else if strings.HasPrefix(file_name, "_rels/") {
			doc_rels = true
		} else if file_name == "[Content_Types].xml" {
			doc_types = true
		}
		tot_files += 1
	}

	if tot_files > 0 && (tot_class+tot_jar)*2 > tot_files {
		is_jar = true
	}

	if is_jar && android_manifest && android_dex {
		return "android/apk"
	} else if is_ipa {
		return "ios/ipa"
	} else if is_jar {
		return "java/jar"
	} else if (doc_props || doc_rels) && doc_types {
		if is_word {
			return "document/office/word"
		} else if is_excel {
			return "document/office/excel"
		} else if is_ppt {
			return "document/office/powerpoint"
		} else if nuspec && psmdcp {
			// It is a nupkg file. Identify as archive/zip for now.
			return "archive/zip"
		} else {
			return "document/office/unknown"
		}
	} else {
		return "archive/zip"
	}
}

func dosIdent(bufStartOfFile []byte, contentPath string, magic string, mime string, currentType string, fallback string) string {
	// Data is too small to be any windows executable, so label it as unknown.
	if len(bufStartOfFile) < 48 {
		return "unknown"
	}
	/// Identify what type of windows binary a file is (dll vs pe and 32 vs 64bit).
	actual_fallback := "executable/windows/dos"
	if len(bufStartOfFile) < 0x40 {
		return actual_fallback
	}
	// Verify header suggests the file is a compiled binary.
	file_header := bufStartOfFile[:0x40]
	if !bytes.Equal(file_header[0:2], []byte("MZ")) {
		return actual_fallback
	}
	p := bytes.NewBuffer(file_header[len(file_header)-4:])
	var header_pos uint32
	var machine_id uint16
	var characteristics uint16
	err := binary.Read(p, binary.LittleEndian, &header_pos)
	if err != nil {
		return actual_fallback
	}
	if int(header_pos) > len(bufStartOfFile)-24 {
		return actual_fallback
	}
	file_header_new_pos := bufStartOfFile[header_pos:]
	if !bytes.Equal(file_header_new_pos[0:4], []byte("PE\x00\x00")) {
		return actual_fallback
	}
	// Determine 64 or 32 bit.
	p = bytes.NewBuffer(file_header_new_pos[4:6])
	err = binary.Read(p, binary.LittleEndian, &machine_id)
	if err != nil {
		return actual_fallback
	}
	var width int
	switch machine_id {
	case 0x014C:
		width = 32
	case 0x8664:
		width = 64
	default:
		return actual_fallback
	}

	// Determine pe or dll.
	p = bytes.NewBuffer(file_header_new_pos[22:24])
	err = binary.Read(p, binary.LittleEndian, &characteristics)
	if err != nil {
		return actual_fallback
	}
	var pe_type string
	if characteristics&0x2000 != 0 {
		pe_type = "dll"
	} else if characteristics&0x0002 != 0 {
		pe_type = "pe"
	} else {
		return actual_fallback
	}

	return fmt.Sprintf("executable/windows/%v%v", pe_type, width)
}

func cartIdent(bufStartOfFile []byte, contentPath string, magic string, mime string, currentType string, fallback string) string {
	// Determine if a file is a cart.
	if len(bufStartOfFile) > 38 {
		firstString := string(bufStartOfFile[0:4])
		version := binary.LittleEndian.Uint16(bufStartOfFile[4:6])
		reserved := binary.LittleEndian.Uint64(bufStartOfFile[6:14])
		if firstString == "CART" && version == 1 && reserved == 0 {
			return "archive/cart"
		}
	}
	return fallback
}

func pdfIdent(bufStartOfFile []byte, contentPath string, magic string, mime string, currentType string, fallback string) string {
	/// Determine if a pdf is password protected or a portfolio.
	contentFile, err := os.Open(contentPath)
	if err != nil {
		st.Logger.Warn().Msg("failed to open file for pdf identification.")
		return fallback
	}
	defer contentFile.Close()

	bufferedReader := bufio.NewReader(contentFile)
	match, err := regexp.MatchReader("/Encrypt", bufferedReader)

	if err == nil && match {
		return "document/pdf/passwordprotected"
		// Portfolios typically contain '/Type/Catalog/Collection
	} else if err != nil {
		st.Logger.Warn().Msg("Regex search of PDF resulted in an error check the regex '/Encrypt'.")
	}
	_, err = contentFile.Seek(0, 0)
	if err != nil {
		st.Logger.Warn().Msg("Failed to seek file during pdf identification.")
		return fallback
	}
	bufferedReaderSecond := bufio.NewReader(contentFile)
	match, err = regexp.MatchReader("/Type/Catalog/Collection", bufferedReaderSecond)
	if err == nil && match {
		return "document/pdf/portfolio"
	} else if err != nil {
		st.Logger.Warn().Msg("Regex search of PDF resulted in an error check the regex '/Type/Catalog/Collection'.")
	}
	return fallback
}

type YaraScanCallback struct {
	highestScore int
	hitType      string
}

func (ysc *YaraScanCallback) RuleMatching(curCtx *yara.ScanContext, rule *yara.Rule) (bool, error) {
	/// Callback function that fires on a yara match and stores the highest scoring hit's type.
	currentType := ""
	var currentScore int
	currentScore = 1 // Default score of 1 for rules that don't have scores.
	for _, meta := range rule.Metas() {
		switch meta.Identifier {
		case "score":
			newCurrentScore, ok := meta.Value.(int)
			if ok {
				currentScore = newCurrentScore
			} else {
				st.Logger.Warn().Msgf("a yara rule isn't working because it's metadata.score can't be coerced into an int value: '%v'", meta.Value)
			}
		case "type":
			newCurrentType, ok := meta.Value.(string)
			if ok {
				currentType = newCurrentType
			} else {
				st.Logger.Warn().Msgf("a yara rule isn't working because it's metadata.type can't be coerced into an string value: '%v'", meta.Value)
			}
		}
	}

	if ysc.highestScore < currentScore && currentType != "" {
		ysc.highestScore = currentScore
		ysc.hitType = currentType
	}

	return false, nil
}

// FUTURE if json identification fails use this. - was required but might not be anymore.
// Stream a file and confirm whether or not it is valid json.
// func streamValidJson(contentPath string) bool {
// 	rawContent, err := os.Open(contentPath)
// 	if err != nil {
// 		log.Print("Warning - unable to open file when attempting to validate json.")
// 		return false
// 	}
// 	defer rawContent.Close()
// 	// Validating json as a sream
// 	decoder := json.NewDecoder(rawContent)
// 	var v interface{}
// 	err = decoder.Decode(&v)
// 	if err != nil {
// 		return false
// 	}
// 	if !decoder.More() {
// 		return true
// 	}
// 	b := make([]byte, 1)
// 	_, err = rawContent.Read(b)
// 	return err == io.EOF
// }

func (cfg *Identify) yaraIdent(bufStartOfFile []byte, contentPath string, magic string, mime string, currentType string, fallback string) string {
	/// Runs yara rules on files that mime and magic couldn't identify.
	// FUTURE if json identification fails use this. - was required but might not be anymore.
	// Check if the file is a misidentified json first before running the yara rules
	// json_valid := streamValidJson(contentPath)
	// if json_valid {
	// 	return "text/json"
	// }

	// Add rules from string
	err := cfg.Yara_Scanner.DefineVariable("mime", mime)
	if err != nil {
		st.Logger.Error().Err(err).Msg("Couldn't compile yara rules (with mime var) skipping yara identification")
		return fallback
	}
	err = cfg.Yara_Scanner.DefineVariable("magic", magic)
	if err != nil {
		st.Logger.Error().Err(err).Msg("Couldn't compile yara rules (with magic var) skipping yara identification")
		return fallback
	}
	err = cfg.Yara_Scanner.DefineVariable("type", currentType)
	if err != nil {
		st.Logger.Error().Err(err).Msg("Couldn't compile yara rules (with type var) skipping yara identification")
		return fallback
	}

	// Run yara rules.
	yscb := YaraScanCallback{highestScore: 0, hitType: fallback}
	cfg.Yara_Scanner.SetCallback(&yscb)
	err = cfg.Yara_Scanner.SetTimeout(time.Second * MAX_YARA_SCAN_TIME_SECONDS).ScanFile(contentPath)

	if err != nil {
		st.Logger.Warn().Msg("yara failed to scan a supplied file.")
	}

	return yscb.hitType
}
