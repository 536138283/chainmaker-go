#!/usr/bin/env bash
#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

set -x
BRANCH=$1
if [[ ! -n $BRANCH ]]; then
				  BRANCH="v3.0.0_qc"
fi
cd ..

go get chainmaker.org/chainmaker/chainconf/v3@${BRANCH}
go get chainmaker.org/chainmaker/common/v3@${BRANCH}
go get chainmaker.org/chainmaker/consensus-maxbft/v3@${BRANCH}
go get chainmaker.org/chainmaker/consensus-abft/v3@${BRANCH}
go get chainmaker.org/chainmaker/consensus-dpos/v3@${BRANCH}
go get chainmaker.org/chainmaker/consensus-raft/v3@${BRANCH}
go get chainmaker.org/chainmaker/consensus-solo/v3@${BRANCH}
go get chainmaker.org/chainmaker/consensus-tbft/v3@${BRANCH}
go get chainmaker.org/chainmaker/consensus-utils/v3@${BRANCH}
go get chainmaker.org/chainmaker/localconf/v3@${BRANCH}
go get chainmaker.org/chainmaker/logger/v3@${BRANCH}
go get chainmaker.org/chainmaker/net-common@v1.3.0_qc
go get chainmaker.org/chainmaker/net-libp2p@v1.3.0_qc
go get chainmaker.org/chainmaker/net-liquid@v1.3.0_qc
go get chainmaker.org/chainmaker/libp2p-pubsub@v1.3.0_qc
go get chainmaker.org/chainmaker/pb-go/v3@${BRANCH}
go get chainmaker.org/chainmaker/protocol/v3@${BRANCH}
go get chainmaker.org/chainmaker/sdk-go/v3@${BRANCH}
go get chainmaker.org/chainmaker/store/v3@${BRANCH}
go get chainmaker.org/chainmaker/store-huge/v3@${BRANCH}
go get chainmaker.org/chainmaker/txpool-batch/v3@${BRANCH}
go get chainmaker.org/chainmaker/txpool-normal/v3@${BRANCH}
go get chainmaker.org/chainmaker/txpool-single/v3@${BRANCH}
go get chainmaker.org/chainmaker/utils/v3@${BRANCH}
go get chainmaker.org/chainmaker/vm-docker-go/v3@${BRANCH}
go get chainmaker.org/chainmaker/vm-engine/v3@${BRANCH}
go get chainmaker.org/chainmaker/vm-native/v3@${BRANCH}
go get chainmaker.org/chainmaker/vm-evm/v3@${BRANCH}
go get chainmaker.org/chainmaker/vm-gasm/v3@${BRANCH}
go get chainmaker.org/chainmaker/vm-wasmer/v3@${BRANCH}
go get chainmaker.org/chainmaker/vm-wxvm/v3@${BRANCH}
go get chainmaker.org/chainmaker/vm/v3@${BRANCH}

go mod tidy

make
make cmc
