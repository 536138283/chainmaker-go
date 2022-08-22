cd $CMC
./cmc client contract user create \
--contract-name=ERC20Go \
--runtime-type=DOCKER_GO \
--byte-code-path=../../testdata/ERC20Go.7z \
--version=1.0 \
--sdk-conf-path=../config/sdk_config.yml \
--admin-key-file-paths=../config/wx-org1.chainmaker.org/certs/user/admin1/admin1.tls.key,../config/wx-org2.chainmaker.org/certs/user/admin1/admin1.tls.key,../config/wx-org3.chainmaker.org/certs/user/admin1/admin1.tls.key \
--admin-crt-file-paths=../config/wx-org1.chainmaker.org/certs/user/admin1/admin1.tls.crt,../config/wx-org2.chainmaker.org/certs/user/admin1/admin1.tls.crt,../config/wx-org3.chainmaker.org/certs/user/admin1/admin1.tls.crt \
--sync-result=true