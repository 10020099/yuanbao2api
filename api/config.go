package api

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

// ServerConfig holds the dynamic server configuration (same as internal/config but for API layer)
var (
	serverConfig     = &ServerConfigData{DeepThinking: false, InternetSearch: false, DefaultModel: "deep_seek_v3"}
	serverConfigLock sync.RWMutex
)

// ServerConfigData represents the server configuration
type ServerConfigData struct {
	DeepThinking   bool   `json:"deepThinking"`
	InternetSearch bool   `json:"internetSearch"`
	DefaultModel   string `json:"defaultModel"`
}

// HandleGetConfig returns the current server configuration
func HandleGetConfig(c *gin.Context) {
	serverConfigLock.RLock()
	defer serverConfigLock.RUnlock()
	c.JSON(http.StatusOK, serverConfig)
}

// HandleSetConfig updates the server configuration
func HandleSetConfig(c *gin.Context) {
	var req ServerConfigData
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	serverConfigLock.Lock()
	defer serverConfigLock.Unlock()

	if req.DeepThinking != serverConfig.DeepThinking {
		serverConfig.DeepThinking = req.DeepThinking
	}
	if req.InternetSearch != serverConfig.InternetSearch {
		serverConfig.InternetSearch = req.InternetSearch
	}
	if req.DefaultModel != "" {
		serverConfig.DefaultModel = req.DefaultModel
	}

	c.JSON(http.StatusOK, serverConfig)
}

// GetServerConfig returns a copy of the current server config
func GetServerConfig() ServerConfigData {
	serverConfigLock.RLock()
	defer serverConfigLock.RUnlock()
	return *serverConfig
}
