cd $CMC
echo "UserA balance:"
./cmc client contract user get \
--contract-name=ERC20Go \
--method=balanceOf \
--sdk-conf-path=../config/sdk_config2.yml \
--params="{\"account\":\"bd3c51417982123a3200570e900f7982133363b3\"}"

echo "UserB balance:"
./cmc client contract user get \
--contract-name=ERC20Go \
--method=balanceOf \
--sdk-conf-path=../config/sdk_config2.yml \
--params="{\"account\":\"8a0c48820698bf17be2fed5642d6273285afb302\"}"


echo "withdraw contract balance:"
./cmc client contract user get \
--contract-name=ERC20Go \
--method=balanceOf \
--sdk-conf-path=../config/sdk_config2.yml \
--params="{\"account\":\"0xd6d64458dc76d02482052bfb8a5b33a72c054c77\"}"