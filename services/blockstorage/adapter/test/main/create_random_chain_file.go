package main

import (
	"flag"
	"fmt"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter/test"
	testUtils "github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"time"
)

type adHocLogger string

func (l *adHocLogger) Log(args ...interface{}) {
	fmt.Println(args...)
}
func (l *adHocLogger) Name() string {
	return string(*l)
}

func main() {
	dir, virtualChain, targetHeight := parseParams()

	logger := adHocLogger("")
	rand := testUtils.NewControlledRand(&logger)
	conf := &randomChainLocalConfig{dir: dir, virtualChainId: virtualChain}

	start := time.Now()
	fmt.Printf("\nusing:\noutput directory: %s\nvirtual chain id: %d\n\nloading adapter and building index...\n", conf.BlockStorageDataDir(), conf.VirtualChainId())

	adapter, release, err := test.NewFilesystemAdapterDriver(conf)
	if err != nil {
		panic(err)
	}
	defer release()

	currentHeight, err := adapter.GetLastBlockHeight()
	if err != nil {
		panic(err)
	}
	prevBlock, err := adapter.GetLastBlock()
	if err != nil {
		panic(err)
	}

	fmt.Printf("indexed %d blocks in %v\n", currentHeight, time.Now().Sub(start))

	for currentHeight < targetHeight {
		nextHeight := currentHeight + 1
		block := builders.RandomizedBlock(nextHeight, rand, prevBlock)

		err := adapter.WriteNextBlock(block)
		if err != nil {
			logger.Log("error writing block to file at height %d. error %s", nextHeight, err)
			panic(err)
		}
		if nextHeight%1000 == 0 {
			logger.Log(fmt.Sprintf("wrote height %d", nextHeight))
		}

		currentHeight++
		prevBlock = block
	}

	fmt.Printf("\n\nblocks file in %s/ now has %d blocks\n\n", conf.BlockStorageDataDir(), currentHeight)
}

func parseParams() (dir string, vchain primitives.VirtualChainId, height primitives.BlockHeight) {
	intHeight := flag.Uint64("height", 100, "target height for blocks file")
	outputDir := flag.String("output", "./gen_data", "target directory for new block file")
	virtualChain := flag.Uint("vchain", 42, "blocks file virtual chain id")
	flag.Parse()
	fmt.Printf("usage: [-output output_folder_name] [-height target_block_height] [-vchain vchain_id]\n\n")
	targetHeight := primitives.BlockHeight(*intHeight)
	return *outputDir, primitives.VirtualChainId(*virtualChain), targetHeight
}

type randomChainLocalConfig struct {
	dir            string
	virtualChainId primitives.VirtualChainId
}

func (l *randomChainLocalConfig) VirtualChainId() primitives.VirtualChainId {
	return l.virtualChainId
}

func (l *randomChainLocalConfig) BlockStorageDataDir() string {
	return l.dir
}

func (l *randomChainLocalConfig) BlockStorageMaxBlockSize() uint32 {
	return 1000000000
}
