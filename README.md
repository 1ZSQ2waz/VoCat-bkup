# Vocat

Vocat（代号）是一套面向 Qualcomm 蜂窝模组（首发 **Quectel EC20**）的**高通模块专业测试工具**，用于对自研 / 定制 EC20 外置模组进行功能验证与故障诊断。

它提供一个集中的 Web 测试环境，覆盖 AT 指令、USSD、短信收发检测、Wi-Fi Calling（VoWiFi）能力检测、eSIM 状态与卡策略管理、上游代理与设备绑定等常用功能，适用于开发者、研究人员、学校与实验室在授权测试环境下验证自研硬件是否工作正常。

> **重要声明：** Vocat 是 source-available（源码可见）软件，仅授权用于研究、教育、开发与硬件功能验证。不得用于商业电信服务、未授权网络接入、冒用他人身份或绕过运营商限制。详见 [LICENSE](LICENSE)。

---

## 概述

围绕 Qualcomm 蜂窝模组（如 Quectel EC20）开发定制硬件时，问题可能来自多个层面：

- USB 通信
- SIM 接口走线
- 模组初始化
- 供电稳定性
- 基带通信
- AT 指令通信
- 短信收发检测
- 运营商兼容性
- IMS / Wi-Fi Calling 能力
- eSIM / EID 相关限制

Vocat 为上述功能提供标准化测试环境，帮助判断自研模组行为是否符合预期。

典型场景：

- 测试新组装的 EC20 USB 转接板
- PCB 贴片完成后验证 EC20 通信
- 验证 SIM 接口功能
- 诊断 AT 指令通信问题
- 检测短信收发能力
- 检查基础运营商功能
- 测试实验室蜂窝硬件
- 蜂窝模组行为的教学演示

---

## 支持硬件

Vocat 主要面向 Qualcomm 蜂窝模组。首发目标平台：

- Quectel EC20
- EC20 Mini PCIe 变体
- 基于 EC20 的定制 USB 转接板
- 自研 EC20 核心 / 转接板

其它 Qualcomm 模组若暴露兼容的 modem 接口与 AT 指令功能，也可能可用。未显式列出的硬件不保证兼容。

---

## 功能

### 1. AT 指令检测

验证所连蜂窝模组是否正确响应标准 AT 指令。示例指令：

```text
AT
ATI
AT+CPIN?
AT+CSQ
AT+COPS?
AT+CREG?
AT+CGREG?
```

可帮助识别：

- USB 通信问题
- 串口配置问题
- 模组初始化失败
- SIM 检测问题
- 注册问题
- 固件通信问题

> 出于安全考虑，一组会改变模组射频 / 分组域状态或直接拨号、发卡的指令（如 `+CFUN=`、`+CGATT=`、`+CGACT=`、`+CUSD=`、`+CMGS`、`ATD`、`ATA`、`ATH` 等）在 Web AT 通道被服务端拦截。需要执行这些操作时，请使用 Telegram 机器人或专用的检测端点。

### 2. USSD 检测

测试模组发送与接收 USSD 请求的能力。该功能主要面向开发用 SIM 卡与授权实验室测试环境。可用性取决于：模组固件、SIM 能力、运营商、网络配置与当前注册状态。

### 3. Wi-Fi Calling（VoWiFi）能力检测

提供与 Wi-Fi Calling 能力相关的诊断检测，检查 modem / IMS / SIM / 网络等信息，辅助判断所连模组是否**具备**支持 VoWiFi 的能力。成功的能力检测**不保证**在特定运营商下 VoWiFi 可用——实际可用性还取决于运营商开通、SIM 订阅、IMS 配置、固件、设备认证、网络策略、ePDG 接入与运营商允许名单等。

Vocat 的 VoWiFi 实现支持**上游 SOCKS5 代理**：可配置每个国家 / 区域对应的代理规则，并将设备绑定到指定上游，绑定变更会触发 VoWiFi 重连。上游代理在接入前会进行真实探测（TCP 连接 + SOCKS5 握手 + **UDP Associate** 探测，VoWiFi 依赖 UDP）。

