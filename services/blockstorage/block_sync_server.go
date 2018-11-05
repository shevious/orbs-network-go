package blockstorage

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/pkg/errors"
)

func (s *service) sourceHandleBlockAvailabilityRequest(ctx context.Context, message *gossipmessages.BlockAvailabilityRequestMessage) error {
	s.logger.Info("received block availability request",
		log.Stringable("petitioner", message.Sender.SenderPublicKey()),
		log.Stringable("requested-first-block", message.SignedBatchRange.FirstBlockHeight()),
		log.Stringable("requested-last-block", message.SignedBatchRange.LastBlockHeight()),
		log.Stringable("requested-last-committed-block", message.SignedBatchRange.LastCommittedBlockHeight()))

	out, err := s.GetLastCommittedBlockHeight(ctx, &services.GetLastCommittedBlockHeightInput{})
	if err != nil {
		return err
	}
	_lastCommittedBlockHeight := out.LastCommittedBlockHeight

	if _lastCommittedBlockHeight <= message.SignedBatchRange.LastCommittedBlockHeight() {
		return nil
	}

	firstAvailableBlockHeight := primitives.BlockHeight(1)
	blockType := message.SignedBatchRange.BlockType()

	response := &gossiptopics.BlockAvailabilityResponseInput{
		RecipientPublicKey: message.Sender.SenderPublicKey(),
		Message: &gossipmessages.BlockAvailabilityResponseMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: s.config.NodePublicKey(),
			}).Build(),
			SignedBatchRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                blockType,
				LastBlockHeight:          _lastCommittedBlockHeight,
				FirstBlockHeight:         firstAvailableBlockHeight,
				LastCommittedBlockHeight: _lastCommittedBlockHeight,
			}).Build(),
		},
	}

	s.logger.Info("sending the response for availability request",
		log.Stringable("petitioner", response.RecipientPublicKey),
		log.Stringable("first-available-block-height", response.Message.SignedBatchRange.FirstBlockHeight()),
		log.Stringable("last-available-block-height", response.Message.SignedBatchRange.LastBlockHeight()),
		log.Stringable("last-committed-available-block-height", response.Message.SignedBatchRange.LastCommittedBlockHeight()),
		log.Stringable("source", response.Message.Sender.SenderPublicKey()),
	)

	_, err = s.gossip.SendBlockAvailabilityResponse(ctx, response)
	return err
}

func (s *service) sourceHandleBlockSyncRequest(ctx context.Context, message *gossipmessages.BlockSyncRequestMessage) error {
	senderPublicKey := message.Sender.SenderPublicKey()
	blockType := message.SignedChunkRange.BlockType()
	firstRequestedBlockHeight := message.SignedChunkRange.FirstBlockHeight()
	lastRequestedBlockHeight := message.SignedChunkRange.LastBlockHeight()

	out, err := s.GetLastCommittedBlockHeight(ctx, &services.GetLastCommittedBlockHeightInput{})
	if err != nil {
		return err
	}
	_lastCommittedBlockHeight := out.LastCommittedBlockHeight

	s.logger.Info("received block sync request",
		log.Stringable("petitioner", message.Sender.SenderPublicKey()),
		log.Stringable("first-requested-block-height", firstRequestedBlockHeight),
		log.Stringable("last-requested-block-height", lastRequestedBlockHeight),
		log.Stringable("last-committed-block-height", _lastCommittedBlockHeight))

	if _lastCommittedBlockHeight <= firstRequestedBlockHeight {
		return errors.New("firstBlockHeight is greater or equal to _lastCommittedBlockHeight")
	}

	if firstRequestedBlockHeight-_lastCommittedBlockHeight > primitives.BlockHeight(s.config.BlockSyncBatchSize()-1) {
		lastRequestedBlockHeight = firstRequestedBlockHeight + primitives.BlockHeight(s.config.BlockSyncBatchSize()-1)
	}

	blocks, firstAvailableBlockHeight, lastAvailableBlockHeight, err := s.GetBlocks(firstRequestedBlockHeight, lastRequestedBlockHeight)
	if err != nil {
		return errors.Wrap(err, "block sync failed reading from block persistence")
	}

	s.logger.Info("sending blocks to another node via block sync",
		log.Stringable("petitioner", senderPublicKey),
		log.Stringable("first-available-block-height", firstAvailableBlockHeight),
		log.Stringable("last-available-block-height", lastAvailableBlockHeight))

	response := &gossiptopics.BlockSyncResponseInput{
		RecipientPublicKey: senderPublicKey,
		Message: &gossipmessages.BlockSyncResponseMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: s.config.NodePublicKey(),
			}).Build(),
			SignedChunkRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                blockType,
				FirstBlockHeight:         firstAvailableBlockHeight,
				LastBlockHeight:          lastAvailableBlockHeight,
				LastCommittedBlockHeight: _lastCommittedBlockHeight,
			}).Build(),
			BlockPairs: blocks,
		},
	}
	_, err = s.gossip.SendBlockSyncResponse(ctx, response)
	return err
}
