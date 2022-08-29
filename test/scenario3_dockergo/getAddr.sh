cd $CMC

# user A
echo "query UserA address: org1 admin1"
./cmc address cert-to-addr ../config/wx-org1.chainmaker.org/certs/user/admin1/admin1.sign.crt

echo "query UserB address: org2 admin1"
./cmc address cert-to-addr ../config/wx-org2.chainmaker.org/certs/user/admin1/admin1.sign.crt

echo "query UserC address: org3 admin1"
./cmc address cert-to-addr ../config/wx-org3.chainmaker.org/certs/user/admin1/admin1.sign.crt

echo "query UserD address: org4 admin1"
./cmc address cert-to-addr ../config/wx-org4.chainmaker.org/certs/user/admin1/admin1.sign.crt