### 4. 短信收发检测

测试所连模组的短信收发能力。检测功能包括：

- 短信能力检测
- 短信存储查看
- 短信发送测试
- 短信接收测试
- modem 短信配置检查

只应使用授权的测试用 SIM 卡。为防止误用，向 `+86` 号段发送短信会被服务端拦截。

### 5. eSIM 与卡策略管理

EID / eSIM 相关验证，面向授权开发与测试环境：

- eSIM 资产盘点：查看本机已写入的 eSIM profile（状态、ICCID、运营商等）
- profile 切换 / 禁用 / 重命名 / 删除
- **eSIM 下载**：通过运营商 websheet 与 GSMA RSP 流程下载 profile（GET + SSE 流式返回下载进度）
- **卡策略（Card Policy）**：按 ICCID 配置 VoWiFi / 飞行模式 / APN / IP 版本等，策略持久化于数据库

结果仅作诊断参考。Vocat 不代表任何移动网络运营商、eSIM 平台、SM-DP+、EUM、GSMA 机构或设备厂商行事。

### 6. 模组信息

在模组支持时，可采集基础 modem 信息：厂商、型号、固件版本、IMEI、SIM 状态、ICCID、IMSI、网络注册状态、服务运营商、信号强度、USB modem 接口。敏感信息只应在授权测试环境下采集。

### 7. 日志与审计

- 实时日志流（SSE）与历史日志查询，支持等级、来源、搜索过滤、自动追尾、暂停、清空与导出
- 日志保留策略可配置：无限制 / 按条数 / 按天数
- 审计事件（auth、config 变更等）落库可查

### 8. 通知

支持 5 个通知渠道：**Telegram、Email、Webhook、Bark、PushPlus**。每个渠道可独立配置与测试连通性。短信到达可触发通知分发（每渠道独立 goroutine）；Webhook 通知附带 HMAC-SHA256 签名与渲染模板。所有外发目标经 SSRF 防护（拦截 localhost、内网、云元数据等）。

### 9. Telegram 机器人

内置 Telegram bot（长轮询），提供指令式交互：查询设备状态、eSIM、切换 profile、VoWiFi 能力检测、短信查看与发送、`/call` 拨号并自动挂断（无语音）。敏感操作有内联键盘二次确认 + 随机令牌 + 2 分钟 TTL。配置在轮询之间热加载。

### 10. 响应式 Web 界面

前端为 React + Vite + Tailwind，支持桌面 / 平板 / 手机多尺寸自适应：手机端日志页控件堆叠 + 横向滚动、设备页列表↔详情互斥切换 + 返回键、主从页容器查询双列布局，以及中英文双语界面。

### 11. 硬件验证流程

可作为自研 EC20 板卡硬件验证流程的一环：

```text
Custom PCB
    ↓
EC20 Module
    ↓
USB Interface
    ↓
Vocat
    ↓
AT / SIM / 短信 / 网络 / VoWiFi / eSIM 诊断
```

PCB 贴片后尤为有用，可帮助判断问题源自硬件、USB 走线、供电、SIM 走线、固件、宿主系统还是运营商侧配置。

---

## 安全与访问控制

Vocat 实现了以下**实际生效**的安全与访问控制机制：

- **认证与会话**：用户名 / 密码登录，`vocat_session`（HttpOnly）+ CSRF 双提交令牌（`vocat_csrf`），SameSite=Strict，`VOCAT_SECURE_COOKIES` 下启用 Secure + HSTS。
- **登录限流**：登录端点限流，防暴力破解。
- **网络访问控制**：可在设置中配置 `internal`（默认，仅 RFC1918 + loopback + link-local + ULA）或 `public` 模式，并维护自定义 CIDR 允许名单；非允许 IP 的请求被 403 拒绝。
- **AT 指令防护**：拦截会修改射频 / 分组域状态或直接拨号发卡的指令。
- **SIM 区域策略**：对 MCC 460 / 461（中国大陆）SIM 卡自动强制飞行模式并写入 `auto_region_block` 卡策略；VoWiFi 启用前亦做同样检查。
- **短信目的端防护**：拦截向 `+86` 号段发送短信。
- **设备数量限制**：单实例最多注册 5 台设备。
- **SSRF 防护**：通知外发目标经地址解析与受限 dialer，拦截内网 / localhost / 云元数据。
- **安全响应头**：X-Content-Type-Options、Referrer-Policy、Permissions-Policy、X-Frame-Options、CSP、HSTS。
- **自更新 SHA256 校验**：CLI 自更新流程对下载的发布产物按 `SHA256SUMS` 校验后再替换二进制。

