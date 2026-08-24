# MacroCanvas

HID 语义宏编排内核 + Vue 3 拖拽画布。开发端口：前端 `31821`，后端诊断 `31822`。

## 1. 如何启动

```bash
docker compose up --build -d
```

浏览器打开 `http://localhost:31821`。账号 `geek` / `phosphor`。

## 2. 使用说明

登录后打开内置 P10 样例。可从左侧拖入节点、连线、热下发后点「执行」。紧急停止独立于宏路径。首次看事件墙须点「授权捕获」并确认其键盘记录语义。时序数字是分档承诺，不是无条件微秒保证。

## 3. 服务列表及API说明

- 用户界面：`http://localhost:31821`
- 后端诊断：`http://localhost:31822/health`
- API 详情：`docs/API.md`
- WebSocket：同源 `/ws/events?token=`

## 4. 测试账号

- 用户名：`geek`
- 密码：`phosphor`
- Token：`mc-dev-31821`

## 5. 题目内容

用 Go 实现基于 HID 语义的键鼠宏可编程内核、串行精密队列与全栈编排画布，Web 热下发，Linux 走 evdev/uinput 内核路径。

## 6. 项目结构

`backend/` Go 内核与 API；`frontend-user/` Vue 3 画布；`tests/` 冒烟与 E2E；`docs/` SSOT。

## 7. API 模拟与切换指南

设备档由环境变量 `DEVICE_MODE` 选择，**真实内核路径已接线**：

| 值 | 行为 |
|---|---|
| `auto`（默认） | 能打开 `/dev/uinput` 则 T-B 真内核虚拟设备；否则降级 T-C 用户态回环 |
| `real` | 必须 T-B，失败则进程退出 |
| `mock` | 强制 T-C，供 CI / 无特权。QA 可用 `DEVICE_MODE=mock docker compose up --build -d` |

T-B 经 `UI_DEV_CREATE` 注册真实 `input_dev`（`BUS_USB`），事件从 `/dev/input/eventN` 读回。T-C 仅用于无特权与测试。可选 `HID_GADGET_ENABLED=true` 打开 `/dev/hidg0`（默认关）。Windows/macOS 宿主拦截是 V2 `mc-agent`（Cgo），不进默认镜像。
