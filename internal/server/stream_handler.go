package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/openclaw/api2openclaw/internal/converter"
)

// StreamHandler SSE 流式响应处理器
type StreamHandler struct {
	normalizer converter.Normalizer
}

// NewStreamHandler 创建流式处理器
func NewStreamHandler(normalizer converter.Normalizer) *StreamHandler {
	return &StreamHandler{
		normalizer: normalizer,
	}
}

// HandleStream 处理流式响应透传
// 从上游读取 → 归一化 → 立即写出，不等待完整响应
func (h *StreamHandler) HandleStream(
	ctx context.Context,
	w http.ResponseWriter,
	upstreamStream <-chan []byte,
	errChan <-chan error,
	modelFamily converter.ModelFamily,
) error {
	// 设置 SSE 响应头 - 禁用缓冲，确保实时推送
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // 关键：禁止 nginx 缓冲

	// 获取 Flusher（用于立即推送）
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported: http.Flusher not available")
	}

	// 记录首 token 状态
	firstTokenReceived := false

	// 逐 chunk 处理
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case rawChunk, ok := <-upstreamStream:
			if !ok {
				// 流结束，发送 [DONE]
				fmt.Fprintf(w, "data: [DONE]\n\n")
				flusher.Flush()
				return nil
			}

			// 归一化 chunk
			normalized, err := h.normalizer.NormalizeStreamChunk(rawChunk, modelFamily)
			if err != nil {
				// 记录错误但继续处理
				h.logChunkError(err, rawChunk)
				continue
			}

			// 序列化为 JSON
			jsonData, err := json.Marshal(normalized)
			if err != nil {
				h.logChunkError(err, rawChunk)
				continue
			}

			// 立即推送，不等待
			fmt.Fprintf(w, "data: %s\n\n", jsonData)
			flusher.Flush()

			// 记录首 token 延迟
			if !firstTokenReceived && len(normalized.Choices) > 0 {
				if normalized.Choices[0].Delta.Content != "" {
					firstTokenReceived = true
				}
			}

		case err := <-errChan:
			if err != nil {
				return fmt.Errorf("upstream stream error: %w", err)
			}
		}
	}
}

// HandleStreamReader 从 Reader 读取流式响应并透传
// 用于上游直接提供 io.Reader 的场景
func (h *StreamHandler) HandleStreamReader(
	ctx context.Context,
	w http.ResponseWriter,
	r io.Reader,
	modelFamily converter.ModelFamily,
) error {
	// 设置 SSE 响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported: http.Flusher not available")
	}

	// 创建 SSE 扫描器
	scanner := newSSEReader(r)
	firstTokenReceived := false
	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		default:
			rawChunk, err := scanner.NextChunk()
			if err == io.EOF {
				// 流结束
				fmt.Fprintf(w, "data: [DONE]\n\n")
				flusher.Flush()
				return nil
			}
			if err != nil {
				return fmt.Errorf("read stream chunk: %w", err)
			}

			// 归一化 chunk
			normalized, err := h.normalizer.NormalizeStreamChunk(rawChunk, modelFamily)
			if err != nil {
				h.logChunkError(err, rawChunk)
				continue
			}

			// 序列化并推送
			jsonData, err := json.Marshal(normalized)
			if err != nil {
				h.logChunkError(err, rawChunk)
				continue
			}

			fmt.Fprintf(w, "data: %s\n\n", jsonData)
			flusher.Flush()

			// 记录首 token 延迟
			if !firstTokenReceived && len(normalized.Choices) > 0 {
				if normalized.Choices[0].Delta.Content != "" {
					firstTokenReceived = true
					firstTokenLatency := time.Since(startTime)
					// 可选：记录到指标
					_ = firstTokenLatency
				}
			}
		}
	}
}

// StreamMetrics 流式响应指标
type StreamMetrics struct {
	FirstTokenLatency time.Duration
	ChunkCount        int
	TotalBytes        int
}

