#!/bin/bash

# Set the fixed minimum tx bytes
minTxBytes=30000000
# minTxBytes=1125000000

# Loop through the specified ranges
for startBlock in 12820000; do
    for compressionAlgo in "brotli"; do
        for brotliQuality in 1 2 3 4 5 6 7 8 9 10 11; do
            for brotliWindow in 10 11 12 13 14 15 16 17 18 19 24; do
                echo "Running with blobs=6, starting-block=$startBlock, minimum-tx-bytes=$minTxBytes, compressionAlgo=$compressionAlgo, brotliQuality=$brotliQuality, brotliWindow=$brotliWindow"
                CGO_CFLAGS='-I /app/compression/brotli/out/installed/include' CGO_LDFLAGS="-L/app/compression/brotli/out/installed/lib -lbrotlicommon" go run main.go -blobs 6 -starting-block $startBlock -minimum-tx-bytes $minTxBytes -compression-algo $compressionAlgo -brotli-quality $brotliQuality -brotli-window $brotliWindow
            done
        done
    done
done
