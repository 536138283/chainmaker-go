cd $CMC
./cmc client contract user create \
--contract-name="rustAsset" \
--runtime-type=WASMER \
--byte-code-path="../../wasm/rust-asset-2.0.0.wasm" \
--version=1.0 \
--params="{\"issue_limit\":\"10000000\",\"total_supply\":\"1000000000\"}"  \
--sdk-conf-path=../config/sdk_config.yml \
--admin-key-file-paths=../config/wx-org1.chainmaker.org/certs/user/admin1/admin1.tls.key,../config/wx-org2.chainmaker.org/certs/user/admin1/admin1.tls.key,../config/wx-org3.chainmaker.org/certs/user/admin1/admin1.tls.key \
--admin-crt-file-paths=../config/wx-org1.chainmaker.org/certs/user/admin1/admin1.tls.crt,../config/wx-org2.chainmaker.org/certs/user/admin1/admin1.tls.crt,../config/wx-org3.chainmaker.org/certs/user/admin1/admin1.tls.crt \
--sync-result=true