**关于 LICENSE 中的其它约束：** [LICENSE](LICENSE) 在法律层面对商用、地域、评估期、SIM 授权等作出约束。其中部分条款（如 14 天评估期、仅限美国地域、运行时完整性校验）属于**许可条款**，由用户依约遵守，Vocat 当前未在代码中对应实现强制技术控制；上文列出的均为代码中实际存在并生效的控制。

用户不得故意移除、绕过、禁用、伪装、修补、干扰或破坏上述安全机制。

---

## SIM 与 eSIM 授权策略

Vocat 只应与以下 SIM / eSIM 资源配合使用：

- 测试用 SIM 卡
- 开发用 SIM 卡
- 实验室用 SIM 卡
- 授权的 eSIM profile
- 用户拥有或被明确授权测试的 SIM/eSIM 资源

未经授权不得使用属于他人的生产用订户凭证。Vocat 可拒绝不满足测试策略的 SIM 卡或 eSIM profile。

---

## 禁止用途

Vocat 不得用于：

- 未授权接入电信网络
- 冒用他人订户或设备
- SIM 克隆
- 未授权的 eSIM 开通
- 使用被盗 / 泄露的订户凭证
- 电信欺诈
- 绕过运营商鉴权
- 绕过运营商合法限制
- 未授权拦截 / 监听
- 大规模群发短信
- 干扰移动网络基础设施
- 商业电信服务
- 未经授权出售 Vocat 访问权限
- 绕过 Vocat 安全控制
- 发布以绕过使用限制为主要目的的修改版本

任何使用须遵守适用法律、法规、运营商政策与授权要求。

---

## 商用与衍生

Vocat 面向个人开发、教育、学术研究、学校 / 大学实验室、非营利研究与授权的电信硬件开发。**未经书面授权不得商用。**

受限商用示例：付费模组检测服务、出售 Vocat 托管实例访问、并入商业电信产品、作为商业 SIM 检测平台运营、倒卖修改版本。

Vocat 为 source-available。可在 [LICENSE](LICENSE) 许可范围内为合法研究 / 教育 / 开发 / 调试 / 硬件兼容测试目的检视与修改源码。分支不得以移除或绕过地域限制、SIM 限制、MCC/MNC 限制、设备数量限制、评估期限制、鉴权机制、完整性校验或防滥用控制为主要目的。再发布的允许修改版本须保留版权声明、许可声明、署名与修改说明。

---

## 隐私与数据安全

Vocat 只应部署在用户有授权访问所测 modem 与订户信息的环境中。诊断信息可能包含：IMEI、ICCID、IMSI、EID、运营商信息、模组信息、固件信息、网络注册状态、信号信息、短信测试数据。

部署运维者有责任妥善保护此类信息。**不要**向公网公开暴露包含敏感电信信息的 Vocat 实例。

部署建议：

- 不要将 modem 控制接口直接暴露到公网
- 远程部署使用强认证
- 收紧容器与串口设备权限
- 保护含订户标识的日志
- 不要将凭证提交到 Git 或写入源码
- 定期审计已部署实例

敏感配置应通过环境变量（见下）或密钥管理系统存储。

---

## 运行要求

推荐环境：

- Linux（amd64、386、arm64 或 armv7）
- 对蜂窝模组的 USB 访问
- 受支持的 Qualcomm modem
- 授权的测试 SIM 或 eSIM
- 需要时的网络连接

---

## 安装

