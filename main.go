package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"yuanbao2api/api"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// Set Gin mode based on environment
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		ginMode = gin.DebugMode
	}
	gin.SetMode(ginMode)

	// Create Gin router
	r := gin.Default()

	// CORS middleware for management panel
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, Accept, X-Requested-With, anthropic-version, x-api-key")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Serve static files (management panel)
	r.Static("/", "./public")

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API routes
	v1 := r.Group("/v1")
	{
		v1.GET("/models", api.HandleOpenAIModels)
		v1.POST("/chat/completions", api.HandleOpenAIChatCompletion)
		v1.POST("/messages", api.HandleAnthropicMessages)
	}

	// Management panel config API
	config := r.Group("/api")
	{
		config.GET("/config", api.HandleGetConfig)
		config.POST("/config", api.HandleSetConfig)
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Printf("Yuanbao2API server starting on port %s", port)
	log.Printf("\n📊 管理面板: http://localhost:%s", port)
	log.Printf("\n配置说明：")
	log.Printf("1. 设置环境变量 YUANBAO_COOKIE（从浏览器复制）")
	log.Printf("2. 可选：设置环境变量 YUANBAO_AGENT_ID（默认: naQivTmsDa）")
	log.Printf("\n✨ 使用临时对话模式，每次请求自动创建新会话")
	log.Printf("\n功能特性：")
	log.Printf("- 深度思考模式（deep_thinking: true）")
	log.Printf("- 联网搜索（internet_search: true）")
	log.Printf("\nAPI 端点：")
	log.Printf("- OpenAI:     POST /v1/chat/completions")
	log.Printf("- Anthropic:  POST /v1/messages")
	log.Printf("- Models:     GET  /v1/models")
	log.Printf("- Config:     GET/POST /api/config")
	log.Printf("\n使用示例：")
	log.Printf("curl http://localhost:%s/v1/chat/completions \\", port)
	log.Printf("  -H \"Content-Type: application/json\" \\")
	log.Printf("  -d '{\"model\":\"deep_seek_v3\",\"messages\":[{\"role\":\"user\",\"content\":\"你好\"}],\"deep_thinking\":true}'")

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
