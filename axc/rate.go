package axc

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	AXCDecimalOnBC  = 8
	AXCDecimalOnAXC = 18
)

// ConvertBCAmountToAXCAmount can only be used to convert AXC decimal
func ConvertBCAmountToAXCAmount(bcAmount int64) *big.Int {
	decimals := sdk.NewIntWithDecimal(1, int(AXCDecimalOnAXC-AXCDecimalOnBC))
	axcAmount := sdk.NewInt(bcAmount).Mul(decimals)
	return axcAmount.BigInt()
}

// ConvertAXCAmountToBCAmount can only be used to convert AXC decimal
func ConvertAXCAmountToBCAmount(axcAmount *big.Int) int64 {
	decimals := sdk.NewIntWithDecimal(1, int(AXCDecimalOnAXC-AXCDecimalOnBC))
	axcAmountInt := sdk.NewIntFromBigInt(axcAmount)
	bcAmount := axcAmountInt.Div(decimals)
	return bcAmount.Int64()
}
