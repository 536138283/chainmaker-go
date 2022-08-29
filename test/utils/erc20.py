"""
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

"""

import json

from utils.cmc_tools_contract import ContractDeal


class Erc20(object):


    def __init__(self,contract_name,abi=None, sync_result=True, sdk_config=None):
        self.contract_name=contract_name
        self.sdk_config = sdk_config
        self.contract=ContractDeal(contract_name, sync_result=sync_result)
        self.pTransfer="{\"to\": \"%s\",\"amount\": \"%d\"}"
        self.pBalanceOf="{\"account\":\"%s\"}"
        self.pApprove="{\"spender\": \"%s\",\"amount\": \"%d\"}"
        self.pTransferFrom="{\"owner\": \"%s\",\"to\": \"%s\",\"amount\": \"%d\"}"
        self.pAllowance="{\"spender\":\"%s\",\"owner\":\"%s\"}"

        self.abi=abi
        if abi:
            self.pTransfer="[{\"address\": \"%s\"},{\"uint256\": \"%d\"}]"
            self.pBalanceOf="[{\"address\":\"%s\"}]"
            self.pApprove="[{\"address\": \"%s\"},{\"uint256\": \"%d\"}]"
            self.pTransferFrom="[{\"address\": \"%s\"},{\"address\": \"%s\"},{\"uint256\": \"%d\"}]"
            self.pAllowance="[{\"address\":\"%s\"},{\"address\":\"%s\"}]"

    def transfer(self, to, amount):
        self.contract.invoke("transfer", self.pTransfer.format(to, amount),
                             sdk_config=self.sdk_config,abi=self.abi)

    def balanceOf(self,addr):
        result = self.contract.get("balanceOf", self.pBalanceOf.format( addr),
                                   sdk_config=self.sdk_config, abi=self.abi)
        balance = json.loads(result).get("contract_result").get("result")
        return balance[1:-1]

    def approve(self, spender, amount):
        self.contract.invoke("approve", self.pApprove.format (spender, amount),
                             sdk_config=self.sdk_config,abi=self.abi)

    def transferFrom(self,_from, to, amount):
        self.contract.invoke("transferFrom", self.pTransferFrom.format (_from,to, amount),
                             sdk_config=self.sdk_config,abi=self.abi)

    def allowance(self,owner,spender):
        result = self.contract.get("allowance", self.pAllowance.format (owner,spender),
                                   sdk_config=self.sdk_config, abi=self.abi)
        allowance_amt = json.loads(result).get("contract_result").get("result")
        return allowance_amt[1:-1]