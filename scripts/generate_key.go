package main

import (
	"context"
	"fmt"
	"log"

	"github.com/openclaw/api2openclaw/internal/auth"
)

func main() {
	// 创建认证管理器（需要数据库连接）
	store, err := auth.NewPostgreSQLStore("host=localhost port=5432 user=api2openclaw password=api2openclaw123 dbname=api2openclaw sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer store.Close()

	mgr := auth.NewManager(store)

	// 创建 API Key 请求
	req := &auth.CreateAPIKeyRequest{
		TenantID:      "default",
		Permissions:   []string{"*"},
		AllowedModels: []string{"*"},
	}

	// 生成新的 API Key
	resp, err := mgr.GenerateAPIKey(context.Background(), req)
	if err != nil {
		log.Fatalf("Failed to generate API key: %v", err)
	}

	fmt.Println("✓ API Key 生成成功")
	fmt.Printf("Key ID: %s\n", resp.KeyID)
	fmt.Printf("Key Secret: %s\n", resp.KeySecret)
	fmt.Printf("完整密钥: %s.%s\n", resp.KeyID, resp.KeySecret)
	fmt.Printf("租户: %s\n", resp.TenantName)
}
