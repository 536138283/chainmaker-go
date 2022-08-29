cd $CMC

echo "A mint 10亿"
./cmc client contract user invoke \
--contract-name=ERC20Go \
--method=mint \
--sdk-conf-path=../config/sdk_config.yml \
--params="{\"account\": \"bd3c51417982123a3200570e900f7982133363b3\",\"amount\": \"1000000000\"}" \
--gas-limit=99999999 \
--sync-result=true
