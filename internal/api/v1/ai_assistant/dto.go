package ai_assistant

import "aigentools-backend/internal/models"

type AnalyzeImageRequest struct {
	ImageURL string `json:"imageUrl" binding:"required"`
	Template string `json:"template" binding:"required"` // "nsfw", "ecommerce", or "custom"
	Prompt   string `json:"prompt"`                      // Required if Template is "custom" (contains Template ID)
}

type AnalyzeImageResponse struct {
	Result string `json:"result"`
}

// OpenAI/Grok compatible request structures for internal use
type ChatMessageContentImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

type ChatMessageContentItem struct {
	Type     string                      `json:"type"`
	Text     string                      `json:"text,omitempty"`
	ImageURL *ChatMessageContentImageURL `json:"image_url,omitempty"`
}

type ChatMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or []ChatMessageContentItem
}

type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Stream      bool          `json:"stream"`
	Temperature float64       `json:"temperature"`
}

type ChatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type CreatePromptRequest struct {
	Code    string `json:"code" binding:"required"`
	Content string `json:"content" binding:"required"`
}

type BatchCreatePromptRequest struct {
	Prompts []CreatePromptRequest `json:"prompts" binding:"required,dive"`
}

type UpdatePromptRequest struct {
	Content string `json:"content" binding:"required"`
}

type PromptListResponse struct {
	Total int64           `json:"total"`
	Items []models.Prompt `json:"items"`
}

type CreateTemplateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Content     string `json:"content" binding:"required"`
	IsPublic    bool   `json:"is_public"`
}

type UpdateTemplateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
	IsPublic    *bool  `json:"is_public"` // Pointer to allow distinguishing between false and nil
}

type TemplateListResponse struct {
	Total int64                   `json:"total"`
	Items []models.PromptTemplate `json:"items"`
}
