cd $CMC

# user A
echo "query UserA address:"
./cmc address cert-to-addr ../config/wx-org1.chainmaker.org/certs/user/admin1/admin1.sign.crt

echo "query UserB address:"
./cmc address cert-to-addr ../config/wx-org1.chainmaker.org/certs/user/client1/client1.sign.crt

echo "query UserC address:"
./cmc address cert-to-addr ../config/wx-org2.chainmaker.org/certs/user/admin1/admin1.sign.crt

echo "query UserA address:"
./cmc address cert-to-addr ../config/wx-org2.chainmaker.org/certs/user/client1/client1.sign.crt
