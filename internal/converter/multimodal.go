package converter

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/url"
	"path/filepath"
	"strings"
)

// MultimodalContent 多模态内容类型
type MultimodalContent struct {
	Type string // "text", "image_url", "audio_url", "video_url"

	// 文本内容
	Text string

	// 图片内容
	ImageURL     string // 完整 URL 或 base64 data URL
	ImageData    []byte // 原始图片数据
	ImageMIME    string // MIME 类型
	ImageBase64  bool   // 是否为 base64 编码

	// 音频内容
	AudioURL    string
	AudioData   []byte
	AudioMIME   string
	AudioBase64 bool

	// 视频内容
	VideoURL    string
	VideoData   []byte
	VideoMIME   string
	VideoBase64 bool
}

// MultimodalMessage 多模态消息
type MultimodalMessage struct {
	Role    string             // "system", "user", "assistant", "tool"
	Content []MultimodalContent // 多模态内容数组
	Name    string             // 工具名称（当 role=tool 时）
	ToolID  string             // 工具调用 ID
}

// OpenAIMultimodalMessage OpenAI 多模态格式消息
type OpenAIMultimodalMessage struct {
	Role       string        `json:"role"`
	Content    interface{}   `json:"content"` // string 或 []ContentPart
	Name       string        `json:"name,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
}

// ContentPart OpenAI 多模态内容块
type ContentPart struct {
	Type     string `json:"type"` // "text", "image_url"
	Text     string `json:"text,omitempty"`
	ImageURL *struct {
		URL    string `json:"url"`
		Detail string `json:"detail,omitempty"` // "low", "high", "auto"
	} `json:"image_url,omitempty"`
}

// MultimodalParser 多模态内容解析器
type MultimodalParser struct {
	// 配置选项
	AllowRemoteURLs bool   // 是否允许远程 URL
	MaxImageSize    int64  // 最大图片大小（字节）
	MaxAudioSize    int64  // 最大音频大小（字节）
	AllowedMIMEs    []string // 允许的 MIME 类型
}

// NewMultimodalParser 创建多模态解析器
func NewMultimodalParser() *MultimodalParser {
	return &MultimodalParser{
		AllowRemoteURLs: true,
		MaxImageSize:    20 * 1024 * 1024, // 20MB
		MaxAudioSize:    50 * 1024 * 1024, // 50MB
		AllowedMIMEs: []string{
			"image/jpeg",
			"image/png",
			"image/gif",
			"image/webp",
			"audio/mpeg",
			"audio/mp4",
			"audio/wav",
			"audio/ogg",
		},
	}
}

// ParseOpenAIMultimodalMessage 解析 OpenAI 格式消息
func (p *MultimodalParser) ParseOpenAIMultimodalMessage(data []byte) (*MultimodalMessage, error) {
	var msg OpenAIMultimodalMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("unmarshal openai message: %w", err)
	}

	result := &MultimodalMessage{
		Role:    msg.Role,
		Name:    msg.Name,
		ToolID:  msg.ToolCallID,
	}

	// 解析 content 字段
	switch v := msg.Content.(type) {
	case string:
		// 纯文本
		result.Content = []MultimodalContent{
			{Type: "text", Text: v},
		}
	case []interface{}:
		// 多模态内容数组
		contentParts, err := p.parseContentParts(v)
		if err != nil {
			return nil, err
		}
		result.Content = contentParts
	default:
		return nil, fmt.Errorf("unsupported content type: %T", msg.Content)
	}

	return result, nil
}

// parseContentParts 解析内容块数组
func (p *MultimodalParser) parseContentParts(parts []interface{}) ([]MultimodalContent, error) {
	var result []MultimodalContent

	for _, part := range parts {
		partMap, ok := part.(map[string]interface{})
		if !ok {
			continue
		}

		contentType, _ := partMap["type"].(string)

		switch contentType {
		case "text":
			text, _ := partMap["text"].(string)
			result = append(result, MultimodalContent{
				Type: "text",
				Text: text,
			})

		case "image_url":
			if imageURL, ok := partMap["image_url"].(map[string]interface{}); ok {
				urlStr, _ := imageURL["url"].(string)
				_, _ = imageURL["detail"].(string) // Reserved for future use

				content := MultimodalContent{
					Type:      "image_url",
					ImageURL:  urlStr,
				}

				// 解析 URL
				if strings.HasPrefix(urlStr, "data:") {
					// Base64 编码的图片
					mime, data, err := parseDataURL(urlStr)
					if err != nil {
						return nil, fmt.Errorf("parse data url: %w", err)
					}
					content.ImageMIME = mime
					content.ImageData = data
					content.ImageBase64 = true
				} else {
					// 远程 URL
					if !p.AllowRemoteURLs {
						return nil, fmt.Errorf("remote urls not allowed")
					}
					content.ImageBase64 = false
					content.ImageMIME = p.detectMIMEFromURL(urlStr)
				}

				result = append(result, content)
			}

		case "audio_url":
			if audioURL, ok := partMap["audio_url"].(map[string]interface{}); ok {
				urlStr, _ := audioURL["url"].(string)

				content := MultimodalContent{
					Type:      "audio_url",
					AudioURL:  urlStr,
				}

				if strings.HasPrefix(urlStr, "data:") {
					mime, data, err := parseDataURL(urlStr)
					if err != nil {
						return nil, fmt.Errorf("parse data url: %w", err)
					}
					content.AudioMIME = mime
					content.AudioData = data
					content.AudioBase64 = true
				} else {
					if !p.AllowRemoteURLs {
						return nil, fmt.Errorf("remote urls not allowed")
					}
					content.AudioBase64 = false
					content.AudioMIME = p.detectMIMEFromURL(urlStr)
				}

				result = append(result, content)
			}

		default:
			// 未知类型，记录但继续处理
			continue
		}
	}

	return result, nil
}

// ToOpenAIFormat 转换为 OpenAI 格式
func (p *MultimodalParser) ToOpenAIFormat(msg *MultimodalMessage) (interface{}, error) {
	if len(msg.Content) == 0 {
		return "", nil
	}

	// 如果只有文本内容，直接返回字符串
	if len(msg.Content) == 1 && msg.Content[0].Type == "text" {
		return msg.Content[0].Text, nil
	}

	// 多模态内容，返回数组
	var parts []ContentPart
	for _, content := range msg.Content {
		switch content.Type {
		case "text":
			parts = append(parts, ContentPart{
				Type: "text",
				Text: content.Text,
			})

		case "image_url":
			imageURL := &struct {
				URL    string `json:"url"`
				Detail string `json:"detail,omitempty"`
			}{
				URL: content.ImageURL,
			}

			if content.ImageBase64 && content.ImageData != nil {
				// 确保 data URL 格式正确
				if !strings.HasPrefix(content.ImageURL, "data:") {
					imageURL.URL = fmt.Sprintf("data:%s;base64,%s",
						content.ImageMIME,
						base64.StdEncoding.EncodeToString(content.ImageData))
				}
			}

			parts = append(parts, ContentPart{
				Type:     "image_url",
				ImageURL: imageURL,
			})
		}
	}

	return parts, nil
}

// ExtractText 提取所有文本内容
func (p *MultimodalParser) ExtractText(msg *MultimodalMessage) string {
	var result strings.Builder
	for _, content := range msg.Content {
		if content.Type == "text" {
			result.WriteString(content.Text)
		}
	}
	return result.String()
}

// CountTokens 估算多模态消息的 token 数量
// 注意：这是粗略估算，实际 token 计数可能因模型而异
func (p *MultimodalParser) CountTokens(msg *MultimodalMessage) int {
	total := 0

	for _, content := range msg.Content {
		switch content.Type {
		case "text":
			// 粗略估算：英文约 4 字符/token，中文约 2 字符/token
			textLen := len(content.Text)
			nonASCII := 0
			for _, r := range content.Text {
				if r > 127 {
					nonASCII++
				}
			}
			asciiLen := textLen - nonASCII
			total += asciiLen/4 + nonASCII/2 + 1

		case "image_url":
			// 图片根据分辨率计算 token
			// low detail: 85 tokens
			// high detail: 根据实际像素计算
			// 这里使用默认值
			total += 255 // 保守估计：high detail
		}
	}

	return total
}

// Validate 验证多模态内容
func (p *MultimodalParser) Validate(msg *MultimodalMessage) error {
	for _, content := range msg.Content {
		switch content.Type {
		case "image_url":
			if content.ImageBase64 && int64(len(content.ImageData)) > p.MaxImageSize {
				return fmt.Errorf("image size %d exceeds limit %d", len(content.ImageData), p.MaxImageSize)
			}
			if content.ImageMIME != "" && !p.isAllowedMIME(content.ImageMIME) {
				return fmt.Errorf("image mime type %s not allowed", content.ImageMIME)
			}

		case "audio_url":
			if content.AudioBase64 && int64(len(content.AudioData)) > p.MaxAudioSize {
				return fmt.Errorf("audio size %d exceeds limit %d", len(content.AudioData), p.MaxAudioSize)
			}
			if content.AudioMIME != "" && !p.isAllowedMIME(content.AudioMIME) {
				return fmt.Errorf("audio mime type %s not allowed", content.AudioMIME)
			}
		}
	}
	return nil
}

// MergeTextContents 合并连续的文本内容
func (p *MultimodalParser) MergeTextContents(msg *MultimodalMessage) {
	if len(msg.Content) < 2 {
		return
	}

	merged := make([]MultimodalContent, 0, len(msg.Content))
	i := 0

	for i < len(msg.Content) {
		if msg.Content[i].Type != "text" {
			merged = append(merged, msg.Content[i])
			i++
			continue
		}

		// 收集连续的文本
		var textBuilder strings.Builder
		for i < len(msg.Content) && msg.Content[i].Type == "text" {
			textBuilder.WriteString(msg.Content[i].Text)
			i++
		}

		merged = append(merged, MultimodalContent{
			Type: "text",
			Text: textBuilder.String(),
		})
	}

	msg.Content = merged
}

// parseDataURL 解析 data URL
func parseDataURL(dataURL string) (mime string, data []byte, err error) {
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("invalid data url format")
	}

	// 解析 MIME 类型和编码
	mimePart := strings.TrimPrefix(parts[0], "data:")
	mimeParts := strings.SplitN(mimePart, ";", 2)
	mime = mimeParts[0]
	if mime == "" {
		mime = "text/plain"
	}

	// 解码数据
	var encoding string
	if len(mimeParts) > 1 {
		encoding = mimeParts[1]
	}

	if encoding == "base64" {
		data, err = base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return "", nil, fmt.Errorf("base64 decode: %w", err)
		}
	} else {
		data = []byte(parts[1])
	}

	return mime, data, nil
}

// detectMIMEFromURL 从 URL 检测 MIME 类型
func (p *MultimodalParser) detectMIMEFromURL(urlStr string) string {
	// 尝试从 URL 路径检测
	u, err := url.Parse(urlStr)
	if err == nil {
		ext := strings.ToLower(filepath.Ext(u.Path))
		switch ext {
		case ".jpg", ".jpeg":
			return "image/jpeg"
		case ".png":
			return "image/png"
		case ".gif":
			return "image/gif"
		case ".webp":
			return "image/webp"
		case ".mp3":
			return "audio/mpeg"
		case ".wav":
			return "audio/wav"
		case ".ogg":
			return "audio/ogg"
		case ".mp4":
			return "audio/mp4"
		}
	}

	// 尝试使用 mime 包
	if ext := filepath.Ext(urlStr); ext != "" {
		if mimeType := mime.TypeByExtension(ext); mimeType != "" {
			return mimeType
		}
	}

	return "application/octet-stream"
}

// isAllowedMIME 检查 MIME 类型是否允许
func (p *MultimodalParser) isAllowedMIME(mime string) bool {
	if len(p.AllowedMIMEs) == 0 {
		return true
	}
	for _, allowed := range p.AllowedMIMEs {
		if strings.HasPrefix(mime, allowed) {
			return true
		}
	}
	return false
}

// MultimodalStreamChunk 多模态流式分块
type MultimodalStreamChunk struct {
	Delta  *MultimodalMessage `json:"delta,omitempty"`
	Finish string             `json:"finish_reason,omitempty"`
}

// ParseMultimodalStreamChunk 解析多模态流式分块
func (p *MultimodalParser) ParseMultimodalStreamChunk(data []byte) (*MultimodalStreamChunk, error) {
	var chunk struct {
		Choices []struct {
			Delta  *OpenAIMultimodalMessage `json:"delta"`
			Finish string         `json:"finish_reason"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(data, &chunk); err != nil {
		return nil, fmt.Errorf("unmarshal stream chunk: %w", err)
	}

	if len(chunk.Choices) == 0 {
		return &MultimodalStreamChunk{}, nil
	}

	choice := chunk.Choices[0]
	result := &MultimodalStreamChunk{
		Finish: choice.Finish,
	}

	if choice.Delta != nil {
		// 转换为多模态消息格式
		deltaData, _ := json.Marshal(choice.Delta)
		msg, err := p.ParseOpenAIMultimodalMessage(deltaData)
		if err == nil {
			result.Delta = msg
		}
	}

	return result, nil
}

