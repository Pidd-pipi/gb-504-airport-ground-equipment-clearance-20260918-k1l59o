请生成 `airport-ground-equipment-clearance`「机场地面设备安全放行」Go 全栈项目，面向机场地勤公司管理机坪车辆、航班周转阶段、检查项和安全放行决定。不要做航班售票、预约、订单或通用工单系统。

## 项目主要需求

复杂度下限：核心实体不少于 3 个、核心页面不少于 4 个、横切关注点不少于 2 个、共享前端组件不少于 3 个、自定义 hooks/utils 不少于 2 个、后端中间件不少于 2 个。

### 核心实体

`Turnaround`（航班周转阶段）、`GroundUnit`（地面设备及状态）、`SafetyCheck`（检查项与证据）、`ClearanceDecision`（放行/限制/撤销）贯穿数据库、Go model/service/handler 和前端 API/store/page。

### 核心页面

`/turnarounds` 周转看板；`/ground-units` 设备状态；`/checks` 检查工作台；`/clearance` 放行审核；`/audit` 审计。`RiskBadge` 在周转和检查页共用，`ClearancePanel` 在检查和放行页共用。

### 横切关注点

JWT/RBAC 需要同步角色表、后端认证授权中间件、前端路由守卫和按钮权限；安全决定审计需要保存状态迁移和操作证据；实现全局错误处理、请求追踪和限流。

### 共享枚举/组件

共享 `UnitState`（available/inspection/blocked/retired）与 `ClearanceState`（pending/cleared/restricted/revoked）。共享 `StatusBadge`、`EvidenceList`、`ConfirmDialog`，hooks 为 `useAuth`、`usePagination`。

### 技术与规模要求

前端 Angular 17 + TypeScript + Vite；后端 Go 1.22 + Gin + GORM；PostgreSQL + Redis。目标 2700–3900 行、26–38 个 `.go` 文件，至少 20 个功能 Go 文件。

### 文件结构强制清单

前端必须拆 `api/stores/types/components/common/hooks/pages/router/utils`；后端必须拆 `model/dto/repository/service/handler/router/middleware/constants/util`，禁止把多个实体写进一个文件。

### 结构红线

严禁合并职责到单一文件；每个实体的 model、service、handler、route 和前端模块独立存放。

### 部署与交付

根目录必须提供 `docker-compose.yml`（顶层 `name: airport-ground-equipment-clearance`，且不写 `version:`）、`.env` 和 `.env.example`（均含 `COMPOSE_PROJECT_NAME=airport-ground-equipment-clearance`）、`README.md`、`frontend/Dockerfile`、`backend/Dockerfile` 和 `frontend/nginx.conf`。前端端口 `18504`、后端端口 `19504`；Nginx 只代理 `/api` 到 `backend:8080`，数据库健康检查、命名卷和 `condition: service_healthy` 依赖必须齐全，并提供真实 `/healthz` 和 Git 初始化。
