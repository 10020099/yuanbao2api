package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"yuanbao2api/internal/models"
	"yuanbao2api/yuanbao"
)

// HandleOpenAIModels returns the list of available models
func HandleOpenAIModels(c *gin.Context) {
	c.JSON(http.StatusOK, models.ModelsResponse{
		Object: "list",
		Data: []models.ModelInfo{
			{
				ID:          "DeepSeek-V3.2",
				Object:      "model",
				Created:     1704067200,
				OwnedBy:     "yuanbao",
				Permission:  []interface{}{},
				Root:        "DeepSeek-V3.2",
				Parent:      nil,
				Description: "DeepSeek V3.2 - 适合深度思考和复杂推理任务",
			},
			{
				ID:          "hunyuan-t1",
				Object:      "model",
				Created:     1704067200,
				OwnedBy:     "yuanbao",
				Permission:  []interface{}{},
				Root:        "hunyuan-t1",
				Parent:      nil,
				Description: "Hunyuan T1 - 腾讯混元大模型，全能处理",
			},
			{
				ID:          "gpt_175B_0404",
				Object:      "model",
				Created:     1704067200,
				OwnedBy:     "yuanbao",
				Permission:  []interface{}{},
				Root:        "gpt_175B_0404",
				Parent:      nil,
				Description: "元宝内部模型标识",
			},
		},
	})
}

// ModelConfig holds the mapping configuration for a model
type ModelConfig struct {
	ChatModelID string
	Model       string
	Name        string
	Description string
}

// MODEL_MAPPING maps model names to their Yuanbao configuration
var MODEL_MAPPING = map[string]ModelConfig{
	"DeepSeek-V3.2": {
		ChatModelID: "deep_seek_v3",
		Model:       "gpt_175B_0404",
		Name:        "DeepSeek V3.2",
		Description: "适合深度思考和复杂推理任务",
	},
	"deep_seek_v3": {
		ChatModelID: "deep_seek_v3",
		Model:       "gpt_175B_0404",
		Name:        "DeepSeek V3.2",
		Description: "适合深度思考和复杂推理任务",
	},
	"deepseek": {
		ChatModelID: "deep_seek_v3",
		Model:       "gpt_175B_0404",
		Name:        "DeepSeek V3.2",
		Description: "适合深度思考和复杂推理任务",
	},
	"hunyuan-t1": {
		ChatModelID: "hunyuan",
		Model:       "gpt_175B_0404",
		Name:        "Hunyuan T1",
		Description: "腾讯混元大模型，全能处理",
	},
	"hunyuan": {
		ChatModelID: "hunyuan",
		Model:       "gpt_175B_0404",
		Name:        "Hunyuan T1",
		Description: "腾讯混元大模型，全能处理",
	},
	"gpt_175B_0404": {
		ChatModelID: "deep_seek_v3",
		Model:       "gpt_175B_0404",
		Name:        "GPT 175B",
		Description: "元宝内部模型",
	},
}

// GetModelConfig returns the model configuration for a given model name
func GetModelConfig(modelName string) ModelConfig {
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(modelName, "-", ""), "_", ""))

	if config, ok := MODEL_MAPPING[modelName]; ok {
		return config
	}

	for key, config := range MODEL_MAPPING {
		keyNormalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "-", ""), "_", ""))
		if keyNormalized == normalized {
			return config
		}
	}

	return MODEL_MAPPING["DeepSeek-V3.2"]
}

// buildYuanbaoRequest builds the Yuanbao API request from parameters
func buildYuanbaoRequest(prompt string, modelConfig ModelConfig, useDeepThinking bool, useInternetSearch bool, agentID string) YuanbaoRequest {
	supportFunctions := []string{}
	if useInternetSearch {
		supportFunctions = append(supportFunctions, "openInternetSearch")
	} else {
		supportFunctions = append(supportFunctions, "closeInternetSearch")
	}

	chatModelID := modelConfig.ChatModelID
	modelID := modelConfig.ChatModelID
	subModelID := ""

	if useDeepThinking {
		if strings.Contains(modelConfig.ChatModelID, "hunyuan") {
			chatModelID = "hunyuan_t1"
			modelID = "hunyuan_t1"
			subModelID = "hunyuan_t1"
		} else {
			chatModelID = "deep_seek"
			modelID = "deep_seek_v3"
			subModelID = "deep_seek"
		}
	}

	displayPrompt := prompt
	if len(prompt) > 2000 {
		displayPrompt = prompt[:2000] + "...[已截断]"
	}

	plugin := "Adaptive"
	if useDeepThinking {
		plugin = ""
	}

	internetSearchFn := "closeInternetSearch"
	if useInternetSearch {
		internetSearchFn = "openInternetSearch"
	}

	extInfo, _ := json.Marshal(map[string]interface{}{
		"modelId":    modelID,
		"subModelId": subModelID,
		"supportFunctions": map[string]string{
			"internetSearch": internetSearchFn,
		},
	})

	return YuanbaoRequest{
		Model:             modelConfig.Model,
		Prompt:            prompt,
		Plugin:            plugin,
		DisplayPrompt:     displayPrompt,
		DisplayPromptType: 1,
		AgentID:           agentID,
		IsTemporary:       true,
		ProjectID:         "",
		ChatModelID:       chatModelID,
		SupportFunctions:  supportFunctions,
		DocOpenID:         "",
		Options: map[string]interface{}{
			"imageIntention": map[string]interface{}{
				"needIntentionModel": true,
				"backendUpdateFlag":  2,
				"intentionStatus":    true,
			},
		},
		Multimedia:        []interface{}{},
		SupportHint:       1,
		ChatModelExtInfo:  string(extInfo),
		ApplicationIDList: []string{},
		Version:           "v2",
		ExtReportParams:   nil,
		IsAtomInput:       false,
		OffsetOfHour:      8,
		OffsetOfMinute:    0,
	}
}

// YuanbaoRequest aliases the yuanbao package's YuanbaoRequest type
type YuanbaoRequest = yuanbao.YuanbaoRequest
