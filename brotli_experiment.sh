#!/bin/bash

# Set the fixed minimum tx bytes
# minTxBytes=30000000
minTxBytes=1125000000

# Loop through the specified ranges
for startBlock in 12820000 12860000 12900000; do
    for compressionAlgo in "brotli"; do
        for brotliQuality in 9 10; do
            for brotliWindow in 23; do
                echo "Running with blobs=6, starting-block=$startBlock, minimum-tx-bytes=$minTxBytes, compressionAlgo=$compressionAlgo, brotliQuality=$brotliQuality, brotliWindow=$brotliWindow"
                # Add flags if the algo is brotli
                if [ $compressionAlgo == "brotli" ]; then
                    CGO_CFLAGS='-I /app/compression/brotli/out/installed/include' CGO_LDFLAGS="-L/app/compression/brotli/out/installed/lib -lbrotlicommon" go run main.go -blobs 6 -starting-block $startBlock -minimum-tx-bytes $minTxBytes -compression-algo $compressionAlgo -brotli-quality $brotliQuality -brotli-window $brotliWindow
                else
                    go run main.go -blobs 6 -starting-block $startBlock -minimum-tx-bytes $minTxBytes -compression-algo $compressionAlgo -brotli-quality $brotliQuality -brotli-window $brotliWindow
                fi
            done
        done
    done
done
