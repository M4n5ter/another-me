package fetchtool

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/go-shiori/go-readability"
	json "github.com/json-iterator/go"
	"github.com/temoto/robotstxt"

	"github.com/m4n5ter/another-me/pkg/common"
	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

const (
	defaultMaxLength              = 10000                                                                                                                           // 默认最大字符数
	defaultTimeoutSeconds         = 30                                                                                                                              // 默认超时时间 (秒)
	defaultUserAgentAutonomous    = "Mozilla/5.0 (compatible; AnotherMe/1.0; +https://github.com/m4n5ter/another-me)"                                               // 自主请求使用的 User-Agent
	defaultUserAgentManualRequest = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36 Edg/135.0.0.0" // 模拟浏览器的手动请求 User-Agent
)

// FetchTool 实现 toolcore.Tool 接口，用于从 URL 获取内容
type FetchTool struct {
	logger  *slog.Logger
	i18nMgr *i18n.Manager
}

// Args 定义了 FetchTool 的参数
type Args struct {
	URL             string `json:"url"`                         // 要获取的 URL
	MaxLength       int    `json:"max_length,omitempty"`        // 返回内容的最大字符数
	StartIndex      int    `json:"start_index,omitempty"`       // 返回内容的起始索引
	Raw             bool   `json:"raw,omitempty"`               // 返回原始 HTML 内容而不是简化后的文本
	IgnoreRobotsTxt bool   `json:"ignore_robots_txt,omitempty"` // 忽略 robots.txt 规则
	UserAgent       string `json:"user_agent,omitempty"`        // 自定义 User-Agent 字符串
	ProxyURL        string `json:"proxy_url,omitempty"`         // 可选的代理 URL
	IsManualRequest bool   `json:"is_manual_request,omitempty"` // 指示请求是否由用户操作手动触发
}

// Result 定义了成功结果的结构
type Result struct {
	URL            string `json:"url"`                        // 已获取的 URL
	Content        string `json:"content"`                    // 获取到的内容（可能已简化/截断）
	OriginalLength int    `json:"original_length"`            // 截断前原始内容的总长度
	ReturnedLength int    `json:"returned_length"`            // 本次调用实际返回的内容长度
	StartIndex     int    `json:"start_index"`                // 本次获取使用的起始索引
	IsTruncated    bool   `json:"is_truncated"`               // 指示返回的内容是否已被截断
	NextStartIndex *int   `json:"next_start_index,omitempty"` // 如果内容被截断，下次获取应使用的 start_index
	Message        string `json:"message,omitempty"`          // 附加的状态信息
	ContentType    string `json:"content_type,omitempty"`     // 响应的 Content-Type 头
}

// NewFetchTool 创建一个新的 FetchTool 实例
func NewFetchTool(i18nMgr *i18n.Manager) *FetchTool {
	return &FetchTool{
		logger:  slog.Default().WithGroup("fetch_tool"),
		i18nMgr: i18nMgr,
	}
}

var _ toolcore.Tool = (*FetchTool)(nil)

// Schema 实现 toolcore.Tool 接口的 Schema 方法
func (t *FetchTool) Schema(ctx context.Context) (toolcore.ToolSchema, error) {
	// 获取不同语言的描述文本
	langs := t.i18nMgr.GetSupportedLanguages()
	descriptions := make(map[string]string, len(langs))
	localizedNames := make(map[string]string, len(langs))

	for _, lang := range langs {
		langCtx := i18n.ContextWithLanguage(ctx, lang)
		descriptions[lang] = t.i18nMgr.T(langCtx, "tool.fetch.description", nil)
		// 工具名称可以本地化为友好显示，但实际调用仍使用规范名称 "fetch"
		localizedNames[lang] = "Fetch"
	}

	// 构建参数定义
	inputParameters := []toolcore.ParameterDefinition{
		common.CreateParamDef(ctx, t.i18nMgr, "url", toolcore.ParamTypeString, true, nil, "tool.fetch.arg.url", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "max_length", toolcore.ParamTypeInteger, false, nil, "tool.fetch.arg.max_length", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "start_index", toolcore.ParamTypeInteger, false, nil, "tool.fetch.arg.start_index", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "raw", toolcore.ParamTypeBoolean, false, nil, "tool.fetch.arg.raw", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "ignore_robots_txt", toolcore.ParamTypeBoolean, false, nil, "tool.fetch.arg.ignore_robots_txt", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "user_agent", toolcore.ParamTypeString, false, nil, "tool.fetch.arg.user_agent", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "proxy_url", toolcore.ParamTypeString, false, nil, "tool.fetch.arg.proxy_url", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "is_manual_request", toolcore.ParamTypeBoolean, false, nil, "tool.fetch.arg.is_manual_request", nil),
	}

	// 返回工具的完整模式
	return toolcore.ToolSchema{
		Name:             "fetch",
		LocalizedNames:   localizedNames,
		Descriptions:     descriptions,
		InputParameters:  inputParameters,
		OutputParameters: t.createOutputParameters(ctx),
	}, nil
}

