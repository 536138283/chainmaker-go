# 使用Docker-go进行智能合约开发



## 部署环境

1. 操作系统

    目前仅支持在Linux系统下部署和运行Docker VM。长安链与Docker VM的通信是基于unix domain socket，在非Linux系统中，会出现权限问题。

2. 软件依赖

    1. docker

       启动长安链之前，确认docker已经启动；

       安装步骤，请参看：https://golang.org/doc/install

       ```bash
       $ docker version
       Client: Docker Engine - Community
        Version:           20.10.7
        API version:       1.41
        Go version:        go1.13.15
        Git commit:        f0df350
        Built:             Wed Jun  2 11:56:38 2021
        OS/Arch:           linux/amd64
        Context:           default
        Experimental:      true
       
       Server: Docker Engine - Community
        Engine:
         Version:          20.10.7
         API version:      1.41 (minimum version 1.12)
         Go version:       go1.13.15
         Git commit:       b0f5bc3
         Built:            Wed Jun  2 11:54:50 2021
         OS/Arch:          linux/amd64
         Experimental:     false
        containerd:
         Version:          1.4.8
         GitCommit:        7eba5930496d9bbe375fdf71603e610ad737d2b2
        runc:
         Version:          1.0.0
         GitCommit:        v1.0.0-0-g84113ee
        docker-init:
         Version:          0.19.0
         GitCommit:        de40ad0
       
       ```
       
    2. 7zip
    
       启动长安链之前，确认7zip已经安装完毕；
    

## 参数说明

1. 启动Docker VM

   ```bash
   $ ./prepare.sh 4 1 11300 12300
   begin check params...
   begin generate certs, cnt: 4
   input consensus type (0-SOLO,1-TBFT(default),3-HOTSTUFF,4-RAFT,5-DPOS): 
   input log level (DEBUG|INFO(default)|WARN|ERROR): 
   ## 开启Docker VM, 请输入YES, 默认为NO：不开启Docker VM
   enable docker vm (YES|NO(default))YES  
   enable docker vm
   begin generate node1 config...
   begin generate node2 config...
   begin generate node3 config...
   begin generate node4 config...
   
   ```



2. 启动参数

    1. chainmaker.yml

       ```yml
       docker:
         # 开启docker vm开关
         enable_dockervm: true    
         # docker image名字， 默认为chainmaker-docker-go-image
         image_name: chainmaker-docker-go-image 
         # docker container名字，默认为chainmaker-docker-go-container，后面的数字根据节点书依次   # 累加
         container_name: chainmaker-docker-go-container1
         # docker container 程序文件夹位置
         docker_container_dir: ../dockervm/dockercontainer
         # docker vm 和主机bind路径
         mount_path: ../data/wx-org1.chainmaker.org/docker-go/mount 
         # rpc 配置
         rpc:
             # unix domain socket 开关，现在仅支持开启unix domain socket
           uds_open: true       
           # rpc 发送最大值，单位MB
           max_send_message_size: 10   
           # rpc 接受最大值，单位MB
           max_recv_message_size: 10   
         # vm 配置
         vm:
             # vm内部接收tx的缓存值
           tx_size: 1000
           # uid 数量
           user_num: 100
           # 每笔交易最大时间
           time_limit: 2
         # pprof 配置
         pprof:
             # pprof 开关
           pprof_enabled: false
       ```

    2. log.yml

       ```yml
       log:
         system: 
           log_level_default: INFO       
           log_levels:
             core: INFO                 
             net: INFO
             # 合约中的日志，需将vm改为debug级别
             vm: INFO                    
             storage: INFO               
           file_path: ../log/system.log
           # docker vm日志文件路径，日志文件名为docker-go.log
           docker_file_path: ../log/docker-go 
           max_age: 365                  
           rotation_time: 1              
           log_in_console: false         
           show_color: true             
         brief:
           log_level_default: INFO
           file_path: ../log/brief.log
           max_age: 365                  
           rotation_time: 1             
           log_in_console: false         
           show_color: true              
         event:
           log_level_default: INFO
           file_path: ../log/event.log
           max_age: 365                  
           rotation_time: 1              
           log_in_console: false         
           show_color: true              
       ```





## 使用docker镜像进行合约开发

1. 拉取镜像

   ```bash
   $ docker pull chainmakerofficial/chainmaker-docker-go-contract:2.0.0_dockervm_alpha
   ```



