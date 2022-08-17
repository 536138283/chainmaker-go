"""
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

"""

import subprocess
import logging
import sys
import os
from config.public_import import *

# 连接Linux，输入Linux命令，可执行Linux命令


# 加入日志
# 获取logger实例
logger = logging.getLogger("baseSpider")
# 指定输出格式
formatter = logging.Formatter('%(asctime)s\
              %(levelname)-8s:%(message)s')
# 文件日志
file_handler = logging.FileHandler("operation_theServer.log")
file_handler.setFormatter(formatter)
# 控制台日志
console_handler = logging.StreamHandler(sys.stdout)
console_handler.setFormatter(formatter)

# 为logge添加具体的日志处理器
logger.addHandler(file_handler)
logger.addHandler(console_handler)

logger.setLevel(logging.INFO)


class TheServerHelper():
    def __init__(self, remote, local_dir='', ftpType='', port=22):

        self.ftpType = ftpType
        self.remote = remote
        self.local_dir = local_dir

    # SSH连接服务器，用于命令执行
    def ssh_connectionServer(self, inputs=[]):
        try:
            # 创建SSH对象
            proc=subprocess.run(self.remote,stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE, shell=True)
            stdout_result = proc.stdout
            stderr_result = proc.stderr
            result = stderr_result if stderr_result else stdout_result
        except Exception as e:
            print(e)
            logger.error("SSHConnection" + self.serverIP + "failed!")

            return False

        # 关闭连接
        print("______执行结果_________")
        linux_result = result.decode()
        return linux_result



if __name__ == "__main__":
    # path = r'/data/go/src/chainmaker.org/chainmaker-go/tools/cmc/common_tools.sh'
    # tsh = TheServerHelper(f'if [ -e {path} ]; then echo "{path} exists";fi')
    # print(tsh.ssh_connectionServer())
    print(TheServerHelper('ls').ssh_connectionServer())
