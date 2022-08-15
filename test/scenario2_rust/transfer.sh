cd $CMC

./cmc client contract user invoke \
--contract-name=ERC20 \
--abi-file-path=../../testdata/erc20.abi \
--method=transfer \
--sdk-conf-path=../config/sdk_config.yml \
--params="[{\"address\": \"0xebe30584c91d648adadfb746d56c0d38dcb2d262\"},{\"uint256\": \"10000000000000000\"}]" \
--sync-result=true