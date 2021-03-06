// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package tcp

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"sync"
)

const MAX_PAYLOADS_IN_MESSAGE = 100000
const MAX_PAYLOAD_SIZE_BYTES = 20 * 1024 * 1024
const SEND_QUEUE_MAX_MESSAGES = 1000
const SEND_QUEUE_MAX_BYTES = 20 * 1024 * 1024

var LogTag = log.String("adapter", "gossip")

type metrics struct {
	incomingConnectionAcceptSuccesses *metric.Gauge
	incomingConnectionAcceptErrors    *metric.Gauge
	incomingConnectionTransportErrors *metric.Gauge
	outgoingConnectionSendErrors      *metric.Gauge
	outgoingConnectionKeepaliveErrors *metric.Gauge
	outgoingConnectionSendQueueErrors *metric.Gauge

	activeIncomingConnections *metric.Gauge
	activeOutgoingConnections *metric.Gauge

	outgoingMessageSize *metric.Histogram
}

type DirectTransport struct {
	config config.GossipTransportConfig
	logger log.Logger

	outgoingPeerQueues map[string]*transportQueue

	mutex                       *sync.RWMutex
	transportListenerUnderMutex adapter.TransportListener
	serverListeningUnderMutex   bool
	serverPort                  int

	metrics        *metrics
	metricRegistry metric.Registry
}

func getMetrics(registry metric.Registry) *metrics {
	return &metrics{
		incomingConnectionAcceptSuccesses: registry.NewGauge("Gossip.IncomingConnection.ListeningOnTCPPortSuccess.Count"),
		incomingConnectionAcceptErrors:    registry.NewGauge("Gossip.IncomingConnection.ListeningOnTCPPortErrors.Count"),
		incomingConnectionTransportErrors: registry.NewGauge("Gossip.IncomingConnection.TransportErrors.Count"),
		outgoingConnectionSendErrors:      registry.NewGauge("Gossip.OutgoingConnection.SendErrors.Count"),
		outgoingConnectionKeepaliveErrors: registry.NewGauge("Gossip.OutgoingConnection.KeepaliveErrors.Count"),
		outgoingConnectionSendQueueErrors: registry.NewGauge("Gossip.OutgoingConnection.SendQueueErrors.Count"),
		activeIncomingConnections:         registry.NewGauge("Gossip.IncomingConnection.Active.Count"),
		activeOutgoingConnections:         registry.NewGauge("Gossip.OutgoingConnection.Active.Count"),
		outgoingMessageSize:               registry.NewHistogram("Gossip.OutgoingConnection.MessageSize.Bytes", MAX_PAYLOAD_SIZE_BYTES),
	}
}

func NewDirectTransport(ctx context.Context, config config.GossipTransportConfig, logger log.Logger, registry metric.Registry) *DirectTransport {
	t := &DirectTransport{
		config:         config,
		logger:         logger.WithTags(LogTag),
		metricRegistry: registry,

		outgoingPeerQueues: make(map[string]*transportQueue),

		mutex:   &sync.RWMutex{},
		metrics: getMetrics(registry),
	}

	// server goroutine
	supervised.GoForever(ctx, t.logger, func() {
		t.serverMainLoop(ctx, t.config.GossipListenPort())
	})

	// client goroutines
	t.mutex.Lock()
	for peerNodeAddress, peer := range t.config.GossipPeers() {
		t.connect(ctx, peerNodeAddress, peer)
	}
	t.mutex.Unlock()

	return t
}

// note that bgCtx MUST be a long-running background context - if it's a short lived context, the new connection will die as soon as
// the context is done
func (t *DirectTransport) connect(bgCtx context.Context, peerNodeAddress string, peer config.GossipPeer) {
	if peerNodeAddress != t.config.NodeAddress().KeyForMap() {
		t.outgoingPeerQueues[peerNodeAddress] = NewTransportQueue(SEND_QUEUE_MAX_BYTES, SEND_QUEUE_MAX_MESSAGES, t.metricRegistry)
		t.outgoingPeerQueues[peerNodeAddress].Disable() // until connection is established

		peerAddress := fmt.Sprintf("%s:%d", peer.GossipEndpoint(), peer.GossipPort())
		t.outgoingPeerQueues[peerNodeAddress].networkAddress = peerAddress

		supervised.GoForever(bgCtx, t.logger, func() {
			t.clientMainLoop(bgCtx, t.outgoingPeerQueues[peerNodeAddress])
		})
	}
}

func (t *DirectTransport) AddPeer(bgCtx context.Context, address primitives.NodeAddress, peer config.GossipPeer) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.config.GossipPeers()[address.KeyForMap()] = peer
	t.connect(bgCtx, address.KeyForMap(), peer)
}

func (t *DirectTransport) RegisterListener(listener adapter.TransportListener, listenerNodeAddress primitives.NodeAddress) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.transportListenerUnderMutex = listener
}

// TODO(https://github.com/orbs-network/orbs-network-go/issues/182): we are not currently respecting any intents given in ctx (added in context refactor)
func (t *DirectTransport) Send(ctx context.Context, data *adapter.TransportData) error {
	switch data.RecipientMode {
	case gossipmessages.RECIPIENT_LIST_MODE_BROADCAST:
		for _, peerQueue := range t.outgoingPeerQueues {
			t.addDataToOutgoingPeerQueue(data, peerQueue)
		}
		return nil
	case gossipmessages.RECIPIENT_LIST_MODE_LIST:
		for _, recipientPublicKey := range data.RecipientNodeAddresses {
			if peerQueue, found := t.outgoingPeerQueues[recipientPublicKey.KeyForMap()]; found {
				t.addDataToOutgoingPeerQueue(data, peerQueue)
			} else {
				return errors.Errorf("unknown recipient public key: %s", recipientPublicKey.String())
			}
		}
		return nil
	case gossipmessages.RECIPIENT_LIST_MODE_ALL_BUT_LIST:
		panic("Not implemented")
	}
	return errors.Errorf("unknown recipient mode: %s", data.RecipientMode.String())
}

func (t *DirectTransport) Port() int {
	return t.serverPort
}

func calcPaddingSize(size uint32) uint32 {
	const contentAlignment = 4
	alignedSize := (size + contentAlignment - 1) / contentAlignment * contentAlignment
	return alignedSize - size
}
