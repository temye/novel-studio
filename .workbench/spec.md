---
domain: system
surface: desktop
purpose: 面向个人作者的 AI 小说创作工作台，帮助作者从设定到章节持续推进作品
owner: 独立作者 / 内容创作者
roles:
  - { name: 作者, opens_daily: true, scope: 自己创建的小说项目 }
subject: 小说项目与其章节创作进度
structure:
  primary: pipeline
  secondary: console
dials: { cadence: daily, input: 7, depth: 7 }
hook:
  text: "你的下一步：继续推进《{project.title}》第 {chapter.number} 章"
  shape: imperative
  fields:
    - { name: project.title, reads: 当前项目名称, writes: user, when: 创建或编辑项目时 }
    - { name: chapter.number, reads: 当前待处理章节, writes: system, when: 根据章节状态自动计算 }
    - { name: ai_task.status, reads: AI 任务状态, writes: system, when: 创建或完成生成任务时 }
cold_start:
  day_1: "先创建一本小说，AI 会从一句话梗概开始帮你搭出第一版世界观。"
  no_project: "还没有小说项目，创建第一本就能开始。"
  no_chapter: "项目已经建立，下一步创建第一章大纲。"
home:
  - hook
  - 当前项目卡片与创作进度
  - 待处理章节列表
  - AI 任务状态
  - 最近编辑记录
entities:
  - name: Project
    fields: [id, title, summary, genre, target_words, current_words, status, created_at, updated_at]
    written_by: { title: user, summary: user, genre: user, target_words: user, current_words: system, status: system, created_at: system, updated_at: system }
    relations: [Project 1-n Chapter, Project 1-n AITask]
  - name: Chapter
    fields: [id, project_id, number, title, outline, content, word_count, status, updated_at]
    written_by: { project_id: system, number: system, title: user, outline: user, content: user, word_count: system, status: user, updated_at: system }
    relations: [Chapter n-1 Project]
  - name: AITask
    fields: [id, project_id, type, status, progress, result, created_at, completed_at]
    written_by: { project_id: system, type: user, status: system, progress: system, result: system, created_at: system, completed_at: system }
    relations: [AITask n-1 Project]
channels:
  - name: 项目总览
    type: today
    weight: primary
    does: 查看创作进度、待处理章节和 AI 任务
    pages:
      - { level: L1, shows: 项目进度卡片 + 待办章节列表 + AI 状态, actions: [新建小说, 继续写作] }
  - name: 世界设定
    type: record
    weight: regular
    does: 管理世界观、角色和故事基础资料
    pages:
      - { level: L1, shows: 设定分类卡片, actions: [新增设定] }
  - name: 剧情大纲
    type: pipeline
    weight: regular
    does: 按卷和章节规划故事推进
    pages:
      - { level: L1, shows: 章节规划看板, actions: [新增章节, AI 生成大纲] }
      - { level: L2, shows: 单章大纲详情, actions: [编辑, 生成正文] }
  - name: 正文写作
    type: tool
    weight: regular
    does: 编辑正文并调用 AI 续写、改写和润色
    pages:
      - { level: L1, shows: 富文本编辑器 + AI 操作栏, actions: [保存, 续写, 润色] }
  - name: 分析中心
    type: review
    weight: occasional
    does: 查看字数、章节完成度和写作趋势
    pages:
      - { level: L1, shows: 指标卡片 + 趋势图, actions: [刷新分析] }
mvp: [登录注册, 项目总览, 新建小说, 项目列表, 世界设定, 角色管理, 地点管理, 道具管理, 章节列表与状态, AI 生成任务演示, Docker 部署]
later: [真实 AI Provider, 富文本编辑器, 一致性分析, 导出发布, PostgreSQL 持久化]
visual: "克制的米白灰工作台，深墨色文字，蓝绿色进度强调；卡片清晰但不堆渐变，章节状态用细色条区分。"
depends_on:
  - { field: AI 生成结果, source: OpenAI 兼容模型服务, exists_today: false, until_then: 使用任务接口和演示状态，不在首屏伪造生成结果 }
seam: { type: none, why: 第一阶段是创作核心闭环，暂不加入付费入口 }
excluded: [多人协作, 付费订阅, 移动端专属界面]
deferred: [PostgreSQL 持久化, Redis 队列, MinIO 附件存储]
---

## 这个台子是给谁的

这是给独立作者使用的桌面网页工作台，围绕“小说项目与章节创作进度”组织。作者每天打开后，最先看到当前作品推进到哪里、下一章要做什么，以及 AI 任务是否完成。

## 每天怎么用

作者登录后进入项目总览，点击待处理章节继续写作，或新建一本小说。第一阶段先完成项目建立和章节状态管理；后续 AI 会接入世界观、章节大纲和正文生成。

## 为什么是这几个频道

- 项目总览：每天打开的默认入口，集中显示进度和下一步。
- 世界设定：承载角色、地点和规则，第一阶段先留出位置。
- 剧情大纲：让章节从列表升级为可推进的创作流程。
- 正文写作：最终创作落点，第一阶段预留接口。
- 分析中心：帮助作者回看产量和完成度，后续实现。

## 已经想过但没做的

第一阶段不加入多人协作、付费和移动端专属体验；第二阶段完成小说资料和章节规划的内存 CRUD，第三阶段加入了演示 AI 任务和 SSE 状态流，真实模型、数据库与队列继续后置。

## 给实现方（不熟悉本规范的 AI 或开发，照这段做即可）

- 这是一个电脑浏览器里的网页工作台，不是 App。首屏最上面要明确告诉作者下一步做什么。
- 首屏按 `home` 顺序排列：项目进度、待处理章节、AI 任务状态、最近活动。
- 没有项目时显示“先创建一本小说，AI 会从一句话梗概开始帮你搭出第一版世界观。”，并提供“新建小说”按钮，不显示空白表格。
- `channels` 是左侧导航；`primary` 的项目总览是默认落地页，并在视觉上明显突出。
- `pipeline` 的核心形态是可推进的章节看板/列表，不要把所有模块做成相同的表格。
- 数据库按 `entities` 创建，`written_by: system` 的字段由服务端计算，不能要求作者手工填写。
- 当前实现到第三阶段基础版：世界设定、角色、地点、道具和章节规划均可在工作台内新增与查看，章节状态可切换，并可启动演示 AI 大纲/正文任务；`later` 功能先显示为规划中或不放入可点击主流程。
- 视觉使用米白灰底、深墨文字、蓝绿色进度色，保持高信息密度和克制卡片。
