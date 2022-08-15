# 场景2 Rust合约的使用
A账户是org1，admin1
B账户是org1，client1
C账户是org2，admin1
D账户是org2，client1
1. A安装合约"../wasm/rust-func-verify-2.0.0.wasm"
2. 查询A账户余额
3. A发起转账给B
4. 查询A账户余额
5. 查询B账户余额
6. A给B、C、D3个账户同时转账
7. 查询A账户余额