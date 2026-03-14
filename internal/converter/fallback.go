package converter

import (
	"encoding/json"
	"fmt"
	"io"
)

// ConversionError 转换错误
type ConversionError struct {
	OriginalError error
	InputData     []byte
	ConverterType string
	RawResponse   []byte
}

// Error 实现 error 接口
func (e *ConversionError) Error() string {
	return fmt.Sprintf("conversion failed in %s: %v", e.ConverterType, e.OriginalError)
}

// Unwrap 实现 errors.Unwrap
func (e *ConversionError) Unwrap() error {
	return e.OriginalError
}

// FallbackConverter 降级转换器
type FallbackConverter struct {
	primary    Converter
	fallback   Converter
	escapeHatch bool // 是否启用逃生舱（返回原始响应）
}

// NewFallbackConverter 创建降级转换器
func NewFallbackConverter(primary, fallback Converter, escapeHatch bool) *FallbackConverter {
	return &FallbackConverter{
		primary:     primary,
		fallback:    fallback,
		escapeHatch: escapeHatch,
	}
}

// Convert 尝试主转换器，失败后使用备用转换器
func (c *FallbackConverter) Convert(data []byte) ([]byte, error) {
	// 尝试主转换器
	result, err := c.primary.Convert(data)
	if err != nil {
		// 主转换器失败，尝试备用
		if c.fallback != nil {
			fallbackResult, fallbackErr := c.fallback.Convert(data)
			if fallbackErr == nil {
				return fallbackResult, nil
			}
		}

		// 逃生舱：返回原始响应
		if c.escapeHatch {
			return c.wrapWithFallbackHeader(data, err)
		}

		return nil, &ConversionError{
			OriginalError: err,
			InputData:     data,
			ConverterType: "primary",
			RawResponse:   data,
		}
	}

	return result, nil
}

// ConvertStream 流式转换（带降级）
func (c *FallbackConverter) ConvertStream(r io.Reader, w io.Writer) error {
	// 尝试主转换器
	err := c.primary.ConvertStream(r, w)
	if err != nil {
		// 主转换器失败，尝试备用
		if c.fallback != nil {
			// 需要重置读取器（如果支持）
			if seeker, ok := r.(io.Seeker); ok {
				seeker.Seek(0, io.SeekStart)
				fallbackErr := c.fallback.ConvertStream(r, w)
				if fallbackErr == nil {
					return nil
				}
			}
		}

		// 逃生舱：写入原始数据
		if c.escapeHatch {
			_, _ = w.Write([]byte(fmt.Sprintf("/* Conversion Error: %v */\n", err)))
			_, _ = io.Copy(w, r)
			return nil
		}

		return &ConversionError{
			OriginalError: err,
			ConverterType: "primary_stream",
		}
	}

	return nil
}

// wrapWithFallbackHeader 包装原始响应带错误头
func (c *FallbackConverter) wrapWithFallbackHeader(data []byte, err error) ([]byte, error) {
	wrapper := map[string]interface{}{
		"content": string(data),
		"conversion_error": err.Error(),
		"fallback_mode": true,
		"raw_response": string(data),
	}

	return json.Marshal(wrapper)
}

// ResilientConverter 韧性转换器（带重试）
type ResilientConverter struct {
	converter Converter
	maxRetries int
}

// NewResilientConverter 创建韧性转换器
func NewResilientConverter(converter Converter, maxRetries int) *ResilientConverter {
	if maxRetries <= 0 {
		maxRetries = 3
	}
	return &ResilientConverter{
		converter:  converter,
		maxRetries: maxRetries,
	}
}

// Convert 带重试的转换
func (c *ResilientConverter) Convert(data []byte) ([]byte, error) {
	var lastErr error

	for i := 0; i <= c.maxRetries; i++ {
		result, err := c.converter.Convert(data)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}

	return nil, &ConversionError{
		OriginalError: fmt.Errorf("after %d retries: %w", c.maxRetries, lastErr),
		ConverterType: "resilient",
		InputData:     data,
	}
}

// ConvertStream 带重试的流式转换
func (c *ResilientConverter) ConvertStream(r io.Reader, w io.Writer) error {
	var lastErr error

	for i := 0; i <= c.maxRetries; i++ {
		err := c.converter.ConvertStream(r, w)
		if err == nil {
			return nil
		}

		// 流式转换重试需要重置读取器
		if seeker, ok := r.(io.Seeker); ok {
			seeker.Seek(0, io.SeekStart)
		} else {
			// 不支持重置，无法重试
			return err
		}

		lastErr = err
	}

	return &ConversionError{
		OriginalError: fmt.Errorf("after %d retries: %w", c.maxRetries, lastErr),
		ConverterType: "resilient_stream",
	}
}

// ConversionResult 转换结果
type ConversionResult struct {
	Success      bool   `json:"success"`
	Data         []byte `json:"data,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	FallbackMode bool   `json:"fallback_mode,omitempty"`
	RawData      []byte `json:"raw_data,omitempty"`
}

// SafeConverter 安全转换器（总是返回结果）
type SafeConverter struct {
	converter   Converter
	escapeHatch bool
}

// NewSafeConverter 创建安全转换器
func NewSafeConverter(converter Converter, escapeHatch bool) *SafeConverter {
	return &SafeConverter{
		converter:   converter,
		escapeHatch: escapeHatch,
	}
}

// Convert 安全转换（总是返回结果）
func (c *SafeConverter) Convert(data []byte) *ConversionResult {
	result, err := c.converter.Convert(data)
	if err == nil {
		return &ConversionResult{
			Success: true,
			Data:    result,
		}
	}

	// 转换失败
	if c.escapeHatch {
		return &ConversionResult{
			Success:      false,
			ErrorMessage: err.Error(),
			FallbackMode: true,
			RawData:      data,
		}
	}

	return &ConversionResult{
		Success:      false,
		ErrorMessage: err.Error(),
	}
}

// GetResponseHeaders 获取响应头（用于转换失败时）
func GetConversionErrorHeaders(err error) map[string]string {
	headers := make(map[string]string)

	if convErr, ok := err.(*ConversionError); ok {
		headers["X-Conversion-Error"] = "true"
		headers["X-Converter-Type"] = convErr.ConverterType
		headers["X-Error-Message"] = convErr.OriginalError.Error()
	} else {
		headers["X-Conversion-Error"] = "true"
		headers["X-Error-Message"] = err.Error()
	}

	return headers
}

// IsConversionError 检查是否是转换错误
func IsConversionError(err error) bool {
	_, ok := err.(*ConversionError)
	return ok
}

// WrapConversionError 包装错误为转换错误
func WrapConversionError(err error, converterType string, data []byte) error {
	if err == nil {
		return nil
	}

	if IsConversionError(err) {
		return err
	}

	return &ConversionError{
		OriginalError: err,
		ConverterType: converterType,
		InputData:     data,
	}
}
