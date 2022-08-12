cd $CMC
echo "A->B转账"
nohup ./cmc client contract user invoke \
--contract-name=ERC20 \
--abi-file-path=../../testdata/erc20.abi \
--method=transfer \
--sdk-conf-path=../config/sdk_config.yml \
--params="[{\"address\": \"0xebe30584c91d648adadfb746d56c0d38dcb2d262\"},{\"uint256\": \"10000000000000000\"}]" \
--sync-result=true &
echo "A->C转账"
nohup ./cmc client contract user invoke \
--contract-name=ERC20 \
--abi-file-path=../../testdata/erc20.abi \
--method=transfer \
--sdk-conf-path=../config/sdk_config.yml \
--params="[{\"address\": \"0xdd04921b54448fb7afb6dd13fe1b2c36ecb7f657\"},{\"uint256\": \"2\"}]" \
--sync-result=true &
echo "A->D转账"
nohup ./cmc client contract user invoke \
--contract-name=ERC20 \
--abi-file-path=../../testdata/erc20.abi \
--method=transfer \
--sdk-conf-path=../config/sdk_config.yml \
--params="[{\"address\": \"0xd6ebce36c3bb5fe42c439e907ff9439b048702f0\"},{\"uint256\": \"3\"}]" \
--sync-result=true &