cd $CMC

echo "A转账给B"
./cmc client contract user invoke \
--contract-name=ERC20 \
--abi-file-path=../../testdata/erc20.abi \
--method=transfer \
--sdk-conf-path=../config/sdk_config.yml \
--params="[{\"address\": \"0x04dd97bbf0b8dca0e9c7c867591903b52fbdf586\"},{\"uint256\": \"100\"}]" \
--gas-limit=99999999 \
--sync-result=true

echo "A转账给withdraw合约"
./cmc client contract user invoke \
--contract-name=ERC20 \
--abi-file-path=../../testdata/erc20.abi \
--method=transfer \
--sdk-conf-path=../config/sdk_config.yml \
--params="[{\"address\": \"0xd6d64458dc76d02482052bfb8a5b33a72c054c77\"},{\"uint256\": \"200\"}]" \
--gas-limit=99999999 \
--sync-result=true