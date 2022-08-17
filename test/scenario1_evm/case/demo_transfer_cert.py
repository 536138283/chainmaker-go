"""
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

"""

import json
import sys
sys.path.append("..")


from case.get_addr import get_user_addr, get_user_balance
from utils.cmc_tools_contract_deal_by_all import ContractDeal


def create_invoke_by_withdraw_erc20():
    print("ERC20合约安装".center(50, "="))
    cd_erc = ContractDeal("ERC20", sync_result=True)
    result_erc = cd_erc.create("EVM", "erc20.bin")
    erc_address = json.loads(result_erc).get("contract_result").get("result").get("address")
    print(erc_address)

    print("withdraw合约安装".center(50, "="))
    cd_withdraw = ContractDeal("withdraw", sync_result=True)
    result_withdraw = cd_withdraw.create("EVM", "withdraw.bin")
    withdraw_address = json.loads(result_withdraw).get("contract_result").get("result").get("address")
    print(withdraw_address)

    print("query UserA address: org1 client1".center(50, "="))
    user_a_address = get_user_addr("1", "client1")
    print("query UserB address: org1 admin1".center(50, "="))
    user_b_address = get_user_addr("1", "admin1")
    print("query UserC address: org2 client1".center(50, "="))
    user_c_address = get_user_addr("2", "client1")
    print("query UserD address: org2 admin1".center(50, "="))
    user_d_address = get_user_addr("2", "admin1")

    print(user_a_address, user_b_address, user_c_address, user_d_address)
    print("A转账给B".center(50, "="))
    cd_erc.invoke("transfer", r"[{\"address\": \"%s\"},{\"uint256\": \"100\"}]" % user_b_address,
                  abi="erc20.abi")
    print("A转账给withdraw合约".center(50, "="))
    cd_erc.invoke("transfer", r"[{\"address\": \"%s\"},{\"uint256\": \"200\"}]" % withdraw_address,
                  abi="erc20.abi")

    print("UserB balance:".center(50, "="))
    balance_b_result = cd_erc.get("balanceOf", r"[{\"address\":\"%s\"}]" % user_b_address, abi="erc20.abi")
    balance_b = get_user_balance(balance_b_result)
    print("withdraw contract balance:".center(50, "="))
    balance_c_result = cd_erc.get("balanceOf", r"[{\"address\":\"%s\"}]" % withdraw_address, abi="erc20.abi")
    balance_c = get_user_balance(balance_c_result)

    print("B调用withdraw合约，提款10".center(50, "="))
    cd_withdraw.invoke("withdraw", r"[{\"address\": \"%s\"},{\"uint256\": \"10\"}]" % erc_address,
                  abi="withdraw.abi", signkey=r"wx-org1.chainmaker.org/certs/user/admin1/admin1.sign.key",
                  signcrt=r"wx-org1.chainmaker.org/certs/user/admin1/admin1.sign.crt")

    print("UserA balance:".center(50, "="))
    balance_a_result = cd_erc.get("balanceOf", r"[{\"address\":\"%s\"}]" % user_a_address, abi="erc20.abi")
    balance_a = get_user_balance(balance_a_result)
    print("UserB balance:".center(50, "="))
    balance_b_result = cd_erc.get("balanceOf", r"[{\"address\":\"%s\"}]" % user_b_address, abi="erc20.abi")
    balance_b = get_user_balance(balance_b_result)
    print("withdraw contract balance:".center(50, "="))
    balance_c_result = cd_erc.get("balanceOf", r"[{\"address\":\"%s\"}]" % withdraw_address, abi="erc20.abi")
    balance_c = get_user_balance(balance_c_result)

    return balance_a, balance_b, balance_c


if __name__ == "__main__":
    print(create_invoke_by_withdraw_erc20())
