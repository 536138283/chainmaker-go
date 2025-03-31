package parallel

import (
	"chainmaker.org/chainmaker/common/v2/ca"
	apiPb "chainmaker.org/chainmaker/pb-go/v2/api"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"sync"
	"sync/atomic"
	"time"
)

// producer 是一个生成器函数，根据给定的方法名执行压力测试构建任务。
// 它持续工作直到满足预设的请求总数（由 threadNum 和 loopNum 共同决定）。
// 通过 select-case 机制监听来自 produceSignal 的通道信号，以控制任务的分配和执行。
// 当接收到特定信号时，会触发不同类型的生产行为：
//   - 若信号值为 -1，表示需要对所有节点执行生产操作。
//   - 否则，根据信号值创建新的 goroutine 针对特定节点执行生产操作。
//
// 参数:
//
//	method string: 压力测试所采用的方法名称，用于获取对应的构建器。
func producer(method string) {
	builder := stressBuilderFactory(method)
	for {
		if requestId >= int64(threadNum*loopNum) {
			interruptSignal = true
			return
		}
		select {
		case index := <-produceSignal:
			if index == -1 {
				for nodeIndex := 0; nodeIndex < nodeNum; nodeIndex++ {
					produce(builder, nodeIndex)
				}
			} else {
				go produce(builder, index)
			}
		}
	}
}

// stressBuilderFactory 是一个工厂函数，根据提供的方法名字符串，
// 创建并返回对应类型的 StressBuilder 实例。此函数支持以下方法：
//   - invokerMethod: 用于创建执行智能合约调用的构建器。
//   - queryMethod: 用于创建执行智能合约查询的构建器。
//   - createContractStr: 用于创建新智能合约的构建器。
//   - upgradeContractStr: 用于升级智能合约的构建器。
//
// 参数:
//
//	method string: 指定要创建的压力测试构建器类型的方法名。
//
// 返回:
//
//	builder StressBuilder: 与指定方法对应的压力测试构建器实例。
//	  可能是 Invoke、Query、Create 或 Upgrade 类型的实例。
func stressBuilderFactory(method string) (builder StressBuilder) {
	switch method {
	case invokerMethod:
		builder = Invoke{}
	case queryMethod:
		builder = Query{}
	case createContractStr:
		builder = Create{}
	case upgradeContractStr:
		builder = Upgrade{}
	}
	return
}

// produce 使用提供的压力测试构建器生成请求参数，并将这些参数放入指定索引的队列中。
// 函数内部通过循环生成多个请求参数，具体数量由 productFactor 决定。
// 在循环过程中，会检查全局的 interruptSignal，若信号为真，则中断循环并提前返回，
// 以此实现对生产过程的外部控制。每生成一个请求参数，都会将其原子递增的唯一 ID（requestId）
// 与参数一同封装进 RequestParam 结构体，并通过通道发送到对应的 paramQueues[index] 中。
// 如果在构建参数时发生错误，该函数会打印错误信息但不会停止其他参数的生成。
//
// 参数:
//
//	builder StressBuilder: 用于生成请求参数的压力测试构建器。
//	index int: 指定将生成的参数放入哪一个索引的队列中。
func produce(builder StressBuilder, index int) {
	for i := 0; i < productFactor; i++ {
		if interruptSignal {
			return
		}
		atomic.AddInt64(&requestId, 1)
		param, err := builder.Build(requestId, index)
		if err != nil {
			fmt.Printf("producer err: %s\n", err.Error())
		}
		paramQueues[index] <- RequestParam{
			Param:     param,
			RequestId: requestId,
		}
	}
}

