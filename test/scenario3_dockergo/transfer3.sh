cd $CMC

echo "A->B转账"
nohup ./cmc client contract user invoke \
--contract-name=ERC20 \
--abi-file-path=../../testdata/erc20.abi \
--method=transfer \
--sdk-conf-path=../config/sdk_config.yml \
--params="[{\"address\": \"0x04dd97bbf0b8dca0e9c7c867591903b52fbdf586\"},{\"uint256\": \"1\"}]" \
--sync-result=true &

echo "A->C转账"
nohup ./cmc client contract user invoke \
--contract-name=ERC20 \
--abi-file-path=../../testdata/erc20.abi \
--method=transfer \
--sdk-conf-path=../config/sdk_config.yml \
--params="[{\"address\": \"0xe48d57bc2570355ed4b039d64705126f4130acad\"},{\"uint256\": \"2\"}]" \
--sync-result=true &

echo "A->D转账"
nohup ./cmc client contract user invoke \
--contract-name=ERC20 \
--abi-file-path=../../testdata/erc20.abi \
--method=transfer \
--sdk-conf-path=../config/sdk_config.yml \
--params="[{\"address\": \"0x1ae351b356e9312fb941f6e5186ed97bcc5d567d\"},{\"uint256\": \"3\"}]" \
--sync-result=true &