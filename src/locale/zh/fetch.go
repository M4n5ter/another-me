package zh

// --- Fetch 工具 ---

const FetchToolDescription = "从给定的 URL 获取内容。它可以简化 HTML 内容或返回原始数据。"

const FetchToolArgURLDescription = "要从中获取内容的 URL。"

const FetchToolArgMaxLengthDescription = "从内容中返回的最大字符数(默认值: 10000)。"

const FetchToolArgStartIndexDescription = "获取内容的起始字符索引(默认值: 0)。用于分页。"

const FetchToolArgRawDescription = "如果为 true, 则返回原始内容(例如 HTML)，而不是尝试简化它(默认值: false)。"

const FetchToolArgIgnoreRobotsTxtDescription = "如果为 true, 则忽略站点的 robots.txt 规则(默认值: false)。请负责任地使用。"

const FetchToolArgUserAgentDescription = "用于请求的自定义 User-Agent 字符串。根据上下文（自主 vs 手动）默认设置。"

const FetchToolArgProxyURLDescription = "用于请求的可选代理 URL(例如，'http://user:pass@host:port')。如果未设置，则使用环境代理。"

const FetchToolArgIsManualRequestDescription = "指示请求是否由用户操作手动触发(影响默认 User-Agent 和 robots.txt 检查行为)。"
