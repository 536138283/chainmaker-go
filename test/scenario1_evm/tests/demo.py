"""
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

"""

import sys
import unittest

sys.path.append("..")

from case.demo_transfer_cert import create_invoke_by_withdraw_erc20


class Test(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.balances = create_invoke_by_withdraw_erc20()

    def test_balance_a_compare(self):
        expect = "[999999999999999999999999700]"
        practical = self.balances[0]
        self.assertEqual(expect, practical, "success")

    def test_balance_b_compare(self):
        expect = "[110]"
        practical = self.balances[1]
        self.assertEqual(expect, practical, "success")

    def test_balance_c_compare(self):
        expect = "[190]"
        practical = self.balances[2]
        self.assertEqual(expect, practical, "success")


if __name__ == '__main__':
    # suite = unittest.TestSuite()
    # suite.addTest(Test('test_balance_a_compare'))
    # suite.addTest(Test('test_balance_b_compare'))
    # suite.addTest(Test('test_balance_c_compare'))
    unittest.main()