// Call 实现 toolcore.Tool 接口的 Call 方法
func (t *FetchTool) Call(ctx context.Context, inputJSON string) (string, error) {
	var args Args
	if err := json.Unmarshal([]byte(inputJSON), &args); err != nil {
		t.logger.Error("解析参数失败", "error", err, "input", inputJSON)
		return "", fmt.Errorf("无效的 JSON 输入: %w", err)
	}

	if args.URL == "" {
		return "", fmt.Errorf("url 参数是必需的")
	}

	parsedURL, err := url.Parse(args.URL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return "", fmt.Errorf("无效或不支持的 URL 方案: %s", args.URL)
	}

	// --- 设置默认值 ---
	if args.MaxLength <= 0 {
		args.MaxLength = defaultMaxLength
	}
	if args.StartIndex < 0 {
		args.StartIndex = 0
	}
	userAgent := args.UserAgent
	if userAgent == "" {
		if args.IsManualRequest {
			userAgent = defaultUserAgentManualRequest // 手动请求使用模拟浏览器的 UA
		} else {
			userAgent = defaultUserAgentAutonomous // 自主请求使用自定义 UA
		}
	}

	// --- HTTP 客户端设置 ---
	client, err := t.setupHTTPClient(args.ProxyURL)
	if err != nil {
		t.logger.Error("设置 HTTP 客户端失败", "error", err, "proxy", args.ProxyURL)
		return "", fmt.Errorf("设置 HTTP 客户端失败: %w", err)
	}

	// --- Robots.txt 检查 ---
	// 仅在非手动、自主请求且未明确忽略时检查 robots.txt。
	if !args.IsManualRequest && !args.IgnoreRobotsTxt {
		allowed, checkErr := t.checkRobotsTxt(ctx, client, parsedURL, userAgent)
		// 对于自主请求，如果 robots.txt 检查过程中出现任何错误（获取失败、解析失败、401/403），都应视为禁止访问。
		if checkErr != nil {
			t.logger.Error("robots.txt 检查失败，禁止获取", "url", args.URL, "userAgent", userAgent, "error", checkErr)
			return "", fmt.Errorf("robots.txt 检查失败 (%s)，禁止获取: %w", args.URL, checkErr)
		} else if !allowed {
			t.logger.Warn("robots.txt 禁止获取", "url", args.URL, "userAgent", userAgent)
			return "", fmt.Errorf("robots.txt 禁止用户代理 '%s' 获取此页面: %s", userAgent, args.URL)
		}
		t.logger.Info("robots.txt 允许获取", "url", args.URL, "userAgent", userAgent)
	}

	// --- 获取内容 ---
	req, err := http.NewRequestWithContext(ctx, "GET", args.URL, nil)
	if err != nil {
		t.logger.Error("创建请求失败", "url", args.URL, "error", err)
		return "", fmt.Errorf("为 %s 创建请求失败: %w", args.URL, err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8") // 模拟浏览器常见的 Accept 头
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")                                                                          // 优先接受英文内容

	t.logger.Info("正在获取 URL", "url", args.URL, "userAgent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		t.logger.Error("获取 URL 失败", "url", args.URL, "error", err)
		return "", fmt.Errorf("获取 %s 失败: %w", args.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.logger.Error("收到非 2xx 状态码", "url", args.URL, "status", resp.Status)
		bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1024)) // 读取部分响应体以获取上下文信息
		if err != nil {
			t.logger.Error("读取响应体失败", "url", args.URL, "error", err)
			return "", fmt.Errorf("从 %s 读取响应体失败: %w", args.URL, err)
		}
		return "", fmt.Errorf("获取 %s 失败: 状态 %s, 响应体: %s", args.URL, resp.Status, string(bodyBytes))
	}

	contentType := resp.Header.Get("Content-Type")
	isHTML := strings.Contains(contentType, "text/html") // 检查 Content-Type 是否表明是 HTML

	var content string // 用于存储最终处理后的内容
	var message string // 用于存储附加的状态信息

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.logger.Error("读取响应体失败", "url", args.URL, "error", err)
		return "", fmt.Errorf("从 %s 读取响应体失败: %w", args.URL, err)
	}
	originalContent := string(bodyBytes)
	originalLength := len([]rune(originalContent)) // 使用 rune 计数以获得更准确的字符数

	switch {
	case !args.Raw && isHTML:
		// 尝试简化 HTML
		t.logger.Debug("尝试简化 HTML 内容", "url", args.URL)
		article, err := readability.FromReader(strings.NewReader(originalContent), parsedURL)

		if err != nil || (article.Content == "" && article.TextContent == "") {
			// readability 执行出错或未提取到任何内容
			content = originalContent
			message = "无法简化 HTML 内容，返回原始内容。"
			if err != nil {
				t.logger.Warn("简化 HTML 时出错", "url", args.URL, "error", err)
				message += fmt.Sprintf(" (Readability 错误: %s)", err)
			} else {
				t.logger.Warn("Readability 未提取到任何内容", "url", args.URL)
				message += " (Readability 未提取到内容)"
			}
			break
		}

		// 处理成功提取内容的情况
		switch {
		case article.Content != "":
			// 优先尝试将简化后的 HTML (article.Content) 转换为 Markdown
			converter := md.NewConverter("", true, nil)
			markdownContent, err := converter.ConvertString(article.Content)
			if err == nil {
				content = markdownContent
				message = "内容已从 HTML 简化并转换为 Markdown。"
				t.logger.Info("成功简化 HTML 并转换为 Markdown", "url", args.URL, "originalLength", originalLength, "markdownLength", len([]rune(content)))
			} else {
				// Markdown 转换失败，回退到纯文本
				t.logger.Warn("将简化后的 HTML 转换为 Markdown 失败，回退到纯文本", "url", args.URL, "error", err)
				content = article.TextContent // 使用纯文本
				message = "内容已从 HTML 简化，但 Markdown 转换失败，返回纯文本。"
				if content == "" {
					// 如果连 TextContent 也没有，则认为简化失败
					t.logger.Warn("简化 HTML 失败，未提取到 Content 或 TextContent", "url", args.URL)
					content = originalContent
					message = "无法简化 HTML 内容，返回原始内容 (readability 未提取到 Content 或 TextContent)。"
				}
			}
		case article.TextContent != "":
			// 如果没有简化后的 HTML (article.Content)，但有纯文本 (article.TextContent)
			t.logger.Info("简化 HTML 时仅获得纯文本内容", "url", args.URL, "originalLength", originalLength, "simplifiedLength", len([]rune(article.TextContent)))
			content = article.TextContent
			message = "内容已从 HTML 简化为纯文本。"
		default:
			// 两个都没有，视为简化失败
			t.logger.Warn("简化 HTML 失败，未提取到 Content 或 TextContent", "url", args.URL)
			content = originalContent
			message = "无法简化 HTML 内容，返回原始内容 (readability 未提取到 Content 或 TextContent)。"
		}
	default:
		// 原始请求或非 HTML 内容
		content = originalContent
		if args.Raw {
			message = "按要求返回原始内容。"
		} else {
			message = fmt.Sprintf("内容类型 (%s) 不是 HTML 或未请求简化，返回原始内容。", contentType)
		}
	}

	// --- 内容截断 ---
	contentRunes := []rune(content) // 使用 rune 进行截断，以正确处理多字节字符
	contentLength := len(contentRunes)
	var returnedContent string
	var nextStartIndex *int
	isTruncated := false

	if args.StartIndex >= contentLength {
		returnedContent = ""
		message += " 起始索引超出内容长度。"
	} else {
		endIndex := min(args.StartIndex+args.MaxLength, contentLength)
		returnedContent = string(contentRunes[args.StartIndex:endIndex])

		if endIndex < contentLength {
			// 如果结束索引小于总长度，说明内容被截断了
			isTruncated = true
			next := endIndex // 下一次开始的索引就是当前的结束索引
			nextStartIndex = &next
			message += fmt.Sprintf(" 内容在 %d 字符处被截断。", args.MaxLength)
		}
	}

	// --- 准备结果 ---
	result := Result{
		URL:            args.URL,
		Content:        returnedContent,
		OriginalLength: contentLength, // 此处是简化后、应用截断逻辑 *之前* 的内容长度
		ReturnedLength: len([]rune(returnedContent)),
		StartIndex:     args.StartIndex,
		IsTruncated:    isTruncated,
		NextStartIndex: nextStartIndex,
		Message:        strings.TrimSpace(message),
		ContentType:    contentType,
	}

	resultJSON, err := json.MarshalToString(result)
	if err != nil {
		t.logger.Error("序列化结果失败", "error", err)
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}

	t.logger.Info("获取成功", "url", args.URL, "returnedLength", result.ReturnedLength, "isTruncated", result.IsTruncated)
	return resultJSON, nil
}

