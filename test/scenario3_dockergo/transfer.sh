cd $CMC

echo "A转账给B"
./cmc client contract user invoke \
--contract-name=ERC20Go \
--method=transfer \
--sdk-conf-path=../config/sdk_config.yml \
--params="{\"to\": \"8a0c48820698bf17be2fed5642d6273285afb302\",\"amount\": \"100\"}" \
--gas-limit=99999999 \
--sync-result=true

echo "A转账给withdraw合约"
./cmc client contract user invoke \
--contract-name=ERC20Go \
--method=transfer \
--sdk-conf-path=../config/sdk_config.yml \
--params="{\"to\": \"0xd6d64458dc76d02482052bfb8a5b33a72c054c77\",\"amount\": \"200\"}" \
--gas-limit=99999999 \
--sync-result=true