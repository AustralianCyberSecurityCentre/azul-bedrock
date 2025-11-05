package events

import "fmt"

// Enumeration of entity subtypes that are available
type BinaryAction string

const (
	//
	// binary events
	//
	// Binary that has been uploaded to Azul
	// Must have entity.datastreams
	// Must not be child of other event types
	ActionSourced BinaryAction = "sourced"

	// A binary has been extracted from another binary, which should be further processed
	// For example, an encoded PE in a PE was extracted
	// Must have entity.datastreams with a label=content stream
	ActionExtracted BinaryAction = "extracted"

	// An alternative binary representation of this file has been produced
	// For example, a screenshot or decompiled source code
	// Must have entity.datastreams with a label!=content stream
	ActionAugmented BinaryAction = "augmented"

	// Metadata for a binary has been mapped into Azul from an external database
	// For example, metadata for a given binary has been found and copied from VirusTotal
	// Must not have entity.datastreams
	// Must not be parent of other event types
	// Must not be a child of other event types
	ActionMapped BinaryAction = "mapped"

	// A binary has been enriched with additional metadata
	// For example, features have been identified by a plugin for this binary
	// Must not have entity.datastreams
	// Must not be parent of other event types
	ActionEnriched BinaryAction = "enriched"
)

// list of 'event' types valid for binary events
var ActionsBinary = []BinaryAction{ActionSourced, ActionExtracted, ActionAugmented, ActionMapped, ActionEnriched}

// list of binary event types that can and must have new 'entity.datastreams' entries
var ActionsBinaryDataOk = []BinaryAction{ActionSourced, ActionExtracted, ActionAugmented}

// This ensures we can check a given action is valid.
var ActionsMap = map[BinaryAction]bool{
	ActionSourced:   true,
	ActionExtracted: true,
	ActionMapped:    true,
	ActionEnriched:  true,
	ActionAugmented: true,
}

// ActionsFromStrings converts binary event type strings to EventType custom typing
func ActionsFromStrings(actions []string) ([]BinaryAction, error) {
	ret := []BinaryAction{}
	for _, et := range actions {
		if _, ok := ActionsMap[BinaryAction(et)]; !ok {
			return nil, fmt.Errorf("invalid event type '%s'", et)
		}
		ret = append(ret, BinaryAction(et))
	}
	return ret, nil
}

func StringsFromActions(actions []BinaryAction) ([]string, error) {
	ret := []string{}
	for _, et := range actions {
		if _, ok := ActionsMap[BinaryAction(et)]; !ok {
			return nil, fmt.Errorf("invalid event type '%s'", et)
		}
		ret = append(ret, string(et))
	}
	return ret, nil
}

func (b BinaryAction) Str() string {
	return string(b)
}
