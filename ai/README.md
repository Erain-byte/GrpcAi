# AI Service with LangChain Integration

## 📚 项目概述

本项目是一个基于 **LangChain** 构建的 AI 微服务,提供智能对话、记忆管理、向量搜索等功能。采用 FastAPI + gRPC 双协议架构,支持与 Gateway 无缝集成。

## 🏗️ 架构设计

```
┌─────────────────────────────────────────────────┐
│              AI Service (Python)                 │
├─────────────────────────────────────────────────┤
│  FastAPI HTTP Layer    │   gRPC Server Layer    │
├─────────────────────────────────────────────────┤
│           LangChain Orchestration Layer          │
│  ┌──────────┬──────────┬──────────┬──────────┐  │
│  │ Chains   │ Agents   │ Memory   │ Tools    │  │
│  └──────────┴──────────┴──────────┴──────────┘  │
├─────────────────────────────────────────────────┤
│  LLM Providers  │  Vector Stores  │ Embeddings  │
│  (OpenAI/etc)   │  (FAISS/Chroma) │             │
├─────────────────────────────────────────────────┤
│     Redis (Memory)  │  MySQL (Persistence)      │
└─────────────────────────────────────────────────┘
```

## 📁 目录结构

```
ai/
├── src/
│   ├── main.py                  # FastAPI 应用入口
│   ├── grpc_service.py          # gRPC 服务实现
│   ├── routes/
│   │   ├── __init__.py
│   │   └── langchain_routes.py  # LangChain HTTP 路由
│   ├── llm/
│   │   ├── __init__.py
│   │   └── provider.py          # LLM 提供者管理 (OpenAI/Claude/Gemini)
│   ├── chains/
│   │   ├── __init__.py
│   │   └── builder.py           # Chain 构建器 (对话链/简单链)
│   ├── agents/
│   │   ├── __init__.py
│   │   └── builder.py           # Agent 构建器 (ReAct Agent)
│   ├── memory/
│   │   ├── __init__.py
│   │   └── manager.py           # 记忆管理器 (Redis 持久化)
│   ├── embeddings/
│   │   ├── __init__.py
│   │   └── manager.py           # 嵌入模型管理
│   ├── vectorstores/
│   │   ├── __init__.py
│   │   └── manager.py           # 向量存储管理 (FAISS/Chroma)
│   └── tools/
│       └── __init__.py          # 自定义工具集
├── config.py                    # Pydantic 配置管理
├── requirements.txt             # Python 依赖
├── .env                         # 环境变量 (不提交到 Git)
├── .env.example                 # 环境变量模板
└── README.md                    # 本文档
```

## 🚀 快速开始

### 1. 安装依赖

```bash
cd ai
pip install -r requirements.txt
```

### 2. 配置环境变量

```bash
cp .env.example .env
# 编辑 .env 文件,填入你的 API Key
```

关键配置项:
```env
# OpenAI API Key (必需)
AI_API_KEY=sk-your-openai-key

# LLM 配置
AI_MODEL_PROVIDER=openai
AI_MODEL=gpt-3.5-turbo
AI_TEMPERATURE=0.7
AI_MAX_TOKENS=2048

# Redis 配置 (用于记忆持久化)
REDIS_HOST=localhost
REDIS_PORT=6379

# 向量数据库配置
VECTOR_STORE_TYPE=faiss
VECTOR_STORE_PATH=./data/vectors
```

### 3. 启动服务

```bash
# 开发模式
python src/main.py

# 生产模式
uvicorn src.main:app --host 0.0.0.0 --port 8003 --workers 4
```

## 📖 API 文档

### HTTP 接口 (FastAPI)

启动后访问: `http://localhost:8003/docs`

#### 1. 标准对话

```bash
POST /langchain/chat
Content-Type: application/json

{
  "message": "你好,请介绍一下 LangChain",
  "conversation_id": "conv_123",
  "model": "gpt-3.5-turbo",
  "use_summary_memory": false,
  "window_size": 10
}
```

**响应:**
```json
{
  "reply": "LangChain 是一个强大的框架...",
  "conversation_id": "conv_123",
  "model": "gpt-3.5-turbo"
}
```

#### 2. 流式对话

```bash
POST /langchain/stream-chat
Content-Type: application/json

{
  "message": "写一首关于 AI 的诗",
  "conversation_id": "conv_456"
}
```

