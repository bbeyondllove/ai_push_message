package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
		Addr string `yaml:"-"` // 不从配置文件读取，而是在加载后计算
	} `yaml:"server"`
	ExternalAPI struct {
		TagPushURL string `yaml:"tag_push_url"`
		APIKey     string `yaml:"api_key"`
	} `yaml:"external_api"`
	SiliconFlow struct {
		APIKey         string `yaml:"api_key"`
		Model          string `yaml:"model"`
		BaseURL        string `yaml:"base_url"`
		MaxTokenLength int    `yaml:"max_token_length"`
	} `yaml:"siliconflow"`
	Log struct {
		Level    string `yaml:"level"`
		Format   string `yaml:"format"`
		Output   string `yaml:"output"`
		FilePath string `yaml:"file_path"`
	} `yaml:"log"`

	DB struct {
		Host            string `yaml:"host"`
		Port            int    `yaml:"port"`
		Username        string `yaml:"username"`
		Password        string `yaml:"password"`
		Database        string `yaml:"database"`
		Charset         string `yaml:"charset"`
		ParseTime       bool   `yaml:"parse_time"`
		DSN             string `yaml:"-"`                 // 不从配置文件读取，而是在加载后计算
		MaxOpenConns    int    `yaml:"max_open_conns"`    // 最大打开连接数
		MaxIdleConns    int    `yaml:"max_idle_conns"`    // 最大空闲连接数
		ConnMaxLifetime int    `yaml:"conn_max_lifetime"` // 连接最大生命周期（分钟）
	} `yaml:"database"`
	Cron struct {
		LookbackDays    int `yaml:"lookback_days"`    // 回溯天数
		ProfileHour     int `yaml:"profile_hour"`     // 每天生成画像的小时（0-23）
		ProfileMin      int `yaml:"profile_min"`      // 每天生成画像的分钟（0-59）
		Concurrency     int `yaml:"concurrency"`      // 用户画像生成并发数
		PushConcurrency int `yaml:"push_concurrency"` // 推送并发数
	} `yaml:"cron"`
	RAG struct {
		URL          string   `yaml:"url"`
		APIKey       string   `yaml:"api_key"`
		KnowledgeIDs []string `yaml:"kb_ids"`
		TopK         int      `yaml:"topk"`        // 搜索返回的最大结果数
		Threshold    float32  `yaml:"threshold"`   // 搜索相似度阈值
		TimeoutSec   int      `yaml:"timeout_sec"` // 请求超时时间,单位:秒
	} `yaml:"rag"`
	LLM struct {
		MaxConcurrency int `yaml:"max_concurrency"` // LLM并发请求数
	} `yaml:"llm"`
	Timeouts struct {
		RequestSec  int `yaml:"request_sec"`  // 请求超时，单位：秒
		ResponseSec int `yaml:"response_sec"` // 响应超时，单位：秒
		IdleSec     int `yaml:"idle_sec"`     // 空闲超时，单位：秒
	} `yaml:"timeouts"`
	Debug struct {
		Enabled            bool `yaml:"enabled"`             // 是否启用debug模式
		RecommendationFreq int  `yaml:"recommendation_freq"` // debug模式下推荐频率，单位：秒
	} `yaml:"debug"`
	Scheduler struct {
		CheckIntervalSec int `yaml:"check_interval_sec"` // 调度器检查间隔（秒）
		DefaultHour      int `yaml:"default_hour"`       // 默认执行小时
		DefaultMinute    int `yaml:"default_minute"`     // 默认执行分钟
	} `yaml:"scheduler"`
}

func Load() *Config {
	// 首先尝试加载.env文件中的环境变量
	_ = godotenv.Load() // 忽略错误，如果.env文件不存在，继续使用系统环境变量

	var cfg Config

	// 尝试从config.yaml文件加载配置
	if data, err := os.ReadFile("config.yaml"); err == nil {
		err = yaml.Unmarshal(data, &cfg)
		if err != nil {
			log.Printf("Error loading config.yaml: %v, falling back to environment variables", err)
			yamlFailed := loadFromEnv()
			return yamlFailed
		}
		log.Println("Loading configuration from config.yaml")

		// 计算 Server.Addr 字段
		cfg.Server.Addr = fmt.Sprintf(":%d", cfg.Server.Port)

		// 从环境变量中加载敏感信息和用户名
		// 数据库用户名和密码
		if envUsername := os.Getenv("DATABASE_USERNAME"); envUsername != "" {
			cfg.DB.Username = envUsername
		}
		if envPassword := os.Getenv("DATABASE_PASSWORD"); envPassword != "" {
			cfg.DB.Password = envPassword
		}

		// RAG API密钥
		if envAPIKey := os.Getenv("RAG_API_KEY"); envAPIKey != "" {
			cfg.RAG.APIKey = envAPIKey
		}

		// SiliconFlow API密钥
		if envAPIKey := os.Getenv("SILICONFLOW_API_KEY"); envAPIKey != "" {
			cfg.SiliconFlow.APIKey = envAPIKey
		}

		// 计算 DB.DSN 字段
		if cfg.DB.DSN == "" {
			// 设置默认值
			if cfg.DB.Charset == "" {
				cfg.DB.Charset = "utf8mb4"
			}

			// 构建DSN
			parseTime := ""
			if cfg.DB.ParseTime {
				parseTime = "&parseTime=true"
			}

			cfg.DB.DSN = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s%s",
				cfg.DB.Username,
				cfg.DB.Password,
				cfg.DB.Host,
				cfg.DB.Port,
				cfg.DB.Database,
				cfg.DB.Charset,
				parseTime)
		}

		return &cfg
	}

	// 如果config.yaml不存在，则完全从环境变量加载配置
	return loadFromEnv()
}

func loadFromEnv() *Config {
	// 当config.yaml加载失败时，创建一个最小配置
	var cfg Config

	// 设置服务器地址
	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Server.Port = p
		}
	}
	cfg.Server.Addr = fmt.Sprintf(":%d", cfg.Server.Port)

	// 只从环境变量中加载敏感信息
	// 数据库配置
	if username := os.Getenv("DATABASE_USERNAME"); username != "" {
		cfg.DB.Username = username
	}
	if password := os.Getenv("DATABASE_PASSWORD"); password != "" {
		cfg.DB.Password = password
	}
	if dsn := os.Getenv("DB_DSN"); dsn != "" {
		cfg.DB.DSN = dsn
	} else if cfg.DB.Host != "" {
		// 只有在没有直接提供DSN且有主机信息时才构建DSN
		parseTime := ""
		if cfg.DB.ParseTime {
			parseTime = "&parseTime=true"
		}
		cfg.DB.DSN = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s%s",
			cfg.DB.Username,
			cfg.DB.Password,
			cfg.DB.Host,
			cfg.DB.Port,
			cfg.DB.Database,
			cfg.DB.Charset,
			parseTime)
	}

	// RAG API密钥
	if apiKey := os.Getenv("RAG_API_KEY"); apiKey != "" {
		cfg.RAG.APIKey = apiKey
	}

	// SiliconFlow API密钥
	if apiKey := os.Getenv("SILICONFLOW_API_KEY"); apiKey != "" {
		cfg.SiliconFlow.APIKey = apiKey
	}

	// 外部API密钥
	if apiKey := os.Getenv("EXTERNAL_API_KEY"); apiKey != "" {
		cfg.ExternalAPI.APIKey = apiKey
	}

	log.Println("配置从环境变量加载，部分配置可能缺失")
	return &cfg
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
