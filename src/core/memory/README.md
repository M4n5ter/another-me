## 工作流程设想:

1. 记忆摄入 (Ingestion):

- 新的外部信息（网页浏览、聊天、用户输入）或内部信息（Agent 思考）产生。
- 创建 MemoryUnit 记录，填充 type, source, content_text, metadata。
- 对 content_text 进行向量化，存入 content_embedding。
- 识别 content_text 中的实体，创建或查找 Entity 节点，并建立 MENTIONS 关系。
- 与上一条 MemoryUnit 建立 SEQUENCED 关系。
- 初步评估 importance。

2. 记忆检索 (Retrieval):

- 当 Agent 需要响应、决策或反思时，根据当前上下文生成检索查询。
- 混合查询:
  - 向量搜索: 在 MemoryUnit, Entity, UserPreference 上使用 content_embedding/representation_embedding 查找语义相似的记忆/偏好。
  - 图遍历: 从相关的 Entity 或最近的 MemoryUnit 出发，沿着 SEQUENCED, MENTIONS, SUPPORTS 等关系扩展，获取上下文相关的记忆。
  - 条件过滤: 使用 timestamp, type, source, importance 等元数据进行过滤。
- 排序与融合: 对检索结果根据相关性、重要性、时间近近性等进行排序，形成输入给 LLM 的上下文。

3. 记忆反思与整合 (Reflection & Consolidation - 定期或触发):

- Agent 检索一段时间内（或关于特定主题）的 MemoryUnit 集合。
- 使用 LLM 分析这些记忆，寻找模式、规律、用户反馈。
- 生成推断:
  - 如果发现一贯的行为模式或明确的用户表达，创建或更新 UserPreference 节点。
  - 建立 SUPPORTS 或 CONTRADICTS 关系连接证据 MemoryUnit。
  - 更新 UserPreference 的 confidence 和 last_updated。
- 生成摘要:
  - 将一组相关的低级 MemoryUnit 总结成一个 type='SUMMARY' 的 MemoryUnit。
  - 建立 ABSTRACTED_INTO 关系。
- 更新重要性: 根据访问频率、是否被用于推断、用户反馈等调整 MemoryUnit 的 importance。