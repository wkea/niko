# Niko

<div align="center">

**把大模型放进你的终端和聊天软件 —— 一个静态二进制，装完就能用。**

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![Linux](https://img.shields.io/badge/Linux-amd64%20%7C%20arm64%20%7C%20armv7-informational?style=flat-square&logo=linux&logoColor=white)](#-安装两条命令)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)
[![No Runtime](https://img.shields.io/badge/runtime-none%20needed-brightgreen?style=flat-square)](#这是什么)

[一键安装](#-安装) · [三分钟上手](#-三分钟上手) · [命令速查](#-命令速查) · [接入聊天软件](#-接入-telegramdiscordslack飞书) · [配置文件](#-配置文件) · [常驻运行](#-常驻运行systemd) · [常见问题](#-常见问题)

</div>

---

## 这是什么

**Niko 是一个用 Go 写的个人 AI 助手。** 它只有一个可执行文件，没有前端页面、没有 Docker、没有 Node/Python 运行时，也不需要图形界面。

你既可以在终端里跟它一问一答，也可以让它常驻后台，通过 Telegram、Discord、飞书、企业微信、QQ 等 **20 个聊天频道** 随时找它，让它在你的机器上读写文件、执行命令、查网页、跑定时任务。

它来自开源项目 [PicoClaw](https://github.com/sipeed/picoclaw)（MIT），本仓库是它的**精简分支**：删掉了 Web 控制台、Docker 编排、CI 与前端构建链，只保留 CLI 与常驻网关；同时去掉了工作区路径与命令执行的白名单限制，行为更直接（详见 [与 PicoClaw 的差异](#-与-picoclaw-的差异)）。

### 一分钟看懂

| 特点 | 说明 |
| --- | --- |
| **单文件部署** | 一个静态二进制，`scp` 上去就能跑；树莓派、VPS、旧安卓手机、嵌入式板卡都行 |
| **三种用法** | `niko -m "..."` 一问一答 · `niko` 交互对话 · `niko gateway` 常驻接频道 |
| **任意模型** | 一条命令接入任意 OpenAI 兼容端点，也内置 OpenAI / Anthropic / DeepSeek / 通义 / Kimi / 智谱 / 火山 / OpenRouter / 本地 Ollama 等配置 |
| **20 个频道** | Telegram、Discord、Slack、飞书、企业微信、微信、QQ、钉钉、Matrix、IRC、LINE、OneBot、MQTT、Teams、VK、WhatsApp、DeltaChat、MaixCAM、Pico |
| **能干活的工具** | 读写文件、执行命令、网页搜索与抓取、定时任务、子 Agent、图片/文件收发 |
| **可扩展** | 接入 MCP 服务、安装 Skill 技能包、Hooks 钩子、长期记忆与工作区人格设定 |

---

## 📦 安装

预编译包只提供 **Linux**（Niko 的目标场景就是 Linux 服务器 / 板卡 / 手机）。Windows 与 macOS 用一行 `go build`，见下方。

### 方式一：一键脚本（推荐）

```bash
curl -fsSL https://raw.githubusercontent.com/wkea/niko/main/install.sh | sh
```

它会自动判断架构、从 Releases 下载对应包、装进 PATH，并在首次运行时初始化配置（已有配置不会被覆盖；重复执行即升级）。

```bash
# 完全手动版（不想 curl | sh 的话）
curl -fsSL -O https://raw.githubusercontent.com/wkea/niko/main/install.sh
sh install.sh
```

常用参数：

| 参数 / 变量 | 作用 |
| --- | --- |
| `--dir DIR` | 安装目录（默认 `/usr/local/bin`，不可写时用 `~/.local/bin`） |
| `--version v0.1.0` | 指定版本，默认取最新 Release |
| `--skip-onboard` | 只装程序，不初始化配置 |
| `NIKO_BASE_URL` | 下载源前缀，国内可指向镜像站 |

### 方式二：手动安装

#### 第一步：确认你的架构

```bash
uname -m
```

| `uname -m` 输出 | 下载哪个包 | 典型设备 |
| --- | --- | --- |
| `x86_64` | `niko-linux-amd64.tar.gz` | 普通 VPS、迷你主机、NAS |
| `aarch64` / `arm64` | `niko-linux-arm64.tar.gz` | 树莓派 3/4/5（64 位系统）、ARM 服务器 |
| `armv7l` | `niko-linux-armv7.tar.gz` | 树莓派 Zero 2 W（32 位系统）、老 ARM 板卡 |

#### 第二步：下载、解压、放进 PATH

以 x86_64 为例（其它架构只需替换包名）：

```bash
# 1. 下载
curl -fL -o niko.tar.gz \
  https://github.com/wkea/niko/releases/latest/download/niko-linux-amd64.tar.gz

# 2. 解压
tar xzf niko.tar.gz

# 3. 安装到 PATH（需要 sudo；也可以放到 ~/.local/bin）
sudo install -m 755 niko /usr/local/bin/niko

# 4. 验证
niko version
```

看到版本信息就装好了。**没有其它依赖**，不需要 Go、Node、Python、数据库。

<details>
<summary>没有 <code>curl</code>？用 <code>wget</code></summary>

```bash
wget -O niko.tar.gz \
  https://github.com/wkea/niko/releases/latest/download/niko-linux-amd64.tar.gz
tar xzf niko.tar.gz
sudo install -m 755 niko /usr/local/bin/niko
```

</details>

<details>
<summary>没有 sudo / 想装在用户目录</summary>

```bash
mkdir -p ~/.local/bin
install -m 755 niko ~/.local/bin/niko
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc && source ~/.bashrc
```

</details>

<details>
<summary>升级怎么做</summary>

重跑一次一键脚本（或重新执行上面的「下载 + 解压 + `sudo install`」）即可。配置和记忆都在 `~/.picoclaw`，不会被覆盖。

> ⚠️ **不要用 `niko update`**：这个自更新命令仍指向上游 `sipeed/picoclaw` 的发布页，会把你的 Niko 替换成上游版本。

</details>

### Windows / macOS：一行编译

需要 [Go](https://go.dev/dl/) 1.25 或更新版本：

```bash
git clone https://github.com/wkea/niko.git
cd niko
go build -o niko ./cmd/picoclaw        # Windows 用: go build -o niko.exe ./cmd/picoclaw
./niko version
```

<details>
<summary>想要带版本号的正式构建（推荐）</summary>

```bash
make build          # 产物: build/picoclaw（Windows: build/picoclaw.exe）
make install        # 装到 ~/.local/bin/niko（仅 Linux/macOS）
```

</details>

### 安卓手机（Termux）

旧手机可以当一台 7×24 小时的助手主机：

```bash
pkg install curl
curl -fL -o niko.tar.gz \
  https://github.com/wkea/niko/releases/latest/download/niko-linux-arm64.tar.gz
tar xzf niko.tar.gz && chmod +x niko
./niko onboard
./niko -m "你好，跑在安卓上"
```

> 若遇到系统目录兼容问题，`pkg install proot` 后用 `termux-chroot ./niko` 运行。更多细节见 [Android Termux 指南](docs/guides/android-termux.md)。

---

## 🚀 三分钟上手

### 1. 初始化

```bash
niko onboard
```

它会在 `~/.picoclaw/` 下生成 `config.json` 和 `workspace/`（人格设定、记忆、技能模板）。

### 2. 接一个模型（一条命令）

```bash
niko model add \
  -b https://api.deepseek.com \
  -k sk-你的DeepSeekKey \
  -m deepseek-flash \
  -n deepseek-flash
```

参数含义：`-b` 接口地址 · `-k` API Key · `-m` 模型 ID（**省略则会列出该端点所有模型让你选**）· `-n` 本地别名。

常见端点：

| 服务 | `-b` 接口地址 | `-m` 示例 |
| --- | --- | --- |
| DeepSeek | `https://api.deepseek.com` | `deepseek-flash` |
| OpenAI | `https://api.openai.com/v1` | `gpt-5.4` |
| OpenRouter | `https://openrouter.ai/api/v1` | 省略 `-m` 交互选择 |
| 智谱 GLM | `https://open.bigmodel.cn/api/paas/v4` | `glm-4.7` |
| 通义千问 | `https://dashscope.aliyuncs.com/compatible-mode/v1` | `qwen-plus` |
| Kimi | `https://api.moonshot.cn/v1` | `moonshot-v1-8k` |
| 本地 Ollama | `http://localhost:11434/v1` | `llama3`（`-k` 随便填） |

> 原生协议（Anthropic、Gemini 直连等）与更多厂商参数见 [模型配置](docs/guides/providers.zh.md)。

### 3. 开始对话

```bash
niko -m "用一句话介绍你自己"      # 一问一答
niko                              # 进入交互模式，Ctrl+C / Ctrl+D 退出
```

### 4. 看看状态

```bash
niko status     # 配置路径、工作区、当前模型
niko model      # 当前默认模型 + 配置里所有可用模型
```

### 5. 让它动手干活

```bash
niko -m "看下当前目录有几个 go 文件，并把数量写进 count.txt"
niko -m "查一下 Go 1.25 的发布时间，存到 notes.md"
```

<details>
<summary>更想手动改配置文件？</summary>

`~/.picoclaw/config.json`（敏感字段会自动分离到同目录的 `.security.yml`）：

```json
{
  "version": 3,
  "agents": {
    "defaults": {
      "model_name": "deepseek-flash"
    }
  },
  "model_list": [
    {
      "model_name": "deepseek-flash",
      "provider": "deepseek",
      "model": "deepseek-flash",
      "api_keys": ["sk-你的Key"]
    }
  ]
}
```

- `model_list` 里每个条目 = 一个可调用模型；`model_name` 是本地别名，`agents.defaults.model_name` 指向当前默认模型。
- 完整的字段模板（所有工具、频道、音色、硬件配置）见 [config/config.example.json](config/config.example.json)。
- 不要用记事本式的富文本编辑器改；保存后可用 `niko status` 校验是否能加载。

</details>

---

## ⌨️ 命令速查

| 命令 | 作用 |
| --- | --- |
| `niko` | 交互式对话 |
| `niko -m "问题"` | 一问一答 |
| `niko -s <会话名>` | 指定会话（默认 `cli:default`） |
| `niko --model <别名>` | 临时换模型 |
| `niko -d` | 调试日志 |
| `niko onboard` | 初始化配置与工作区（别名 `o`） |
| `niko onboard --enc` | 初始化并开启 API Key 加密（生成密钥，需输入口令） |
| `niko model` | 查看默认模型与模型列表 |
| `niko model <别名>` | 切换默认模型 |
| `niko model add -b … -k …` | 接入任意 OpenAI 兼容端点 |
| `niko status` | 配置/工作区/模型状态（别名 `s`） |
| `niko gateway` | 常驻运行，接入聊天频道（别名 `g`） |
| `niko gateway -d` | 常驻 + 调试日志 |
| `niko gateway --host 0.0.0.0` | 临时改监听地址 |
| `niko auth login openai` | OAuth 登录（支持 `openai` / `anthropic` / `antigravity`） |
| `niko cron add -n 名字 -c "0 9 * * *" -m "内容"` | 新增定时任务（也支持 `-e 秒数`） |
| `niko cron list` / `enable` / `disable` / `remove` | 管理定时任务 |
| `niko mcp add <名字> <命令或URL>` / `list` / `test` / `remove` | 管理 MCP 服务 |
| `niko skills list` / `install` / `search` | 查看与安装技能包 |
| `niko migrate --from openclaw` | 从其它 claw 类助手迁移配置 |
| `niko config reset -f` | 重置为出厂配置 |
| `niko version` | 版本信息（别名 `v`） |

---

## 💬 接入 Telegram、Discord、Slack、飞书

终端聊天只是开胃菜。让 `niko gateway` 常驻，它就会把消息总线接上各种聊天软件。

### 以 Telegram 为例（5 步）

1. 在 Telegram 里找 **@BotFather** → `/newbot` → 拿到 Bot Token（形如 `123456:ABC...`）
2. 找 **@userinfobot** 发一条消息，拿到你的数字 ID（用来限制谁能用这个 bot）
3. 编辑 `~/.picoclaw/config.json`，启用频道：

   ```json
   {
     "channel_list": {
       "telegram": {
         "enabled": true,
         "type": "telegram",
         "allow_from": ["你的数字ID"],
         "settings": {
           "token": "你的BotToken"
         }
       }
     }
   }
   ```

4. 启动网关：

   ```bash
   niko gateway
   ```

5. 在 Telegram 里私聊你的 bot，直接说话即可。

> 🔒 `allow_from` 留空等于**所有人都能调用你的 bot**。上线前一定填上自己的 ID（或显式写 `["*"]` 表示你确实要开放）。

<details>
<summary>把它挂到后台 / 看日志</summary>

```bash
nohup niko gateway >> ~/.picoclaw/gateway.out 2>&1 &
tail -f ~/.picoclaw/logs/gateway.log
```

长期运行建议用下面的 systemd 方案。

</details>

### 其它频道

频道配置都在 `config.json` 的 `channel_list` 下，结构一致：`enabled` + `type` + `settings`。

| 频道 | `type` | 需要填的 `settings` |
| --- | --- | --- |
| Discord | `discord` | `token`（Bot Token） |
| Slack | `slack` | `bot_token` + `app_token`（Socket Mode） |
| 飞书 / Lark | `feishu` | `app_id` + `app_secret`（`is_lark` 区分国际版） |
| 企业微信 | `wecom` | `bot_id` + `secret` |
| 微信 | `weixin` | 运行 `niko auth weixin` 扫码登录 |
| QQ | `qq` | `app_id` + `app_secret` |
| 钉钉 | `dingtalk` | `client_id` + `client_secret` |
| Matrix | `matrix` | `homeserver` + `user_id` + `access_token` |
| IRC | `irc` | 服务器地址与昵称 |
| LINE | `line` | `channel_secret` + `channel_access_token` |
| OneBot（QQ 机器人框架） | `onebot` | `ws_url`（如 `ws://127.0.0.1:3001`） |
| MQTT | `mqtt` | broker 地址与凭据 |
| Teams / Slack 群机器人 | `teams_webhook` / `slack_webhook` | `webhooks` 列表 |
| MaixCAM | `maixcam` | 摄像头推流地址 |
| WhatsApp | `whatsapp` | `bridge_url`（或编译 `whatsapp_native` 免桥接） |
| DeltaChat / VK | `deltachat` / `vk` | 对应账号凭据 |

逐频道的开通步骤见 [聊天软件接入](docs/guides/chat-apps.zh.md)。

### 网关参数与端口

| 项 | 默认 | 说明 |
| --- | --- | --- |
| 监听地址 | `localhost` | 来自 `gateway.host`，可被 `--host` 或 `PICOCLAW_GATEWAY_HOST` 覆盖 |
| 端口 | `18790` | 来自 `gateway.port` |
| 日志级别 | `warn` | `gateway.log_level`：`debug`/`info`/`warn`/`error`/`fatal` |
| 健康检查 | — | `GET http://localhost:18790/health`、`/ready`；`POST /reload` 热重载 |
| 日志文件 | `~/.picoclaw/logs/gateway.log` | 崩溃日志 `gateway_panic.log` |

> 只有把 `host` 改成 `0.0.0.0` 或使用 `--host 0.0.0.0`，服务才会对外监听。放在公网前请务必配置好 `allow_from`，或套一层反向代理 + 鉴权。

---

## 🔁 常驻运行（systemd）

让网关开机自启、崩了自动拉起：

```bash
sudo tee /etc/systemd/system/niko.service >/dev/null <<'EOF'
[Unit]
Description=Niko personal AI assistant gateway
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=你的用户名
WorkingDirectory=/home/你的用户名
Environment=PICOCLAW_HOME=/home/你的用户名/.picoclaw
ExecStart=/usr/local/bin/niko gateway
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now niko
sudo systemctl status niko
```

常用操作：

```bash
journalctl -u niko -f            # 实时日志
tail -f ~/.picoclaw/logs/gateway.log
sudo systemctl restart niko      # 改完配置重启
```

---

## ⚙️ 配置文件

### 配置放在哪

| 环境变量 | 作用 | 默认 |
| --- | --- | --- |
| `PICOCLAW_HOME` | 数据根目录 | `~/.picoclaw` |
| `PICOCLAW_CONFIG` | 直接指定配置文件路径 | `$PICOCLAW_HOME/config.json` |
| `PICOCLAW_GATEWAY_HOST` | 网关监听地址 | `gateway.host` |
| `PICOCLAW_LOG_LEVEL` | 日志级别 | `gateway.log_level` |
| `TZ` | 时区，影响 cron 与时间戳 | 系统时区 |

```bash
# 示例：把所有数据放到 /opt/niko，配置用另一份
PICOCLAW_HOME=/opt/niko PICOCLAW_CONFIG=/srv/niko/main.json niko gateway
```

### 目录结构

```
~/.picoclaw/
├── config.json          # 主配置（不含密钥）
├── .security.yml        # 密钥等敏感字段，自动写入
├── logs/
│   └── gateway.log
└── workspace/           # Agent 的工作区
    ├── AGENT.md         # 行为准则（可直接编辑，运行中自动生效）
    ├── SOUL.md          # 性格设定
    ├── USER.md          # 你的偏好
    ├── memory/MEMORY.md # 长期记忆
    ├── skills/          # 技能包
    ├── sessions/        # 会话历史
    └── cron/            # 定时任务数据
```

> `AGENT.md` / `SOUL.md` / `USER.md` / `memory/MEMORY.md` 改动会按文件修改时间自动加载，**不用重启网关**。

### 配置项速览

| 顶层键 | 作用 |
| --- | --- |
| `version` | 配置版本（当前 `3`） |
| `agents` | 默认模型、工作区、上下文与重试等参数 |
| `model_list` | 模型清单与凭证 |
| `channel_list` | 聊天频道 |
| `tools` | 各工具开关与参数 |
| `gateway` | 监听地址、端口、日志级别 |
| `heartbeat` | 周期性心跳任务 |
| `voice` / `devices` | 语音、硬件设备 |
| `hooks` / `events` / `evolution` | 钩子、事件、自进化 |

常用字段：

| 字段 | 默认 | 说明 |
| --- | --- | --- |
| `agents.defaults.model_name` | — | 当前默认模型（必填） |
| `agents.defaults.workspace` | `~/.picoclaw/workspace` | 工作区路径 |
| `agents.defaults.max_tokens` | `32768` | 单次回复上限 |
| `agents.defaults.max_tool_iterations` | `50` | 单轮最多工具调用次数 |
| `tools.exec.timeout_seconds` | `60` | 命令执行超时 |
| `tools.exec.allow_remote` | `true` | 是否允许来自聊天频道的命令执行请求；设 `false` 则仅本机 CLI 可用 |
| `tools.web.enabled` | `true` | 网页搜索/抓取 |

---

## ❓ 常见问题

**`niko: command not found`**
没装到 PATH。用绝对路径运行（`/usr/local/bin/niko`），或把所在目录加进 `PATH`。

**提示没有默认模型 / 启动后不回复**
先 `niko model` 看默认模型有没有设；没有就执行一次 `niko model add …`，或手动填 `agents.defaults.model_name`。

**报 401 / 403 / 模型不存在**
`-b` 接口地址与 `-k` Key 不匹配，或 `-m` 模型 ID 写错。省略 `-m` 重新跑一次 `niko model add`，让它列出可用模型。

**端口 18790 被占用**
改 `config.json` 的 `gateway.port`，或 `PICOCLAW_GATEWAY_HOST` 换地址。

**时间和定时任务对不上**
设置 `TZ`，例如 systemd 里加 `Environment=TZ=Asia/Shanghai`。

**网关起来了，但 bot 不理我**
依次检查：`channel_list` 里该频道 `enabled` 是否为 `true`、token 是否正确、你的 ID 是否在 `allow_from` 里、`~/.picoclaw/logs/gateway.log` 有没有报错。

**会不会把我的数据传出去**
配置、记忆、会话都在本地 `~/.picoclaw`。只有对话内容会按你的配置发给你选的模型服务商。

**它能删我的文件吗**
能。本分支**移除了工作区路径限制与命令白名单**，Agent 拥有与你运行它的账户相同的权限。请在虚拟机、容器或专用账户里运行，重要数据做好备份。

---

## 🧬 与 PicoClaw 的差异

| 项 | 上游 PicoClaw | 本仓库（Niko） |
| --- | --- | --- |
| 形态 | CLI + Gateway + Web 控制台 + 桌面启动器 | **CLI + Gateway** |
| 依赖 | 前端需 Node/pnpm；Docker 编排可选 | **仅一个二进制** |
| 工作区限制 | `restrict_to_workspace` 等开关 | 移除，不受限 |
| 命令白名单 | `exec` deny/allow patterns | 移除，不拦截 |
| 根命令 / 安装名 | `picoclaw` | `niko` |
| 默认模型 | 视配置 | `deepseek-flash` |

保留的仍是上游的引擎：Agent 循环、多渠道、多 Provider、MCP、Cron、Skill、Hooks、记忆。上游文档里的配置项在本仓库同样生效，见下方[更多文档](#-更多文档)。

---

## 🛠 开发

```bash
make deps      # 拉依赖
make build     # 构建到 build/picoclaw
make test      # 跑测试
make fmt lint  # 格式化 + 静态检查
make build-all # 交叉编译 linux{amd64,arm,arm64,loong64,riscv64,mipsle} 与 darwin/windows/netbsd
```

| 目录 | 内容 |
| --- | --- |
| `cmd/picoclaw/` | CLI 入口与各子命令（`agent`、`gateway`、`model`、`cron`、`mcp`…） |
| `pkg/agent/` | Agent 循环、会话、子任务 |
| `pkg/providers/` | 各模型协议适配 |
| `pkg/channels/` | 各聊天频道适配 |
| `pkg/tools/` | 工具实现（文件、命令、网页、定时…） |
| `pkg/config/` | 配置结构与迁移 |
| `workspace/` | 内置工作区模板（`onboard` 时释放） |

---

## 📚 更多文档

| 文档 | 内容 |
| --- | --- |
| [配置指南](docs/guides/configuration.zh.md) | 全部配置项、工作区、Session、技能 |
| [模型与提供商](docs/guides/providers.zh.md) | 各家 API 参数、回退与负载均衡 |
| [聊天软件接入](docs/guides/chat-apps.zh.md) | Telegram/Discord/Slack/飞书等逐步开通 |
| [模型路由](docs/guides/routing-guide.zh.md) | 简单问题走便宜模型 |
| [子任务](docs/guides/spawn-tasks.zh.md) | 让 Agent 分派子 Agent |
| [硬件兼容性](docs/guides/hardware-compatibility.zh.md) | 已验证的开发板清单 |
| [Android Termux](docs/guides/android-termux.md) | 旧手机部署 |
| [架构说明](docs/architecture/README.md) | 路由、会话、事件、钩子设计 |

> 部分上游文档仍以 `picoclaw` 作为命令名，把 `picoclaw` 换成 `niko` 即可。

---

## 🙏 致谢与许可

- 上游项目：[sipeed/picoclaw](https://github.com/sipeed/picoclaw)，以及所有 PicoClaw 贡献者。
- 灵感来源：[NanoBot](https://github.com/HKUDS/nanobot)。
- 本项目以 [MIT License](LICENSE) 发布，保留上游版权声明。

<div align="center">

**装上它，然后让 AI 去干活。**

</div>
