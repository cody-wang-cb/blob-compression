#!/bin/bash

# Set the fixed minimum tx bytes
minTxBytes=60000000

# Loop through the specified ranges
for startBlock in 12840000 12860000 12880000; do
    for compressionAlgo in "zlib" "brotli"; do
        echo "Running with blobs=6, starting-block=$startBlock, minimum-tx-bytes=$minTxBytes, compressionAlgo=$compressionAlgo"
        go run main.go -blobs 6 -starting-block $startBlock -minimum-tx-bytes $minTxBytes -compression-algo $compressionAlgo
    done
done
