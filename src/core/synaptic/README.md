# 神经突触(Synaptic)

神经突触是 `another-me` 的感知器官，它负责感受外部世界，并唤醒 `another-me`。

## Synaptic 的核心

1. 轻量与高效: Synaptic 需要 7x24 小时运行（或按需启动），因此必须设计得足够轻量，资源消耗低，避免产生高昂的持续运行成本。它不应该运行完整的 LLM 推理。
2. 可配置与可编程: another-me (Agent 主体) 必须能够动态地指示 Synaptic 要监控什么内容、使用什么条件进行触发。
3. 多源信息接入: 需要能够接入多种信息源（社交媒体、新闻 RSS、API、网页变化检测等）。
4. 基于规则与简单模型的过滤: 主要依赖规则匹配、关键词检测、以及可能训练好的小型专用模型（例如，判断新闻相关性的简单分类器）来进行初步判断，而不是复杂的 LLM 分析。
5. 可靠的唤醒机制: 当触发条件满足时，需要有可靠的方式唤醒 another-me，并传递必要的上下文信息。

## Synaptic 的架构和组件

可以设想由以下几个核心组件构成:

1. **监控任务管理器 (Monitor Task Manager):**
- 职责: 维护一个由 another-me 定义和管理的监控任务列表。
- 数据结构:
  - MonitorTask 表:
    - id: 任务 ID。
    - owner_agent_id: 所属的 another-me 实例 ID。
    - task_description: 人类可读的任务描述 (如 "监控 Elon Musk 的推特")。
    - source_type: string (枚举: TWITTER_USER, RSS_FEED, WEBSITE_CHANGE, API_ENDPOINT, KEYWORD_SEARCH 等)。
    - source_target: string (具体目标，如推特用户名、RSS URL、网站 URL、API 地址、搜索关键词)。
    - check_interval_seconds: integer (检查频率)。
    - trigger_conditions: object (触发唤醒的条件，见下文详述)。
    - last_check_time: datetime。
    - last_known_state: object (可选，用于状态比较，如上次检查的推文 ID、网站内容的哈希值)。
    - is_active: boolean。
    - priority: integer (可选，用于调度)。
- 接口: 提供给 another-me 添加、删除、修改、暂停/恢复监控任务的 API。

2. **信息源采集器 (Source Collectors):**
- 职责: 针对不同 source_type 实现具体的采集逻辑。每个采集器都是轻量级的。
- 例子:
  - TwitterCollector: 使用 Twitter API (或爬虫，注意合规性) 获取指定用户的最新推文。
  - RSSCollector: 解析指定的 RSS Feed 获取最新条目。
  - WebChangeCollector: 抓取指定网页，计算内容哈希或特定区域内容，与 last_known_state 比较。
  - APICollector: 调用指定的 API 端点，获取数据。
  - NewsKeywordCollector: 使用新闻聚合 API (如 NewsAPI) 或搜索引擎搜索指定关键词。
- 输出: 将采集到的原始数据（如新推文列表、新闻条目列表、网页内容变化标记）传递给触发器引擎。

3. **触发器引擎 (Trigger Engine):**
- 职责: 接收来自采集器的原始数据，并根据对应 MonitorTask 中定义的 trigger_conditions 进行判断。
- trigger_conditions 结构 (示例用JSON表示，也可以是Go结构体):

```json
        {
          "strategy": "ANY", // 或者 "ALL"
          "rules": [
            {
              "type": "NEW_ITEM_DETECTED" // 简单的检测到新内容就触发
            },
            {
              "type": "KEYWORD_MATCH", // 内容中包含特定关键词
              "keywords": ["AI", "startup", "funding"],
              "match_logic": "ANY" // 或 "ALL"
            },
            {
              "type": "REGEX_MATCH", // 内容匹配正则表达式
              "pattern": "important announcement"
            },
            {
              "type": "SENTIMENT_SCORE", // 简单情感分析（如果需要，可选）
              "threshold": 0.7, // 如，正面情绪超过 0.7
              "direction": "POSITIVE" // 或 "NEGATIVE", "ANY"
            },
            {
               "type": "SIMPLE_CLASSIFIER", // 使用预训练的小型模型判断相关性
               "model_id": "news_topic_classifier_xyz",
               "expected_label": "TECH_BUSINESS",
               "confidence_threshold": 0.8
            },
            {
              "type": "WEBSITE_CONTENT_CHANGED", // 网页内容变化
              "comparison_method": "HASH" // 或 "SELECTOR_CONTENT"
            }
            // ... 可以扩展更多规则类型
          ]
        }
```