// parallelStart 分批次并发启动所有线程执行消耗操作。
// 首先，计算每批启动的线程数量，确保至少启动一批，即使线程总数少于10。
// 然后，基于总爬升时间分配每批启动之间的间隔，单位为秒。
// 函数通过两层循环来控制并发启动：
//   - 外层循环按每批count个线程启动。
//   - 内层循环在每批内实际启动线程，并计数直至达到当前批次应启动的数量。
//
// 启动完每批线程后，程序会暂停指定的interval时间，之后继续启动下一批，直到所有线程都被启动。
//
// 参数:
//
//	threads []*Thread: 一个线程切片，每个元素代表一个待启动的线程实体，包含消耗操作的方法consume。
func parallelStart(threads []*Thread) {
	// 计算每批启动的线程数量，若总数小于10，则至少启动1个
	count := threadNum / 10
	if count == 0 {
		count = 1
	}
	// 计算每批启动间隔，确保总爬升时间内均匀分布启动时间
	interval := time.Duration(climbTime/count) * time.Second
	// 主循环以控制启动批次
	for index := 0; index < threadNum; {
		// 内循环控制每批内线程的并发启动
		for j := 0; j < 10; j++ {
			go threads[index].consume()
			index++
			if index >= threadNum {
				break
			}
		}
		time.Sleep(interval)
	}
}

// listenAndExit 监听完成信号和超时，以控制并发任务的有序退出。
// 该函数等待所有任务完成或到达预设的超时时间，然后执行必要的清理工作并终止相关信号通道。
//
// 参数:
//
//	timeoutChan chan struct{}: 超时信号通道，用于通知任务超时。
//	doneChan chan struct{}: 完成信号通道，每次有任务完成时接收信号。
//	printTicker *time.Ticker: 定时器，周期性执行某些打印操作，函数结束前需停止。
//
// 功能说明:
// 1. 初始化一个超时定时器，周期为 timeout 秒。
// 2. 使用 sync.Once 确保超时逻辑只执行一次，避免重复发送超时信号。
// 3. 循环监听 doneChan 和 timeoutTicker.C 的信号：
//   - 收到 doneChan 信号，表示有一个任务完成，累加完成计数。
//   - 若超时（从 timeoutTicker 收到信号），则通过 once 执行超时处理：
//     向 timeoutChan 发送 threadNum 次信号，通知所有未完成的任务超时。
//
// 4. 当所有任务完成（doneCount 达到 threadNum）或发生超时，执行以下操作：
//   - 设置全局中断信号为 true。
//   - 停止 printTicker 防止资源泄露。
//   - 关闭各个通道以结束相关协程的执行。
func listenAndExit(timeoutChan, doneChan chan struct{}) {
	doneCount := 0
	timeoutTicker := time.NewTicker(time.Duration(timeout) * time.Second)
	once := sync.Once{}
	for {
		if doneCount >= threadNum {
			break
		}
		select {
		case <-doneChan:
			doneCount++
		case <-timeoutTicker.C:
			once.Do(func() {
				for i := 0; i < threadNum; i++ {
					timeoutChan <- struct{}{}
				}
			})
		}
	}
	interruptSignal = true
	close(timeoutChan)
	close(doneChan)
	close(produceSignal)
	timeoutTicker.Stop()
}

// Thread 结构体表示一个执行特定任务的线程或协程实例，
// 包含了执行循环次数、通信通道、统计信息以及与 gRPC 服务的连接信息。
type Thread struct {
	id           int                   // 是线程的唯一标识符，用于区分不同的线程实例
	loopNum      int                   // 指定线程需要执行循环操作的次数
	doneChan     chan struct{}         // 是一个通道，用于传递完成信号。当线程完成其指定任务后，会向此通道发送信号
	timeoutChan  chan struct{}         // 用于接收超时信号。当接收到信号时，表明当前线程可能需要根据超时情况做出响应或终止操作
	statistician *Statistician         // 是一个指向 Statistician 实例的指针，用于收集和统计线程执行过程中的性能数据或其他相关信息
	conns        []*grpc.ClientConn    // conn 表示与 gRPC 服务建立的客户端连接，通过此连接可以发起 RPC 调用 conns与clients是相互对应关系
	clients      []apiPb.RpcNodeClient // 是 gRPC 客户端，类型为 apiPb.RpcNodeClient，封装了对远程节点服务的所有调用能力
	index        int                   // 用来线程去消费的队列索引
}