Vocat 提供两种部署形态：**二进制 + systemd**（推荐，开箱即用随机初始密码与自更新）与 **Docker**（容器化，适合隔离运行）。Vocat 不随附 `docker-compose.yml` 或 `.env.example`；如需编排或环境文件，请自行创建。

### 方式 1 — 一键安装脚本（二进制 + systemd）

```bash
curl -fsSL <INSTALL_SCRIPT_URL> | sudo bash
```

用官方 install.sh 脚本的 raw 链接替换 `<INSTALL_SCRIPT_URL>`。脚本会：选择语言（中 / 英）、检测架构（amd64 / 386 / arm64 / armv7）、下载二进制与 `SHA256SUMS` 并校验、创建 `vocat` 系统用户、写入 systemd unit、首次安装时生成 32 位随机管理员密码（写入仅一次显示）。详见 [scripts/install.sh](scripts/install.sh)。

安装指定版本：

```bash
sudo bash install.sh 0.1.0
```

强制重装相同版本：

```bash
sudo bash install.sh --force
```

安装完成后，服务默认监听 `0.0.0.0:7575`，浏览器访问 `http://<host>:7575`，用户名 `admin`，首次密码见终端一次性输出。

### 方式 2 — Docker

仓库根目录提供 [Dockerfile](Dockerfile)，多阶段构建：`node:20-alpine` 编译前端 → `golang:1.25-alpine` 交叉编译 Go（含 buildinfo ldflags，并通过 `go:embed` 将前端打进二进制）→ `alpine:3.20` 运行时（非 root `vocat` 用户，uid/gid 1000）。

构建并运行：

```bash
git clone <REPOSITORY_URL>
cd vocat
docker build -t vocat .
docker run -d --name vocat -p 7575:7575 \
  -v vocat-data:/opt/vocat/data \
  --device /dev/ttyUSB0 \
  vocat
```

容器默认值：`VOCAT_ADDR=0.0.0.0:7575`，`VOCAT_DATABASE_PATH=/opt/vocat/data/vocat.db`，`VOLUME /opt/vocat/data`，`EXPOSE 7575`。**默认管理员为 `admin` / `admin`，请登录后立即修改**（Web 设置或 `docker exec ... vocat menu`）。

### USB 设备访问

容器需直通蜂窝 modem 所在的串口设备：

```yaml
services:
  vocat:
    devices:
      - /dev/ttyUSB0:/dev/ttyUSB0
      - /dev/ttyUSB1:/dev/ttyUSB1
      - /dev/ttyUSB2:/dev/ttyUSB2
      - /dev/ttyUSB3:/dev/ttyUSB3
```

实际设备名取决于宿主系统、模组固件、USB composition 与驱动配置。可查看可用串口：

```bash
ls /dev/ttyUSB*
ls /dev/ttyACM*
```

---

## 配置

Vocat 通过 `VOCAT_*` 环境变量配置，可选地用 JSON 配置文件（路径由 `VOCAT_CONFIG` 指定，严格反序列化，字段可见 `internal/config`）。环境变量优先级高于配置文件。

| 环境变量 | 默认值 | 说明 |
|---|---|---|
| `VOCAT_ADDR` | `0.0.0.0:7575` | 监听地址与端口 |
| `VOCAT_DATABASE_PATH` | `./data/vocat.db`（Docker：`/opt/vocat/data/vocat.db`） | SQLite 数据库路径 |
| `VOCAT_ADMIN_USERNAME` | `admin` | 管理员用户名 |
| `VOCAT_ADMIN_PASSWORD` | `admin` | 管理员密码（首次安装脚本会随机生成） |
| `VOCAT_SESSION_TTL` | `24h`（5m–720h） | 会话有效期 |
| `VOCAT_SECURE_COOKIES` | `false` | 启用 Secure cookie + HSTS，HTTPS 对外部署时建议开启 |
| `VOCAT_SHUTDOWN_TIMEOUT` | `10s` | 优雅关闭超时 |
| `VOCAT_MAX_REQUEST_BODY_BYTES` | `1048576`（1m–10m） | 请求体大小上限 |
| `VOCAT_CONFIG` | — | JSON 配置文件路径 |
| `VOCAT_REPO` | `your-org/vocat` | install.sh / CLI 自更新使用的 GitHub repo（owner/name） |

