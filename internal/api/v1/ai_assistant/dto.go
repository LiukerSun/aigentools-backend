package ai_assistant

type AnalyzeImageRequest struct {
	ImageURL string `json:"imageUrl" binding:"required"`
	Template string `json:"template" binding:"required,oneof=nsfw ecommerce"`
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
