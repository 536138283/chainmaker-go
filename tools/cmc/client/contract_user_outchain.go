/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/sym"
	"github.com/mr-tron/base58"
)

//链外数据存储，先将文件读取后计算出fileHash，然后使用密钥对文件内容加密，加密后的数据和文件哈希一起存入数据库
//敏感词过滤，维护在filter表中的敏感词，如果要存证的链外数据包含敏感词，存证失败。如果已经存证的数据包含敏感词，读取失败

//// File 外部文件
//type File struct {
//	FileHash []byte `gorm:"size:128;primaryKey;default:''"`
//	FileData []byte `gorm:"type:longblob"`
//}

// getFilterList 获得敏感词列表
// @param db
// @return []string
func getFilterList(db *sql.DB) []string {
	rows, err := db.Query("SELECT obj_value FROM filter")
	if err != nil || rows.Err() != nil {
		log.Fatalln(err)
	}
	filterList := make([]string, 0)
	defer rows.Close()
	for rows.Next() {
		var s string
		err = rows.Scan(&s)
		if err != nil {
			log.Fatalln(err)
		}
		filterList = append(filterList, s)
	}
	return filterList
}

// invokeOutUserContract 存证链外数据
// @return error
func invokeOutUserContract() error {
	//将读取的文件计算哈希，然后对称加密，最后存入数据库
	dsn := dbUser + ":" + dbPass + "@tcp(" + dbHost + ":" + dbPort + ")/" + dbName + "?loc=Local&parseTime=true"

	var db *sql.DB
	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		fmt.Printf("open db fail:%s", err)
	}

	err = db.Ping()
	if err != nil {
		fmt.Printf("ping db fail:%s", err)
	}
	db.SetMaxOpenConns(200)
	db.SetMaxIdleConns(20)
	defer db.Close()
	filterList := getFilterList(db)
	outFileBytes, err := ioutil.ReadFile(outFilePath)
	if err != nil {
		return err
	}
	//检查是否含有敏感词
	for _, filter := range filterList {
		if strings.Contains(string(outFileBytes), filter) {
			fmt.Printf("Contains sensitive data(%s)\n", filter)
			return nil
		}
	}
	outFileHash := sha256.Sum256(outFileBytes)
	fileHash := base58.Encode(outFileHash[:])

	key, err := base58.Decode(sm4Key)
	if err != nil {
		return err
	}
	sm4, err := sym.GenerateSymKey(crypto.SM4, key)
	if err != nil {
		return err
	}
	outFileEncrypt, err := sm4.Encrypt(outFileBytes)
	if err != nil {
		return err
	}
	//保存文件Hash和加密后的数据到数据库
	sql := "insert into files(file_hash,file_data) values('?','?')"
	_, err = db.Exec(sql, fileHash, base58.Encode(outFileEncrypt))
	if err != nil {
		log.Println("exec failed:", err, ", sql:", sql)
	}
	//对文件Hash进行存证
	//先将文件Hash放入Parameter中
	kvsMap := make(map[string]string)
	err = json.Unmarshal([]byte(params), &kvsMap)
	if err != nil {
		return err
	}
	kvsMap["file_hash"] = fileHash
	fmt.Printf("[file_hash:%s]\n", fileHash)
	pJson, _ := json.Marshal(kvsMap)
	params = string(pJson)
	//调用合约
	return invokeUserContract()
}

// getOutUserContract 查询链外数据
// @return error
func getOutUserContract() error {
	//参数准备
	key, err := base58.Decode(sm4Key)
	if err != nil {
		return fmt.Errorf("file data err:%s", err)
	}
	sm4, err := sym.GenerateSymKey(crypto.SM4, key)
	if err != nil {
		return fmt.Errorf("file data err:%s", err)
	}

	dsn := dbUser + ":" + dbPass + "@tcp(" + dbHost + ":" + dbPort + ")/" + dbName + "?loc=Local&parseTime=true"
	//读取数据库
	var db *sql.DB
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		fmt.Printf("open db fail:%s", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		fmt.Printf("ping db fail:%s", err)
	}
	db.SetMaxOpenConns(200)
	db.SetMaxIdleConns(20)
	filterList := getFilterList(db)

	pairs := make(map[string]interface{})
	if params != "" {
		err := json.Unmarshal([]byte(params), &pairs)
		if err != nil {
			return err
		}
	}

	//先查询数据库文件数据，并解密出明文
	rows, err := db.Query("SELECT file_data FROM files where file_hash='?'", pairs["file_hash"])
	if err != nil || rows.Err() != nil {
		log.Fatalln(err)
	}
	defer rows.Close()
	f := false
	for rows.Next() {
		var s string
		err = rows.Scan(&s)
		s1, err := base58.Decode(s)
		if err != nil {
			return fmt.Errorf("file data err")
		}

		if len(s1) < 16 {
			return fmt.Errorf("file data err")
		}
		ss, err := sm4.Decrypt(s1)
		if err != nil {
			return fmt.Errorf("file data err")
		}
		outFileHash := sha256.Sum256([]byte(ss))
		fileHash := base58.Encode(outFileHash[:])
		if fileHash != pairs["file_hash"] {
			return fmt.Errorf("get error file")
		}
		//检查敏感词
		for _, filter := range filterList {
			if strings.Contains(string(ss), filter) {
				fmt.Printf("Contains sensitive data(%s)\n", s)
				return nil
			}
		}
		//解密后明文写出到文件
		err = ioutil.WriteFile(outFilePath, []byte(ss), os.ModePerm)
		if err != nil {
			return err
		}
		f = true
		break

	}

	if !f {
		return fmt.Errorf("lost file")
	}
	//存证的链上查询
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()

	resp, err := client.QueryContract(contractName, method, util.ConvertParameters(pairs), -1)
	if err != nil {
		return fmt.Errorf("query contract failed, %s", err.Error())
	}
	util.PrintPrettyJson(resp)
	return nil
}
