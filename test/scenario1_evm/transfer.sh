cd $CMC

./cmc client contract user invoke \
--contract-name=ERC20 \
--abi-file-path=../../testdata/erc20.abi \
--method=transfer \
--sdk-conf-path=../config/sdk_config.yml \
--params="[{\"address\": \"0x04dd97bbf0b8dca0e9c7c867591903b52fbdf586\"},{\"uint256\": \"10000000000000000\"}]" \
--sync-result=true