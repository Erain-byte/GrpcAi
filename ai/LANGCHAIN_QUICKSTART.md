# LangChain 快速入门指南

## 🎯 核心概念

LangChain 是一个用于开发 **LLM 应用**的框架,提供以下核心组件:

```
┌─────────────────────────────────────────────┐
│           LangChain 生态系统                 │
├──────────┬──────────┬───────────┬───────────┤
│ Chains   │ Agents   │ Memory    │ Tools     │
│ (链)     │ (智能体) │ (记忆)    │ (工具)    │
├──────────┴──────────┴───────────┴───────────┤
│         LLM + Prompts + Embeddings          │
└─────────────────────────────────────────────┘
```

## 📚 五大核心组件

### 1. **Chains (链)** - 组合多个操作

```python
from langchain.chains import ConversationChain
from src.llm.provider import get_llm

# 最简单的链: Prompt → LLM → Output
llm = get_llm()
chain = ConversationChain(llm=llm)
response = chain.invoke({"input": "你好"})
```

**常见链类型:**
- `ConversationChain`: 对话链 (带记忆)
- `RetrievalQA`: RAG 问答链
- `LLMChain`: 基础链

### 2. **Agents (智能体)** - 自主决策 + 工具调用

```python
from langchain.agents import create_react_agent, AgentExecutor
from langchain_core.tools import Tool

# 定义工具
def search_web(query: str) -> str:
    """搜索网络"""
    return f"搜索结果: {query}"

tools = [
    Tool(
        name="web_search",
        func=search_web,
        description="搜索网络获取最新信息"
    )
]

# 创建 Agent
llm = get_llm()
agent = create_react_agent(llm, tools)
executor = AgentExecutor(agent=agent, tools=tools)

# Agent 自动决定使用哪个工具
result = executor.invoke({
    "input": "2024年最流行的编程语言是什么?"
})
```

**Agent 工作流程:**
```
用户问题 → Agent 思考 → 选择工具 → 执行工具 → 分析结果 → 回答用户
```

### 3. **Memory (记忆)** - 保持对话上下文

```python
from src.memory.manager import memory_manager

# 三种记忆模式:

# 1. Buffer Memory - 保存所有历史
memory = memory_manager.create_buffer_memory("conv_123")

# 2. Summary Memory - 自动总结长对话 (节省 Token)
memory = memory_manager.create_summary_memory(llm, "conv_123")

# 3. Window Memory - 只保留最近 K 条消息
memory = memory_manager.create_window_memory("conv_123", k=10)
```

**Redis 存储示例:**
```json
Key: conversation:conv_123:messages
Value: [
  {"role": "human", "content": "你好"},
  {"role": "ai", "content": "你好!有什么可以帮助你的?"},
  {"role": "human", "content": "介绍一下 Python"},
  {"role": "ai", "content": "Python 是一种..."}
]
```

### 4. **Tools (工具)** - 扩展 LLM 能力

```python
from langchain_core.tools import Tool

# 自定义工具
def calculate(expression: str) -> str:
    """计算数学表达式"""
    try:
        return str(eval(expression))
    except:
        return "计算错误"

tool = Tool(
    name="calculator",
    func=calculate,
    description="计算数学表达式。输入如: '2+2' 或 '10*5'"
)

# 内置工具
from langchain_community.tools import DuckDuckGoSearchRun
search_tool = DuckDuckGoSearchRun()
```

**常用工具:**
- `DuckDuckGoSearchRun`: 网络搜索
- `WikipediaQueryRun`: Wikipedia 查询
- `PythonREPLTool`: Python 代码执行
- 自定义数据库查询工具

### 5. **Vector Stores (向量存储)** - 语义搜索

```python
from src.vectorstores.manager import VectorStoreManager
from langchain_core.documents import Document

# 1. 创建向量库
vector_store = VectorStoreManager.create_vector_store()

# 2. 添加文档
docs = [
    Document(
        page_content="Python 是一种高级编程语言",
        metadata={"source": "wiki"}
    ),
    Document(
        page_content="Java 是一种面向对象的编程语言",
        metadata={"source": "wiki"}
    )
]
VectorStoreManager.add_documents(vector_store, docs)

# 3. 相似度搜索
results = VectorStoreManager.similarity_search(
    vector_store,
    query="什么是 Python?",
    k=2
)

for doc in results:
    print(doc.page_content)
    print(doc.metadata)
```

**支持的向量数据库:**
- FAISS (本地,快速)
- Chroma (轻量级)
- Pinecone (云服务)
- Milvus (大规模)

## 🔄 典型工作流

### 工作流 1: 简单对话

```python
from src.chains.builder import get_conversation_chain

# 创建对话链
chain = get_conversation_chain(conversation_id="user_001")

# 多轮对话
response1 = chain.invoke({"input": "你好"})
print(response1["response"])  # "你好!有什么可以帮助你的?"

response2 = chain.invoke({"input": "我叫张三"})
print(response2["response"])  # "你好张三!很高兴认识你"

response3 = chain.invoke({"input": "我叫什么名字?"})
print(response3["response"])  # "你叫张三" ← 记住上下文!
```

### 工作流 2: RAG 知识库问答

```python
from langchain.chains import RetrievalQA
from src.vectorstores.manager import get_vector_store
from src.llm.provider import get_llm

# 1. 加载向量库
vector_store = get_vector_store()

# 2. 创建检索器
retriever = vector_store.as_retriever(search_kwargs={"k": 3})

# 3. 创建 QA 链
qa_chain = RetrievalQA.from_chain_type(
    llm=get_llm(),
    retriever=retriever,
    return_source_documents=True,
)

# 4. 回答问题
result = qa_chain.invoke({
    "query": "公司的年假政策是什么?"
})

print(result["result"])  # AI 生成的答案
print(result["source_documents"])  # 引用的文档
```

