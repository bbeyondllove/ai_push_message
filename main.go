package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/swaggo/swag" // 导入 swag

	"ai_push_message/config"
	"ai_push_message/db"
	_ "ai_push_message/docs" // 导入 swagger 文档
	"ai_push_message/handlers"
	"ai_push_message/logger"
	"ai_push_message/scheduler"
)

func main() {
	cfg := config.Load()

	// 初始化日志系统
	if err := logger.Init(cfg); err != nil {
		log.Fatalf("init logger failed: %v", err)
	}
	logger.Info("日志系统初始化成功", "level", cfg.Log.Level, "format", cfg.Log.Format, "output", cfg.Log.Output)

	if err := db.InitMySQLWithConfig(cfg); err != nil {
		logger.Error("初始化MySQL失败", "error", err)
		os.Exit(1)
	}
	logger.Info("MySQL连接成功",
		"max_open_conns", cfg.DB.MaxOpenConns,
		"max_idle_conns", cfg.DB.MaxIdleConns,
		"conn_max_lifetime", cfg.DB.ConnMaxLifetime)

	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	handlers.RegisterRoutes(r, cfg)

	// start cron
	scheduler.Start(cfg)

	serverAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	logger.Info("服务器启动", "address", serverAddr)
	logger.Info("Swagger文档可访问", "url", fmt.Sprintf("http://%s/swagger/index.html", serverAddr))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Server.Port), r))
}
