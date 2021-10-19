### ABFT

ABFT(Asynchronous Byzantine Fault Tolerance) 异步拜占庭容错共识算法，能够在异部网络环境，拜占庭节点小于总节点数1/3时，保证系统的安全运行。ABFT基于HoneyBadger BFT共识算法，HoneyBadger BFT的核心创新点在于发现了在异步、拜占庭环境下，原子广播`ABC(Atomic Broadcast)` 问题可以分解为分解成一个核心模块异步共同子集`ACS(Asynchronous Common Subset)`，然后将 ACS 分解成了可靠广播 `RBC(Reliable Broadcast)` + 拜占庭二进制共识`BBA(Byzantine Binary Agreement)`两个子模块，并且分别针对这两个子模块找到了两个比较优化的实现。
通过模块化的设计，保证了在异步、拜占庭环境下，各个节点按相同顺序收到相同的消息。

异步网络模型是ABFT共识的重要特性，同步、半同步、异步是节点之间消息传输的底层网络模型。

* 同步网络`(synchronous)`：假设网络中的消息能够在一个已知的时间 Δ 内到达。即最大消息延迟是确定。比如Bitcoin、Ethereum基于的Pow共识协议，一致性和活性都采用同步假设
* 半同步网络`(partially synchronous)`：网络中消息某限定时间后到达所有共识节点的的概率与时间的关系是已知的，假设在一个 GST（global stabilization time）事件之后，消息在 Δ 时间内到达。比如Raft、PBFT 都是基于半同步网络假设设计的共识协议，这些协议的关注点可以更多的放在安全性（safety），活性由 failure detector 来保证。在 PBFT 中每个 replica 都要维护一个 timer，一旦 timeout 就会触发 view change 协议选举新的 leader。这里的 timer 就起到了一个 failure detector 的作用。这些协议虽然能够保证在任何网络情况下系统的安全性，但在异步网络下会丧失活性
* 异步网络`(asynchrony)`：正常节点发出消息，在一个时间间隔内可以送达目标节点，但是该时间间隔未知，即最大消息延迟未知。异步共识协议则完全不需要考虑 timer 的设置。为了保证协议的活性，异步协议需要引入随机源，简单来说就是当协议无法达成共识的时候，借助上帝抛骰子的方式随机选择一个结果作为最终结果

#### 模块设计

2. ABFT共识模块具体实现主要由以下几个组件组成，分别是：异步共同子集 ACS、可靠广播 `RBC`、拜占庭二进制共识 `BBA`、消息传输 `msgSender`，组件之间互相配合在 `ConsensusABFTImpl` 实现了ABFT共识算法。
   * 异步共同子集 `ACS(Asynchronous Common Subset)`：用于让各个节点按相同顺序收到相同的消息，每个节点的数据集合分别是U<sub>1</sub>, U<sub>2</sub>, ..., U<sub>n</sub>，节点之间通过一些通讯之后每个节点上都得到一个相同的集合U=U<sub>1</sub>∪U<sub>2</sub> ... ∪U<sub>n</sub> 
   * 可靠广播 `RBC(Reliable Broadcast)`：用来确保源节点发送的交易批次能够可靠地发送到网络中的所有节点
   * 拜占庭二进制共识 `BBA(Byzantine Binary Agreement)`：在所有节点之间进行一轮共识，让所有节点对于 0 或 1 达成一致，得到一个最终确认的二进制数值，由这个二进制的对应的位来决定哪个交易批次会被最终确认
   * 消息发送者 `msgSender(Message Sender)`：实现了节点之间消息发送、接受的逻辑

<img src="../images/Consensus-abft-protocol.png"  alt="abft-共识协议" style="zoom:100%;"/>

#### 共识流程

ABFT 的共识流程如下图所示，ABFT共识利用异步共同子集 `ACS` 实现原子广播 `ABC` 的。ABFT共识不区分主节点和从节点，所有节点都公平的接受交易，每个节点都随机从自己的交易池选取一批交易为一个区块贡献一部分交易，可靠广播 `RBC` 模块确保每个节点贡献的交易批次能够达到所有节点，拜占庭二进制共识`BBA` 模块的作用就是确定哪些节点贡献的交易批次最终达成共识，被包含在区块中。

