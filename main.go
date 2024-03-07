package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-batcher/compressor"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rlp"
	"log"
	"math/big"
	"os"
	"time"
)

const ONEBLOB = 130044

var channelConfig = batcher.ChannelConfig{
	SeqWindowSize:      3600, // from base deploy script json
	ChannelTimeout:     300,  // from base deploy script json
	MaxChannelDuration: 600,  // 2 hrs
	SubSafetyMargin:    4,
	MaxFrameSize:       ONEBLOB, // default 1 blob
	CompressorConfig: compressor.Config{
		TargetFrameSize:  ONEBLOB, // default 1 blob
		TargetNumFrames:  1,
		ApproxComprRatio: 0.4,
		Kind:             "shadow",
	},
	BatchType: derive.SpanBatchType, // use SpanBatchType after Delta fork
}

var rollupConfig = rollup.Config{
	Genesis:   rollup.Genesis{L2: eth.BlockID{Number: 0}},
	L2ChainID: big.NewInt(8453),
}

// Note: have to override the channel definition to make it work
func buildChannelBuilder(numberOfBlobs int) *batcher.ChannelBuilder {
	channelConfig := channelConfig
	channelConfig.MaxFrameSize = uint64(ONEBLOB * numberOfBlobs)
	channelConfig.CompressorConfig.TargetFrameSize = uint64(ONEBLOB * numberOfBlobs)
	cb, err := batcher.NewChannelBuilder(channelConfig, rollupConfig)
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

	flag.IntVar(&numberOfBlobs, "blobs", 1, "Number of blobs to compress")
	flag.IntVar(&startBlock, "starting-block", 11443817, "Starting block number")
	flag.IntVar(&minimumTxBytes, "minimum-tx-bytes", 4500000, "Minimum number of tx bytes to compress")

	flag.Parse()

	fmt.Println("Starting block: ", startBlock)
	fmt.Println("Number of blobs: ", numberOfBlobs)
	fmt.Println("Minimum tx bytes: ", minimumTxBytes)

	// Open the file for writing
	file, err := os.OpenFile("results.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize the channel builder
	cb := buildChannelBuilder(numberOfBlobs).Cb

	// Connect to the local geth node
	clientLocation := "/data/geth.ipc"
	client, err := ethclient.Dial(clientLocation)
	if err != nil {
		// Cannot connect to local node for some reason
		log.Fatal(err)
	}

	totalOutputtedTxSize := 0
	totalProcessedTxSize := 0
	numBlockProcessed := 0
	for i := startBlock; totalProcessedTxSize < minimumTxBytes; i++ {
		block, err := client.BlockByNumber(context.Background(), big.NewInt(int64(i)))
		if err != nil {
			log.Fatal(err)
		}
		_, err = cb.AddBlock(block)
		// If we encounter an error (channel full), output the frames and print the total size of the frames
		if err != nil {
			fmt.Println("Channel full, outputting frames")
			fmt.Println("Processed tx size ", totalProcessedTxSize)
			fmt.Println("Number of block processed ", i-startBlock)
			numBlockProcessed = i - startBlock
			cb.OutputFrames()
			cb.Reset()
			// Update total tx size
			totalOutputtedTxSize = totalProcessedTxSize
			i -= 1
			continue
		}
		// Calculate the total size of all non-deposit transactions
		totalProcessedTxSize += calculateTxBytes(block)
	}

	// Get all the outputted frame size
	totalFrameSize := cb.OutputBytes()
	fmt.Println("total frames size: ", totalFrameSize)
	fmt.Println("total tx size: ", totalOutputtedTxSize)
	fmt.Println("compression ratio: ", float64(totalFrameSize)/float64(totalOutputtedTxSize))

	resultString := fmt.Sprintf("[%s] Starting block: %d\nNumber of blobs: %d\nMinimum tx bytes: %d\nTotal frames size: %d\nTotal tx size: %d\nCompression ratio: %f\nNumber block processed: %d\n\n", time.Now().Format(time.RFC3339), startBlock, numberOfBlobs, minimumTxBytes, totalFrameSize, totalOutputtedTxSize, float64(totalFrameSize)/float64(totalOutputtedTxSize), numBlockProcessed)
	file.WriteString(resultString)

	defer client.Close()
}
