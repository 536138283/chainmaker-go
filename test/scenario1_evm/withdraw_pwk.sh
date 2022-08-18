cd $CMC

echo "B调用withdraw合约，提款10"
./cmc client contract user invoke \
--contract-name=withdraw \
--abi-file-path=../../testdata/withdraw.abi \
--method=withdraw \
--sdk-conf-path=../config/sdk_config.yml \
--user-signkey-file-path=../config/wx-org2.chainmaker.org/keys/user/admin/admin.key \
--org-id=wx-org2.chainmaker.org \
--params="[{\"address\": \"44924fd364217285e4cad818292c7ac37c0a345b\"},{\"uint256\": \"10\"}]" \
--sync-result=true