切勿将真实密码提交到仓库。

---

## 快速开始

安装后：

1. 将 EC20 模组连到测试主机。
2. 确认操作系统检测到 modem（`lsusb` / `ls /dev/ttyUSB*`）。
3. 插入授权的测试 SIM 卡。
4. 启动 Vocat 并登录 Web（默认 `admin`，密码见安装输出）。
5. 选择检测到的 modem 接口。
6. 运行基础模组检测。
7. 查看诊断结果。

推荐检测顺序：

```text
USB 检测
      ↓
AT 通信
      ↓
模组信息
      ↓
SIM 检测
      ↓
网络注册
      ↓
USSD 检测
      ↓
短信收发检测
      ↓
VoWiFi 诊断
      ↓
eSIM / EID 验证
```

### 首次检测建议

先验证基础通信，再跑高级诊断：

```text
AT      → 期望 OK
ATI     → 返回型号 / 固件
```

若模组无响应，检查：USB 线缆、USB D+/D− 走线、模组供电、串口选择、USB 驱动、模组启动状态、PCB 焊接、地线连接。

---

## 命令行

Vocat 二进制支持子命令：

```bash
vocat                  # 前台运行 Web 服务（默认）
vocat version          # 查看版本与构建时间
vocat update           # 自更新（从 GitHub Releases 拉取，SHA256 校验后原子替换；仅 Linux）
vocat menu             # root 交互菜单：改密 / 重启服务 / 卸载
vocat help             # 用法
```

`vocat update` 支持 `--check`（仅检查）、`--force`、`--repo`、`--target`、`--token` 等标志。Web 端的「检查更新」当前为有意留空的 no-op（不接可信更新源），自更新仅通过 CLI 进行。

`vocat menu` 要求 root 与交互式 TTY，用于在无 Web 访问时执行运维操作。

---

## 故障排查

### 模组未识别

```bash
lsusb
ls /dev/ttyUSB*
```

可能原因：USB 走线错误、模组供电不足、缺 USB 驱动、USB 线损坏、模组未启动、USB composition 不对、PCB 贴片问题。

### AT 指令无响应

确认选择了正确的串口——EC20 可能暴露多个串口，并非每个都用于 AT 指令。执行 `AT`，正常应返回 `OK`。

### SIM 未识别

```text
AT+CPIN?
```

响应可能为 SIM ready / PIN required / SIM unavailable。若未检测到，检查 SIM_VDD / SIM_DATA / SIM_CLK / SIM_RST / 地线 / SIM 插座焊接 / SIM 方向。

### 网络注册失败

```text
AT+CSQ
AT+COPS?
AT+CREG?
AT+CGREG?
```

注册取决于 SIM、网络可用性、运营商政策、支持频段、天线、固件与授权状态。

### 短信不工作

检查：SIM 注册、短信能力、信号质量、运营商支持、正确的 modem 端口；若目标是 `+86` 号段会被服务端拦截。

---

## 项目范围

Vocat 是诊断工具。它**不是**：

- 移动网络 / MVNO
- 运营商开通平台
- SM-DP+ / SM-DS
- eSIM 发行方
- SIM 克隆平台
- 电信拦截平台
- 运营商鉴权绕过工具

本项目用于辅助授权开发者进行硬件验证。

---

## 负责任使用

蜂窝模组与受监管的电信基础设施交互。测试前请确保：

1. 你拥有或被授权使用该硬件。
2. 你被授权使用该 SIM/eSIM profile。
3. 网络允许拟进行的测试活动。
4. 你的测试符合适用法律。
5. 你的设备不干扰电信基础设施。

存疑时，请使用隔离或运营商认可的实验室环境。

---

## 许可

Vocat 依据 **Vocat Research & Evaluation License** 分发。这是 source-available 许可，**不是** OSI 认证的开源许可。访问源码不自动授予：商用、再发布不受限的修改版本、移除防滥用控制、绕过地域限制或绕过 SIM 限制的权利。