2. 请指定你本机的工作目录$WORK_DIR，例如/data/workspace/contract，挂载到docker容器中以方便后续进行必要的一些文件拷贝

   ```bash
   $ docker run -it --name chainmaker-docker-go-contract -v <WORK_DIR>:/home chainmakerofficial/chainmaker-docker-go-contract:2.0.0_dockervm_alpha bash
   ```



3. 编译合约，压缩合约

   ```bash
   $ cd data/
   $ tar xvf /data/contract_docker_go_template.tar.gz
   $ cd contract_docker_go
   $ ./build.sh
   please input contract name, contract name should be same as name in tx: 
   <contract_name> #此处contract_name必须和交易中的合约名一致
   please input zip file: 
   <zip_file_name> #建议与contract_name保持一致
   ...
   ```



4. 编译，压缩好的文件位置在

   ```bash
   /home/contract_docker_go/target/release/<contract_name>.7z
   ```



## 实例说明

实现功能：

1. 存储文件哈希，文件名

2. 通过文件哈希查询该条交易

   ```go
   package main
   
   import (
   	"encoding/json"
   	"log"
   	"strconv"
   
   	"chainmaker.org/chainmaker-contract-sdk-docker-go/pb/protogo"
   	"chainmaker.org/chainmaker-contract-sdk-docker-go/shim"
   )
   
   // 用户合约名称
   type FactContract struct {
   }
   
   // 存证对象
   type Fact struct {
   	FileHash string `json:"FileHash"`
   	FileName string `json:"FileName"`
   	Time     int32  `json:"Time"`
   }
   
   // 新建存证对象
   func NewFact(FileHash string, FileName string, time int32) *Fact {
   	fact := &Fact{
   		FileHash: FileHash,
   		FileName: FileName,
   		Time:     time,
   	}
   	return fact
   }
   
   // 部署合约
   func (f *FactContract) InitContract(stub shim.CMStubInterface) protogo.Response {
   
   	return shim.Success([]byte("Init Success"))
   
   }
   
   // 调用合约
   func (f *FactContract) InvokeContract(stub shim.CMStubInterface) protogo.Response {
   
   	// 获取参数，方法名通过参数传递
   	method := stub.GetArgs()["method"]
   
   	switch method {
   	case "save":
   		return f.save(stub)
   	case "findByFileHash":
   		return f.findByFileHash(stub)
   	default:
   		return shim.Error("invalid method")
   	}
   
   }
   
   func (f *FactContract) save(stub shim.CMStubInterface) protogo.Response {
   	params := stub.GetArgs()
   
   	// 获取参数
   	fileHash := params["file_hash"]
   	fileName := params["file_name"]
   	timeStr := params["time"]
   	time, err := strconv.Atoi(timeStr)
   	if err != nil {
   		msg := "time is [" + timeStr + "] not int"
   		stub.Log(msg)
   		return shim.Error(msg)
   	}
   
   	// 构建结构体
   	fact := NewFact(fileHash, fileName, int32(time))
   
   	// 序列化
   	factBytes, _ := json.Marshal(fact)
   
   	// 发送事件
   	stub.EmitEvent("topic_vx", []string{fact.FileHash, fact.FileName})
   
   	// 存储数据
   	err = stub.PutState([]byte("fact_hash" + fact.FileHash), factBytes)
   	if err != nil {
   		return shim.Error("fail to save fact")
   	}
   
   	// 记录日志
   	stub.Log("[save] FileHash=" + fact.FileHash)
   	stub.Log("[save] FileName=" + fact.FileName)
   
   	// 返回结果
   	return shim.Success([]byte(fact.FileName + fact.FileHash))
   
   }
   
   func (f *FactContract) findByFileHash(stub shim.CMStubInterface) protogo.Response {
   	// 获取参数
   	FileHash := stub.GetArgs()["file_hash"]
       
   	// 查询结果
   	result, err := stub.GetState([]byte("fact_hash" + FileHash))
   	if err != nil {
   		return shim.Error("failed to call get_state")
   	}
   
   	// 反序列化
   	var fact Fact
   	_ = json.Unmarshal(result, &fact)
   
   	// 记录日志
   	stub.Log("[find_by_file_hash] FileHash=" + fact.FileHash)
   	stub.Log("[find_by_file_hash] FileName=" + fact.FileName)
   	
       // 返回结果
   	return shim.Success(result)
   }
   
   func main() {
   	err := shim.Start(new(FactContract))
   	if err != nil {
   		log.Fatal(err)
   	}
   }
   
   ```
