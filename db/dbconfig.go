package db

import (
	"fmt"
)

type DBConfig struct {
	Host     string
	Port     int
	DBName   string
	User     string
	Password string
}

// Generate DSN
func (c DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		c.Host, c.Port, c.User, c.Password, c.DBName,
	)
}