**响应 (SSE 格式):**
```
data: {"chunk":"在","conversation_id":"conv_456","done":false}

data: {"chunk":"数字","conversation_id":"conv_456","done":false}

data: {"chunk":"","conversation_id":"conv_456","done":true}
```

#### 3. 获取对话历史

```bash
GET /langchain/history/conv_123?limit=50
```

#### 4. 删除对话

```bash
DELETE /langchain/conversation/conv_123
```

#### 5. 列出可用模型

```bash
GET /langchain/models
```

### gRPC 接口

需要先生成 Protobuf 代码:

```bash
cd proto
./gen.sh
```

然后在 Python 中调用:

```python
import grpc
from proto.ai import ai_pb2, ai_pb2_grpc

channel = grpc.insecure_channel('localhost:9003')
stub = ai_pb2_grpc.AiServiceStub(channel)

response = stub.Chat(ai_pb2.ChatRequest(
    message="你好",
    conversation_id="conv_123"
))

print(response.reply)
```

## 🔧 核心模块说明

### 1. LLM Provider (`src/llm/provider.py`)

支持多模型切换:

```python
from src.llm.provider import LLMProvider

# 使用 OpenAI
llm = LLMProvider.create_llm(provider="openai", model="gpt-4")

# 使用 Anthropic Claude
llm = LLMProvider.create_llm(provider="anthropic", model="claude-3-opus")

# 使用 Google Gemini
llm = LLMProvider.create_llm(provider="google", model="gemini-pro")
```

### 2. Memory Manager (`src/memory/manager.py`)

三种记忆模式:

```python
from src.memory.manager import memory_manager

# Buffer Memory (完整历史)
memory = memory_manager.create_buffer_memory("conv_123")

# Summary Memory (自动总结长对话)
memory = memory_manager.create_summary_memory(llm, "conv_123")

# Window Memory (只保留最近 K 条)
memory = memory_manager.create_window_memory("conv_123", k=10)
```

**Redis 存储结构:**
```
Key: conversation:{conversation_id}:messages
Value: JSON Array [
  {"role": "human", "content": "你好"},
  {"role": "ai", "content": "你好!有什么可以帮助你的?"}
]
TTL: 30 days
```

### 3. Chain Builder (`src/chains/builder.py`)

```python
from src.chains.builder import ChainBuilder

# 带记忆的对话链
chain = ChainBuilder.create_conversation_chain(
    conversation_id="conv_123",
    system_prompt="你是一个专业的 Python 程序员助手",
    use_summary=False,
)

response = chain.invoke({"input": "如何优化这段代码?"})
```

### 4. Agent Builder (`src/agents/builder.py`)

```python
from src.agents.builder import AgentBuilder

# 创建带搜索功能的 Agent
agent = AgentBuilder.create_search_agent(
    enable_web_search=True,
    enable_wikipedia=True,
)

result = agent.invoke({
    "input": "查找 2024 年最新的 AI 技术趋势"
})
```

### 5. Vector Store (`src/vectorstores/manager.py`)

```python
from src.vectorstores.manager import VectorStoreManager
from langchain_core.documents import Document

# 创建向量存储
vector_store = VectorStoreManager.create_vector_store()

# 添加文档
docs = [
    Document(page_content="LangChain 是一个框架", metadata={"source": "doc1"}),
]
VectorStoreManager.add_documents(vector_store, docs)

# 相似度搜索
results = VectorStoreManager.similarity_search(
    vector_store,
    query="什么是 LangChain?",
    k=5
)
```

## 🎯 使用场景示例

### 场景 1: 客服机器人

```python
from src.chains.builder import get_conversation_chain

# 创建客服链
chain = get_conversation_chain(
    conversation_id="customer_001",
    system_prompt="""你是公司客服助手。
规则:
1. 语气友好专业
2. 回答简洁明了
3. 不知道的问题引导用户联系人工客服""",
    window_size=20,  # 只保留最近 20 条消息
)

# 处理用户问题
response = chain.invoke({"input": "我的订单什么时候发货?"})
```

### 场景 2: RAG 知识库问答

```python
from src.vectorstores.manager import get_vector_store, VectorStoreManager
from src.chains.builder import ChainBuilder
from langchain.chains import RetrievalQA

# 加载向量库
vector_store = get_vector_store()

# 创建检索器
retriever = vector_store.as_retriever(search_kwargs={"k": 3})

# 创建 QA 链
qa_chain = RetrievalQA.from_chain_type(
    llm=get_llm(),
    retriever=retriever,
    return_source_documents=True,
)

# 回答问题
result = qa_chain.invoke({"query": "公司的休假政策是什么?"})
print(result["result"])
print("来源:", result["source_documents"])
```

