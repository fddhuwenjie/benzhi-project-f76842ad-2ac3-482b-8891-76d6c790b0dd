# 环境样品交接异常闭环服务

本项目是面向环境监测样品流转的版本化 HTTP JSON API。服务把样品档案草稿修订与提交预检、逐站及批量保管交接、进度和时限风险查询、温控/封签/时限/路线异常识别、质量调查队列与认领、整改证据补充链复核和关闭归档串成一条可追溯流程。业务快照、交接幂等结果和追加式审计事件保存在嵌入式 SQLite 中。

## 环境要求

- Go 1.23 或更高版本
- Linux CGO 工具链和系统运行库 `libsqlite3.so.0`

项目自带面向 `database/sql` 的最小 SQLite 驱动适配，不需要下载第三方 Go 模块。

## 构建与测试

```bash
go build ./cmd/server
go test ./...
```

## 运行

```bash
go run ./cmd/server -addr=127.0.0.1:19081 -db=sample-chain.db
```

默认监听 `127.0.0.1:19081`，只允许回环地址。`-addr` 优先于 `PORT`；未传 `-addr` 时可将 `PORT` 设置为端口号，服务会监听 `127.0.0.1:<PORT>`。`-db` 指定 SQLite 文件路径，默认是当前目录的 `sample-chain.db`。

运行完整 HTTP 自检并主动退出：

```bash
go run ./cmd/server -selfcheck -addr=127.0.0.1:19081
```

自检会启动真实监听器并完成档案创建、提交、正常交接、异常交接、调查认领、原因登记、证据接受、关闭和审计查询。

## API 约定

除健康检查和只读查询外，业务请求使用 `X-Actor` 和 `X-Role` 表示操作人及角色。角色包括 `custodian`、`receiver`、`quality_reviewer` 和 `responsible_person`。

修改请求通过 `If-Match` 或 JSON 的 `revision` 字段提供预期修订号。交接请求还必须使用 `Idempotency-Key`；同一标识和同一载荷返回原始结果，同一标识配合不同载荷返回 `IDEMPOTENCY_CONFLICT`。所有响应包含 `X-Request-ID`，也可以由客户端通过同名请求头传入关联标识。

主要路由：

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/healthz` | 健康检查 |
| `POST` | `/api/v1/dossiers` | 创建档案草稿 |
| `PATCH` | `/api/v1/dossiers/{dossier_id}` | 使用条件修订号修改草稿并记录字段差异 |
| `POST` | `/api/v1/dossiers/{dossier_id}/submit/preflight` | 汇总提交条件并返回规范化预览 |
| `POST` | `/api/v1/dossiers/{dossier_id}/submit` | 校验并提交冻结档案 |
| `POST` | `/api/v1/dossiers/{dossier_id}/transfers` | 幂等登记交接并识别异常 |
| `POST` | `/api/v1/dossiers/{dossier_id}/transfers/batch` | 原子、幂等地补录连续交接批次 |
| `GET` | `/api/v1/dossiers/{dossier_id}/transfers/progress` | 查询路线进度、当前责任与时限风险 |
| `GET` | `/api/v1/investigations/queue` | 按条件和稳定游标查询调查工作队列 |
| `POST` | `/api/v1/investigations/{investigation_id}/claim` | 认领异常调查 |
| `POST` | `/api/v1/investigations/{investigation_id}/release` | 当前认领人释放尚无结论的调查 |
| `POST` | `/api/v1/investigations/{investigation_id}/conclusion` | 登记原因、影响与处置要求 |
| `POST` | `/api/v1/investigations/{investigation_id}/evidence` | 提交带 SHA-256 摘要及可选前序关联的整改证据 |
| `POST` | `/api/v1/evidence/{evidence_id}/review` | 接受或驳回整改证据 |
| `GET` | `/api/v1/dossiers/{dossier_id}/close/preflight` | 汇总关闭归档条件且不写业务数据 |
| `POST` | `/api/v1/dossiers/{dossier_id}/close` | 重检交接链并关闭归档 |
| `GET` | `/api/v1/dossiers/{dossier_id}` | 查询完整档案视图 |
| `GET` | `/api/v1/dossiers/{dossier_id}/audit` | 查询不可变审计轨迹 |

请求体上限为 1 MiB。单笔交接和批量补录都要求 `Idempotency-Key`，批量中的任一链路阻断会回滚整个批次。调查队列仅供 `quality_reviewer` 查询，页大小为 1 到 100。关闭仅允许在交接链完整、调查已有结论、证据全部复核且最终证据已接受时执行，关闭后的档案拒绝所有业务修改。
