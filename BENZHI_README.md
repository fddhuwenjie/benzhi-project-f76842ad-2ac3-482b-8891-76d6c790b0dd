# BENZHI_README

## 项目说明
- 项目：benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd
- 项目用途：完整实现环境监测样品监管链异常闭环 HTTP 服务，覆盖档案冻结、逐站交接、规则识别、质量调查、整改证据复核、SQLite 审计持久化和关闭归档。
- Go 工具链：`golang:1.23.0`
- 前端工具链：无

## 项目描述
- 项目名称：环境样品交接异常闭环服务
- 项目概述：面向环境监测样品流转场景的 HTTP API 服务，将样品档案提交、保管交接、异常识别、原因调查、整改证据复核和归档关闭串成一条可追溯的业务流程。
- 核心工作流：采样保管人员创建样品档案并提交冻结基础信息，交接双方逐站登记保管转移；系统发现温控、封签、时限或路线异常后进入待调查状态，质量审核人员记录原因与处置要求，责任人员提交整改证据，审核通过后关闭并归档完整监管链。
- 对外接口：提供版本化 HTTP JSON API，覆盖样品档案、交接记录、异常调查、整改证据、关闭归档和审计查询；服务支持 `-addr=127.0.0.1:<port>`，也支持用 `PORT` 端口号绑定 `127.0.0.1:<PORT>`，默认监听 `127.0.0.1:19081`，并提供可自行结束的 `-selfcheck` 模式。

## 标准构建、运行和测试命令
进入容器后执行：
cd '/app' && GOTOOLCHAIN=local go build ./...
cd /app && GOTOOLCHAIN=local go run ./cmd/server -selfcheck -addr=127.0.0.1:19081
cd '/app' && GOTOOLCHAIN=local go test ./...

## Docker 构建和进入容器
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd-amd64 linux/amd64
./build_benzhi_docker.sh benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd-arm64 linux/arm64
docker run -it benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd-amd64:latest

## 题目验证命令
1. 预期退出码 0：`go test ./...`
2. 预期退出码 0：`go run ./cmd/server -selfcheck -addr=127.0.0.1:19081`
