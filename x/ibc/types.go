package ibc

import (
	sdk "github.com/aximchain/axc-cosmos-sdk/types"
)

type packageRecord struct {
	destChainID sdk.ChainID
	channelID   sdk.ChannelID
	sequence    uint64
}

type packageCollector struct {
	collectedPackages []packageRecord
}

func newPackageCollector() *packageCollector {
	return &packageCollector{
		collectedPackages: nil,
	}
}
