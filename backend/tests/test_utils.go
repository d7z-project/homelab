package tests

import (
	"homelab/pkg/common"
	"log"

	"gopkg.d7z.net/middleware/kv"
)

// SetupTestDB 初始化一个内存数据库用于测试
// 返回一个清理函数，用于在测试结束时关闭数据库
func SetupTestDB() func() {
	db, err := kv.NewKVFromURL("memory://")
	if err != nil {
		log.Fatalf("failed to create test db: %v", err)
	}

	// 保存旧的 DB 引用以便恢复（如果需要）
	oldDB := common.DB
	common.DB = db

	return func() {
		db.Close()
		common.DB = oldDB
	}
}
