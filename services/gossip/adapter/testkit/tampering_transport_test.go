// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package testkit

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/memory"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type tamperingHarness struct {
	senderKey string
	transport *TamperingTransport
	listener  *MockTransportListener
}

func newTamperingHarness(tb testing.TB, ctx context.Context) *tamperingHarness {
	senderAddress := "sender"
	listenerAddress := "listener"
	listener := &MockTransportListener{}
	logger := log.DefaultTestingLogger(tb).WithTags(log.String("adapter", "transport"))

	genesisValidatorNodes := make(map[string]config.ValidatorNode)
	genesisValidatorNodes[senderAddress] = config.NewHardCodedValidatorNode(primitives.NodeAddress(senderAddress))
	genesisValidatorNodes[listenerAddress] = config.NewHardCodedValidatorNode(primitives.NodeAddress(listenerAddress))

	transport := NewTamperingTransport(logger, memory.NewTransport(ctx, logger, genesisValidatorNodes))

	transport.RegisterListener(listener, primitives.NodeAddress(listenerAddress))

	return &tamperingHarness{
		senderKey: senderAddress,
		transport: transport,
		listener:  listener,
	}
}

func (c *tamperingHarness) send(ctx context.Context, payloads [][]byte) {
	c.broadcast(ctx, c.senderKey, payloads)
}

func (c *tamperingHarness) broadcast(ctx context.Context, sender string, payloads [][]byte) error {
	return c.transport.Send(ctx, &adapter.TransportData{
		RecipientMode:     gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		SenderNodeAddress: primitives.NodeAddress(sender),
		Payloads:          payloads,
	})
}

func TestFailingTamperer(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		c := newTamperingHarness(t, ctx)

		c.transport.Fail(anyMessage())

		c.send(ctx, nil)

		c.listener.ExpectNotReceive()

		ok, err := c.listener.Verify()
		if !ok {
			t.Fatal(err)
		}
	})
}

func TestPausingTamperer(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		c := newTamperingHarness(t, ctx)

		digits := make(chan byte, 10)

		odds := c.transport.Pause(oddNumbers())

		c.listener.WhenOnTransportMessageReceived(mock.Any).Call(func(ctx context.Context, payloads [][]byte) {
			digits <- payloads[0][0]
		}).Times(10)

		for b := 0; b < 10; b++ {
			c.send(ctx, [][]byte{{byte(b)}})
		}

		for b := 0; b < 5; b++ {
			if <-digits%2 != 0 {
				t.Errorf("got odd number while odds should be paused")
			}
		}

		odds.StopTampering(ctx)

		for b := 0; b < 5; b++ {
			if <-digits%2 != 1 {
				t.Errorf("got even number while odds should be released")
			}
		}
	})
}

func TestLatchingTamperer_WaitBlocksUntilSend(t *testing.T) {
	t.Parallel()

	test.WithContext(func(ctx context.Context) {
		c := newTamperingHarness(t, ctx)

		latch := c.transport.LatchOn(anyMessage())
		c.listener.WhenOnTransportMessageReceived(mock.Any)

		requireOperationBlocking(t, ctx, func() { latch.Wait() }, func() { c.send(ctx, nil) }, "latch.Wait()", "tamperingHarness.send()")
		requireOperationBlocking(t, ctx, func() { latch.Wait() }, func() { c.send(ctx, nil) }, "latch.Wait()", "tamperingHarness.send()")
	})
}

func TestLatchingTamperer_SendBlocksUntilWait(t *testing.T) {
	t.Parallel()

	test.WithContext(func(ctx context.Context) {
		c := newTamperingHarness(t, ctx)

		latch := c.transport.LatchOn(anyMessage())
		c.listener.WhenOnTransportMessageReceived(mock.Any)

		requireOperationBlocking(t, ctx, func() { c.send(ctx, nil) }, func() { latch.Wait() }, "tamperingHarness.send()", "latch.Wait()")
		requireOperationBlocking(t, ctx, func() { c.send(ctx, nil) }, func() { latch.Wait() }, "tamperingHarness.send()", "latch.Wait()")
	})
}

func requireOperationBlocking(t *testing.T, ctx context.Context, op1, op2 func(), op1Name, op2Name string) {
	infinity := 500 * time.Millisecond
	immediately := 50 * time.Millisecond

	doneOp1 := make(chan struct{})
	go func() {
		op1()
		close(doneOp1)
	}()

	timeout, _ := context.WithTimeout(ctx, infinity)
	select {
	case <-doneOp1:
		t.Fatalf("expected %s to block before %s", op1Name, op2Name)
	case <-timeout.Done():
		t.Logf("%s blocks before %s", op1Name, op2Name)
	}

	op2StartTime := time.Now()
	op2()
	op2EndTime := time.Now()

	require.WithinDurationf(t, op2EndTime, op2StartTime, immediately, "expected %s to return immediately when calling %s", op1Name, op2Name)

	timeout, _ = context.WithTimeout(ctx, infinity)
	select {
	case <-doneOp1:
		require.WithinDurationf(t, time.Now(), op2EndTime, immediately, "expected %s to return fast after %s", op1Name, op2Name)
	case <-timeout.Done(): // done testing
		t.Fatalf("expected %s to return immediately after %s", op1Name, op2Name)
	}
}

func oddNumbers() MessagePredicate {
	return func(data *adapter.TransportData) bool {
		return data.Payloads[0][0]%2 == 1
	}
}

func anyMessage() MessagePredicate {
	return func(data *adapter.TransportData) bool {
		return true
	}
}
