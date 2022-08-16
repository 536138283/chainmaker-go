cd $CMC
echo "UserA balance:"
./cmc client contract user get \
--contract-name=ERC20 \
--abi-file-path=../../testdata/erc20.abi \
--method=balanceOf \
--sdk-conf-path=../config/sdk_config2.yml \
--params="[{\"address\":\"0x83a5bc9b4d2db58249c670720a9860e9476a5424\"}]"

echo "UserB balance:"
./cmc client contract user get \
--contract-name=ERC20 \
--abi-file-path=../../testdata/erc20.abi \
--method=balanceOf \
--sdk-conf-path=../config/sdk_config2.yml \
--params="[{\"address\":\"0x04dd97bbf0b8dca0e9c7c867591903b52fbdf586\"}]"


echo "withdraw contract balance:"
./cmc client contract user get \
--contract-name=ERC20 \
--abi-file-path=../../testdata/erc20.abi \
--method=balanceOf \
--sdk-conf-path=../config/sdk_config2.yml \
--params="[{\"address\":\"0xd6d64458dc76d02482052bfb8a5b33a72c054c77\"}]"