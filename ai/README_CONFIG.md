# AI Service Configuration Guide

## 📋 配置管理

AI 服务使用 `.env` 文件管理所有配置项，符合 Python 项目的最佳实践。

### 🔧 配置文件说明

| 文件 | 用途 | 是否提交到 Git |
|------|------|--------------|
| `.env.example` | 配置模板，包含所有配置项及注释 | ✅ 是 |
| `.env` | 实际配置文件，包含敏感信息 | ❌ 否 |

---

## 🚀 快速开始

### **步骤 1：复制配置模板**

```bash
cd ai
cp .env.example .env
```

### **步骤 2：编辑配置文件**

打开 `.env` 文件，修改以下关键配置：

```bash
# ⚠️ 必须修改的配置
AI_API_KEY=sk-your-actual-openai-api-key-here  # 替换为你的 OpenAI API Key

# 可选修改（根据环境调整）
DB_HOST=localhost        # 数据库地址
DB_PASSWORD=123456       # 数据库密码
REDIS_HOST=localhost     # Redis 地址
JWT_SECRET=your-secret   # JWT 密钥
```

### **步骤 3：安装依赖**

```bash
pip install -r requirements.txt
```

### **步骤 4：验证配置**

```bash
python config.py
```

如果输出 `✅ Configuration loaded successfully!`，说明配置正确。

---

## 📝 配置项详解

### **1. 服务配置**

```env
SERVICE_NAME=ai-service
SERVICE_HOST=localhost
SERVICE_PORT=8003
GRPC_PORT=9003
SERVICE_VERSION=1.0.0
```

---

### **2. 数据库配置（MySQL）**

```env
DB_DRIVER=mysql
DB_HOST=localhost
DB_PORT=3306
DB_USERNAME=root
DB_PASSWORD=123456
DB_NAME=ai_db
DB_MAX_IDLE_CONNS=10
DB_MAX_OPEN_CONNS=100
DB_CONN_MAX_LIFETIME=3600
```

---

### **3. Redis 配置**

**单节点模式**：
```env
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_POOL_SIZE=100
REDIS_CLUSTER_ADDRESSES=
```

**集群模式**：
```env
REDIS_CLUSTER_ADDRESSES=host1:6379,host2:6379,host3:6379
```

---

### **4. JWT 配置**

```env
JWT_SECRET=g0-s3cr3t-k3y-for-jwt-t0k3n-auth3nt1c4t10n-2024
JWT_EXPIRE=24h
```

⚠️ **安全提示**：生产环境必须使用强随机字符串作为 `JWT_SECRET`。

---

### **5. Consul 配置**

**单节点模式**：
```env
CONSUL_HOST=localhost
CONSUL_PORT=8500
CONSUL_TOKEN=
CONSUL_SCHEME=http
CONSUL_ADDRESSES=
```

**集群模式**：
```env
CONSUL_ADDRESSES=consul1:8500,consul2:8500,consul3:8500
```

---

### **6. AI 模型配置**

```env
AI_MODEL_PROVIDER=openai
AI_API_KEY=sk-your-actual-openai-api-key-here
AI_MODEL=gpt-3.5-turbo
AI_MAX_TOKENS=2048
AI_TEMPERATURE=0.7
```

**支持的模型提供商**：
- `openai`: OpenAI GPT 系列
- `anthropic`: Anthropic Claude
- `azure`: Azure OpenAI
- `local`: 本地部署的模型（如 Llama）

---

### **7. TLS/mTLS 配置（可选）**

**开发环境（默认）**：
```env
GRPC_USE_TLS=false
```

**生产环境（双向 mTLS）**：
```env
GRPC_USE_TLS=true
GRPC_INSECURE_SKIP_VERIFY=false
GRPC_CERT_FILE=/etc/certs/ai-service/client.crt
GRPC_KEY_FILE=/etc/certs/ai-service/client.key
GRPC_CA_FILE=/etc/certs/ai-service/ca.crt
GRPC_SERVER_NAME=ai-service
```

📖 **证书路径规范**：`/etc/certs/{service-name}/`

---

## 🔒 安全最佳实践

### **1. 保护敏感信息**

✅ **正确做法**：
```bash
# .env 文件已加入 .gitignore，不会被提交
echo ".env" >> .gitignore
```

❌ **错误做法**：
```bash
# 不要将 .env 提交到版本控制系统
git add .env  # ❌ 禁止！
```

---

### **2. 使用环境变量覆盖**

在生产环境中，建议使用系统环境变量而非 `.env` 文件：

```bash
export AI_API_KEY=sk-prod-key-xxx
export DB_PASSWORD=prod-password-xxx
python main.py
```

优先级：**系统环境变量 > .env 文件 > 默认值**

---

### **3. 定期轮换密钥**

建议每 90 天轮换一次以下密钥：
- `JWT_SECRET`
- `AI_API_KEY`
- `DB_PASSWORD`

---

## 🧪 测试配置

### **测试配置加载**

```bash
python config.py
```

**预期输出**：
```
✅ Configuration loaded successfully!
Service: ai-service
Database: mysql://root:123456@localhost:3306/ai_db
Redis: redis://localhost:6379/0
AI Model: gpt-3.5-turbo
TLS Enabled: False
```

---

### **验证必填配置**

配置加载器会自动验证必填项：
- `JWT_SECRET`
- `AI_API_KEY`（当使用 OpenAI 时）

如果缺少必填项，会抛出 `ValueError` 异常。

---

## 📂 目录结构

```
ai/
├── .env                 # 实际配置（不提交到 Git）
├── .env.example         # 配置模板（提交到 Git）
├── config.py            # 配置加载器
├── requirements.txt     # Python 依赖
└── README.md            # 本文档
```

---

## 🔗 相关文档

- [Gateway TLS 配置指南](../gateway/README.md#-安全通信)
- [微服务安全通信规范](../../SKILL.md)
- [Python dotenv 官方文档](https://pypi.org/project/python-dotenv/)

---

## ❓ 常见问题

### **Q1: 为什么使用 .env 而不是 YAML？**

A: Python 生态系统中，`.env` 文件是管理环境变量的标准方式，具有以下优势：
- ✅ 与 Docker、Kubernetes 等容器化平台天然兼容
- ✅ 支持通过系统环境变量覆盖
- ✅ 避免硬编码敏感信息
- ✅ 广泛的社区支持和工具链

---

### **Q2: 如何在 Docker 中使用？**

A: 在 `docker-compose.yml` 中引用 `.env` 文件：

```yaml
services:
  ai-service:
    build: .
    env_file:
      - .env
    environment:
      - AI_API_KEY=${AI_API_KEY}  # 从 .env 读取
```

---

### **Q3: 如何切换不同环境的配置？**

A: 为每个环境创建独立的 `.env` 文件：

```bash
.env.development   # 开发环境
.env.staging       # 测试环境
.env.production    # 生产环境
```

启动时指定：
```bash
ENV_FILE=.env.production python main.py
```

---

### **Q4: 证书文件应该放在哪里？**

A: 遵循统一规范 `/etc/certs/{service-name}/`：

```bash
sudo mkdir -p /etc/certs/ai-service
sudo cp client.crt client.key ca.crt /etc/certs/ai-service/
sudo chmod 600 /etc/certs/ai-service/*
```

然后在 `.env` 中配置：
```env
GRPC_CERT_FILE=/etc/certs/ai-service/client.crt
GRPC_KEY_FILE=/etc/certs/ai-service/client.key
GRPC_CA_FILE=/etc/certs/ai-service/ca.crt
```
