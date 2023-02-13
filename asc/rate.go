package asc

import (
	"math/big"

	sdk "github.com/aximchain/axc-cosmos-sdk/types"
)

const (
	AXCDecimalOnFC  = 8
	AXCDecimalOnASC = 18
)

// ConvertFCAmountToASCAmount can only be used to convert AXC decimal
func ConvertFCAmountToASCAmount(fcAmount int64) *big.Int {
	decimals := sdk.NewIntWithDecimal(1, int(AXCDecimalOnASC-AXCDecimalOnFC))
	ascAmount := sdk.NewInt(fcAmount).Mul(decimals)
	return ascAmount.BigInt()
}

// ConvertASCAmountToFCAmount can only be used to convert AXC decimal
func ConvertASCAmountToFCAmount(ascAmount *big.Int) int64 {
	decimals := sdk.NewIntWithDecimal(1, int(AXCDecimalOnASC-AXCDecimalOnFC))
	ascAmountInt := sdk.NewIntFromBigInt(ascAmount)
	fcAmount := ascAmountInt.Div(decimals)
	return fcAmount.Int64()
}