// threadFactory 函数负责创建并初始化一组 Thread 实例，每个实例代表一个工作线程，准备就绪以执行特定任务。
// 参数:
// - number: 指定要创建的工作线程数量。
// - doneChan: 一个通道，各线程通过它来通知其已完成任务。
// - timeoutChan: 用于向线程发送超时信号的通道。
// - statistician: 一个指向 Statistician 对象的指针，用于跨线程收集运行时统计数据。
// 返回:
// - []*Thread: 初始化后的线程实例切片。
// - error: 如果在初始化过程中遇到错误（如 gRPC 连接失败），则返回具体的错误信息。
// 功能实现:
// 1. 首先，根据指定的数量创建一个空的线程切片。
// 2. 循环遍历，为每个线程分配唯一 ID、循环执行次数以及其他共享资源（doneChan, timeoutChan, statistician）。
// 3. 计算每个线程对应的主机索引（基于线程 ID 和主机列表长度取模），用于 gRPC 连接初始化。
// 4. 初始化每个线程的 gRPC 连接，使用 TLS 参数和计算出的主机索引。
// 5. 若 gRPC 连接成功，则创建相应的 gRPC 客户端实例并将其附加到线程上。
// 6. 将构建好的线程添加到线程切片中。
// 7. 若在任一环节遇到错误，则立即停止创建过程并返回错误。
func threadFactory(number int, doneChan, timeoutChan chan struct{}, statistician *Statistician) ([]*Thread, error) {
	threads := make([]*Thread, number)
	var err error
	for i := 0; i < number; i++ {
		t := &Thread{id: i, loopNum: loopNum, doneChan: doneChan, timeoutChan: timeoutChan, statistician: statistician}
		t.index = t.id % len(hosts)
		t.conns, err = t.initGRPCConnect(useTLS, t.index)
		if err != nil {
			return nil, err
		}
		for _, conn := range t.conns {
			t.clients = append(t.clients, apiPb.NewRpcNodeClient(conn))
		}
		threads[i] = t
	}
	return threads, nil
}

// consume 是 Thread 类型的方法，负责处理单个线程的工作流程，包括循环发送请求、处理响应、统计以及错误日志记录。
// 功能描述:
// 1. 循环处理: 方法会在给定的循环次数 (t.loopNum) 内持续工作。
// 2. 超时与中断控制:
//   - 监听 t.timeoutChan，一旦收到信号，线程将通过 t.doneChan 发送完成信号并提前返回。
//   - 内部的 select 语句检查 paramQueues[t.index] 是否关闭，若关闭则同样结束当前线程。
//
// 3. 请求发送与统计:
//   - 从 paramQueues[t.index] 队列接收请求参数，若队列非空且未中断，尝试发送交易至 gRPC 客户端。
//   - 成功发送后，更新统计信息，包括总请求数、请求耗时及节点 ID，并可能触发生产者线程继续生产数据。
//
// 4. 错误处理与日志:
//   - 发生错误时，根据配置 (recordLog) 记录错误日志，包含线程 ID、循环 ID 和节点 ID 等详细信息。
//
// 5. 延迟与节流:
//   - 每次循环后，线程会暂停 sleepTime 毫秒，以控制请求频率。
//
// 结束条件:
// - 执行完所有循环次数。
// - 收到超时信号。
// - 请求队列关闭。
// 最终，无论循环是否自然结束，都会通过 t.doneChan 发送完成信号。
func (t *Thread) consume() {
	for i := 0; i < t.loopNum; i++ {
		select {
		case <-t.timeoutChan:
			// 超时，结束线程
			t.doneChan <- struct{}{}
			return
		default:
			// 尝试从队列接收请求参数
			select {
			case req, ok := <-paramQueues[t.index]:
				// 如果chan 关闭，被分配到该chan的线程也一起关闭
				if !ok {
					t.doneChan <- struct{}{}
					return
				}
				// 如果队列中数量小于生产因子，发送生产信号
				if len(paramQueues[t.index]) < productFactor && !interruptSignal {
					produceSignal <- t.index
				}
				start := time.Now()

				var err error
				err = sendTx(t.clients[loopNum%nodeNum], orgIDs[t.index], i, req.Param)
				// 结果进入结果集
				atomic.AddUint32(&t.statistician.totalCount, 1)
				t.statistician.reqStatC <- &reqStat{
					success: err == nil,
					elapsed: time.Since(start).Milliseconds(),
					nodeId:  t.index,
				}
				if recordLog && err != nil {
					log.Errorf("threadId: %d, loopId: %d, nodeId: %d, err: %s", t.id, i, t.index, err)
				}
				time.Sleep(time.Duration(sleepTime) * time.Millisecond)
			}
		}
	}
	// 自然循环结束，发送完成信号
	t.doneChan <- struct{}{}
}

