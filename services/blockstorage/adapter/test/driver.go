package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TODO V1 TBD - do we want to fuss with simulating io errors? (tampering FS)
// TODO V1 can we detect errors that indicate we need to open a writing file handle?
// TODO V1 file format includes a file version, vchain id, network id, and if it doesn't match don't run!

const blocksFilename = "blocks"

func NewFilesystemAdapterDriver(conf config.FilesystemBlockPersistenceConfig) (adapter.BlockPersistence, func(), error) {
	ctx, cancelCtx := context.WithCancel(context.Background())

	persistence, err := adapter.NewFilesystemBlockPersistence(ctx, conf, log.GetLogger(), metric.NewRegistry())
	if err != nil {
		return nil, nil, err
	}

	closeAdapter := func() {
		cancelCtx()
		time.Sleep(500 * time.Millisecond) // time to release any lock
	}

	return persistence, closeAdapter, nil
}

type localConfig struct {
	dir string
}

func newTempFileConfig() *localConfig {
	dirName, err := ioutil.TempDir("", "contract_test_block_persist")
	if err != nil {
		panic(err)
	}
	return &localConfig{
		dir: dirName,
	}
}
func (l *localConfig) BlockStorageDataDir() string {
	return l.dir
}

func (l *localConfig) BlockStorageMaxBlockSize() uint32 {
	return 64 * 1024 * 1024
}

func (l *localConfig) cleanDir() {
	_ = os.RemoveAll(l.BlockStorageDataDir()) // ignore errors - nothing to do
}

func getFileSize(t *testing.T, conf *localConfig) int64 {
	blocksFile, err := os.Open(filepath.Join(conf.BlockStorageDataDir(), blocksFilename))
	require.NoError(t, err)
	blocksFileInfo2, err := blocksFile.Stat()
	require.NoError(t, err)
	err = blocksFile.Close()
	require.NoError(t, err)
	return blocksFileInfo2.Size()
}

func truncateFile(t *testing.T, conf *localConfig, size int64) {
	blocksFile, err := os.OpenFile(filepath.Join(conf.BlockStorageDataDir(), blocksFilename), os.O_RDWR, 0666)
	require.NoError(t, err)
	err = blocksFile.Truncate(size)
	require.NoError(t, err)
	err = blocksFile.Close()
	require.NoError(t, err)
}

func flipBitInFile(t *testing.T, conf *localConfig, offset int64, bitMask byte) {
	blocksFile, err := os.OpenFile(filepath.Join(conf.BlockStorageDataDir(), blocksFilename), os.O_RDWR, 0666)
	require.NoError(t, err)
	b := make([]byte, 1)
	n, err := blocksFile.ReadAt(b, offset)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
	b[0] = b[0] ^ bitMask
	n, err = blocksFile.WriteAt(b, offset)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
	err = blocksFile.Close()
	require.NoError(t, err)
}

func writeRandomBlocksToFile(t *testing.T, conf *localConfig, numBlocks int32, ctrlRand *test.ControlledRand) []*protocol.BlockPairContainer {
	fsa, closeAdapter, err := NewFilesystemAdapterDriver(conf)
	require.NoError(t, err)
	defer closeAdapter()

	blockChain := builders.RandomizedBlockChain(numBlocks, ctrlRand)

	for _, block := range blockChain {
		err = fsa.WriteNextBlock(block)
		require.NoError(t, err)
	}
	return blockChain
}
