package cli

import (
	"github.com/aximchain/axc-cosmos-sdk/client"
	"github.com/aximchain/axc-cosmos-sdk/codec"
	"github.com/spf13/cobra"
)

var scStoreKey = "sc"
var slashingStoreName = "slashing"

func AddCommands(root *cobra.Command, cdc *codec.Codec) {
	slashingCmd := &cobra.Command{
		Use:   "slashing",
		Short: "slashing validators",
	}

	slashingCmd.AddCommand(
		client.PostCommands(
			GetCmdAscSubmitEvidence(cdc),
			GetCmdSideChainUnjail(cdc),
		)...)

	slashingCmd.AddCommand(
		client.GetCommands(
			GetCmdQuerySideChainSigningInfo(slashingStoreName, cdc),
			GetCmdQuerySideChainSlashRecord(slashingStoreName, cdc),
			GetCmdQuerySideChainSlashRecords(cdc),
			GetCmdQueryAllSideSlashRecords(slashingStoreName, cdc),
		)...)

	root.AddCommand(slashingCmd)
}
