cd $CMC

echo "A->B转账"
nohup ./cmc client contract user invoke \
--contract-name=ERC20Go \
--method=transfer \
--sdk-conf-path=../config/sdk_config.yml \
--params="[{\"address\": \"8a0c48820698bf17be2fed5642d6273285afb302\"},{\"uint256\": \"1\"}]" \
--sync-result=true &

echo "A->C转账"
nohup ./cmc client contract user invoke \
--contract-name=ERC20Go \
--method=transfer \
--sdk-conf-path=../config/sdk_config.yml \
--params="[{\"address\": \"7406a39c054f270d11c79be3db5f2c8ee7e9ce47\"},{\"uint256\": \"2\"}]" \
--sync-result=true &

echo "A->D转账"
nohup ./cmc client contract user invoke \
--contract-name=ERC20Go \
--method=transfer \
--sdk-conf-path=../config/sdk_config.yml \
--params="[{\"address\": \"a891e56a05c4021e7eb6f56b27ba91a17f3e10df\"},{\"uint256\": \"3\"}]" \
--sync-result=true &