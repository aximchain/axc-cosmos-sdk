package tx

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/client/utils"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

const (
	flagTags    = "tag"
	flagAny     = "any"
	flagPage    = "page"
	flagPerPage = "perPage"
)

// default client command to search through tagged transactions
func SearchTxCmd(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "txs",
		Short: "Search for all transactions that match the given tags.",
		Long: strings.TrimSpace(`
Search for transactions that match the given tags. By default, transactions must match ALL tags
passed to the --tags option. To match any transaction, use the --any option.

For example:

$ gaiacli tendermint txs --tag test1,test2 --page 0 --perPage 30

will match any transaction tagged with both test1,test2. To match a transaction tagged with either
test1 or test2, use:

$ gaiacli tendermint txs --tag test1,test2 --any --page 0 --perPage 30
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			tags := viper.GetStringSlice(flagTags)
			page := viper.GetInt(flagPage)
			perPage := viper.GetInt(flagPerPage)

			cliCtx := context.NewCLIContext().WithCodec(cdc)

			txs, err := searchTxs(cliCtx, cdc, tags, page, perPage)
			if err != nil {
				return err
			}

			var output []byte
			if cliCtx.Indent {
				output, err = cdc.MarshalJSONIndent(txs, "", "  ")
			} else {
				output, err = cdc.MarshalJSON(txs)
			}

			if err != nil {
				return err
			}

			fmt.Println(string(output))
			return nil
		},
	}

	cmd.Flags().StringP(client.FlagNode, "n", "tcp://localhost:26657", "Node to connect to")
	viper.BindPFlag(client.FlagNode, cmd.Flags().Lookup(client.FlagNode))
	cmd.Flags().String(client.FlagChainID, "", "Chain ID of Tendermint node")
	viper.BindPFlag(client.FlagChainID, cmd.Flags().Lookup(client.FlagChainID))
	cmd.Flags().Bool(client.FlagTrustNode, false, "Trust connected full node (don't verify proofs for responses)")
	viper.BindPFlag(client.FlagTrustNode, cmd.Flags().Lookup(client.FlagTrustNode))
	cmd.Flags().StringSlice(flagTags, nil, "Comma-separated list of tags that must match")
	cmd.Flags().Bool(flagAny, false, "Return transactions that match ANY tag, rather than ALL")
	return cmd
}

func searchTxs(cliCtx context.CLIContext, cdc *codec.Codec, tags []string, page, perPage int) ([]Info, error) {
	if len(tags) == 0 {
		return nil, errors.New("must declare at least one tag to search")
	}

	// XXX: implement ANY
	query := strings.Join(tags, " AND ")

	// get the node
	node, err := cliCtx.GetNode()
	if err != nil {
		return nil, err
	}

	prove := !cliCtx.TrustNode

	res, err := node.TxSearch(query, prove, page, perPage)
	if err != nil {
		return nil, err
	}

	if prove {
		for _, tx := range res.Txs {
			err := ValidateTxResult(cliCtx, tx)
			if err != nil {
				return nil, err
			}
		}
	}

	info, err := FormatTxResults(cdc, res.Txs)
	if err != nil {
		return nil, err
	}

	return info, nil
}

// parse the indexed txs into an array of Info
func FormatTxResults(cdc *codec.Codec, res []*ctypes.ResultTx) ([]Info, error) {
	var err error
	out := make([]Info, len(res))
	for i := range res {
		out[i], err = formatTxResult(cdc, res[i])
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

/////////////////////////////////////////
// REST

// Search Tx REST Handler
func SearchTxRequestHandlerFn(cliCtx context.CLIContext, cdc *codec.Codec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tag := r.FormValue("tag")
		if tag == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("You need to provide at least a tag as a key=value pair to search for. Postfix the key with _bech32 to search bech32-encoded addresses or public keys"))
			return
		}
		pageStr := r.FormValue("page")
		if pageStr == "" {
			pageStr = "0"
		}
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("page parameter is not a valid non-negative integer"))
			return
		}

		perPageStr := r.FormValue("perPage")
		if perPageStr == "" {
			perPageStr = "30" // should be consistent with tendermint/tendermint/rpc/core/pipe.go:19
		}
		perPage, err := strconv.Atoi(perPageStr)
		if err != nil || perPage <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("perPage parameter is not a valid positive integer"))
			return
		}

		keyValue := strings.Split(tag, "=")
		key := keyValue[0]

		value, err := url.QueryUnescape(keyValue[1])
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, sdk.AppendMsgToErr("could not decode address", err.Error()))
			return
		}

		if strings.HasSuffix(key, "_bech32") {
			bech32address := strings.Trim(value, "'")
			prefix := strings.Split(bech32address, "1")[0]
			bz, err := sdk.GetFromBech32(bech32address, prefix)
			if err != nil {
				utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
				return
			}

			tag = strings.TrimRight(key, "_bech32") + "='" + sdk.AccAddress(bz).String() + "'"
		}

		txs, err := searchTxs(cliCtx, cdc, []string{tag}, page, perPage)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		if len(txs) == 0 {
			w.Write([]byte("[]"))
			return
		}

		utils.PostProcessResponse(w, cdc, txs, cliCtx.Indent)
	}
}