// stop 停止线程并关闭其关联的连接
func (t *Thread) stop() {
	for _, conn := range t.conns {
		err := conn.Close()
		if err != nil {
			return
		}
	}
}

// initGRPCConnect 初始化 gRPC 连接给定的 Thread 实例。
// 参数:
// - useTLS: 布尔值，指示是否使用 TLS 安全连接。
// - index: 整数，用于从配置的主机列表、CA 路径、用户证书路径和用户密钥路径中选择相应的项。
//
// 函数行为:
// 1. **选择地址: 根据提供的索引值选取主机地址。
// 2. TLS 配置:
//   - 如果 useTLS 为 true，根据主机名（或默认值）配置 TLS 客户端。
//   - 从对应索引获取 CA 路径、用户证书路径和用户密钥路径，创建 TLS 客户端凭证。
//   - 使用这些凭证通过 gRPC 的 WithTransportCredentials 方法建立安全连接。
//
// 3. 非 TLS 连接:
//   - 如果 useTLS 为 false，则直接使用 grpc.Dial 并通过 WithInsecure 方法创建不安全的连接。
//
// 4. 错误处理:
//   - 在初始化 TLS 凭证或建立连接过程中遇到的任何错误都将被捕获并作为错误返回。
//
// 返回:
// - *grpc.ClientConn: 成功建立的 gRPC 客户端连接。
// - error: 初始化连接过程中发生的错误，如果成功则为 nil。
func (t *Thread) initGRPCConnect(useTLS bool, index int) ([]*grpc.ClientConn, error) {
	// 设置 gRPC 服务端地址
	var conns []*grpc.ClientConn
	for i := 0; i < len(hosts); i++ {
		url := hosts[i]
		// 根据 useTLS 配置选择不同的连接方式
		if useTLS {
			var serverName string
			// 初始化 serverName，如果没有指定 hostnames，则使用默认域名
			if hostnamesString == "" {
				serverName = "chainmaker.org"
			} else {
				if len(hosts) != len(hostnames) {
					return nil, errors.New("required len(hosts) == len(hostnames)")
				}
				// 根据索引选择对应的主机名
				serverName = hostnames[index]
			}
			// 使用 TLS 凭证创建安全的 gRPC 连接
			tlsClient := ca.CAClient{
				ServerName: serverName,
				CaPaths:    []string{caPaths[index]},
				CertFile:   userCrtPaths[index],
				KeyFile:    userKeyPaths[index],
			}
			if userEncCrtPathsString != "" && userEncKeyPathsString != "" {
				tlsClient.EncKeyBytes = encKeyBytes[i]
				tlsClient.EncCertBytes = encCrtBytes[i]
				tlsClient.EncKeyFile = encKeyPaths[i]
				tlsClient.EncCertFile = encCrtPaths[i]
			}
			// 获取 TLS 凭证
			c, err := tlsClient.GetCredentialsByCA()
			if err != nil {
				return nil, err
			}
			conn, err := grpc.Dial(url, grpc.WithTransportCredentials(*c))
			if err != nil {
				return nil, err
			}
			conns = append(conns, conn)
		} else {
			conn, err := grpc.Dial(url, grpc.WithInsecure())
			if err != nil {
				return nil, err
			}
			conns = append(conns, conn)
		}
	}
	return conns, nil
}
