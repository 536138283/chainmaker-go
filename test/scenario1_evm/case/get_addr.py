"""
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

"""

import json
import sys
sys.path.append("..")

from utils.cmc_tools_query import query_address


def get_user_addr(org, user):
    result_addr = query_address(org, user)
    address = result_addr.get("ethereum").get("address")
    return address


def get_user_balance(result):
    balance = json.loads(result).get("contract_result").get("result")
    return balance


if __name__ == "__main__":
    get_user_addr("1", "client1")