**RAG 流程:**
```
用户问题 → 向量化 → 检索相关文档 → 拼接 Prompt → LLM 生成答案
```

### 工作流 3: Agent 数据分析

```python
from src.agents.builder import AgentBuilder
from langchain_core.tools import Tool

# 定义数据库查询工具
def query_sales_data(question: str) -> str:
    """查询销售数据"""
    # 实际实现中连接数据库
    if "最高" in question:
        return "产品A销售额最高,达到100万"
    return "未找到相关数据"

tools = [
    Tool(
        name="sales_query",
        func=query_sales_data,
        description="查询销售数据。输入自然语言问题。"
    )
]

# 创建 Agent
agent = AgentBuilder.create_basic_agent(tools=tools)

# 执行查询
result = agent.invoke({
    "input": "上个月哪个产品销售额最高?"
})

print(result["output"])
# Agent 会自动:
# 1. 理解问题
# 2. 调用 sales_query 工具
# 3. 分析返回结果
# 4. 生成最终答案
```

## 🚀 实战示例

### 示例 1: 客服机器人

```python
from src.chains.builder import ChainBuilder

# 创建客服链
chain = ChainBuilder.create_conversation_chain(
    conversation_id="customer_001",
    system_prompt="""你是电商客服助手。
规则:
1. 语气友好专业
2. 订单问题引导用户提供订单号
3. 退货政策: 7天无理由退货
4. 不知道的问题转人工客服""",
    window_size=15,  # 保留最近15条消息
)

# 处理客户咨询
questions = [
    "我的订单什么时候发货?",
    "订单号是 ORD123456",
    "可以退货吗?",
]

for q in questions:
    response = chain.invoke({"input": q})
    print(f"Q: {q}")
    print(f"A: {response['response']}\n")
```

### 示例 2: 文档问答系统

```python
from langchain.document_loaders import PyPDFLoader
from langchain.text_splitter import RecursiveCharacterTextSplitter
from src.vectorstores.manager import get_vector_store
from src.embeddings.manager import get_embeddings

# 1. 加载 PDF 文档
loader = PyPDFLoader("company_handbook.pdf")
documents = loader.load()

# 2. 分割文档 (避免超过 LLM 上下文限制)
text_splitter = RecursiveCharacterTextSplitter(
    chunk_size=1000,
    chunk_overlap=200
)
chunks = text_splitter.split_documents(documents)

# 3. 向量化并存储
embeddings = get_embeddings()
vector_store = get_vector_store()
vector_store.add_documents(chunks)

# 4. 创建问答链
from langchain.chains import RetrievalQA
qa_chain = RetrievalQA.from_chain_type(
    llm=get_llm(),
    retriever=vector_store.as_retriever()
)

# 5. 回答问题
answer = qa_chain.invoke({
    "query": "员工请假流程是什么?"
})
print(answer["result"])
```

### 示例 3: 多 Agent 协作

```python
from src.agents.builder import AgentBuilder

# Agent 1: 研究员 (负责搜集信息)
researcher = AgentBuilder.create_search_agent()

# Agent 2: 作家 (负责撰写文章)
writer_chain = ChainBuilder.create_simple_chain(
    system_prompt="你是一个专业的科技文章作家。根据提供的资料撰写通俗易懂的文章。"
)

# 协作流程
# 1. 研究员搜集资料
research_result = researcher.invoke({
    "input": "查找 2024 年 AI 领域的重大突破"
})

# 2. 作家撰写文章
article = writer_chain.invoke({
    "input": f"基于以下资料写一篇文章:\n{research_result['output']}"
})

print(article)
```

## 💡 最佳实践

### 1. 选择合适的记忆模式

| 场景 | 推荐记忆 | 原因 |
|------|---------|------|
| 短对话 (<10轮) | Buffer Memory | 完整保留上下文 |
| 长对话 (>20轮) | Summary Memory | 节省 Token,避免超限 |
| 实时聊天 | Window Memory | 低延迟,只关注最近内容 |

### 2. Prompt 工程技巧

```python
# ❌ 差的 Prompt
prompt = "介绍一下 Python"

# ✅ 好的 Prompt
prompt = """请用简洁的语言介绍 Python 编程语言,包括:
1. 主要特点 (3-5点)
2. 适用场景
3. 学习建议

要求:
- 适合初学者阅读
- 每点不超过50字
- 使用 bullet points 格式"""
```

### 3. 错误处理

```python
from langchain.callbacks import StdOutCallbackHandler

try:
    response = chain.invoke({"input": message})
except Exception as e:
    logger.error(f"Chain execution failed: {e}")
    response = {"response": "抱歉,我遇到了一些问题,请稍后重试。"}
```

### 4. 性能优化

```python
# 使用异步 API
async def async_chat(message: str):
    response = await chain.ainvoke({"input": message})
    return response

# 缓存常见回答
from functools import lru_cache

@lru_cache(maxsize=100)
def get_cached_answer(question: str):
    return chain.invoke({"input": question})
```

## 📖 学习资源

- **官方文档**: https://python.langchain.com/
- **LangSmith**: https://smith.langchain.com/ (调试和监控)
- **LangGraph**: https://langchain-ai.github.io/langgraph/ (复杂工作流)
- **Awesome LangChain**: https://github.com/kyrolabs/awesome-langchain

## 🎓 下一步

1. ✅ 运行测试脚本: `python test_langchain.py`
2. ✅ 启动服务: `python src/main.py`
3. ✅ 访问 API 文档: http://localhost:8003/docs
4. ✅ 尝试示例代码
5. ✅ 阅读完整文档: README.md

---

**祝你在 LangChain 的学习之路上取得成功! 🚀**
