cd $CMC
./cmc client contract user get \
--contract-name=ERC20 \
--abi-file-path=../../testdata/erc20.abi \
--method=balanceOf \
--sdk-conf-path=../config/sdk_config.yml \
--params="[{\"address\":\"0x5a3e1a768e3ad7f49f7207c1cc113a337dfaa4ba\"}]"