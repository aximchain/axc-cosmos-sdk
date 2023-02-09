package ibc

import (
	"fmt"

	sdk "github.com/aximchain/axc-cosmos-sdk/types"
)

const (
	separator                    = "::"
	ibcEventType                 = "IBCPackage"
	ibcPackageInfoAttributeKey   = "IBCPackageInfo"
	ibcPackageInfoAttributeValue = "%d" + separator + "%d" + separator + "%d" // destChainID channelID sequence
)

func buildIBCPackageAttributeValue(sideChainID sdk.ChainID, channelID sdk.ChannelID, sequence uint64) string {
	return fmt.Sprintf(ibcPackageInfoAttributeValue, sideChainID, channelID, sequence)
}
