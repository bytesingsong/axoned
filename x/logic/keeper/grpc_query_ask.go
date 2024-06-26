package keeper

import (
	goctx "context"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/axone-protocol/axoned/v8/x/logic/meter"
	"github.com/axone-protocol/axoned/v8/x/logic/types"
	"github.com/axone-protocol/axoned/v8/x/logic/util"
)

var defaultSolutionsLimit = sdkmath.OneUint()

func (k Keeper) Ask(ctx goctx.Context, req *types.QueryServiceAskRequest) (response *types.QueryServiceAskResponse, err error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if req == nil {
		return nil, errorsmod.Wrap(types.InvalidArgument, "request is nil")
	}

	limits := k.limits(ctx)
	if err := checkLimits(req, limits); err != nil {
		return nil, err
	}

	sdkCtx = withGasMeter(sdkCtx, limits)
	defer func() {
		if r := recover(); r != nil {
			if gasError, ok := r.(storetypes.ErrorOutOfGas); ok {
				response, err = nil, errorsmod.Wrapf(
					types.LimitExceeded, "out of gas: %s <%s> (%d/%d)",
					types.ModuleName, gasError.Descriptor, sdkCtx.GasMeter().GasConsumed(), sdkCtx.GasMeter().Limit())

				return
			}

			panic(r)
		}
	}()
	sdkCtx.GasMeter().ConsumeGas(sdkCtx.GasMeter().GasConsumed(), types.ModuleName)

	return k.execute(
		sdkCtx,
		req.Program,
		req.Query,
		util.DerefOrDefault(req.Limit, defaultSolutionsLimit))
}

// withGasMeter returns a new context with a gas meter that has the given limit.
// The gas meter is go-router-safe.
func withGasMeter(sdkCtx sdk.Context, limits types.Limits) sdk.Context {
	gasMeter := meter.WithSafeMeter(storetypes.NewGasMeter(limits.MaxGas.Uint64()))

	return sdkCtx.WithGasMeter(gasMeter)
}
