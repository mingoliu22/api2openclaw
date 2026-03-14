package router

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/openclaw/api2openclaw/internal/models"
)

// Forwarder 后端请求转发器
type Forwarder struct {
	httpClient *http.Client
	converter  Converter
	metrics    MetricsReporter
}

// Converter 格式转换器接口
type Converter interface {
	Convert(data []byte) ([]byte, error)
	ConvertStream(r io.Reader, w io.Writer) error
}

// MetricsReporter 指标报告接口
type MetricsReporter interface {
	RecordModelRequest(model, backend string, statusCode int, duration time.Duration)
	RecordModelError(model, backend, errorType string)
	RecordTokens(apiKeyID, model string, promptTokens, completionTokens int)
}

// NewForwarder 创建转发器
func NewForwarder(converter Converter, metrics MetricsReporter) *Forwarder {
	return &Forwarder{
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		converter: converter,
		metrics:   metrics,
	}
}

// ForwardRequest 转发请求到后端
func (f *Forwarder) ForwardRequest(ctx context.Context, backend *models.Backend, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	startTime := time.Now()

	// 构建后端请求
	backendReq := f.buildBackendRequest(req)

	// 序列化请求体
	body, err := json.Marshal(backendReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 创建 HTTP 请求
	url := backend.BaseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	if backend.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+backend.APIKey)
	}
	for k, v := range backend.Headers {
		httpReq.Header.Set(k, v)
	}

	// 发送请求
	log.Printf("[Forwarder] Sending request to %s (model: %s)", backend.ID, req.Model)
	resp, err := f.httpClient.Do(httpReq)
	if err != nil {
		if f.metrics != nil {
			f.metrics.RecordModelError(req.Model, backend.ID, "connection_error")
		}
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		if f.metrics != nil {
			f.metrics.RecordModelError(req.Model, backend.ID, "read_error")
		}
		return nil, fmt.Errorf("read response: %w", err)
	}

	duration := time.Since(startTime)

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		if f.metrics != nil {
			f.metrics.RecordModelRequest(req.Model, backend.ID, resp.StatusCode, duration)
			f.metrics.RecordModelError(req.Model, backend.ID, "http_error")
		}
		return nil, fmt.Errorf("backend returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var response ChatCompletionResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// 记录指标
	if f.metrics != nil {
		f.metrics.RecordModelRequest(req.Model, backend.ID, resp.StatusCode, duration)
		if response.Usage != nil {
			// TODO: 从上下文获取 apiKeyID
			f.metrics.RecordTokens("", req.Model, response.Usage.PromptTokens, response.Usage.CompletionTokens)
		}
	}

	log.Printf("[Forwarder] Request completed in %v (tokens: %d)", duration, response.Usage.TotalTokens)

	return &response, nil
}