<img src="../images/Consensus-abft-process.png"  alt="abft-共识流程" style="zoom:100%;"/>

#### 可靠广播 RBC

1. 可靠广播 `RBC` 可以确保源节点能够可靠地将消息发送到网络中的所有节点。具体来说，`RBC` 主要有以下三个性质：
   * 一致性（Agreement）任意两个正确节点都收到来自源节点的相同的消息
   * 全局性（Totality）只要有一个节点收到了来自源节点的消息，那么所有正确的节点最终都能收到这个消息
   * 有效性（Validity）如果源节点是正确的，那么所有正确的节点收到的消息一定与源节点发送的消息一致

2. RBC 主要分成 `Val`、`Echo`、`Ready` 三个阶段，其中后两个阶段各经历一次 all-to-all 的广播。
3. RBC协议通过纠删码算法降低节点间的数据传输，ABFT 中采用了基于(N-2f，N)的纠删码模式，即将一个数据块进行编码后，可以将其分成 N 份，其中只要任意 N-2f 份组合可以恢复整个数据块。

<img src="../images/Consensus-abft-rbc.png"  alt="abft-可靠广播" style="zoom:100%;"/>

#### 拜占庭二进制共识 BBA

1. 正确的`Byzantine Binary Agreement` 算法应该满足以下三个条件:
   * Agreement：任意一个诚实节点输出b，那么所有诚实节点都输出b
   * Validity：任意一个诚实节点输出b，那么至少一个诚实节点的输入是b
   * Termination：如果所有诚实节点都有输入，那么所有诚实节点都会有输出
2. BBA 的实现原理：BBA完成一次需要超过2/3的参与节点同意是否在区块中包含某批次交易。如果只收到1/3-2/3的节点之间投票Y，节点无法达成一致，BBA需要借助外部随机源做决定才能正确终止。这个随机源就是 BBA 的核心组件`(Common Coin，CC)`，我们也可以将 CC 理解成抛硬币，只有 0 和 1 两个值。只抛一次硬币可能还是无法达成共识，那么就不停掷，最终会出现所有人都达成一致的结果。Common Coin 有很多实现方案，ABFT针对其模块化的设计，采用了基于阈值签名的 CC 方案。每个节点对一个共同的字符串进行签名并广播给其它节点，当节点收到来自其它 f+1 个节点的签名时，就可以将这些签名聚合成一个签名，并将这个签名作为随机源。

#### 门限加密(Threshold Encrytion)

1. 因为恶意节点的存在可能干扰 `Binary Byzantine Agreement` 的结果。因此，ABFT 提出了门限加密的方式来避免最终的交易集受到攻击。
2. 门限加密的原理是允许任何节点使用一把主公钥来加密一条信息，但是解密则需要网络中所有节点来共同合作，只有当 f + 1 个诚实节点共同合作才能获得解密秘钥，从而得到消息原文。在这之前，任何攻击者都无法解密获得消息的原文。
3. 具体过程如下：
   1. `TPKE.Setup(1<sup>λ</sup>)→PK,{SK<sub>i</sub>}`：创建一把公钥PK、同时为每个节点生成一个私钥SK<sub>i</sub>。
   2. `TPKE.Enc(PK,m)→C`：用这把公钥PK对明文m进行加密，加密结果是C。
   3. `TPKE.DecShare(SK<sub>i</sub>,C)→σ<sub>i</sub>`：每个节点用其私钥SK<sub>i</sub> 解密C得到中间结果σ<sub>i</sub>。
   4. `TPKE.Dec(PK,C,{i,σ<sub>i</sub>})→m`：用f+1 个中间结果σ<sub>i</sub> 配合PK，就可以解密C得到明文m。

注：当前chainmaker 版本未实现门限加密
#### 配置参数

```yaml
#共识配置
consensus:
  # 共识类型(0-SOLO,1-TBFT,2-MBFT,3-HOTSTUFF,4-RAFT,5-DPoS,6-ABFT,10-POW)
  type: 6
```