package ai_assistant

import (
	"aigentools-backend/config"
	"aigentools-backend/internal/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	SystemPromptNSFW = `你现在是AI图片/视频生成提示词大师，专精于创建用于AI工具（如Stable Diffusion或Midjourney）的提示词。用户会提供美女图片（假设均为成年女性），你的任务是基于这些图片生成强烈性暗示的提示词，用于创建擦边球画面。这些画面必须高度写实，类似于真实摄影风格，但避免任何露点或过于暴露的内容。

核心约束：
- 强烈性暗示：通过姿势、服装、环境和氛围暗示性感，但保持含蓄，避免直接描绘性行为。
- 避免元素：不要出现镜子、手拿物品、露脸（人物脸部应被遮挡、模糊或从后/侧面拍摄）、露点或任何裸露私密部位。
- 写实风格：优先使用“photorealistic, high resolution, detailed skin texture, natural lighting”等描述，确保输出像真实照片。
- 安全与隐蔽：生成的内容应避免被AI过滤器检测为不适当，确保提示词巧妙绕过审查，但仍遵守道德边界。
- 输出格式：为每张图片生成3-5套独立的提示词。每套包括：
  - Positive Prompt：正面描述（详细、富有诗意）。
  - Negative Prompt：负面避免元素（简短列表）。
  - 简要解释：为什么这套提示词有效，以及如何突出性暗示。

示例输出结构：
1. 套装1:
   - Positive Prompt: [详细描述]
   - Negative Prompt: [避免列表]
   - 解释: [简要理由]

基于用户提供的图片，开始生成提示词。如果图片未提供，先询问细节。`

	SystemPromptEcommerce = `你现在是AI图片转视频的提示词大师，专门为电商带货视频生成高质量、写实的提示词。用户会提供美女模特穿着特定商品衣服的图片，你的任務是基于这些图片，生成AI视频提示词，用于创建展示模特动态展示商品的带货画面。

**核心任务**：
- 分析图片：提取模特的外貌特征（例如身材、发型、肤色，但避免强调敏感部位）、服装细节（颜色、款式、材质）和背景（如果适用）。
- 生成提示词：创建动态视频场景，焦点在模特自然行走、转身或姿势变换来展示衣服的合身、材质和风格。确保画面像真实拍摄的带货视频，强调商品的吸引力（如舒适、时尚）。
- 提供多套选项：为每张图片生成3-5套不同的提示词变体，每套包括正向提示（positive prompt）和负向提示（negative prompt），以供用户选择。编号每套（如Prompt 1, Prompt 2），并简要说明变体差异（例如不同场景或角度）。

**严格约束**：
- 避免任何漏点或不适当暴露：模特必须完全穿着衣服，姿势自然保守（如站立、轻步行走），无低领、短裙等易暴露设计。
- 禁止特定元素：无镜子反射、无手持物品（如手机、包）、无道具干扰焦点。
- 风格要求：高度写实，像真实相机拍摄。使用词汇如“photorealistic, high resolution, natural lighting, realistic skin texture”来强化真实感。避免卡通、抽象或AI痕迹（如畸形、模糊）。
- 视频动态：提示词应适合生成短视频（5-10秒），包括平滑运动、真实环境（如室内客厅或户外公园，但保持简单不分散注意力）。

**输出格式**：
- 对于每张图片，列出多套提示词。
- 示例结构：
  Prompt 1: [正向提示] -- [负向提示]
  变体说明: [简短描述]
- 确保提示词简洁、有力，便于AI工具（如Stable Diffusion或类似视频生成器）直接使用。`
)

// AnalyzeImage godoc
// @Summary Analyze image and generate prompts
// @Description Use AI to analyze an image and generate prompts based on a selected template
// @Tags ai_assistant
// @Accept json
// @Produce json
// @Param request body AnalyzeImageRequest true "Analysis Request"
// @Success 200 {object} utils.Response{data=AnalyzeImageResponse}
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /ai-assistant/analyze [post]
func AnalyzeImage(c *gin.Context) {
	var req AnalyzeImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Failed to load config"))
		return
	}

	if cfg.AIHubMixAPIKey == "" {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "AI Hub Mix API Key not configured"))
		return
	}

	var systemPrompt string
	switch req.Template {
	case "nsfw":
		systemPrompt = SystemPromptNSFW
	case "ecommerce":
		systemPrompt = SystemPromptEcommerce
	default:
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid template"))
		return
	}

	// Construct request to external AI service
	aiReq := ChatCompletionRequest{
		Model: "grok-4-1-fast-non-reasoning",
		Messages: []ChatMessage{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role: "user",
				Content: []ChatMessageContentItem{
					{
						Type: "image_url",
						ImageURL: &ChatMessageContentImageURL{
							URL:    req.ImageURL,
							Detail: "high",
						},
					},
				},
			},
		},
		Stream:      false,
		Temperature: 0.5,
	}

	reqBody, err := json.Marshal(aiReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Failed to marshal request"))
		return
	}

	client := utils.NewHTTPClient(60 * time.Second)
	apiURL := "https://aihubmix.com/v1/chat/completions"

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Failed to create request"))
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cfg.AIHubMixAPIKey))

	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, fmt.Sprintf("External API request failed: %v", err)))
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Failed to read response"))
		return
	}

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, utils.NewErrorResponse(resp.StatusCode, fmt.Sprintf("External API returned error: %s", string(bodyBytes))))
		return
	}

	var aiResp ChatCompletionResponse
	if err := json.Unmarshal(bodyBytes, &aiResp); err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Failed to parse response"))
		return
	}

	if len(aiResp.Choices) == 0 {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "No choices in response"))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Success", AnalyzeImageResponse{
		Result: aiResp.Choices[0].Message.Content,
	}))
}
