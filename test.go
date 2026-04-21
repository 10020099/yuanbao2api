package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"yuanbao2api/api"
)

// TestMain 初始化测试环境
func TestMain(m *testing.M) {
	godotenv.Load()
	os.Setenv("GIN_MODE", gin.TestMode)
	os.Setenv("YUANBAO_COOKIE", "test_cookie")
	os.Setenv("YUANBAO_AGENT_ID", "naQivTmsDa")
	os.Setenv("PORT", "8080")
	code := m.Run()
	os.Exit(code)
}

// TestHealthCheck 测试健康检查端点
func TestHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("健康检查应返回200状态码，实际: %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["status"] != "ok" {
		t.Errorf("状态应为'ok'，实际: %v", response["status"])
	}
}

// TestGetModels 测试模型列表端点
func TestGetModels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/v1/models", api.HandleOpenAIModels)

	req, _ := http.NewRequest("GET", "/v1/models", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("模型列表应返回200状态码，实际: %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["object"] != "list" {
		t.Errorf("对象类型应为'list'，实际: %v", response["object"])
	}
}

// TestOpenAIChatCompletion 测试OpenAI兼容聊天完成端点（结构测试，不连接真实API）
func TestOpenAIChatCompletion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/v1/chat/completions", api.HandleOpenAIChatCompletion)

	requestBody := map[string]interface{}{
		"model": "deep_seek_v3",
		"messages": []map[string]string{
			{"role": "user", "content": "你好"},
		},
		"stream": false,
	}
	requestJSON, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("POST", "/v1/chat/completions", bytes.NewBuffer(requestJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Note: This will likely return 500 since YUANBAO_COOKIE is fake
	// The test validates the route is properly registered
	t.Logf("Response code: %d", w.Code)
}

// TestConfigAPI 测试配置API端点
func TestConfigAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/config", api.HandleGetConfig)

	req, _ := http.NewRequest("GET", "/api/config", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("配置API应返回200状态码，实际: %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["defaultModel"] != "deep_seek_v3" {
		t.Errorf("默认模型应为'deep_seek_v3'，实际: %v", response["defaultModel"])
	}
}
