# 防重放中间件使用指南

## 📋 概述

防重放攻击（Replay Attack）是指攻击者截获合法请求后，重复发送该请求以达到恶意目的。防重放中间件通过以下机制防止此类攻击：

1. **时间戳验证**：确保请求在有效时间窗口内
2. **Nonce 唯一性检查**：每个请求必须有唯一的 nonce，且只能使用一次
3. **签名验证**（可选）：使用 HMAC-SHA256 验证请求完整性

## 🔧 配置说明

### 配置文件 (`etc/gateway.yaml`)

```yaml
# Anti-Replay Configuration (防重放配置)
anti_replay:
  enabled: false                    # 是否启用防重放（开发环境建议关闭，生产环境启用）
  timestamp_tolerance: 300          # 时间戳容差（秒），默认5分钟
  nonce_cache_size: 10000           # Nonce 缓存大小
  nonce_expire_time: 600            # Nonce 过期时间（秒），默认10分钟
```

### 配置项说明

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `enabled` | bool | false | 是否启用防重放功能。**开发环境建议设为 false**，生产环境设为 true |
| `timestamp_tolerance` | int | 300 | 时间戳容差（秒）。请求时间戳与服务器时间的最大允许偏差 |
| `nonce_cache_size` | int | 10000 | Nonce 缓存的最大容量。超过此数量会自动清理过期的 nonce |
| `nonce_expire_time` | int | 600 | Nonce 的过期时间（秒）。超过此时间的 nonce 会被自动清理 |

## 🚀 客户端实现示例

### Go 客户端

```go
package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
)

// GenerateNonce 生成随机 nonce
func GenerateNonce() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateSignature 生成签名
func GenerateSignature(timestamp string, nonce string, secretKey string) string {
	message := fmt.Sprintf("%s:%s", timestamp, nonce)
	
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(message))
	
	return hex.EncodeToString(h.Sum(nil))
}

func MakeSecureRequest(url string, secretKey string) (*http.Response, error) {
	// 生成时间戳和 nonce
	timestamp := time.Now().Unix()
	nonce, err := GenerateNonce()
	if err != nil {
		return nil, err
	}
	
	timestampStr := fmt.Sprintf("%d", timestamp)
	signature := GenerateSignature(timestampStr, nonce, secretKey)
	
	// 创建请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	// 添加安全请求头
	req.Header.Set("X-Timestamp", timestampStr)
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature", signature)
	
	// 发送请求
	client := &http.Client{}
	return client.Do(req)
}
```

### JavaScript/TypeScript 客户端

```typescript
import crypto from 'crypto';

interface SecureRequestOptions {
  url: string;
  method?: string;
  secretKey: string;
  body?: any;
}

/**
 * 生成随机 nonce
 */
function generateNonce(): string {
  return crypto.randomBytes(16).toString('hex');
}

/**
 * 生成 HMAC-SHA256 签名
 */
function generateSignature(timestamp: string, nonce: string, secretKey: string): string {
  const message = `${timestamp}:${nonce}`;
  return crypto
    .createHmac('sha256', secretKey)
    .update(message)
    .digest('hex');
}

/**
 * 发起安全请求（带防重放保护）
 */
async function makeSecureRequest(options: SecureRequestOptions): Promise<Response> {
  const { url, method = 'GET', secretKey, body } = options;
  
  // 生成时间戳和 nonce
  const timestamp = Math.floor(Date.now() / 1000).toString();
  const nonce = generateNonce();
  const signature = generateSignature(timestamp, nonce, secretKey);
  
  // 构建请求头
  const headers: Record<string, string> = {
    'X-Timestamp': timestamp,
    'X-Nonce': nonce,
    'X-Signature': signature,
    'Content-Type': 'application/json',
  };
  
  // 发起请求
  const response = await fetch(url, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });
  
  return response;
}

// 使用示例
const result = await makeSecureRequest({
  url: 'https://api.example.com/api/v1/user/profile',
  method: 'POST',
  secretKey: 'your-secret-key',
  body: { name: 'John' },
});
```

### Python 客户端

```python
import hmac
import hashlib
import time
import secrets
import requests

def generate_nonce():
    """生成随机 nonce"""
    return secrets.token_hex(16)

def generate_signature(timestamp, nonce, secret_key):
    """生成 HMAC-SHA256 签名"""
    message = f"{timestamp}:{nonce}"
    signature = hmac.new(
        secret_key.encode('utf-8'),
        message.encode('utf-8'),
        hashlib.sha256
    ).hexdigest()
    return signature

def make_secure_request(url, secret_key, method='GET', data=None):
    """发起安全请求（带防重放保护）"""
    # 生成时间戳和 nonce
    timestamp = str(int(time.time()))
    nonce = generate_nonce()
    signature = generate_signature(timestamp, nonce, secret_key)
    
    # 构建请求头
    headers = {
        'X-Timestamp': timestamp,
        'X-Nonce': nonce,
        'X-Signature': signature,
        'Content-Type': 'application/json',
    }
    
    # 发起请求
    if method.upper() == 'GET':
        response = requests.get(url, headers=headers)
    elif method.upper() == 'POST':
        response = requests.post(url, headers=headers, json=data)
    else:
        raise ValueError(f"Unsupported method: {method}")
    
    return response

# 使用示例
response = make_secure_request(
    url='https://api.example.com/api/v1/user/profile',
    secret_key='your-secret-key',
    method='POST',
    data={'name': 'John'}
)
print(response.json())
```

