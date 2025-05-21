# Browser Agent 示例

本示例展示了如何使用浏览器工具(browsertool)创建一个能够自动化执行网页任务的智能代理。

## 功能特点

- 访问网站并进行网页浏览
- 截取网页截图
- 点击页面元素
- 填写表单内容
- 执行JavaScript脚本
- 提取网页内容
- 调试网页等

## 示例结构

本示例包含两个版本：

1. **基础版本** (basic/main.go) - 展示基本的浏览器自动化功能
2. **高级版本** (advanced/main.go) - 展示更复杂的浏览器任务和自定义配置

## 使用方法

### 环境要求

- Go 1.20+
- Chrome/Chromium浏览器（会自动搜索系统中已安装的浏览器）
- DEEPSEEK_API_KEY环境变量（使用Deepseek模型）

### 运行基础示例

```bash
# 设置API密钥
export DEEPSEEK_API_KEY=your_api_key_here

# 运行基础示例
cd examples/browser_agent/basic
go run main.go
```

### 运行高级示例

```bash
# 设置API密钥
export DEEPSEEK_API_KEY=your_api_key_here

# 运行高级示例
cd examples/browser_agent/advanced
go run main.go
```

## 代码说明

### 基础示例功能

基础示例展示了如何使用默认配置创建浏览器工具，并执行简单的网页任务：
- 访问百度首页
- 截图保存
- 搜索关键词
- 截取搜索结果页面

### 高级示例功能

高级示例展示了如何使用自定义配置创建浏览器工具，执行更复杂的任务：
- 自定义浏览器窗口大小和超时设置
- 使用有头模式（可视界面）
- 执行多步骤、复杂任务
- 使用JavaScript提取网页内容
- 任务完成后关闭浏览器

## 自定义浏览器任务

修改代码中的userInput字符串可以定义不同的浏览器任务，AI代理会尝试理解并执行这些任务。

## 浏览器工具接口

浏览器工具支持以下操作类型：
- navigate - 导航到指定URL
- screenshot - 截取网页或元素截图
- click - 点击页面元素
- fill - 填写表单
- select - 选择下拉选项
- hover - 鼠标悬停
- evaluate - 执行JavaScript代码
- debug - 调试操作 