// Copy 创建多模态消息的深拷贝
func (msg *MultimodalMessage) Copy() *MultimodalMessage {
	if msg == nil {
		return nil
	}

	copied := &MultimodalMessage{
		Role:   msg.Role,
		Name:   msg.Name,
		ToolID: msg.ToolID,
	}

	copied.Content = make([]MultimodalContent, len(msg.Content))
	for i, content := range msg.Content {
		copied.Content[i] = MultimodalContent{
			Type:         content.Type,
			Text:         content.Text,
			ImageURL:     content.ImageURL,
			ImageMIME:    content.ImageMIME,
			ImageBase64:  content.ImageBase64,
			AudioURL:     content.AudioURL,
			AudioMIME:    content.AudioMIME,
			AudioBase64:  content.AudioBase64,
		}

		if content.ImageData != nil {
			copied.Content[i].ImageData = make([]byte, len(content.ImageData))
			copy(copied.Content[i].ImageData, content.ImageData)
		}

		if content.AudioData != nil {
			copied.Content[i].AudioData = make([]byte, len(content.AudioData))
			copy(copied.Content[i].AudioData, content.AudioData)
		}
	}

	return copied
}

// MultimodalConverter 多模态转换器
type MultimodalConverter struct {
	parser  *MultimodalParser
	config  *ConverterConfig
	fallback Converter
}