## 🧪 测试示例

### cURL 测试

```bash
# 1. 生成时间戳
TIMESTAMP=$(date +%s)

# 2. 生成随机 nonce
NONCE=$(openssl rand -hex 16)

# 3. 生成签名（假设密钥为 "test-secret-key"）
SIGNATURE=$(echo -n "${TIMESTAMP}:${NONCE}" | openssl dgst -sha256 -hmac "test-secret-key" | awk '{print $NF}')

# 4. 发起请求
curl -X GET "http://localhost:8080/api/v1/user/profile" \
  -H "X-Timestamp: ${TIMESTAMP}" \
  -H "X-Nonce: ${NONCE}" \
  -H "X-Signature: ${SIGNATURE}" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Postman 测试

1. 在 Pre-request Script 中添加：

```javascript
// 生成时间戳
const timestamp = Math.floor(Date.now() / 1000).toString();

// 生成随机 nonce
const nonce = require('crypto').randomBytes(16).toString('hex');

// 生成签名（需要设置环境变量 SECRET_KEY）
const message = `${timestamp}:${nonce}`;
const signature = CryptoJS.HmacSHA256(message, pm.environment.get("SECRET_KEY")).toString(CryptoJS.enc.Hex);

// 设置请求头
pm.request.headers.add({
    key: 'X-Timestamp',
    value: timestamp
});
pm.request.headers.add({
    key: 'X-Nonce',
    value: nonce
});
pm.request.headers.add({
    key: 'X-Signature',
    value: signature
});
```

2. 在环境变量中设置 `SECRET_KEY`

## ⚠️ 注意事项

### 1. 开发环境 vs 生产环境

**开发环境：**
```yaml
anti_replay:
  enabled: false  # 关闭防重放，方便调试
```

**生产环境：**
```yaml
anti_replay:
  enabled: true   # 启用防重放，保障安全
  timestamp_tolerance: 300
  nonce_cache_size: 10000
  nonce_expire_time: 600
```

### 2. 时钟同步

- 确保客户端和服务器的时间同步（建议使用 NTP）
- 如果时间偏差过大，请求会被拒绝
- 可通过 `timestamp_tolerance` 调整容差范围

### 3. Nonce 生成策略

- **必须使用加密安全的随机数生成器**
- 推荐长度：至少 16 字节（32 个十六进制字符）
- 禁止使用时间戳、序列号等可预测的值

❌ **错误的 nonce 生成方式：**
```go
nonce := fmt.Sprintf("%d", time.Now().Unix()) // 可预测！
nonce := "request_001"                         // 可预测！
```

✅ **正确的 nonce 生成方式：**
```go
bytes := make([]byte, 16)
rand.Read(bytes)
nonce := hex.EncodeToString(bytes) // 不可预测
```

### 4. 性能考虑

- Nonce 缓存会占用内存（默认 10000 条记录）
- 高并发场景可适当增大 `nonce_cache_size`
- 定期清理过期 nonce（自动进行）

### 5. 签名验证（可选）

当前实现中签名验证是**可选的**。如果需要更强的安全性：

1. 在服务端配置密钥
2. 客户端使用该密钥生成签名
3. 服务端验证签名是否正确

修改 `anti_replay.go` 中的注释部分即可启用。

## 🔍 故障排查

### 问题 1: 请求被拒绝，提示 "Missing required headers"

**原因：** 客户端未发送必需的请求头

**解决：** 确保发送以下三个请求头：
- `X-Timestamp`
- `X-Nonce`
- `X-Signature`

### 问题 2: 请求被拒绝，提示 "Request timestamp expired"

**原因：** 客户端时间与服务器时间偏差过大

**解决：**
1. 同步客户端和服务器时间（使用 NTP）
2. 或增大 `timestamp_tolerance` 配置

### 问题 3: 请求被拒绝，提示 "Duplicate request detected"

**原因：** 相同的 nonce 被重复使用

**解决：**
1. 确保每次请求都生成新的 nonce
2. 检查是否有重试逻辑复用了旧 nonce

### 问题 4: 开发时调试困难

**解决：** 临时关闭防重放功能
```yaml
anti_replay:
  enabled: false
```

## 📊 中间件执行顺序

```
1. Recovery (异常恢复)
2. Tracing (链路追踪)
3. Logger (日志记录)
4. CORS (跨域处理)
5. Anti-Replay (防重放) ← 新增
6. Rate Limit (限流)
7. JWT Auth (认证)
8. Circuit Breaker (熔断)
```

防重放中间件在 **CORS 之后、限流之前** 执行，这样可以：
- ✅ 尽早拦截恶意请求
- ✅ 避免无效请求消耗限流配额
- ✅ 减少后端服务压力

## 🎯 最佳实践

1. **生产环境必须启用**：`enabled: true`
2. **合理设置时间容差**：根据网络延迟调整（通常 5 分钟足够）
3. **监控 nonce 缓存命中率**：如果频繁出现重复 nonce，可能是攻击行为
4. **定期轮换密钥**：如果启用了签名验证
5. **结合 HTTPS 使用**：防止中间人窃取请求信息
6. **记录安全事件**：对失败的防重放检查进行日志记录和告警

---

**最后更新：** 2026-06-02  
**版本：** v1.0.0
