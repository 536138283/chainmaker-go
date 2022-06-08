#!/usr/bin/env bash
#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
function lint_check() {
  cd ${cm}/$1
  echo "cd ${cm}/$1"
  golangci-lint run ./...

  #计算注释覆盖率，需要安装gocloc： go install github.com/hhatto/gocloc/cmd/gocloc@latest
  comment_coverage=$(gocloc --include-lang=Go --output-type=json --not-match=".*_test\.go" . | jq '(.total.comment-.total.files*5)/(.total.code+.total.comment)*100')
  echo "注释率：${comment_coverage}%"
  # 如果测试覆盖率低于N，认为执行失败
  (( $(awk "BEGIN {print (${comment_coverage} >= $2)}") )) || (echo "$1 注释覆盖率: ${comment_coverage} 低于 $2%"; exit 1)
}
set -e

cm=$(pwd)

if [[ $cm == *"scripts" ]] ;then
  cm=$cm/..
fi

if [ -n "$1" ] ;then
  echo "check lint and comment cover: $1."
  lint_check "$1" 15
else
#   lint_check "module/accesscontrol" 15
   lint_check "module/blockchain" 15
  lint_check "module/consensus" 15
#  lint_check "module/core" 15
  lint_check "module/net" 12
  lint_check "module/rpcserver" 4
#  lint_check "module/snapshot" 15
#  lint_check "module/sync" 15
#  lint_check "module/txfilter" 15
  lint_check "module/subscriber" 10
  lint_check "module/txpool" 15
#  lint_check "tools/cmc" 15
fi
