package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	// 数据库连接
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		"localhost", 5432, "api2openclaw", "api2openclaw123", "api2openclaw", "disable",
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	// 读取迁移文件
	migrations := []string{
		"migrations/002_create_admin_users.up.sql",
		"migrations/003_create_models.up.sql",
		"migrations/004_create_api_keys.up.sql",
		"migrations/005_create_request_logs.up.sql",
	}

	for _, file := range migrations {
		content, err := os.ReadFile(file)
		if err != nil {
			log.Printf("Warning: could not read %s: %v", file, err)
			continue
		}

		_, err = db.Exec(string(content))
		if err != nil {
			log.Printf("Warning: migration %s failed: %v", file, err)
		} else {
			log.Printf("Migration %s completed", file)
		}
	}

	log.Println("All migrations completed")
}
