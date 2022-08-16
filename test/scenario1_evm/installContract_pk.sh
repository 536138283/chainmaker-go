cd $CMC
./cmc client contract user create \
--abi-file-path=../../testdata/erc20.abi \
--contract-name=ERC20 \
--runtime-type=EVM \
--byte-code-path=../../testdata/erc20.bin \
--version=1.0 \
--sdk-conf-path=../config/sdk_config.yml \
--sync-result=true



./cmc client contract user create \
--abi-file-path=../../testdata/withdraw.abi \
--contract-name=withdraw \
--runtime-type=EVM \
--byte-code-path=../../testdata/withdraw.bin \
--version=1.0 \
--sdk-conf-path=../config/sdk_config.yml \
--sync-result=true