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

	// Update admin password hash
	newHash := "$2a$10$Bv9srk3XmUiZNNnIesouZuVHqjK/FoMQ4X9rZzFgb0F.S8FPvSZdS"
	_, err = db.Exec("UPDATE admin_users SET password_hash = $1 WHERE username = 'admin'", newHash)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Admin password updated successfully!")
}