完整条款见 [LICENSE](LICENSE)。

---

## 免责声明

Vocat 以授权研究、教育、开发与电信硬件测试为目的提供。软件按 **"AS IS"** 提供，不附带任何明示或暗示的担保。作者、维护者、贡献者与分发者不对使用或滥用 Vocat 造成的损失负责，包括但不限于：SIM 卡损坏、eSIM profile 丢失、SIM 停用、modem / 基带故障、PCB / 蜂窝模组 / 宿主设备损坏、网络服务中断、运营商 / 账户限制、数据丢失、服务中断、监管后果与未授权的电信活动。

用户有责任确保其使用 Vocat 符合适用的法律、法规、电信要求、运营商政策、网络政策与合同义务。项目维护者对在受限地域或环境中未授权部署或运行 Vocat 不承担责任。

---

## 安全问题

发现 Vocat 的安全问题，请**不要**立即在公开 issue 中发布利用细节，而应私下联系项目维护者。报告宜包含：问题描述、受影响版本、复现条件、潜在影响、可选的缓解方案。请勿在报告中附带真实订户凭证、SIM 密钥、鉴权密钥或个人信息。

---

## 贡献

欢迎与合法硬件测试和诊断相关的贡献，例如：更多 modem 兼容、更好的 EC20 检测、AT 指令诊断改进、USB 检测改进、短信检测改进、文档改进、UI 改进、Bug 修复、Docker 改进、硬件兼容文档。

以禁用或绕过项目安全 / 防滥用限制为主要目的的贡献不予接受。提交 PR 前：测试变更、说明改动、描述测试所用硬件、避免附敏感订户信息、确保贡献符合项目许可。

---

## 开发状态

Vocat 是实验性开发与硬件测试项目。版本间，接口、兼容性、命令、配置格式与安全机制可能变更。请勿将 Vocat 用于生产电信基础设施。

---

## FAQ

### Vocat 是开源项目吗？
Vocat 是 source-available，但不是 OSI 认证开源许可。源码可在 Vocat Research & Evaluation License 授权范围内检视与修改。

### 可以商用吗？
未经书面授权不行。默认许可仅授权研究、开发、教育与非营利测试用途。

### 可以用日常 SIM 卡吗？
Vocat 设计用于授权测试或开发用 SIM/eSIM。使用生产订户凭证可能受部署策略限制（例如对中国大陆 SIM 自动飞行模式）。

### Vocat 能解锁蜂窝模组吗？
不能。Vocat 用于诊断与功能验证。

### Vocat 会绕过运营商限制吗？
不会。Vocat 不用于绕过运营商鉴权、开通要求、认证要求或网络安全控制。

### VoWiFi 检测成功就等于 VoWiFi 可用吗？
不是。运营商侧开通与认证要求仍可能阻止 VoWiFi 实际可用。

### Vocat 只支持 EC20 吗？
EC20 是首发与主要开发目标。后续版本可能支持更多 Qualcomm 蜂窝模组。

### 默认密码是什么？
二进制 + systemd 首次安装脚本会生成 32 位随机密码并仅显示一次；Docker 镜像默认 `admin` / `admin`，须登录后立即修改。

---

## 致谢

Vocat 可能与第三方开发的硬件、软件、协议或技术交互。所有第三方名称、商标、产品名与公司名归其各自所有者所有。对 Qualcomm、Quectel、EC20、移动网络运营商、GSMA 技术等的引用仅用于识别与互操作说明。Vocat 与上述机构无关联、未被赞助、未被背书，除非另有明确声明。

---

## 联系

用于：安全报告、研究合作、教育使用、延长测试授权、商用授权咨询、附加地域授权——请通过官方 Vocat 项目仓库或指定项目联系渠道联系维护者。

---

## 最终声明

下载、编译、安装、修改或运行 Vocat 即表示你承诺确保你的使用是授权的，并符合适用许可、法律、电信法规、运营商政策与测试要求。若不同意项目许可条款或使用限制，请勿使用本软件。
