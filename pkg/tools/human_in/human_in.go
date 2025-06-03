package humanintool

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	json "github.com/json-iterator/go"

	"github.com/m4n5ter/another-me/pkg/common"
	"github.com/m4n5ter/another-me/pkg/i18n"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// 人类介入工具，用来请求人类介入，人类介入的内容会作为工具调用的结果
// 这个工具被调用时，应该需要和客户端进行通信，客户端可以设置超时时间，在此期间该工具会阻塞住，直到得到响应（超时或成功介入）

// HumanInTool 人类介入工具结构体
type HumanInTool struct {
	logger     *slog.Logger
	i18nMgr    *i18n.Manager
	commChan   chan HumanResponse // 与客户端通信的通道
	pendingReq chan HumanRequest  // 待处理的人类介入请求
}

// HumanInArgs 人类介入工具的输入参数
type HumanInArgs struct {
	Message     string   `json:"message"`                // 向人类显示的消息
	Timeout     int      `json:"timeout,omitempty"`      // 超时时间（秒），默认300秒
	RequestType string   `json:"request_type,omitempty"` // 请求类型：input, confirmation, decision
	Options     []string `json:"options,omitempty"`      // 选择项（用于decision类型）
}

// HumanInResult 人类介入工具的输出结果
type HumanInResult struct {
	Status        string `json:"status"`         // 状态：success, timeout, cancelled
	HumanResponse string `json:"human_response"` // 人类的回应
	ResponseTime  int    `json:"response_time"`  // 响应时间（秒）
	RequestType   string `json:"request_type"`   // 请求类型
	TimedOut      bool   `json:"timed_out"`      // 是否超时
}

// HumanRequest 发送给客户端的请求
type HumanRequest struct {
	ID          string   `json:"id"`
	Message     string   `json:"message"`
	RequestType string   `json:"request_type"`
	Options     []string `json:"options,omitempty"`
	Timeout     int      `json:"timeout"`
	Timestamp   int64    `json:"timestamp"`
}

// HumanResponse 客户端的响应
type HumanResponse struct {
	ID           string `json:"id"`
	Response     string `json:"response"`
	Status       string `json:"status"` // success, timeout, cancelled
	ResponseTime int    `json:"response_time"`
}

// 请求类型常量
const (
	RequestTypeInput        = "input"        // 需要人类输入文本
	RequestTypeConfirmation = "confirmation" // 需要人类确认（是/否）
	RequestTypeDecision     = "decision"     // 需要人类从选项中选择
)

// NewHumanInTool 创建人类介入工具实例
func NewHumanInTool(i18nMgr *i18n.Manager) *HumanInTool {
	return &HumanInTool{
		logger:     slog.Default().WithGroup("human_in_tool"),
		i18nMgr:    i18nMgr,
		commChan:   make(chan HumanResponse, 10),
		pendingReq: make(chan HumanRequest, 10),
	}
}

// NewHumanInToolWithChannels 创建带自定义通道的人类介入工具实例（用于测试或自定义客户端）
func NewHumanInToolWithChannels(i18nMgr *i18n.Manager, commChan chan HumanResponse, pendingReq chan HumanRequest) *HumanInTool {
	return &HumanInTool{
		logger:     slog.Default().WithGroup("human_in_tool"),
		i18nMgr:    i18nMgr,
		commChan:   commChan,
		pendingReq: pendingReq,
	}
}

// 接口实现检查
var _ toolcore.Tool = (*HumanInTool)(nil)

// Schema 实现 toolcore.Tool 接口的 Schema 方法
func (t *HumanInTool) Schema(ctx context.Context) (toolcore.ToolSchema, error) {
	// 获取多语言支持
	langs := t.i18nMgr.GetSupportedLanguages()
	descriptions := make(map[string]string, len(langs))
	localizedNames := make(map[string]string, len(langs))

	for _, lang := range langs {
		langCtx := i18n.ContextWithLanguage(ctx, lang)
		descriptions[lang] = t.i18nMgr.T(langCtx, "tool.human_in.description", nil)
		localizedNames[lang] = t.i18nMgr.T(langCtx, "tool.human_in.name", nil)
	}

	// 定义请求类型枚举
	requestTypes := []any{RequestTypeInput, RequestTypeConfirmation, RequestTypeDecision}

	// 定义输入参数
	inputParameters := []toolcore.ParameterDefinition{
		common.CreateParamDef(ctx, t.i18nMgr, "message", toolcore.ParamTypeString, true, nil, "tool.human_in.arg.message", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "timeout", toolcore.ParamTypeInteger, false, nil, "tool.human_in.arg.timeout", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "request_type", toolcore.ParamTypeString, false,
			Some(requestTypes), "tool.human_in.arg.request_type", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "options", toolcore.ParamTypeArray, false, nil, "tool.human_in.arg.options", nil),
	}

	return toolcore.ToolSchema{
		Name:             "human_in",
		LocalizedNames:   localizedNames,
		Descriptions:     descriptions,
		InputParameters:  inputParameters,
		OutputParameters: t.createOutputParameters(ctx),
	}, nil
}

