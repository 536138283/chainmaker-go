cd $CMC

./cmc gas recharge \
--sdk-conf-path=../config/sdk_config.yml \
--address=4aab4aaad419e5cf5e145bf9ba854c17f2ff7955 \
--amount=1000000000 \
--sync-result=true


./cmc client contract user create \
--abi-file-path=../../testdata/erc20.abi \
--contract-name=ERC20Go \
--runtime-type=EVM \
--byte-code-path=../../testdata/erc20.bin \
--version=1.0 \
--sdk-conf-path=../config/sdk_config.yml \
--gas-limit=99999999 \
--sync-result=true



./cmc client contract user create \
--abi-file-path=../../testdata/withdraw.abi \
--contract-name=withdraw \
--runtime-type=EVM \
--byte-code-path=../../testdata/withdraw.bin \
--version=1.0 \
--sdk-conf-path=../config/sdk_config.yml \
--gas-limit=99999999 \
--sync-result=true