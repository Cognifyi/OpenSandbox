# OpenSandbox Browser Sandbox 部署文档

本文档描述 OpenSandbox Browser Sandbox 的部署步骤，包括开发环境设置、Docker 镜像构建、服务配置、EvoCrawl API 集成以及 OverlayFS 快照优化。

## 目录

- [前置要求](#前置要求)
- [开发环境设置](#开发环境设置)
- [Phase 1: Docker 镜像构建](#phase-1-docker-镜像构建)
- [Phase 2: OpenSandbox Server 部署](#phase-2-opensandbox-server-部署)
- [Phase 3: EvoCrawl API 集成](#phase-3-evocrawl-api-集成)
- [Phase 4: OverlayFS 快照优化](#phase-4-overlayfs-快照优化)
- [监控与维护](#监控与维护)
- [故障排查](#故障排查)

## 前置要求

### 系统要求

- **操作系统**: Linux (Ubuntu 20.04+ / CentOS 8+)
- **Docker**: 20.10+
- **Docker Compose**: 2.0+
- **内存**: 最少 4GB，推荐 8GB+
- **磁盘**: 最少 20GB 可用空间

### 依赖服务

- **OpenSandbox Server**: 用于管理 sandbox 生命周期
- **EvoCrawl API**: 用于浏览器会话管理
- **PostgreSQL/Supabase**: 用于会话持久化（EvoCrawl）

## 开发环境设置

### 前置条件

- **Python 3.10+**
- **uv** - Python 包管理器 ([安装指南](https://github.com/astral-sh/uv))
- **Docker** - 用于运行 sandboxes

### 快速设置

```bash
# 1. 导航到 server 目录
cd /home/user/code/labs/OpenSandbox/server

# 2. 安装依赖
uv sync

# 3. 复制示例配置文件
cp server/opensandbox_server/examples/browser.config.toml ~/.sandbox.toml

# 4. 编辑配置文件
vim ~/.sandbox.toml
```

### 配置文件说明

配置文件位置：`~/.sandbox.toml`

**关键配置项**：

```toml
[server]
host = "0.0.0.0"
port = 8080
api_key = "your-secret-api-key"  # 生产环境必须设置

[log]
level = "INFO"  # 开发环境使用 "DEBUG"

[runtime]
type = "docker"
execd_image = "opensandbox/execd:latest"

[docker]
network_mode = "bridge"  # 或 "host"

[browser]
enable_overlayfs_snapshots = true  # 默认启用 OverlayFS
```

### 运行开发服务器

```bash
cd /home/user/code/labs/OpenSandbox/server
uv run python -m opensandbox_server.main
```

更多开发细节请参考 [server/DEVELOPMENT.md](server/DEVELOPMENT.md) 和 [CONTRIBUTING.md](CONTRIBUTING.md)。

## Phase 1: Docker 镜像构建

### 1.1 构建 browser-cdp 镜像

```bash
cd /home/user/code/labs/OpenSandbox/sandboxes/browser-cdp

# 赋予构建脚本执行权限
chmod +x build.sh

# 构建镜像（默认 latest 标签）
./build.sh

# 或指定标签
TAG=v1.0.0 ./build.sh
```

### 1.2 镜像说明

构建完成后将生成两个镜像：

- `opensandbox/browser-cdp-base:latest` - 基础镜像（包含 Chromium、Playwright、Puppeteer）
- `opensandbox/browser-cdp:latest` - 最终镜像（集成 execd）

### 1.3 推送到镜像仓库

```bash
# 推送到 Docker Hub
docker push opensandbox/browser-cdp:latest
docker push opensandbox/browser-cdp-base:latest

# 或推送到私有仓库
docker tag opensandbox/browser-cdp:latest your-registry.com/opensandbox/browser-cdp:latest
docker push your-registry.com/opensandbox/browser-cdp:latest
```

## Phase 2: OpenSandbox Server 部署

### 2.1 使用 Browser 专用配置

我们提供了预配置的 browser 专用配置文件，可直接使用：

```bash
# 复制 browser 专用配置
cp /home/user/code/labs/OpenSandbox/server/opensandbox_server/examples/browser.config.toml ~/.sandbox.toml

# 启动服务器
cd /home/user/code/labs/OpenSandbox/server
uv run python -m opensandbox_server.main
```

**browser.config.toml** 已包含以下预配置：
- Docker 运行时配置
- browser-cdp 镜像支持
- OverlayFS 快照优化（默认启用）
- Bridge 网络模式

### 2.2 生产环境部署

#### 使用 systemd

创建 systemd 服务文件：

```bash
sudo vim /etc/systemd/system/opensandbox.service
```

`/etc/systemd/system/opensandbox.service`:

```ini
[Unit]
Description=OpenSandbox Lifecycle Server
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
User=opensandbox
WorkingDirectory=/opt/opensandbox/server
Environment="PATH=/opt/opensandbox/.venv/bin"
ExecStart=/opt/opensandbox/.venv/bin/python -m opensandbox_server.main
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

启动服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable opensandbox
sudo systemctl start opensandbox
sudo systemctl status opensandbox
```

#### 使用 Docker Compose

创建 `docker-compose.yml`:

```yaml
version: '3.8'

services:
  opensandbox-server:
    build:
      context: ./server
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ~/.sandbox.toml:/root/.sandbox.toml:ro
    environment:
      - SANDBOX_CONFIG_PATH=/root/.sandbox.toml
    restart: unless-stopped
    depends_on:
      - docker
```

## Phase 3: EvoCrawl API 集成

### 3.1 配置环境变量

编辑 EvoCrawl 的 `.env` 文件：

```bash
# Browser Service Configuration
BROWSER_SERVICE_URL=http://opensandbox-server:8080
BROWSER_SERVICE_TYPE=opensandbox  # 关键配置：启用 OpenSandbox 类型
BROWSER_SERVICE_API_KEY=your-api-key  # 可选
```

**注意**：OverlayFS 优化由 OpenSandbox Server 自动注入，无需在 EvoCrawl 中额外配置。当 Server 检测到使用 `browser-cdp` 镜像时，会自动注入 `ENABLE_OVERLAYFS_SNAPSHOTS=true` 环境变量。

### 3.2 验证集成

```bash
# 重启 EvoCrawl API
cd /home/user/code/labs/evocrawl/apps/api
npm run dev

# 测试浏览器创建
curl -X POST http://localhost:3002/api/v2/browser \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "ttl": 600,
    "activityTtl": 300
  }'
```

预期响应：

```json
{
  "success": true,
  "id": "session-id",
  "cdpUrl": "ws://host:port/proxy/9222",
  "liveViewUrl": "",
  "interactiveLiveViewUrl": "",
  "expiresAt": "2026-04-27T00:00:00Z"
}
```

## Phase 4: OverlayFS 快照优化

### 4.1 OverlayFS 自动启用

**重要**：OverlayFS 快照功能在 browser 专用配置中**默认启用**。

当 OpenSandbox Server 检测到使用 `browser-cdp` 镜像时，会自动注入 `ENABLE_OVERLAYFS_SNAPSHOTS=true` 环境变量。

**配置方式**：

在 `~/.sandbox.toml` 中配置：

```toml
[browser]
enable_overlayfs_snapshots = true  # 默认为 true
```

**禁用方式**（不推荐）：

```toml
[browser]
enable_overlayfs_snapshots = false
```

### 4.2 Docker 特权模式要求

OverlayFS 需要容器以特权模式运行。OpenSandbox Server 会自动处理此配置。

**手动运行时需要添加**：

```yaml
# Docker Compose
services:
  browser-cdp:
    image: opensandbox/browser-cdp:latest
    privileged: true  # OverlayFS 需要
    volumes:
      - /var/lib/overlay:/var/lib/overlay
```

```bash
# Docker Run
docker run -d \
  --name browser-cdp \
  --privileged \
  -v /var/lib/overlay:/var/lib/overlay \
  opensandbox/browser-cdp:latest
```

### 4.3 快照管理 API

启用 OverlayFS 后，可使用以下 API 管理快照：

**创建快照**：
```bash
curl -X POST http://localhost:8080/browser/snapshot \
  -H "Content-Type: application/json" \
  -d '{"name": "base-snapshot"}'
```

**回滚快照**：
```bash
curl -X POST http://localhost:8080/browser/snapshot/rollback/base-snapshot
```

**列出快照**：
```bash
curl http://localhost:8080/browser/snapshot
```

### 4.4 性能目标

- 冷启动（无快照）：3-5秒
- 热启动（有快照）：1-2秒
- 回滚启动：< 1秒

## 监控与维护

### 5.1 健康检查

```bash
# 检查 OpenSandbox Server
curl http://localhost:8080/ping

# 检查 execd
curl http://localhost:8080/metrics

# 检查浏览器会话
curl http://localhost:8080/browser
```

### 5.2 日志查看

```bash
# OpenSandbox Server 日志
sudo journalctl -u opensandbox -f

# Docker 容器日志
docker logs -f browser-cdp

# execd 日志
docker exec browser-cdp journalctl -u execd -f
```

### 5.3 资源监控

```bash
# CPU/内存使用
docker stats browser-cdp

# 磁盘使用（OverlayFS）
du -sh /var/lib/overlay
```

### 5.4 清理策略

**定期清理过期快照**：
```bash
# 删除 7 天前的快照
find /var/lib/overlay/snapshots -type d -mtime +7 -exec rm -rf {} \;
```

**清理未使用的浏览器会话**：
```bash
# 通过 EvoCrawl API 清理
curl -X DELETE http://localhost:3002/api/v2/browser/{sessionId}
```

## 故障排查

### 6.1 浏览器启动失败

**症状**：浏览器会话创建失败

**排查步骤**：
```bash
# 检查 Chromium 是否安装
docker exec browser-cdp which chromium-browser

# 检查 CDP 端口是否被占用
docker exec browser-cdp netstat -tlnp | grep 9222

# 查看浏览器启动日志
docker exec browser-cdp cat /tmp/browser-*.log
```

### 6.2 OverlayFS 挂载失败

**症状**：`ENABLE_OVERLAYFS_SNAPSHOTS=true` 但无法挂载

**排查步骤**：
```bash
# 检查 privileged 模式
docker inspect browser-cdp | grep Privileged

# 检查内核支持
uname -r  # 需要 3.18+
modprobe overlay

# 检查目录权限
ls -la /var/lib/overlay
```

### 6.3 CDP 连接失败

**症状**：无法连接到 CDP WebSocket

**排查步骤**：
```bash
# 检查 execd 代理中间件
curl http://localhost:8080/proxy/9222

# 检查防火墙规则
sudo iptables -L -n | grep 9222

# 测试 CDP 端点
wscat -c ws://localhost:8080/proxy/9222
```

### 6.4 内存不足

**症状**：容器 OOM Killed

**解决方案**：
```yaml
# Docker Compose 增加内存限制
services:
  browser-cdp:
    deploy:
      resources:
        limits:
          memory: 4Gi
```

## 安全建议

### 7.1 网络隔离

```yaml
# 使用 Docker 网络
networks:
  browser-network:
    driver: bridge
    internal: false  # 允许外部访问
```

### 7.2 访问控制

```bash
# 配置 EXECD_ACCESS_TOKEN
docker run -e EXECD_ACCESS_TOKEN=your-secret-token ...

# 在 EvoCrawl 中配置
BROWSER_SERVICE_API_KEY=your-secret-token
```

### 7.3 资源限制

```yaml
# 限制 CPU 和内存
deploy:
  resources:
    limits:
      cpus: '2'
      memory: 4Gi
    reservations:
      cpus: '1'
      memory: 2Gi
```

## 性能调优

### 8.1 并发优化

```bash
# 增加 Docker 守护进程并发
sudo vim /etc/docker/daemon.json
```

```json
{
  "max-concurrent-downloads": 10,
  "max-concurrent-uploads": 10
}
```

### 8.2 缓存优化

```bash
# 使用 BuildKit 缓存
DOCKER_BUILDKIT=1 docker build ...

# 或在 build.sh 中启用
export DOCKER_BUILDKIT=1
```

### 8.3 镜像优化

```dockerfile
# 使用多阶段构建减少镜像大小
# 在 Dockerfile_base 中清理不必要的文件
RUN apt-get autoremove -y && rm -rf /var/lib/apt/lists/*
```

## 附录

### A. 环境变量完整列表

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `EXECD_PORT` | 8080 | execd HTTP 端口 |
| `EXECD_ACCESS_TOKEN` | (空) | execd 访问令牌 |
| `BROWSER_LAUNCH_ARGS` | (预设) | Chromium 启动参数 |
| `BROWSER_CDP_PORT` | 0 | CDP 端口（0=随机） |
| `ENABLE_OVERLAYFS_SNAPSHOTS` | false | 启用 OverlayFS 快照 |
| `OVERLAY_BASE_DIR` | /var/lib/overlay | OverlayFS 基础目录 |

### B. API 端点列表

| 端点 | 方法 | 说明 |
|------|------|------|
| `/browser` | POST | 创建浏览器会话 |
| `/browser/kill/:sessionId` | DELETE | 销毁浏览器会话 |
| `/browser` | GET | 列出浏览器会话 |
| `/browser/snapshot` | POST | 创建快照（可选） |
| `/browser/snapshot/rollback/:name` | POST | 回滚快照（可选） |
| `/browser/snapshot` | GET | 列出快照（可选） |

### C. 相关文档

- [OpenSandbox 架构设计](../cogni-plan/4.功能设计/OpenSandbox-OverlayFS快照与回滚架构设计-v1.md)
- [OpenSandbox README](README.md)
- [execd 开发文档](components/execd/DEVELOPMENT.md)

---

**文档版本**: v1.0  
**最后更新**: 2026-04-26  
**维护者**: SuperAIHuman Labs