- 逻辑: 对每条采集到的数据应用规则，如果满足 strategy (ANY 或 ALL) 定义的条件，则判定为触发。

4. **唤醒调度器 (Wake-up Scheduler):**
- 职责: 接收来自触发器引擎的触发信号，并负责实际唤醒 another-me。
- 唤醒机制:
  - 消息队列: 向与 another-me 关联的特定消息队列（如 Redis Stream, RabbitMQ, Kafka, 或 channel）发送一条包含触发信息（任务 ID、触发的数据、触发原因）的消息。another-me 在启动时或空闲时监听此队列。
  - API 回调: 调用 another-me 预先注册的回调 API。
  - 数据库标记: 在数据库中设置一个特定的状态标记，another-me 定期检查此标记。 (相对延迟较高)
- 传递上下文: 将触发的 MonitorTask 信息、导致触发的原始数据、以及可能的初步分析结果（如匹配的关键词）一起传递给 another-me。
- 合并与去重 (可选): 如果短时间内有大量相似触发，可以考虑合并或去重，避免过于频繁地唤醒 Agent。

5. **任务调度器 (Task Scheduler):**
- 职责: 根据 MonitorTask 中定义的 check_interval_seconds 和 priority，调度信息源采集器的执行。可以使用轻量级的调度库或依赖操作系统的 cron。

## 工作流程示例(监控特定推特用户):

1. another-me 通过 API 向 Monitor Task Manager 添加一个任务: source_type='TWITTER_USER', source_target='@some_user', check_interval_seconds=300, trigger_conditions={'strategy': 'ANY', 'rules': [{'type': 'NEW_ITEM_DETECTED'}]}。
2. Task Scheduler 每 300 秒触发 TwitterCollector 检查 @some_user 的推特。
3. TwitterCollector 获取最新的推文 ID 列表，与数据库中该任务的 last_known_state (上次的最新推文 ID) 比较。
4. 如果发现新的推文 ID，TwitterCollector 将新推文的数据发送给 Trigger Engine。
5. Trigger Engine 根据任务的 trigger_conditions (检测到新条目)，判定为触发。
6. Trigger Engine 通知 Wake-up Scheduler。
7. Wake-up Scheduler 将包含任务信息和新推文内容的消息发送到 another-me 的消息队列。
8. another-me 主进程（如果正在运行且空闲，或下次启动时）从队列中读取到消息，被唤醒，并根据传递的上下文进行处理（如阅读推文、存入记忆、判断是否需要进一步行动或向用户汇报）。
9. (可选) another-me 在处理完后，可以更新 MonitorTask 的 last_known_state。

## 其他唤醒机制:

除了 Synaptic 主动触发，还可以有：

- 定时唤醒: another-me 可以设定在特定时间（如每天早上 8 点）自动唤醒，执行例行任务（如检查日程、阅读晨间新闻摘要）。这可以通过任务调度器直接安排 another-me 的唤醒逻辑。
- 用户主动唤醒: 用户通过应用界面或命令行直接启动/连接 another-me。
- 外部事件回调: 其他系统（如日历服务、邮件服务）通过 webhook 或 API 调用触发 another-me 唤醒。

通过这种设计，Synaptic 可以作为一个高效、灵活的外部感知系统，让 another-me 在休眠时也能保持对关键信息的关注，并在必要时被智能地唤醒，大大提高了 Agent 的自主性和实用性，同时有效控制了运行成本。