cd $CMC

./cmc gas recharge \
--sdk-conf-path=../config/sdk_config.yml \
--address=4535541d506884371f25246c1badc96c3d9267c7 \
--amount=1000000000 \
--sync-result=true


echo "B调用withdraw合约，提款10"
./cmc client contract user invoke \
--contract-name=withdraw \
--abi-file-path=../../testdata/withdraw.abi \
--method=withdraw \
--sdk-conf-path=../config/sdk_config.yml \
--user-signkey-file-path="../config/node2/admin/admin2/admin2.key" \
--params="[{\"address\": \"44924fd364217285e4cad818292c7ac37c0a345b\"},{\"uint256\": \"10\"}]" \
--gas-limit=99999999 \
--sync-result=true
