- 使用预生成inputdata压测solidity合约
```shell
./cmc parallel invoke \
--loopNum=10000 \
--printTime=5 \
--threadNum=10 \
--timeout=100000 \
--sleepTime=100 \
--climbTime=5 \
--use-tls=true \
--contract-name=1 \
--method=pressCreateDid \
--org-IDs=wx-org1.chainmaker.org \
--hosts="127.0.0.1:12301" \
--user-keys=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
--user-crts=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
--sign-keys=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key \
--sign-crts=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
--org-ids=wx-org1.chainmaker.org \
--ca-path="./testdata/crypto-config/wx-org1.chainmaker.org/ca" \
--input-data-file=./input.data
```