// setupHTTPClient 配置并返回一个带有代理和超时的 HTTP 客户端。
func (t *FetchTool) setupHTTPClient(proxyURLStr string) (*http.Client, error) {
	// 克隆默认 Transport 以避免修改全局设置
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 100                 // 增加最大空闲连接数
	transport.MaxIdleConnsPerHost = 10           // 限制每个主机的最大空闲连接数
	transport.IdleConnTimeout = 90 * time.Second // 增加空闲连接超时时间

	if proxyURLStr != "" {
		// 如果指定了代理 URL
		proxyURL, err := url.Parse(proxyURLStr)
		if err != nil {
			return nil, fmt.Errorf("无效的代理 URL %s: %w", proxyURLStr, err)
		}
		transport.Proxy = http.ProxyURL(proxyURL) // 设置代理
		t.logger.Info("使用代理", "proxy_url", proxyURLStr)
	} else {
		// 如果未指定代理 URL，则遵循环境变量中的设置 (HTTP_PROXY, HTTPS_PROXY)
		transport.Proxy = http.ProxyFromEnvironment
		t.logger.Debug("如果环境变量中设置了代理，则使用该代理")
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   defaultTimeoutSeconds * time.Second, // 设置请求超时
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// 自定义重定向策略
			// 允许重定向，但记录日志，并限制重定向次数
			if len(via) >= 10 {
				t.logger.Warn("请求重定向次数过多，停止重定向", "url", req.URL.String(), "redirect_count", len(via))
				return http.ErrUseLastResponse // 返回最后一个响应，避免无限重定向
			}
			t.logger.Debug("跟随重定向", "to", req.URL.String(), "from", via[len(via)-1].URL.String())
			return nil // 返回 nil 表示允许重定向
		},
	}
	return client, nil
}

