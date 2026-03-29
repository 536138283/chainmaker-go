# ChainMaker HTTP 代理 RPC 功能与 API 说明

## 1. 功能定位

ChainMaker 的 RPC 服务本质是 **gRPC 服务**。在 `rpc.gateway.enabled=true` 时，节点会在同一个端口上额外提供一层 HTTP/JSON 代理（grpc-gateway），把 HTTP 请求转发为内部 gRPC 调用。

这层能力适用于：

- 只能发 HTTP 请求、不能直接走 gRPC 的客户端；
- 希望通过 REST 风格接口快速联调链上交易提交与版本查询；
- WebSocket 场景下通过网关接入订阅接口。

> 注意：该代理并不是独立监听新端口，而是与 gRPC 复用同一个 RPC 端口。

---

## 2. 服务端实现机制（工作原理）

### 2.1 入口与复用

RPC Server 初始化时会创建“混合服务（mix server）”：

1. 创建 gRPC server；
2. （可选）创建 gateway handler；
3. 使用 `GrpcHandlerFunc` 按协议分流：
   - `HTTP/2 + Content-Type=application/grpc` => 走原生 gRPC；
   - 其他 HTTP 请求 => 走 gateway handler。

因此同一监听地址可同时接收 gRPC 与 HTTP 请求。

### 2.2 Gateway 到 gRPC 的转发

gateway 本身不会直接执行业务，而是作为本地客户端转发到 `127.0.0.1:<rpc.port>` 的 gRPC 服务。

- 当 RPC TLS 开启时，gateway 会加载链上信任根证书和本地证书，以 TLS 拨号到本地 gRPC；
- 当 TLS 关闭时，gateway 以 insecure 方式拨号。

### 2.3 WebSocket 代理

gateway 启用时，HTTP server 会套一层 `wsproxy.WebsocketProxy`，用于支持 WebSocket 转发，并可通过配置控制最大响应缓存体积（`gateway.max_resp_body_size`，单位 MB）。

---

## 3. 配置说明

配置位于 `chainmaker.yml` 的 `rpc` 段。

```yaml
rpc:
  provider: grpc
  host: 0.0.0.0
  port: 12301

  gateway:
    enabled: false
    max_resp_body_size: 16
```

关键项：

- `rpc.gateway.enabled`
  - `false`：仅提供 gRPC；
  - `true`：同端口开启 HTTP/JSON 代理能力。
- `rpc.gateway.max_resp_body_size`
  - WebSocket 代理最大响应缓存大小（MB）。

---

## 4. HTTP 代理 API 清单

根据当前依赖的 `pb-go/v2@v2.3.7`（`rpc_node.pb.gw.go`）可映射出以下 HTTP 路由：

| HTTP 方法 | 路径 | 对应 RPC | 说明 |
|---|---|---|---|
| `POST` | `/v1/sendrequest` | `RpcNode.SendRequest` | 发送交易请求（异步返回） |
| `POST` | `/v1/sendrequestsync` | `RpcNode.SendRequestSync` | 发送交易请求并等待执行结果（同步模式） |
| `GET` | `/v1/subscribe` | `RpcNode.SubscribeWS` | 订阅 WebSocket 入口 |
| `GET` | `/v1/getversion` | `RpcNode.GetChainMakerVersion` | 查询节点版本 |

> 说明：`Subscribe`（gRPC 流式）不是 HTTP 普通接口；HTTP 侧对应的是 `SubscribeWS` 路由。

---

## 5. 主要 API 行为说明

### 5.1 `POST /v1/sendrequest`

- 入参：`common.TxRequest` JSON；
- 行为：构造交易并调用 `invoke(..., isSync=false)`；
- 返回：`common.TxResponse`，代表受理/校验结果，不等待最终上链执行完成。

### 5.2 `POST /v1/sendrequestsync`

- 入参：`common.TxRequest` JSON；
- 行为：调用 `invoke(..., isSync=true)`；
- 返回：`common.TxResponse`，会等待同步交易结果（等待超时受 `rpc.sync_tx_result_timeout` 控制）。

### 5.3 `GET /v1/subscribe`

- 对应 `SubscribeWS(rawTxReq)`：先解析 `RawTxRequest.RawTx`，再复用 `Subscribe` 主流程；
- 最终支持的订阅方法取决于交易 payload method：
  - `SUBSCRIBE_BLOCK`
  - `SUBSCRIBE_TX`
  - `SUBSCRIBE_CONTRACT_EVENT`

### 5.4 `GET /v1/getversion`

- 对应 `GetChainMakerVersion`，返回节点版本信息。

---

## 6. 鉴权与限流注意事项

HTTP 代理并不会绕过链上权限校验：

- 交易请求仍然会进入 `validate(tx)`（证书、签名、权限等校验）；
- RPC 的全局/IP 限流、黑名单等中间件仍然生效；
- 订阅通道有独立的 subscriber 限流逻辑。

---

## 7. 客户端“HTTP 代理”相关配置（SDK 侧）

除服务端 gateway 外，SDK 配置也支持通过上游 HTTP 代理访问 RPC 目标（例如企业网络代理场景）：

```yaml
# 格式为：http://[username:password@]proxyhost:proxyport
# proxy: http://myproxy:8080
```

该配置位于 SDK 配置样例中，属于客户端网络出口代理能力，与服务端 `rpc.gateway` 是两个不同层面的“代理”。

---

## 8. 联调建议

1. 在节点 `chainmaker.yml` 开启 `rpc.gateway.enabled=true`；
2. 重启节点后先用 `GET /v1/getversion` 验证 HTTP 代理已生效；
3. 再联调 `sendrequest/sendrequestsync`；
4. 订阅场景优先验证 `subscribe` 路由可达与 WebSocket 链路稳定性；
5. 生产环境建议配合网关前置层（如 Nginx/Ingress）做鉴权、限流与观测。

