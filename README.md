# Novel Studio

AI 小说创作工作台第一阶段。默认端口：`9008`。

## 本地运行

需要 Go 1.23+：

```bash
cd backend
go run ./cmd/server
```

浏览器打开 <http://localhost:9008>。当前创作数据仍默认保存在 `backend/data/novel-studio.json`；SQLite 迁移目标为 `backend/data/novel-studio.db`，待依赖可用后接入。可通过 `NOVEL_DATA_FILE` 指定 JSON 路径。Docker Compose 会将数据目录挂载到项目根目录的 `data/`，重建容器后仍会保留。首次启动会自动创建演示项目。

## AI 模型配置

复制 `.env.example` 为 `.env`，填写 `AI_BASE_URL`、`AI_API_KEY` 和 `AI_MODEL` 即可使用 OpenAI 兼容接口。未配置密钥时，系统使用内置演示生成器。

## Docker

```bash
docker compose up --build
```

打开 <http://localhost:9008>。

## 当前阶段：第三阶段基础版

- 登录 / 注册演示
- 项目总览
- 创建小说项目
- 项目卡片与创作进度
- 章节基础列表
- 世界设定、角色、地点、道具管理
- 章节规划和状态切换
- 相关内存 CRUD API
- AI 大纲/正文生成任务演示
- AI 任务状态查询与 SSE 流式接口
- 正文版本记录与项目分析接口
- 正文一致性检查和 Markdown/TXT 导出
- 本地 JSON 持久化（SQLite 迁移规划中）
- 存储层预留 PostgreSQL 适配边界
- 项目编辑与级联删除 API（`PUT/PATCH/DELETE /api/projects/:id`）
- Go API 与前端静态壳

SQLite、PostgreSQL、Redis 和 MinIO 将在后续多用户/云端阶段接入；当前数据访问仍基于本地 JSON，迁移时应通过可替换存储层，避免业务逻辑绑定 SQLite 专属语法。
