package keeper

import (
	sdk "github.com/aximchain/axc-cosmos-sdk/types"
	"github.com/aximchain/axc-cosmos-sdk/x/gov"
	"github.com/aximchain/axc-cosmos-sdk/x/paramHub/types"
)

func (keeper *Keeper) getLastFCParamChanges(ctx sdk.Context) *types.FCChangeParams {
	var latestProposal *gov.Proposal
	lastProposalId := keeper.GetLastFCParamChangeProposalId(ctx)
	keeper.govKeeper.Iterate(ctx, nil, nil, gov.StatusPassed, lastProposalId.ProposalID, true, func(proposal gov.Proposal) bool {
		if proposal.GetProposalType() == gov.ProposalTypeParameterChange {
			latestProposal = &proposal
			return true
		}
		return false
	})

	if latestProposal != nil {
		var changeParam types.FCChangeParams
		strProposal := (*latestProposal).GetDescription()
		err := keeper.cdc.UnmarshalJSON([]byte(strProposal), &changeParam)
		if err != nil {
			keeper.Logger(ctx).Error("Get broken data when unmarshal FCParamsChange msg, will skip.", "proposalId", (*latestProposal).GetProposalID(), "err", err)
			return nil
		}
		// SetLastFCParamChangeProposalId first. If invalid, the proposal before it will not been processed too.
		keeper.SetLastFCParamChangeProposalId(ctx, types.LastProposalID{ProposalID: (*latestProposal).GetProposalID()})
		if err := changeParam.Check(); err != nil {
			keeper.Logger(ctx).Error("The FCParamsChange proposal is invalid, will skip.", "proposalId", (*latestProposal).GetProposalID(), "param", changeParam, "err", err)
			return nil
		}
		return &changeParam
	}
	return nil
}

func (keeper *Keeper) GetLastFCParamChangeProposalId(ctx sdk.Context) types.LastProposalID {
	var id types.LastProposalID
	keeper.paramSpace.GetIfExists(ctx, ParamStoreKeyFCLastParamsChangeProposalID, &id)
	return id
}

func (keeper *Keeper) SetLastFCParamChangeProposalId(ctx sdk.Context, id types.LastProposalID) {
	keeper.paramSpace.Set(ctx, ParamStoreKeyFCLastParamsChangeProposalID, &id)
	return
}

func (keeper *Keeper) GetFCParams(ctx sdk.Context) ([]types.FCParam, sdk.Error) {
	params := make([]types.FCParam, 0)
	for _, subSpace := range keeper.GetSubscriberFCParamSpace() {
		param := subSpace.Proto()
		subSpace.ParamSpace.GetParamSet(ctx, param)
		params = append(params, param)
	}
	return params, nil
}