// ForwardStreamRequest 转发流式请求
func (f *Forwarder) ForwardStreamRequest(ctx context.Context, backend *models.Backend, req *ChatCompletionRequest, apiKeyID string) (<-chan StreamChunk, <-chan error) {
	chunkChan := make(chan StreamChunk, 16)
	errChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errChan)

		startTime := time.Now()

		// 构建后端请求
		backendReq := f.buildBackendRequest(req)
		backendReq["stream"] = true

		// 序列化请求体
		body, err := json.Marshal(backendReq)
		if err != nil {
			errChan <- fmt.Errorf("marshal request: %w", err)
			return
		}

		// 创建 HTTP 请求
		url := backend.BaseURL + "/chat/completions"
		httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		if err != nil {
			errChan <- fmt.Errorf("create request: %w", err)
			return
		}

		// 设置请求头
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "text/event-stream")
		if backend.APIKey != "" {
			httpReq.Header.Set("Authorization", "Bearer "+backend.APIKey)
		}
		for k, v := range backend.Headers {
			httpReq.Header.Set(k, v)
		}

		// 发送请求
		log.Printf("[Forwarder] Sending stream request to %s (model: %s)", backend.ID, req.Model)
		resp, err := f.httpClient.Do(httpReq)
		if err != nil {
			if f.metrics != nil {
				f.metrics.RecordModelError(req.Model, backend.ID, "connection_error")
			}
			errChan <- fmt.Errorf("send request: %w", err)
			return
		}
		defer resp.Body.Close()

		// 检查响应状态
		if resp.StatusCode != http.StatusOK {
			if f.metrics != nil {
				f.metrics.RecordModelRequest(req.Model, backend.ID, resp.StatusCode, time.Since(startTime))
				f.metrics.RecordModelError(req.Model, backend.ID, "http_error")
			}
			bodyBytes, _ := io.ReadAll(resp.Body)
			errChan <- fmt.Errorf("backend returned status %d: %s", resp.StatusCode, string(bodyBytes))
			return
		}

		// 读取 SSE 流
		scanner := newSSEScanner(resp.Body)
		chunkIndex := 0
		totalTokens := 0

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
			}

			data := scanner.Bytes()
			if len(data) == 0 {
				continue
			}

			// 解析 SSE 数据
			var streamResp struct {
				ID      string `json:"id"`
				Object  string `json:"object"`
				Created int64  `json:"created"`
				Model   string `json:"model"`
				Choices []struct {
					Index        int                    `json:"index"`
					Delta        map[string]interface{} `json:"delta"`
					FinishReason string                 `json:"finish_reason"`
				} `json:"choices"`
			}

			if err := json.Unmarshal(data, &streamResp); err != nil {
				log.Printf("[Forwarder] Failed to parse stream chunk: %v, data: %s", err, string(data))
				continue
			}

			// 转换 chunk
			chunk := StreamChunk{
				ID:      streamResp.ID,
				Object:  streamResp.Object,
				Created: streamResp.Created,
				Model:   streamResp.Model,
			}

			if len(streamResp.Choices) > 0 {
				choice := streamResp.Choices[0]
				chunk.Choices = []Choice{
					{
						Index:        choice.Index,
						Delta:        &Message{Role: "assistant", Content: ""},
						FinishReason: choice.FinishReason,
					},
				}

				// 提取 content
				if content, ok := choice.Delta["content"]; ok {
					if contentStr, ok := content.(string); ok {
						chunk.Choices[0].Delta.Content = contentStr
					}
				}
			}

			chunkIndex++
			chunkChan <- chunk
		}

		if err := scanner.Err(); err != nil {
			if f.metrics != nil {
				f.metrics.RecordModelError(req.Model, backend.ID, "stream_read_error")
			}
			errChan <- fmt.Errorf("read stream: %w", err)
			return
		}

		// 记录指标
		duration := time.Since(startTime)
		if f.metrics != nil {
			f.metrics.RecordModelRequest(req.Model, backend.ID, 200, duration)
			if totalTokens > 0 {
				f.metrics.RecordTokens(apiKeyID, req.Model, 0, totalTokens)
			}
		}

		log.Printf("[Forwarder] Stream completed in %v (chunks: %d)", duration, chunkIndex)
	}()

	return chunkChan, errChan
}

// buildBackendRequest 构建后端请求
func (f *Forwarder) buildBackendRequest(req *ChatCompletionRequest) map[string]interface{} {
	backendReq := make(map[string]interface{})
	backendReq["model"] = req.Model
	backendReq["messages"] = req.Messages

	if req.Stream {
		backendReq["stream"] = true
	}

	if req.Temperature > 0 {
		backendReq["temperature"] = req.Temperature
	}

	if req.MaxTokens > 0 {
		backendReq["max_tokens"] = req.MaxTokens
	}

	if req.TopP > 0 {
		backendReq["top_p"] = req.TopP
	}

	if len(req.Stop) > 0 {
		backendReq["stop"] = req.Stop
	}

	return backendReq
}

// ChatCompletionRequest 聊天完成请求
type ChatCompletionRequest struct {
	Model       string `json:"model" binding:"required"`
	Messages    []Message `json:"messages" binding:"required"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
	Stop        []string `json:"stop,omitempty"`
	Stream      bool    `json:"stream,omitempty"`
}

// Message 消息
type Message struct {
	Role    string `json:"role" binding:"required"`
	Content string `json:"content" binding:"required"`
}

// ChatCompletionResponse 聊天完成响应
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

// Choice 选择
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message,omitempty"`
	Delta        *Message `json:"delta,omitempty"`
	FinishReason string  `json:"finish_reason,omitempty"`
}

// Usage 使用量
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamChunk 流式分块
type StreamChunk struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
}

// sseScanner SSE 流扫描器
type sseScanner struct {
	reader  *bufio.Reader
	current string
	err     error
}

// newSSEScanner 创建 SSE 扫描器
func newSSEScanner(r io.Reader) *sseScanner {
	return &sseScanner{reader: bufio.NewReader(r)}
}

// Scan 扫描下一个 SSE 数据块
func (s *sseScanner) Scan() bool {
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			s.err = err
			return false
		}

		// 跳过注释行和空行
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// 检查是否是 data 行
		if strings.HasPrefix(line, "data:") {
			s.current = strings.TrimPrefix(line, "data:")
			s.current = strings.TrimSpace(s.current)
			return true
		}
	}
}

// Bytes 获取当前数据
func (s *sseScanner) Bytes() []byte {
	return []byte(s.current)
}

// Err 获取错误
func (s *sseScanner) Err() error {
	return s.err
}
