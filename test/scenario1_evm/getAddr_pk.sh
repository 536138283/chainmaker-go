cd $CMC

# user A
echo "query UserA address: node1 admin"
./cmc address pk-to-addr ../config/node1/admin/admin1/admin1.pem

echo "query UserB address: node2 admin"
./cmc address pk-to-addr ../config/node2/admin/admin2/admin2.pem

echo "query UserC address: node3 admin"
./cmc address pk-to-addr ../config/node3/admin/admin3/admin3.pem

echo "query UserD address: node4 admin"
./cmc address pk-to-addr ../config/node4/admin/admin4/admin4.pem