// HandleStreamWithMetrics 带指标收集的流式处理
func (h *StreamHandler) HandleStreamWithMetrics(
	ctx context.Context,
	w http.ResponseWriter,
	upstreamStream <-chan []byte,
	errChan <-chan error,
	modelFamily converter.ModelFamily,
) (*StreamMetrics, error) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	metrics := &StreamMetrics{
		ChunkCount: 0,
		TotalBytes: 0,
	}
	startTime := time.Now()
	firstTokenReceived := false

	for {
		select {
		case <-ctx.Done():
			return metrics, ctx.Err()

		case rawChunk, ok := <-upstreamStream:
			if !ok {
				fmt.Fprintf(w, "data: [DONE]\n\n")
				flusher.Flush()
				return metrics, nil
			}

			metrics.ChunkCount++
			metrics.TotalBytes += len(rawChunk)

			normalized, err := h.normalizer.NormalizeStreamChunk(rawChunk, modelFamily)
			if err != nil {
				h.logChunkError(err, rawChunk)
				continue
			}

			jsonData, err := json.Marshal(normalized)
			if err != nil {
				h.logChunkError(err, rawChunk)
				continue
			}

			fmt.Fprintf(w, "data: %s\n\n", jsonData)
			flusher.Flush()

			if !firstTokenReceived && len(normalized.Choices) > 0 {
				if normalized.Choices[0].Delta.Content != "" {
					firstTokenReceived = true
					metrics.FirstTokenLatency = time.Since(startTime)
				}
			}

		case err := <-errChan:
			if err != nil {
				return metrics, fmt.Errorf("upstream error: %w", err)
			}
		}
	}
}

// logChunkError 记录 chunk 处理错误
func (h *StreamHandler) logChunkError(err error, rawChunk []byte) {
	// 使用结构化日志记录
	// TODO: 集成结构化日志系统
	fmt.Printf("[StreamHandler] Chunk error: %v, data: %s\n", err, string(rawChunk))
}

// === SSE Reader ===

// sseReader SSE 流读取器
type sseReader struct {
	reader *bufioReader
	current []byte
	err    error
}

// newSSEReader 创建 SSE 读取器
func newSSEReader(r io.Reader) *sseReader {
	return &sseReader{
		reader: newBufioReader(r),
	}
}

// NextChunk 读取下一个 SSE 数据块
func (r *sseReader) NextChunk() ([]byte, error) {
	for {
		line, err := r.reader.readLine()
		if err != nil {
			return nil, err
		}

		line = trimSpace(line)
		if len(line) == 0 || hasPrefix(line, ":") {
			// 跳过空行和注释
			continue
		}

		if hasPrefix(line, "data:") {
			data := trimPrefix(line, "data:")
			data = trimSpace(data)
			return []byte(data), nil
		}
	}
}

// === 简单的字符串操作（避免导入 strings） ===

type bufioReader struct {
	r   io.Reader
	buf []byte
	pos int
}

func newBufioReader(r io.Reader) *bufioReader {
	return &bufioReader{r: r, buf: make([]byte, 4096)}
}

func (r *bufioReader) readLine() ([]byte, error) {
	line := make([]byte, 0, 256)
	for {
		if r.pos >= len(r.buf) {
			_, err := r.r.Read(r.buf)
			if err != nil {
				if err == io.EOF && len(line) > 0 {
					return line, nil
				}
				return line, err
			}
			r.pos = 0
		}

		b := r.buf[r.pos]
		r.pos++
		line = append(line, b)

		if b == '\n' {
			return line, nil
		}
	}
}

func trimSpace(b []byte) []byte {
	start := 0
	end := len(b)
	for start < end && (b[start] == ' ' || b[start] == '\t' || b[start] == '\r' || b[start] == '\n') {
		start++
	}
	for end > start && (b[end-1] == ' ' || b[end-1] == '\t' || b[end-1] == '\r' || b[end-1] == '\n') {
		end--
	}
	return b[start:end]
}

func hasPrefix(b []byte, prefix string) bool {
	if len(b) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if b[i] != prefix[i] {
			return false
		}
	}
	return true
}

func trimPrefix(b []byte, prefix string) []byte {
	if !hasPrefix(b, prefix) {
		return b
	}
	return b[len(prefix):]
}
