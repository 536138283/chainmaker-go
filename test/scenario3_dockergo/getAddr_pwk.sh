cd $CMC

# user A
echo "query UserA address: org1 admin"
./cmc address pk-to-addr ../config/wx-org1.chainmaker.org/keys/user/admin/admin.pem

echo "query UserB address: org2 admin"
./cmc address pk-to-addr ../config/wx-org2.chainmaker.org/keys/user/admin/admin.pem

echo "query UserC address: org3 admin"
./cmc address pk-to-addr ../config/wx-org3.chainmaker.org/keys/user/admin/admin.pem

echo "query UserD address: org4 admin"
./cmc address pk-to-addr ../config/wx-org4.chainmaker.org/keys/user/admin/admin.pem
