package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/acceptance/callcontract"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestIncomingTransactionTriggersExactlyOneBlock(t *testing.T) {
	newHarness().
		WithEmptyBlockTime(1*time.Second).
		Start(t, func(tb testing.TB, ctx context.Context, network *NetworkHarness) {
			contract := callcontract.NewContractClient(network)
			time.Sleep(100 * time.Millisecond)
			heightBeforeTx, _ := network.BlockPersistence(0).GetLastBlockHeight()

			_, txHash := contract.Transfer(ctx, 0, 43, 5, 6)
			network.WaitForTransactionInState(ctx, txHash)

			heightAfterTx, _ := network.BlockPersistence(0).GetLastBlockHeight()

			require.Equal(t, uint64(heightBeforeTx)+1, uint64(heightAfterTx), "incoming transaction triggered closure of more than one block")
		})
}

func TestIncomingTransactionTriggersImmediateBlockClosure(t *testing.T) {
	newHarness().
		WithEmptyBlockTime(1*time.Hour). // so that the test fails on timeout if block doesn't close immediately on incoming transaction
		WithLogFilters(log.ExcludeEntryPoint("BlockSync")).
		Start(t, func(tb testing.TB, ctx context.Context, network *NetworkHarness) {
			contract := callcontract.NewContractClient(network)
			_, txHash := contract.Transfer(ctx, 0, 43, 5, 6)
			network.WaitForTransactionInState(ctx, txHash)

		})
}
