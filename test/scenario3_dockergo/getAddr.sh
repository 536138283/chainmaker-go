cd $CMC

# user A
echo "query UserA address: org1 client1"
./cmc address cert-to-addr ../config/wx-org1.chainmaker.org/certs/user/client1/client1.sign.crt

echo "query UserB address: org1 admin1"
./cmc address cert-to-addr ../config/wx-org1.chainmaker.org/certs/user/admin1/admin1.sign.crt

echo "query UserC address: org2 client1"
./cmc address cert-to-addr ../config/wx-org2.chainmaker.org/certs/user/client1/client1.sign.crt

echo "query UserD address: org2 admin1"
./cmc address cert-to-addr ../config/wx-org2.chainmaker.org/certs/user/admin1/admin1.sign.crt