// checkRobotsTxt 获取并检查目标 URL 的 robots.txt，判断指定的 userAgent 是否被允许访问。
// 返回值: (是否允许访问, 错误信息)
// 注意：对于自主请求，调用方应将任何非 nil 的错误视为禁止访问。
func (t *FetchTool) checkRobotsTxt(ctx context.Context, client *http.Client, targetURL *url.URL, userAgent string) (bool, error) {
	// 构建 robots.txt 的 URL (通常在网站根目录下)
	robotsURL := targetURL.Scheme + "://" + targetURL.Host + "/robots.txt"
	t.logger.Info("检查 robots.txt", "url", robotsURL, "target", targetURL.String())

	req, err := http.NewRequestWithContext(ctx, "GET", robotsURL, nil)
	if err != nil {
		// 无法创建请求本身就是一个错误
		return false, fmt.Errorf("创建 robots.txt 请求失败: %w", err)
	}
	// 获取 robots.txt 时使用哪个 User-Agent？
	// 为了与实际的抓取检查保持一致，这里也使用提供的 User-Agent。
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		// 获取 robots.txt 时发生网络错误
		t.logger.Error("获取 robots.txt 时发生网络错误", "robots_url", robotsURL, "error", err)
		// 对于自主代理，网络错误通常意味着无法确认是否允许，应视为禁止
		return false, fmt.Errorf("获取 robots.txt (%s) 时发生网络错误: %w", robotsURL, err)
	}
	defer resp.Body.Close()

	// 检查状态码
	switch resp.StatusCode {
	case http.StatusOK: // 200 OK - 继续解析
	case http.StatusUnauthorized, http.StatusForbidden: // 401, 403
		t.logger.Warn("robots.txt 禁止访问 (401/403)", "robots_url", robotsURL, "status", resp.Status)
		// 明确禁止访问 robots.txt，应视为禁止获取目标 URL
		return false, fmt.Errorf("服务器禁止访问 robots.txt (%s): status %s", robotsURL, resp.Status)
	case http.StatusNotFound: // 404
		t.logger.Info("robots.txt 未找到 (404)，允许获取", "robots_url", robotsURL)
		return true, nil // robots.txt 不存在，通常允许访问
	default:
		// 其他 4xx/5xx 错误
		t.logger.Warn("获取 robots.txt 时收到非预期状态码，允许获取", "robots_url", robotsURL, "status", resp.Status)
		// 无法获取有效的 robots.txt，采取宽容策略，允许访问，但可能记录错误
		return true, nil // 暂时采取宽容策略
	}

	// 使用 robotstxt 库解析响应体
	robotsData, err := robotstxt.FromResponse(resp)
	if err != nil {
		t.logger.Error("解析 robots.txt 失败", "robots_url", robotsURL, "error", err)
		// 解析失败意味着无法确认规则，对于自主代理应视为禁止
		return false, fmt.Errorf("解析 robots.txt (%s) 失败: %w", robotsURL, err)
	}

	// 检查指定的路径是否对我们的 User-Agent 开放。
	path := targetURL.Path
	if path == "" {
		path = "/"
	}
	allowed := robotsData.TestAgent(path, userAgent)
	t.logger.Info("robots.txt 检查结果", "allowed", allowed, "target_path", path, "userAgent", userAgent)
	return allowed, nil
}

// createOutputParameters 创建输出参数定义
func (t *FetchTool) createOutputParameters(_ context.Context) []toolcore.ParameterDefinition {
	// 可以从i18n系统获取这些描述，但为简化起见，这里使用硬编码的描述
	contentDesc := map[string]string{
		"en": "The fetched content, possibly simplified or truncated",
		"zh": "获取的内容，可能已简化或截断",
	}

	urlDesc := map[string]string{
		"en": "The URL that was fetched",
		"zh": "已获取的 URL",
	}

	originalLengthDesc := map[string]string{
		"en": "Total length of the original content before truncation",
		"zh": "截断前原始内容的总长度",
	}

	return []toolcore.ParameterDefinition{
		{
			Name:        "url",
			Type:        toolcore.ParamTypeString,
			Description: urlDesc,
			Required:    true,
		},
		{
			Name:        "content",
			Type:        toolcore.ParamTypeString,
			Description: contentDesc,
			Required:    true,
		},
		{
			Name:        "original_length",
			Type:        toolcore.ParamTypeInteger,
			Description: originalLengthDesc,
			Required:    true,
		},
	}
}
