"""
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

"""

import json

from utils.cmc_tools_contract import ContractDeal


class Erc20(object):


    def __init__(self,contract_name,abi, sync_result=True, sdk_config=None):
        self.contract_name=contract_name
        self.sdk_config = sdk_config
        self.contract=ContractDeal(contract_name, sync_result=sync_result)
        self.abi=abi

    def transfer(self, to, amount):
        self.contract.invoke("transfer", r"[{\"address\": \"%s\"},{\"uint256\": \"%d\"}]" % (to, amount),
                      sdk_config=self.sdk_config,abi=self.abi)

    def balanceOf(self,addr):
        result = self.contract.get("balanceOf", r"[{\"address\":\"%s\"}]" % addr,
                                      sdk_config=self.sdk_config, abi=self.abi)
        balance = json.loads(result).get("contract_result").get("result")
        return balance[1:-1]

    def approve(self, spender, amount):
        self.contract.invoke("tranapprovesfer", r"[{\"address\": \"%s\"},{\"uint256\": \"%d\"}]" % (spender, amount),
                             sdk_config=self.sdk_config,abi=self.abi)

    def transferFrom(self,_from, to, amount):
        self.contract.invoke("transferFrom", r"[{\"address\": \"%s\"},{\"address\": \"%s\"},{\"uint256\": \"%d\"}]" % (_from,to, amount),
                             sdk_config=self.sdk_config,abi=self.abi)

    def allowance(self,owner,spender):
        result = self.contract.get("allowance", r"[{\"address\":\"%s\"},{\"address\":\"%s\"}]" % (owner,spender),
                                   sdk_config=self.sdk_config, abi=self.abi)
        allowance_amt = json.loads(result).get("contract_result").get("result")
        return allowance_amt[1:-1]