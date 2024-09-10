package main

import (
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"ser163.cn/earthworm/config"
	"ser163.cn/earthworm/feishu"
	"ser163.cn/earthworm/read"
)

func main() {
	conf := config.GetConfig()
	db, err := ConnectDatabase(conf)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	// 获取业务本地数据
	Mysqldb, err := ConnectMysqlDatabase(conf)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()
	defer Mysqldb.Close()
	// 获取需要更新的数据
	readClient := read.NewReadLib(Mysqldb, db)

	records, err := readClient.Transfer()
	if err != nil {
		log.Fatalf("Error transfer from read: %v", err)
		os.Exit(1)
	}
	// 调用飞书方法
	feishuClient := feishu.NewFeiShuLib(db)

	// 新建飞书任务字段
	_, err = feishuClient.NewBatchCreateRecord(records)
	if err != nil {
		log.Fatalf("Error creating records: %v", err)
	}

	// 更新本地记录
	err = readClient.UploadLocalRecord()
	if err != nil {
		log.Fatalf("Error uploading record: %v", err)
		os.Exit(1)
	}

}