### 场景 3: 数据分析 Agent

```python
from src.agents.builder import AgentBuilder
from langchain_core.tools import Tool

# 自定义工具
def query_database(query: str) -> str:
    """查询数据库"""
    # 实现数据库查询逻辑
    return "查询结果..."

tools = [
    Tool(
        name="database_query",
        func=query_database,
        description="查询业务数据库。输入 SQL 查询语句。"
    ),
]

# 创建数据分析 Agent
agent = AgentBuilder.create_basic_agent(tools=tools)

result = agent.invoke({
    "input": "上个月销售额最高的产品是什么?"
})
```

## 🔐 安全配置

### JWT 认证

所有请求需要在 Header 中携带 JWT Token:

```python
from fastapi import Depends, HTTPException
from jose import jwt

def verify_token(token: str = Header(...)):
    try:
        payload = jwt.decode(token, config.jwt_secret, algorithms=["HS256"])
        return payload
    except:
        raise HTTPException(status_code=401, detail="Invalid token")
```

### TLS/mTLS 配置

编辑 `.env`:

```env
GRPC_USE_TLS=true
GRPC_CERT_FILE=/etc/certs/ai-service/server.crt
GRPC_KEY_FILE=/etc/certs/ai-service/server.key
GRPC_CA_FILE=/etc/certs/ca.crt
```

## 📊 性能优化

### 1. 连接池

```python
# Redis 连接池
redis_pool = redis.ConnectionPool(
    host=config.redis_host,
    port=config.redis_port,
    max_connections=config.redis_pool_size,
)
```

### 2. 缓存策略

```python
from functools import lru_cache

@lru_cache(maxsize=100)
def get_cached_response(question: str) -> str:
    """缓存常见问题的回答"""
    # ...
```

### 3. 异步处理

```python
async def async_chat(message: str):
    """异步对话处理"""
    chain = get_conversation_chain()
    response = await chain.ainvoke({"input": message})
    return response
```

## 🧪 测试

```bash
# 运行测试
pytest tests/ -v

# 测试覆盖率
pytest --cov=src tests/
```

## 🐛 故障排查

### 问题 1: OpenAI API 调用失败

**症状:** `OpenAIError: Invalid API key`

**解决:**
1. 检查 `.env` 中的 `AI_API_KEY` 是否正确
2. 确认账户余额充足
3. 检查网络连接

### 问题 2: Redis 连接失败

**症状:** `redis.exceptions.ConnectionError`

**解决:**
```bash
# 检查 Redis 是否运行
redis-cli ping

# 启动 Redis
redis-server
```

### 问题 3: FAISS 索引加载失败

**症状:** `OSError: Unable to load FAISS index`

**解决:**
```bash
# 删除损坏的索引文件
rm -rf ./data/vectors/*

# 重启服务会自动重建
```

## 📈 监控与日志

### 结构化日志

```python
import structlog

logger = structlog.get_logger()

logger.info("chat_request_received", 
    conversation_id="conv_123",
    message_length=len(message)
)
```

### Prometheus 指标 (可选)

```python
from prometheus_fastapi_instrumentator import Instrumentator

Instrumentator().instrument(app).expose(app)
```

访问 `http://localhost:8003/metrics` 查看指标。

## 🔄 后续优化方向

1. **多模态支持**: 集成 GPT-4V 处理图片
2. **函数调用**: 利用 OpenAI Function Calling
3. **Agent 编排**: 使用 LangGraph 构建复杂工作流
4. **评估系统**: 集成 LangSmith 进行质量监控
5. **私有模型**: 支持本地部署的 Llama/Mistral

## 📝 开发规范

1. **类型提示**: 所有函数必须添加类型注解
2. **文档字符串**: 使用 Google 风格 docstring
3. **错误处理**: 使用 try-except 捕获异常并记录日志
4. **配置管理**: 通过 Pydantic Settings 管理配置
5. **依赖注入**: 使用 FastAPI Depends 进行依赖管理

## 🤝 贡献指南

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 📄 License

MIT License

---

**技术支持**: 如有问题请提 Issue 或联系开发团队