## 调用示例

调用Docker-Go的合约与之前保持一致，可以通过cmc或者Go SDK，通过发交易的方式进行合约的部署和调用。
区别在于，调用合约的具体方法需要放入参数中，并且runtime-type为DOCKER_GO。以下示例的准备工作请按官网文档进行。

1. 使用cmc工具
    ``` bash
   ## 创建合约
   ./cmc client contract user create \
    --contract-name=contract_fact \
    --runtime-type=DOCKER_GO \
    --byte-code-path=./testdata/docker-go-demo/contract_fact.7z \
    --version=1.0 \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --admin-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org4.chainmaker.org/user/admin1/admin1.tls.key \
    --admin-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org4.chainmaker.org/user/admin1/admin1.tls.crt \
    --sync-result=true \
    --params="{}"
   
   ## 调用合约
   ./cmc client contract user invoke \
    --contract-name=contract_fact \
    --method=invoke_contract \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --params="{\"method\":\"save\",\"file_name\":\"name007\",\"file_hash\":\"ab3456df5799b87c77e7f88\",\"time\":\"6543234\"}" \
    --sync-result=true
   
   ## 查询合约
   ./cmc client contract user get \
    --contract-name=contract_fact \
    --method=invoke_contract \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --params="{\"method\":\"findByFileHash\",\"file_hash\":\"ab3456df5799b87c77e7f88\"}"
    ```

2. 使用Go SDK
    ``` go
   // 创建合约
   func testUserContractCreate(client *sdk.ChainClient, withSyncResult bool, isIgnoreSameContract bool, usernames ...string) {

        resp, err := createUserContract(client, factContractName, factVersion, factByteCodePath,
            common.RuntimeType_DOCKER_GO, []*common.KeyValuePair{}, withSyncResult, usernames...)
        if !isIgnoreSameContract {
            if err != nil {
                log.Fatalln(err)
            }
        }

        fmt.Printf("CREATE claim contract resp: %+v\n", resp)
    }
   
   func createUserContract(client *sdk.ChainClient, contractName, version, byteCodePath string, runtime common.RuntimeType, kvs []*common.KeyValuePair, withSyncResult bool, usernames ...string) (*common.TxResponse, error) {

        payload, err := client.CreateContractCreatePayload(contractName, version, byteCodePath, runtime, kvs)
        if err != nil {
            return nil, err
        }

        endorsers, err := examples.GetEndorsers(payload, usernames...)
        if err != nil {
            return nil, err
        }

        resp, err := client.SendContractManageRequest(payload, endorsers, createContractTimeout, withSyncResult)
        if err != nil {
            return nil, err
        }

        err = examples.CheckProposalRequestResp(resp, true)
        if err != nil {
            return nil, err
        }

        return resp, nil
    }
   
   // 调用合约
   // 调用或者查询合约时，method参数请设置为 invoke_contract，此方法会调用合约的InvokeContract方法，再通过参数获得具体方法
   func testUserContractInvoke(client *sdk.ChainClient, method string, withSyncResult bool) (string, error) {

        curTime := strconv.FormatInt(time.Now().Unix(), 10)

        fileHash := uuid.GetUUID()
        kvs := []*common.KeyValuePair{
            {
                Key: "method",
                Value: []byte("save"),
            },
            {
                Key:   "time",
                Value: []byte(curTime),
            },
            {
                Key:   "file_hash",
                Value: []byte(fileHash),
            },
            {
                Key:   "file_name",
                Value: []byte(fmt.Sprintf("file_%s", curTime)),
            },
        }

        err := invokeUserContract(client, factContractName, method, "", kvs, withSyncResult)
        if err != nil {
            return "", err
        }

        return fileHash, nil
    }
   
   func invokeUserContract(client *sdk.ChainClient, contractName, method, txId string, kvs []*common.KeyValuePair, withSyncResult bool) error {

        resp, err := client.InvokeContract(contractName, method, txId, kvs, -1, withSyncResult)
        if err != nil {
            return err
        }

        if resp.Code != common.TxStatusCode_SUCCESS {
            return fmt.Errorf("invoke contract failed, [code:%d]/[msg:%s]\n", resp.Code, resp.Message)
        }

        if !withSyncResult {
            fmt.Printf("invoke contract success, resp: [code:%d]/[msg:%s]/[txId:%s]\n", resp.Code, resp.Message, resp.ContractResult.Result)
        } else {
            fmt.Printf("invoke contract success, resp: [code:%d]/[msg:%s]/[contractResult:%s]\n", resp.Code, resp.Message, resp.ContractResult)
        }

        return nil
    }
    ```

