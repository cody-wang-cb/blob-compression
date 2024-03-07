#!/bin/bash

# Set the fixed minimum tx bytes
minTxBytes=60000000

# Loop through the specified ranges
for startBlock in 11440000 11460000 11480000; do
    for blobs in {1..6}; do
        echo "Running with blobs=$blobs, starting-block=$startBlock, minimum-tx-bytes=$minTxBytes"
        go run main.go -blobs $blobs -starting-block $startBlock -minimum-tx-bytes $minTxBytes
    done
done
