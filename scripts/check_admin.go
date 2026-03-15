package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		"localhost", 5432, "api2openclaw", "api2openclaw123", "api2openclaw", "disable",
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM admin_users").Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Admin users count: %d\n", count)

	var username, passwordHash string
	err = db.QueryRow("SELECT username, password_hash FROM admin_users LIMIT 1").Scan(&username, &passwordHash)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Username: %s\n", username)
	fmt.Printf("Password hash: %s\n", passwordHash)
}