## 代码编写规则

1. 代码入口

   ```go
   // sdk代码中，有且仅有一个main()方法
   func main() {  
       // main()方法中，下面的代码为必须代码，不建议修改main()方法当中的代码
       // 其中，TestContract为用户实现合约的具体名称
   	err := shim.Start(new(TestContract))
   	if err != nil {
   		log.Fatal(err)
   	}
   }
   ```



2. 合约必要代码

   ```go
   // 合约结构体，合约名称需要写入main()方法当中
   type TestContract struct {
   }
   
   // 合约必须实现下面两个方法：
   // InitContract(stub shim.CMStubInterface) protogo.Response
   // InvokeContract(stub shim.CMStubInterface) protogo.Response
   
   // 用于合约的部署和升级
   // @param stub: 合约接口
   // @return: 	合约返回结果，包括Success和Error
   func (t *TestContract) InitContract(stub shim.CMStubInterface) protogo.Response {
   
   	return shim.Success([]byte("Init Success"))
   
   }
   
   // 用于合约的调用
   // @param stub: 合约接口
   // @return: 	合约返回结果，包括Success和Error
   func (t *TestContract) InvokeContract(stub shim.CMStubInterface) protogo.Response {
   
   	return shim.Success([]byte("Invoke Success"))
   
   }
   ```



## 编译说明

用户如果手工编译，需要将SDK和智能合约放入同一个文件夹，同时保证是在Linux环境下编译，在此文件夹的当前路径执行如下编译命令：

```bash
go build -o contract_name

7z a zip_file_name contract_name
```

在编译合约时，首先使用golang编译程序，**保证contract_name和接下来发起交易使用的合约名字一致**

编译后使用7zip对编译好的程序进行压缩，目前用户上传的合约都是可执行文件，**保证合约编译的环境也安装好7zip工具**



## 接口描述

用户与链交互接口

```go
type CMStubInterface interface {
    // GetArgs get arg from transaction parameters
    // @return: 参数map
    GetArgs() map[string]string
    // GetState get [key] from chain and db
    // @param key: 获取的参数名
    // @return1: 获取结果
    // @return2: 获取错误信息
    GetState(key []byte) ([]byte, error)
    // PutState put [key, value] to chain
    // @param1 key: 参数名
    // @param2 value: 参数值
    // @return1: 上传参数错误信息
    PutState(key []byte, value []byte) error 
    // DelState delete [key] to chain
    // @param1 key: 删除的参数名 
    //@return1：删除参数的错误信息 
    DelState(key []byte) error 
    // GetCreatorOrgId get tx creator org id 
    //@return1: 合约创建者的组织ID
    // @return2: 获取错误信息
    GetCreatorOrgId() (string, error)
    // GetCreatorRole get tx creator role
    // @return1: 合约创建者角色
    // @return2: 获取错误信息
    GetCreatorRole() (string, error)
    // GetCreatorPk get tx creator pk
    // @return1: 合约创建者的公钥
    // @return2: 获取错误信息
    GetCreatorPk() (string, error)
    // GetSenderOrgId get tx sender org id
    // @return1: 交易发起者的组织ID
    // @return2: 获取错误信息
    GetSenderOrgId() (string, error)
    // GetSenderRole get tx sender role
    // @return1: 交易发起者的角色
    // @return2: 获取错误信息
    GetSenderRole() (string, error)
    // GetSenderPk get tx sender pk
    // @return1: 交易发起者的公钥
    // @return2: 获取错误信息
    GetSenderPk() (string, error)
    // GetBlockHeight get tx block height
    // @return1: 当前块高度
    // @return2: 获取错误信息
    GetBlockHeight() (int, error)
    // GetTxId get current tx id
    // @return1: 交易ID
    // @return2: 获取错误信息
    GetTxId() (string, error)
    // EmitEvent emit event, you can subscribe to the event using the SDK
    // @param1 topic: 合约事件的主题
    // @param2 data: 合约事件的数据，参数数量不可大于16
    EmitEvent(topic string, data []string)
    // Log record log to chain server
    // @param message: 事情日志的信息
    Log(message string)
}

```

