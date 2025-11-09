# 推送服务系统

基于用户画像的内容推荐和推送服务系统，分析用户数据生成画像，并根据画像搜索知识库推送相关内容。

## 功能特点

1. **用户画像生成**：
   - 分析用户社区发帖和群聊消息
   - 生成带权重的关键词标签
   - 支持实时和定时生成
   - 存在则更新，不存在则创建

2. **推荐内容生成**：
   - 基于用户画像关键词搜索知识库
   - 按关键词权重优先级搜索
   - 支持实时和定时生成
   - 存在则更新，不存在则创建

3. **智能推送系统**：
   - HTTP接口推送到第三方服务器
   - 统一推送流程，避免重复推送
   - 智能群发机制：对无推荐内容的用户发送热门话题
   - 推送状态记录和错误处理

4. **日志系统**：
   - 统一的日志记录
   - 多级别和多输出目标支持
   - 详细的推送过程跟踪

## 系统架构

### 核心流程
系统采用**三步统一流程**设计：
```
1. 用户画像生成 → 2. 推荐内容生成 → 3. 内容推送
```

- **正常模式**：每天0点执行完整流程
- **Debug模式**：按配置间隔执行完整流程

### 模块架构
- **配置管理**：`config`模块负责加载和管理配置
- **数据库操作**：`repository`模块负责数据库操作
- **服务层**：`services`模块提供核心业务逻辑
- **API接口**：`handlers`模块提供HTTP接口
- **定时任务**：`scheduler`模块负责定时任务调度
- **日志系统**：`logger`模块提供统一的日志记录
- **工具函数**：`utils`模块提供通用工具函数

## 接口说明

### 用户画像接口
- `POST /api/profile/generate`：生成所有用户画像
- `POST /api/profile/generate/{cid}`：生成指定用户画像
- `GET /api/profile/check/{cid}`：验证用户画像

### 推荐内容接口
- `POST /api/recommendation/generate`：为所有用户生成推荐
- `POST /api/recommendation/generate/{cid}`：为指定用户生成推荐
- `GET /api/recommendation/{cid}`：获取用户推荐内容
- `POST /api/recommendation/refresh/{cid}`：强制刷新用户推荐内容

### 推送接口
- `POST /api/push/user/{cid}`：为指定用户推送
- `POST /api/push/all`：为所有用户推送

## 特性功能

### Debug模式
- **正常模式**：每天0点执行完整推荐流程（画像生成 → 推荐生成 → 推送）
- **Debug模式**：可配置任意时间间隔（秒）执行完整流程
- **一键切换**：通过修改`debug.enabled`即可在开发和生产环境间切换

### 智能推送系统
- **统一流程**：移除重复推送任务，所有推送统一在完整流程中处理
- **并发推送**：支持可配置的并发推送，显著提高推送性能
- **智能群发**：对于无推荐内容的用户，自动发送一条热门话题群发消息（cid=""）
- **性能优化**：使用数据库查询直接判断是否需要群发，避免不必要的遍历

### 备用推荐策略
- **智能降级**：当基于画像的推荐内容为空时，自动使用群聊热门话题
- **时效性保证**：只获取前一天的热门话题数据，确保内容新鲜度
- **统一处理**：所有推荐内容生成场景都遵循相同的空结果处理逻辑

## 配置说明

系统配置支持配置文件和环境变量两种方式：
- `config.yaml`：主要配置
- `.env`：敏感信息（数据库密码、API密钥）

### 关键配置项

**Debug模式配置**：
```yaml
debug:
  enabled: false              # debug模式开关，true=按间隔执行，false=每天0点执行
  recommendation_freq: 30     # debug模式下完整流程执行频率（秒）
```

**推送配置**：
```yaml
external_api:
  tag_push_url: "http://example.com/api/push"  # 第三方推送接口
  api_key: "${EXTERNAL_API_KEY}"              # API密钥

cron:
  lookback_days: 30           # 回溯天数
  profile_hour: 0             # 每天生成画像的小时（0-23）
  profile_min: 0              # 每天生成画像的分钟（0-59）
  concurrency: 10             # 用户画像生成并发数
  push_concurrency: 5         # 推送并发数，避免对第三方服务器造成过大压力
```

**日志配置**：
```yaml
log:
  level: "debug"              # 日志级别
  format: "json"              # 日志格式
  output: "both"              # 输出目标（stdout/file/both）
  file_path: "logs/ai_push_message.log"
```

## 运行方式

```bash
# 编译
go build -o ai_push_message

# 运行
./ai_push_message
```

### 开发环境（Debug模式）
```bash
# 1. 修改config.yaml启用debug模式
# debug:
#   enabled: true
#   recommendation_freq: 30  # 30秒执行一次完整流程（画像生成 → 推荐生成 → 推送）

# 2. 编译和运行
go build -o ai_push_message
./ai_push_message

 