package types

import (
	"bytes"
	"encoding/json"
	"time"

	tcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	tmtypes "github.com/tendermint/tendermint/types"
)

// SortedJSON takes any JSON and returns it sorted by keys. Also, all white-spaces
// are removed.
// This method can be used to canonicalize JSON to be returned by GetSignBytes,
// e.g. for the ledger integration.
// If the passed JSON isn't valid it will return an error.
func SortJSON(toSortJSON []byte) ([]byte, error) {
	var c interface{}
	var err error
	if !IsUpgrade(FixSignBytesOverflow) {
		err = json.Unmarshal(toSortJSON, &c)
	} else {
		decoder := json.NewDecoder(bytes.NewReader(toSortJSON))
		decoder.UseNumber()
		err = decoder.Decode(&c)
	}
	if err != nil {
		return nil, err
	}
	js, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	return js, nil
}

// MustSortJSON is like SortJSON but panic if an error occurs, e.g., if
// the passed JSON isn't valid.
func MustSortJSON(toSortJSON []byte) []byte {
	js, err := SortJSON(toSortJSON)
	if err != nil {
		panic(err)
	}
	return js
}

// Slight modification of the RFC3339Nano but it right pads all zeros and drops the time zone info
const SortableTimeFormat = "2006-01-02T15:04:05.000000000"

// Formats a time.Time into a []byte that can be sorted
func FormatTimeBytes(t time.Time) []byte {
	return []byte(t.UTC().Round(0).Format(SortableTimeFormat))
}

// Parses a []byte encoded using FormatTimeKey back into a time.Time
func ParseTimeBytes(bz []byte) (time.Time, error) {
	str := string(bz)
	t, err := time.Parse(SortableTimeFormat, str)
	if err != nil {
		return t, err
	}
	return t.UTC().Round(0), nil
}

// DefaultChainID returns the chain ID from the genesis file if present. An
// error is returned if the file cannot be read or parsed.
//
// TODO: This should be removed and the chainID should always be provided by
// the end user.
func DefaultChainID() (string, error) {
	cfg, err := tcmd.ParseConfig()
	if err != nil {
		return "", err
	}

	doc, err := tmtypes.GenesisDocFromFile(cfg.GenesisFile())
	if err != nil {
		return "", err
	}

	return doc.ChainID, nil
}
