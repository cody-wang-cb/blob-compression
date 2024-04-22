#!/bin/bash

# Set the fixed minimum tx bytes
# minTxBytes=30000000
minTxBytes=1125000000


chain="OP"
for startBlock in 117500000 118200000; do
    for compressionAlgo in "brotli"; do
        for brotliQuality in 11; do
            for brotliWindow in 23; do
                echo "Running with blobs=6, starting-block=$startBlock, minimum-tx-bytes=$minTxBytes, compressionAlgo=$compressionAlgo, brotliQuality=$brotliQuality, brotliWindow=$brotliWindow"
                # Add flags if the algo is brotli
                if [ $compressionAlgo == "brotli" ]; then
                    CGO_CFLAGS='-I /app/blob-compression/brotli/out/installed/include' CGO_LDFLAGS="-L/app/blob-compression/brotli/out/installed/lib -lbrotlicommon" go run main.go -blobs 6 -starting-block $startBlock -minimum-tx-bytes $minTxBytes -compression-algo $compressionAlgo -brotli-quality $brotliQuality -brotli-window $brotliWindow -chain $chain
                else
                    go run main.go -blobs 6 -starting-block $startBlock -minimum-tx-bytes $minTxBytes -compression-algo $compressionAlgo -brotli-quality $brotliQuality -brotli-window $brotliWindow -chain $chain
                fi
            done
        done
    done
done