// NewMultimodalConverter 创建多模态转换器
func NewMultimodalConverter(config *ConverterConfig) *MultimodalConverter {
	return &MultimodalConverter{
		parser:  NewMultimodalParser(),
		config:  config,
		fallback: NewDeepSeekConverter(config),
	}
}

// Convert 转换多模态响应
func (c *MultimodalConverter) Convert(data []byte) ([]byte, error) {
	// 尝试检测是否为多模态响应
	if c.isMultimodalResponse(data) {
		return c.convertMultimodal(data)
	}

	// 回退到普通转换器
	return c.fallback.Convert(data)
}

// ConvertStream 转换多模态流式响应
func (c *MultimodalConverter) ConvertStream(r io.Reader, w io.Writer) error {
	// 读取第一个块以检测是否为多模态
	buf := make([]byte, 1024)
	n, err := io.ReadFull(r, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return err
	}

	if c.isMultimodalResponse(buf[:n]) {
		// 创建多 reader，一个用于检测，一个用于实际读取
		multiReader := io.MultiReader(bytes.NewReader(buf[:n]), r)
		return c.convertMultimodalStream(multiReader, w)
	}

	// 回退到普通流式转换
	multiReader := io.MultiReader(bytes.NewReader(buf[:n]), r)
	return c.fallback.ConvertStream(multiReader, w)
}

