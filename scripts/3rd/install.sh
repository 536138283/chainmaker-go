#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

tar -zxvf glibc-2.18.tar.gz
cd glibc-2.18
mkdir build
cd build
# Adapt to gmake version 4.x
sed -i "s/3.\[89\]\*./3.[89]* | 4.* )/g" ../configure
../configure --prefix=/usr
make -j4
sudo make install
