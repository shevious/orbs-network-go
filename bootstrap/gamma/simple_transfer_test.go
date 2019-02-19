package gamma

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/acceptance/callcontract"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestSimpleTransfer(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		network := NewDevelopmentNetwork(ctx, log.DefaultTestingLogger(t))
		contract := callcontract.NewContractClient(network)

		t.Log("doing a simple transfer")

		contract.Transfer(ctx, 0, 17, 5, 6)

		t.Log("making sure balance is correct")

		require.True(t, test.Eventually(1*time.Second, func() bool {
			return 17 == contract.GetBalance(ctx, 0, 6)
		}), "expected balance to reflect the transfer")

	})
	time.Sleep(5 * time.Millisecond) // give context dependent goroutines 5 ms to terminate gracefully
}