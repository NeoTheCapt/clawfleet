# Clawfleet 部署/排障 Checklist（防止“代码有但线上看不到”的坑）

> 目标：遇到“某功能线上没有/不生效”时，**先确认运行态产物**，再看源码。
> 这份清单优先用于：Vue 前端（web/dist）+ Go 后端（二进制）+ Nginx 反代 的部署模式。

---

## 0. 先定性：这次是怎么部署的？

- **A. Git/CI 部署**：服务器从 git 拉代码构建/发布
- **B. 本地 build 手工部署**：本地构建 `web/dist`、`clawfleet-server`，再 scp/rsync 到服务器

> 如果是 **B**，那么“是否 git commit/push”不应成为阻塞点；关键是**服务器是否拿到了最新产物**。

---

## 1. 先画清楚请求链路（10 秒）

典型链路：

1) 用户浏览器 → `https://fleet.example.com`
2) Nginx（可能在 Docker 容器里）→ 反代到 `host.docker.internal:8090`
3) `clawfleet-server`（systemd）监听 `:8090`
4) `clawfleet-server` 静态文件目录：**`web/dist/`**（SPA）

如果链路不清楚，后面很容易在“错误层面”排查（例如纠结前端代码，但实际上是旧 dist）。

---

## 2. 最快定责：先看“线上产物”有没有（前端）

在服务器上确认 `web/dist` 是否真的包含该功能（比看源码更快）：

- 查页面入口：
  - `ls -la /data/clawfleet/web/dist`
  - `sed -n '1,120p' /data/clawfleet/web/dist/index.html`

- grep 关键字符串（推荐）：
  - `grep -R --line-number "🤖 Bots" /data/clawfleet/web/dist/assets | head`
  - `grep -R --line-number "/api/companies/" /data/clawfleet/web/dist/assets | grep bots | head`

判定：
- grep 为空 → **线上静态产物没更新**（优先修部署）
- grep 命中 → 前端产物 OK，继续查后端 API

---

## 3. 再看“线上二进制”有没有（后端 API）

### 3.1 确认实际运行的二进制路径

- `systemctl status clawfleet-cp --no-pager -l`
- `ps aux | grep clawfleet-server | grep -v grep`
- `ss -lntp | grep 8090`

> 确保你替换/重启的是**真正被 systemd 使用的那个二进制**。

### 3.2 API 探测必须带鉴权（JWT Cookie）

很多接口是 JWT 保护的，直接 curl 会得到 `unauthorized`，这不是服务坏了。

流程：
1) `POST /api/login` 获取 `Set-Cookie: token=...`
2) curl 其它接口时带上 `Cookie: token=...`

判定：
- `/api/companies/:id/bots` 404 → **后端二进制没包含 bots 路由**（需要更新 server 二进制）
- 200 但返回空/null → API 存在，继续检查 DB/migration/数据

---

## 4. Nginx/反代验证（避免“我访问的不是它”）

如果 Nginx 在容器里，且按 Host 分流，建议用 Host 头测试：

- `curl -k https://127.0.0.1/ -H "Host: fleet.example.com" | head`
- `curl -k https://127.0.0.1/api/health -H "Host: fleet.example.com"`

判定：
- Host 头正确时返回正常 → 反代链路 OK
- Host 头不正确/返回空 → 说明你测到的是 default_server/别的站点

---

## 5. 部署动作的标准化（建议）

### 5.1 前端

- 本地：`cd web && npm run build`
- 部署：**确保覆盖的是服务器的 `web/dist/`**
  - 服务器上存在 `web/assets` 和 `web/dist` 时，务必记住：Go 服务端默认只 serve `web/dist/`

### 5.2 后端

- 本地：`go build -o clawfleet-server ./cmd/server`
- 部署：覆盖到服务器 systemd 配置里的 `ExecStart` 指向路径，例如：
  - `/data/clawfleet/bin/clawfleet-server`
- 然后：`systemctl restart clawfleet-cp`

---

## 6. 事故复盘结论（这次 Bots 管理）

- 误区：先纠结“代码/是否 commit”，而实际部署是“本地 build 手工部署”
- 真因：
  1) 线上 `web/dist` 是旧 build（因此前端没有 Bots tab）
  2) 即便更新前端，线上后端二进制也没更新（因此 `/api/.../bots` 404）
- 正解：用 2/3/4 步快速定责：**先 grep dist，再测鉴权 API，再确认二进制路径**
