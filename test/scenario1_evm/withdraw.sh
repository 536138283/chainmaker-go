cd $CMC

echo "B调用withdraw合约，提款10"
./cmc client contract user invoke \
--contract-name=withdraw \
--abi-file-path=../../testdata/withdraw.abi \
--method=withdraw \
--sdk-conf-path=../config/sdk_config.yml \
--user-signkey-file-path=../config/wx-org1.chainmaker.org/certs/user/admin1/admin1.sign.key \
--user-signcrt-file-path=../config/wx-org1.chainmaker.org/certs/user/admin1/admin1.sign.crt \
--params="[{\"address\": \"44924fd364217285e4cad818292c7ac37c0a345b\"},{\"uint256\": \"10\"}]" \
--sync-result=true
