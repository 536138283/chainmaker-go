module github.com/libp2p/go-libp2p-core

go 1.14

require (
	chainmaker.org/chainmaker-go/common v0.0.0
	github.com/btcsuite/btcd v0.21.0-beta
	github.com/coreos/go-semver v0.3.0
	github.com/gogo/protobuf v1.3.1
	github.com/ipfs/go-cid v0.0.7
	github.com/jbenet/goprocess v0.1.4
	github.com/libp2p/go-buffer-pool v0.0.2
	github.com/libp2p/go-flow-metrics v0.0.3
	github.com/libp2p/go-msgio v0.0.6
	github.com/libp2p/go-openssl v0.0.7
	github.com/minio/sha256-simd v0.1.1
	github.com/mr-tron/base58 v1.2.0
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/multiformats/go-multihash v0.0.14
	github.com/multiformats/go-varint v0.0.6
	github.com/tjfoc/gmsm v1.4.1
	go.opencensus.io v0.22.4
)

replace chainmaker.org/chainmaker-go/common => ../../../../common