// Call 实现 toolcore.Tool 接口的 Call 方法
func (t *HumanInTool) Call(ctx context.Context, inputJSON string) (string, error) {
	var args HumanInArgs
	if err := json.Unmarshal([]byte(inputJSON), &args); err != nil {
		t.logger.Error("解析参数失败", "error", err, "input", inputJSON)
		return "", fmt.Errorf("无效的 JSON 输入: %w", err)
	}

	// 参数验证
	if args.Message == "" {
		return "", fmt.Errorf("message 参数不能为空")
	}

	// 设置默认值
	if args.Timeout <= 0 {
		args.Timeout = 300 // 默认5分钟超时
	}
	if args.RequestType == "" {
		args.RequestType = RequestTypeInput
	}

	// 验证请求类型
	switch args.RequestType {
	case RequestTypeInput, RequestTypeConfirmation, RequestTypeDecision:
		// 有效的请求类型
	default:
		return "", fmt.Errorf("无效的请求类型: %s", args.RequestType)
	}

	// 如果是决策类型，验证选项
	if args.RequestType == RequestTypeDecision && len(args.Options) == 0 {
		return "", fmt.Errorf("决策类型请求必须提供选项")
	}

	// 生成请求ID
	requestID := fmt.Sprintf("human_req_%d", time.Now().UnixNano())

	// 创建人类介入请求
	request := HumanRequest{
		ID:          requestID,
		Message:     args.Message,
		RequestType: args.RequestType,
		Options:     args.Options,
		Timeout:     args.Timeout,
		Timestamp:   time.Now().Unix(),
	}

	t.logger.Info("发送人类介入请求", "request_id", requestID, "message", args.Message, "type", args.RequestType)

	// 发送请求到客户端
	select {
	case t.pendingReq <- request:
		// 请求发送成功
	case <-time.After(5 * time.Second):
		return "", fmt.Errorf("无法发送人类介入请求：客户端可能未连接")
	}

	// 等待人类回应或超时
	startTime := time.Now()
	timeout := time.Duration(args.Timeout) * time.Second

	select {
	case response := <-t.commChan:
		// 检查响应是否匹配当前请求
		if response.ID != requestID {
			t.logger.Warn("收到不匹配的响应", "expected_id", requestID, "received_id", response.ID)
			return "", fmt.Errorf("收到不匹配的响应")
		}

		// 计算实际响应时间
		actualResponseTime := int(time.Since(startTime).Seconds())

		result := HumanInResult{
			Status:        response.Status,
			HumanResponse: response.Response,
			ResponseTime:  actualResponseTime,
			RequestType:   args.RequestType,
			TimedOut:      false,
		}

		t.logger.Info("收到人类回应", "request_id", requestID, "status", response.Status, "response_time", actualResponseTime)

		resultJSON, err := json.MarshalToString(result)
		if err != nil {
			t.logger.Error("序列化结果失败", "error", err)
			return "", fmt.Errorf("序列化结果失败: %w", err)
		}
		return resultJSON, nil

	case <-time.After(timeout):
		// 超时处理
		t.logger.Warn("人类介入请求超时", "request_id", requestID, "timeout", args.Timeout)

		result := HumanInResult{
			Status:        "timeout",
			HumanResponse: "",
			ResponseTime:  args.Timeout,
			RequestType:   args.RequestType,
			TimedOut:      true,
		}

		resultJSON, err := json.MarshalToString(result)
		if err != nil {
			t.logger.Error("序列化超时结果失败", "error", err)
			return "", fmt.Errorf("序列化超时结果失败: %w", err)
		}
		return resultJSON, nil

	case <-ctx.Done():
		// 上下文取消
		t.logger.Info("人类介入请求被取消", "request_id", requestID)

		result := HumanInResult{
			Status:        "cancelled",
			HumanResponse: "",
			ResponseTime:  int(time.Since(startTime).Seconds()),
			RequestType:   args.RequestType,
			TimedOut:      false,
		}

		resultJSON, err := json.MarshalToString(result)
		if err != nil {
			t.logger.Error("序列化取消结果失败", "error", err)
			return "", fmt.Errorf("序列化取消结果失败: %w", err)
		}
		return resultJSON, nil
	}
}

// createOutputParameters 创建输出参数定义
func (t *HumanInTool) createOutputParameters(ctx context.Context) []toolcore.ParameterDefinition {
	statusDesc := map[string]string{
		"en": "Request status: success, timeout, or cancelled",
		"zh": "请求状态：success（成功）、timeout（超时）或cancelled（取消）",
	}

	responseDesc := map[string]string{
		"en": "Human response content",
		"zh": "人类回应内容",
	}

	responseTimeDesc := map[string]string{
		"en": "Response time in seconds",
		"zh": "响应时间（秒）",
	}

	requestTypeDesc := map[string]string{
		"en": "Type of request",
		"zh": "请求类型",
	}

	timedOutDesc := map[string]string{
		"en": "Whether the request timed out",
		"zh": "请求是否超时",
	}

	return []toolcore.ParameterDefinition{
		{
			Name:        "status",
			Type:        toolcore.ParamTypeString,
			Description: statusDesc,
			Required:    true,
		},
		{
			Name:        "human_response",
			Type:        toolcore.ParamTypeString,
			Description: responseDesc,
			Required:    true,
		},
		{
			Name:        "response_time",
			Type:        toolcore.ParamTypeInteger,
			Description: responseTimeDesc,
			Required:    true,
		},
		{
			Name:        "request_type",
			Type:        toolcore.ParamTypeString,
			Description: requestTypeDesc,
			Required:    true,
		},
		{
			Name:        "timed_out",
			Type:        toolcore.ParamTypeBoolean,
			Description: timedOutDesc,
			Required:    true,
		},
	}
}

// GetPendingRequestChannel 获取待处理请求通道（供客户端使用）
func (t *HumanInTool) GetPendingRequestChannel() <-chan HumanRequest {
	return t.pendingReq
}

// SendResponse 发送人类回应（供客户端使用）
func (t *HumanInTool) SendResponse(response HumanResponse) error {
	select {
	case t.commChan <- response:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("发送回应超时")
	}
}
