package db

import (
	"database/sql"
	"time"

	"ai_push_message/config"

	_ "github.com/go-sql-driver/mysql"
)

var (
	DB *sql.DB // 数据库连接
)

// InitMySQL 初始化数据库连接
func InitMySQL(dsn string) error {
	var err error
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	return DB.Ping()
}

// InitMySQLWithConfig 使用配置初始化数据库连接池
func InitMySQLWithConfig(cfg *config.Config) error {
	var err error
	DB, err = sql.Open("mysql", cfg.DB.DSN)
	if err != nil {
		return err
	}

	// 从配置读取连接池参数，提供默认值保护
	maxOpenConns := cfg.DB.MaxOpenConns
	if maxOpenConns <= 0 {
		maxOpenConns = 50 // 默认最大连接数
	}

	maxIdleConns := cfg.DB.MaxIdleConns
	if maxIdleConns <= 0 {
		maxIdleConns = 10 // 默认最大空闲连接数
	}

	connMaxLifetime := cfg.DB.ConnMaxLifetime
	if connMaxLifetime <= 0 {
		connMaxLifetime = 60 // 默认连接最大生命周期（分钟）
	}

	DB.SetMaxOpenConns(maxOpenConns)
	DB.SetMaxIdleConns(maxIdleConns)
	DB.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Minute)

	return DB.Ping()
}
