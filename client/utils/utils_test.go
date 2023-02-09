package utils

import (
	"testing"

	"github.com/aximchain/axc-cosmos-sdk/cmd/gaia/app"
	sdk "github.com/aximchain/axc-cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
)

func TestParseQueryResponse(t *testing.T) {
	cdc := app.MakeCodec()
	sdkResBytes := cdc.MustMarshalBinaryLengthPrefixed(sdk.Result{})
	_, err := parseQueryResponse(cdc, sdkResBytes)
	assert.Nil(t, err)
	_, err = parseQueryResponse(cdc, []byte("fuzzy"))
	assert.NotNil(t, err)
}