// isMultimodalResponse 检测是否为多模态响应
func (c *MultimodalConverter) isMultimodalResponse(data []byte) bool {
	var resp map[string]interface{}
	if err := json.Unmarshal(data, &resp); err != nil {
		return false
	}

	// 检查消息内容是否包含图片 URL
	if choices, ok := resp["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].([]interface{}); ok {
					for _, item := range content {
						if itemMap, ok := item.(map[string]interface{}); ok {
							if itemType, ok := itemMap["type"].(string); ok {
								if itemType == "image_url" || itemType == "audio_url" {
									return true
								}
							}
						}
					}
				}
			}
		}
	}

	return false
}

// convertMultimodal 转换多模态响应
func (c *MultimodalConverter) convertMultimodal(data []byte) ([]byte, error) {
	var resp struct {
		Choices []struct {
			Message OpenAIMultimodalMessage `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal multimodal response: %w", err)
	}

	if len(resp.Choices) == 0 {
		return []byte{}, nil
	}

	// 解析多模态消息
	msg, err := c.parser.ParseOpenAIMultimodalMessageMessage(&resp.Choices[0].Message)
	if err != nil {
		return nil, err
	}

	// 提取文本用于输出
	text := c.parser.ExtractText(msg)
	template := c.config.Templates.Message
	output := fmt.Sprintf(template, text)

	if c.config.IncludeUsage {
		usage := fmt.Sprintf("\n\nusage: prompt=%d completion=%d total=%d",
			resp.Usage.PromptTokens,
			resp.Usage.CompletionTokens,
			resp.Usage.TotalTokens)
		output += usage
	}

	return []byte(output), nil
}

// convertMultimodalStream 转换多模态流式响应
func (c *MultimodalConverter) convertMultimodalStream(r io.Reader, w io.Writer) error {
	decoder := json.NewDecoder(r)

	for {
		var chunk map[string]interface{}
		if err := decoder.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("decode stream chunk: %w", err)
		}

		// 提取 delta.content 中的文本
		if choices, ok := chunk["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if delta, ok := choice["delta"].(map[string]interface{}); ok {
					if content, ok := delta["content"].([]interface{}); ok {
						// 提取文本部分
						for _, item := range content {
							if itemMap, ok := item.(map[string]interface{}); ok {
								if itemType, ok := itemMap["type"].(string); ok && itemType == "text" {
									if text, ok := itemMap["text"].(string); ok {
										fmt.Fprintf(w, c.config.Templates.StreamChunk, text)
									}
								}
							}
						}
					} else if contentStr, ok := delta["content"].(string); ok {
						// 简单字符串内容
						fmt.Fprintf(w, c.config.Templates.StreamChunk, contentStr)
					}
				}
			}
		}
	}

	return nil
}

// ParseOpenAIMultimodalMessageMessage 辅助方法：从 OpenAIMultimodalMessage 解析多模态消息
func (p *MultimodalParser) ParseOpenAIMultimodalMessageMessage(msg *OpenAIMultimodalMessage) (*MultimodalMessage, error) {
	result := &MultimodalMessage{
		Role:   msg.Role,
		Name:   msg.Name,
		ToolID: msg.ToolCallID,
	}

	switch v := msg.Content.(type) {
	case string:
		result.Content = []MultimodalContent{{Type: "text", Text: v}}
	case []interface{}:
		parts, err := p.parseContentParts(v)
		if err != nil {
			return nil, err
		}
		result.Content = parts
	}

	return result, nil
}
