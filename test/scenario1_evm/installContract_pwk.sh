cd $CMC
./cmc client contract user create \
--abi-file-path=../../testdata/erc20.abi \
--contract-name=ERC20 \
--runtime-type=EVM \
--byte-code-path=../../testdata/erc20.bin \
--version=1.0 \
--sdk-conf-path=../config/sdk_config.yml \
--admin-key-file-paths=../config/wx-org1.chainmaker.org/keys/user/admin/admin.key,../config/wx-org2.chainmaker.org/keys/user/admin/admin.key,../config/wx-org3.chainmaker.org/keys/user/admin/admin.key \
--admin-org-ids=wx-org1.chainmaker.org,wx-org2.chainmaker.org,wx-org3.chainmaker.org \
--sync-result=true


#
#./cmc client contract user create \
#--abi-file-path=../../testdata/withdraw.abi \
#--contract-name=withdraw \
#--runtime-type=EVM \
#--byte-code-path=../../testdata/withdraw.bin \
#--version=1.0 \
#--sdk-conf-path=../config/sdk_config.yml \
#--admin-key-file-paths=../config/wx-org1.chainmaker.org/certs/user/admin1/admin1.tls.key,../config/wx-org2.chainmaker.org/certs/user/admin1/admin1.tls.key,../config/wx-org3.chainmaker.org/certs/user/admin1/admin1.tls.key \
#--admin-crt-file-paths=../config/wx-org1.chainmaker.org/certs/user/admin1/admin1.tls.crt,../config/wx-org2.chainmaker.org/certs/user/admin1/admin1.tls.crt,../config/wx-org3.chainmaker.org/certs/user/admin1/admin1.tls.crt \
#--sync-result=true