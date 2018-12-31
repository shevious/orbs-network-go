package e2e

import (
	"fmt"
	"github.com/orbs-network/orbs-client-sdk-go/codec"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/contracts"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestDeploymentOfNativeContract(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	runMultipleTimes(t, func(t *testing.T) {

		h := newHarness()
		lt := time.Now()
		printTestTime(t, "started", &lt)

		h.waitUntilTransactionPoolIsReady(t)
		printTestTime(t, "first block committed", &lt)

		counterStart := uint64(time.Now().UnixNano())
		contractName := fmt.Sprintf("CounterFrom%d", counterStart)

		printTestTime(t, "send deploy - start", &lt)

		// TODO remove Eventually loop once node can handle requests at block height 0
		require.True(t, test.Eventually(5*time.Second, func() bool {
			dcExResult, dcTxStatus, dcErr := h.deployNativeContract(OwnerOfAllSupply, contractName, []byte(contracts.NativeSourceCodeForCounter(counterStart)))
			return dcErr == nil &&
				dcTxStatus == codec.TRANSACTION_STATUS_COMMITTED &&
				dcExResult == codec.EXECUTION_RESULT_SUCCESS
		}), "expected contract to deploy successfully within 5 seconds")
		printTestTime(t, "send deploy - end", &lt)

		// check counter
		ok := test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
			printTestTime(t, "call method - start", &lt)
			response, err2 := h.callMethod(OwnerOfAllSupply.PublicKey(), contractName, "get")
			printTestTime(t, "call method - end", &lt)

			if err2 == nil && response.ExecutionResult == codec.EXECUTION_RESULT_SUCCESS {
				return response.OutputArguments[0] == counterStart
			}
			return false
		})
		require.True(t, ok, "get counter should return counter start")

		// transaction to add to the counter
		amount := uint64(17)

		printTestTime(t, "send transaction - start", &lt)
		response, _, err := h.sendTransaction(OwnerOfAllSupply.PublicKey(), OwnerOfAllSupply.PrivateKey(), contractName, "add", uint64(amount))
		printTestTime(t, "send transaction - end", &lt)

		require.NoError(t, err, "add transaction should not return error")
		require.Equal(t, codec.TRANSACTION_STATUS_COMMITTED, response.TransactionStatus)
		require.Equal(t, codec.EXECUTION_RESULT_SUCCESS, response.ExecutionResult)

		// check counter
		ok = test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
			response, err := h.callMethod(OwnerOfAllSupply.PublicKey(), contractName, "get")

			if err == nil && response.ExecutionResult == codec.EXECUTION_RESULT_SUCCESS {
				return response.OutputArguments[0] == counterStart+amount
			}
			return false
		})

		require.True(t, ok, "get counter should return counter start plus added value")
		printTestTime(t, "done", &lt)

	})
}
