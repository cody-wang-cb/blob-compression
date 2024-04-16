package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-batcher/compressor"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rlp"
)

const ONEBLOB = eth.MaxBlobDataSize

var channelConfig = batcher.ChannelConfig{
	SeqWindowSize:      3600, // from base deploy script json
	ChannelTimeout:     300,  // from base deploy script json
	MaxChannelDuration: 600,  // 2 hrs
	SubSafetyMargin:    4,
	MaxFrameSize:       ONEBLOB - 1, // default 1 blob
	CompressorConfig: compressor.Config{
		ApproxComprRatio: 0.4,
		Kind:             "shadow",
	},
	BatchType: derive.SpanBatchType, // use SpanBatchType after Delta fork
}

func u64Ptr(v uint64) *uint64 {
	return &v
}

var rollupConfig = rollup.Config{
	Genesis:     rollup.Genesis{L2: eth.BlockID{Number: 0}},
	L2ChainID:   big.NewInt(8453),
	EcotoneTime: u64Ptr(1710374401),
}

// Note: have to override the channel definition to make it work
func buildChannelBuilder(numberOfBlobs int, compressionAlgo string, brotliQuality int, brotliWindow int) *batcher.ChannelBuilder {
	channelConfig := channelConfig
	channelConfig.MaxFrameSize = uint64((eth.MaxBlobDataSize - 1) * numberOfBlobs)
	channelConfig.MultiFrameTxs = true
	channelConfig.TargetNumFrames = 6
	channelConfig.CompressorConfig.CompressionAlgo = compressionAlgo
	channelConfig.CompressorConfig.TargetOutputSize = uint64(ONEBLOB * numberOfBlobs)
	channelConfig.CompressorConfig.BrotliQuality = brotliQuality
	channelConfig.CompressorConfig.BrotliWindow = brotliWindow
	cb, err := batcher.NewChannelBuilder(channelConfig, rollupConfig, 10)
	if err != nil {
		log.Fatal(err)
	}

	return cb
}

func calculateTxBytes(block *types.Block) int {
	totalTxSize := 0
	for _, tx := range block.Transactions() {
		// ignore deposit type
		if tx.Type() == types.DepositTxType {
			continue
		}
		txData, err := rlp.EncodeToBytes(tx)
		if err != nil {
			panic(err)
		}
		totalTxSize += len(txData)
	}
	return totalTxSize
}

func main() {
	var numberOfBlobs int
	var startBlock int
	var minimumTxBytes int
	var compressionAlgo string
	var brotliQuality int
	var brotliWindow int

	flag.IntVar(&numberOfBlobs, "blobs", 6, "Number of blobs to compress")
	flag.IntVar(&startBlock, "starting-block", 11443817, "Starting block number")
	flag.IntVar(&minimumTxBytes, "minimum-tx-bytes", 450000000, "Minimum number of tx bytes to compress")
	flag.StringVar(&compressionAlgo, "compression-algo", "zlib", "Compression algorithm to use")
	flag.IntVar(&brotliQuality, "brotli-quality", 6, "Brotli quality")
	flag.IntVar(&brotliWindow, "brotli-window", 22, "Brotli window size")

	flag.Parse()

	fmt.Println("Starting block: ", startBlock)
	fmt.Println("Number of blobs: ", numberOfBlobs)
	fmt.Println("Minimum tx bytes: ", minimumTxBytes)
	fmt.Println("Compression algo: ", compressionAlgo)
	fmt.Println("Brotli quality: ", brotliQuality)
	fmt.Println("Brotli window: ", brotliWindow)

	// Open the file for writing
	file, err := os.OpenFile("results.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize the channel builder
	cb := buildChannelBuilder(numberOfBlobs, compressionAlgo, brotliQuality, brotliWindow)

	// Connect to the local geth node
	clientLocation := "/data/geth.ipc"
	client, err := ethclient.Dial(clientLocation)
	if err != nil {
		// Cannot connect to local node for some reason
		log.Fatal(err)
	}

	totalProcessedTxSize := 0
	totalBlocks := 0
	var i int;
	for i = startBlock; totalProcessedTxSize < minimumTxBytes; i++ {
		// If we encounter an error (channel full), output the frames and print the total size of the frames
		// fmt.Println(cb.OutputBytes(), cb.InputBytes(), cb.ReadyBytes(), cb.PendingFrames())
		block, err := client.BlockByNumber(context.Background(), big.NewInt(int64(i)))
		if err != nil {
			log.Fatal(err)
		}
		_, err = cb.AddBlock(block)
		if err != nil {
			fmt.Println("Channel full, outputting frames")
			fmt.Println("Processed tx size ", totalProcessedTxSize)
			fmt.Println("Number of block processed ", i-startBlock)
			cb.OutputFrames()
			cb.Reset()
			i--
			continue
		}
		// Calculate the total size of all non-deposit transactions
		totalProcessedTxSize += calculateTxBytes(block)
	}

	// close the channel so that the last batches are compressed and ready to be outputted
	cb.Close()
	cb.OutputFrames()
	cb.Reset()
	// Get all the outputted frame size
	totalFrameSize := cb.OutputBytes()
	totalBlocks = i - startBlock
	fmt.Println("total frames size: ", totalFrameSize)
	fmt.Println("total tx size: ", totalProcessedTxSize)
	fmt.Println("compression ratio: ", float64(totalFrameSize)/float64(totalProcessedTxSize))

	if compressionAlgo == "brotli" {
		resultString := fmt.Sprintf("[%s] Starting block: %d\nNumber of blobs: %d\nMinimum tx bytes: %d\nTotal frames size: %d\nTotal tx size: %d\nCompression ratio: %f\nCompression Algo: %s\nTotal Blocks: %d\nBrotli Quality: %d\nBrotli Window: %d\n\n", time.Now().Format(time.RFC3339), startBlock, numberOfBlobs, minimumTxBytes, totalFrameSize, totalProcessedTxSize, float64(totalFrameSize)/float64(totalProcessedTxSize), compressionAlgo, totalBlocks, brotliQuality, brotliWindow)
		file.WriteString(resultString)
	} else {
		resultString := fmt.Sprintf("[%s] Starting block: %d\nNumber of blobs: %d\nMinimum tx bytes: %d\nTotal frames size: %d\nTotal tx size: %d\nCompression ratio: %f\nCompression Algo: %s\nTotal Blocks: %d\n\n", time.Now().Format(time.RFC3339), startBlock, numberOfBlobs, minimumTxBytes, totalFrameSize, totalProcessedTxSize, float64(totalFrameSize)/float64(totalProcessedTxSize), compressionAlgo, totalBlocks)
		file.WriteString(resultString)
	}


	defer client.Close()
}