# 机场地面设备安全放行

面向机场地勤公司的航班周转安全控制台。系统把地面设备状态、周转阶段、逐项检查证据、放行决定和审计记录连成一条可追溯业务链，不包含售票、预约、订单或通用工单功能。

## 快速启动

```bash
docker compose up -d --build
```

| 入口 | 地址 |
| --- | --- |
| Angular 控制台 | http://localhost:18504 |
| Gin API | http://localhost:19504/api/v1 |
| 健康检查 | http://localhost:19504/healthz |

预置账号：

根目录开发用 `.env` 显式设置了 `SEED_DEMO_DATA=true`，因此空库会创建以下演示账号。生产环境必须使用 `.env.example` 为基线，保持 `SEED_DEMO_DATA=false`，并通过受控的身份供应流程创建首个管理员。

| 角色 | 手机号 | 密码 |
| --- | --- | --- |
| 管理员 | `13800000001` | `Admin@123` |
| 安全放行员 | `13800000002` | `User@123` |
| 机坪检查员 | `13800000003` | `User@123` |
| 地勤操作员 | `13800000004` | `User@123` |

## 业务能力

- `/turnarounds`：建立航班周转阶段，分配地面设备和首个检查项，跟踪周转状态与风险等级。
- `/ground-units`：登记牵引车、地面电源、传送带等设备，维护 `available / inspection / blocked / retired` 状态。
- `/checks`：逐项记录检查结论、说明和证据；检查复核使用事务锁且结论不可改写，未处理或失败检查会阻断完全放行。
- `/clearance`：形成 `cleared / restricted / revoked` 决定；完全放行同时要求全部关联设备可用，限制放行必须填写运行条件，紧急撤销不受未完成检查阻断。
- `/audit`：查询所有写操作；放行状态迁移额外保存前态、后态、依据、证据和 request id。

JWT 与 RBAC 同时覆盖路由和页面按钮。系统不开放匿名注册，只有管理员能通过受保护的用户接口创建账号和指定角色。管理员、安全放行员可建立周转和形成决定；检查员可提交检查结论并上报设备异常，但不能自行恢复或退役设备；普通操作员拥有只读视图。后端还提供统一错误响应、请求追踪、限流和结构化日志。

## 技术结构

```text
backend/
  cmd/server
  internal/{model,dto,repository,service,handler,router,middleware,constants,util}
frontend/src/
  api/ stores/ types/ components/common/ hooks/ pages/ router/ utils/
database/init.sql
docker-compose.yml
```

- 前端：Angular 17、TypeScript、Angular Material，生产构建使用 Angular application builder。
- 后端：Go 1.22、Gin、GORM、JWT；关键状态迁移、业务数据和审计证据在同一数据库事务中提交。
- 数据：PostgreSQL 16、Redis 7，均配置健康检查和命名卷。
- 部署：Nginx 仅将 `/api` 代理到 `backend:8080`，其余路径回退到 Angular SPA。

四个核心实体均保持独立的 model、repository、service、handler 和 route 文件。`RiskBadge` 同时用于周转和检查页，`ClearancePanel` 同时用于检查和放行页；`StatusBadge`、`EvidenceList`、`ConfirmDialog` 为共享组件。

## API

所有业务接口使用 `/api/v1` 前缀，响应格式为 `{"code":0,"message":"ok","data":...}`。

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| POST | `/auth/login` | 登录并签发 JWT | 公开 |
| GET / PUT | `/users/me` | 当前用户 / 修改姓名 | 登录 |
| GET / POST | `/users` | 用户列表 / 管理员创建账号 | 管理角色 / 管理员 |
| GET / POST | `/ground-units` | 查询 / 登记设备 | 登录 / 管理角色 |
| PATCH | `/ground-units/:id/state` | 设备状态迁移（带版本号） | 管理、检查角色 |
| GET / POST | `/turnarounds` | 查询 / 建立周转 | 登录 / 管理角色 |
| PATCH | `/turnarounds/:id/status` | 周转状态迁移（带版本号） | 管理角色 |
| GET / POST | `/checks` | 查询 / 增加检查项 | 登录 / 检查角色 |
| PATCH | `/checks/:id/review` | 提交结论和证据 | 检查角色 |
| GET / POST | `/clearance` | 查询 / 形成放行决定 | 登录 / 管理角色 |
| GET | `/audit` | 查询审计记录 | 管理角色 |

所有列表接口均为服务端分页，`page_size` 最大为 200；前端分页器不会再截断第 100 条之后的数据。`/healthz` 会真实探测 PostgreSQL 与 Redis，任一依赖不可用时返回 HTTP 503。

登录示例：

```bash
curl -s http://localhost:19504/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"phone":"13800000001","password":"Admin@123"}'
```

## 本地验证

```bash
cd backend
go test ./...
go vet ./...
go build ./...

cd ../frontend
npm ci
npm run build

cd ..
docker compose config --quiet
```

停止并清理演示数据：

```bash
docker compose down -v --remove-orphans
```

## 环境变量

根目录 `.env` 提供本地默认值，`.env.example` 用于部署参考。关键变量包括 `COMPOSE_PROJECT_NAME=airport-ground-equipment-clearance`、`FRONTEND_PORT=18504`、`BACKEND_PORT=19504`、PostgreSQL 连接、Redis 端口、JWT 密钥和每分钟限流阈值。
