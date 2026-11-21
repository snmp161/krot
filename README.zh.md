# krot

[English](README.md) | [Русский](README.ru.md) | **简体中文** | [繁體中文](README.zh-Hant.md) | [Español](README.es.md) | [Português](README.pt.md) | [한국어](README.ko.md) | [日本語](README.ja.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [Deutsch](README.de.md)

`krot` 是一款用于在 Linux 服务器上管理 **WireGuard** 和 **AmneziaWG**
（1.x、2.0、3.1）隧道的命令行工具：创建和删除接口、管理客户端（peer）、
生成客户端配置、二维码和 AmneziaVPN 密钥（`vpn://…`）。

名称 **krot** 在俄语和其他一些斯拉夫语中意为“鼹鼠”——一种挖掘隧道的动物。

- 单个静态二进制文件，无依赖（Go，`CGO_ENABLED=0`）。
- 客户端操作**不会重启或重建接口**：仅通过精确的
  `wg|awg set <if> peer …` 调用完成——其他客户端不受影响。
- 接口配置是唯一事实来源：无需额外数据库；krot 元数据存储在注释中，
  非 krot 创建的配置会被读取并保留。
- 输出可预测：面向人类的表格，面向自动化的 `--json` 和退出码。
- krot 不配置 NAT、转发或防火墙规则。

完整技术规范：[`docs/SPEC.md`](docs/SPEC.md)。

## 系统要求

| | |
|---|---|
| 操作系统 | Ubuntu 22.04 / 24.04 / 26.04、Debian 12 / 13、AlmaLinux（Rocky、RHEL）9 / 10 |
| 架构 | `amd64`、`arm64` |
| `--type wg` | `wireguard-tools`（`wg`、`wg-quick`）、WireGuard 内核模块 |
| `--type awg*` | `amneziawg-tools`（`awg`、`awg-quick`）和 AmneziaWG 内核模块**或** `amneziawg-go` |
| 始终需要 | `iproute2`（`ip`）、`systemd` |
| 权限 | root（`--help`、`--version`、`--doctor`、`--udp-checker` 除外；`--udp-listener` 仅在端口 `<1024` 或 `/proc/net/udp{,6}` 不可读时需要 root） |

krot 不会自动安装上述任何组件。如果缺少必需程序，会以退出码 `6`
退出，并给出需要安装的软件包名称。

### AmneziaWG 版本

在执行 `--create` 之前，krot 会检查已安装的 AmneziaWG 是否支持请求的类型：

| 类型 | amneziawg-tools | 内核模块 | netlink 家族 | amneziawg-go |
|---|---|---|---|---|
| `awg`（1.0） | 1.0.20240213 | — | ≥ 1 | 0.1.0 |
| `awg` 带 `--i1`（1.5） | 1.0.20250901 | 1.0.20251009 | ≥ 2 | 0.2.13 |
| `awg2`（2.0） | 1.0.20250901 | 1.0.20251009 | ≥ 2 | 0.2.15 |
| `awg3`（3.1） | 3.1.20260812 | 3.1.20260812 | ≥ 3 | 3.1.20260812 |

某些构建版本会报告占位版本号（例如，1.x 工具的 `awg --version`
几乎总是打印 `1.0.20210914`，使用 `make` 构建的模块报告 `1.0.0`）。
这类版本会产生警告而非错误；只有在版本被可靠识别且过旧时才会返回
退出码 `6`。如果版本确实过旧，单元将无法启动，krot 会回滚更改。

## 安装

请从发布页面 <https://github.com/snmp161/krot/releases> 下载软件包或
二进制文件进行安装，并务必验证校验和。

```sh
VERSION=0.7.3
BASE=https://github.com/snmp161/krot/releases/download/v$VERSION

# 下载软件包和校验和列表
wget "$BASE/krot_${VERSION}_amd64.deb" "$BASE/SHA256SUMS"
# 验证：应输出 "krot_..._amd64.deb: OK"
sha256sum --check --ignore-missing SHA256SUMS
# 安装
sudo dpkg -i "krot_${VERSION}_amd64.deb"
```

基于 RPM 的系统（AlmaLinux、Rocky、RHEL）：

```sh
wget "$BASE/krot-${VERSION}-1.x86_64.rpm" "$BASE/SHA256SUMS"
sha256sum --check --ignore-missing SHA256SUMS
sudo rpm -i "krot-${VERSION}-1.x86_64.rpm"
```

不使用软件包的安装方式：下载 `krot-linux-amd64`（或 `krot-linux-arm64`）
和 `SHA256SUMS`，以同样方式验证，然后执行
`sudo install -m 0755 krot-linux-amd64 /usr/bin/krot`。

```sh
wget "$BASE/LICENSE" "$BASE/THIRD_PARTY_NOTICES"
```

对于 `arm64`，请使用 `*_arm64.deb`、`*.aarch64.rpm` 和 `krot-linux-arm64` 文件。

## 使用方法

```
krot <操作> [标志]
```

每次运行只能执行一个操作；标志仅支持长格式 `--flag value` 或
`--flag=value`。不适用于该操作的标志会导致错误（退出码 `2`）。
完整帮助：`krot --help`。

### 隧道

```sh
# 创建 AmneziaWG 2.0（双栈）；名称自动选择：awg0、awg1、…
sudo krot --create --type awg2 --ip 10.8.0.1/24,fd00:8::1/64 --port 51820 \
          --endpoint vpn.example.com --dns 1.1.1.1,9.9.9.9

# 使用显式名称和 MTU 创建 WireGuard
sudo krot --create --type wg --interface wg0 --ip 10.9.0.1/24 --port 51821 \
          --endpoint 203.0.113.10 --mtu 1420

# 带 I1 签名包的 AmneziaWG 1.5（krot 验证 CPS 语法和长度——最多 2048 字符）
sudo krot --create --type awg --ip 10.10.0.1/24 --port 51822 --endpoint vpn.example.com \
          --i1 '<b 0xf6ab3267fa><r 16>'

# AmneziaWG 3.1（始终启用 Header Protection、RandomTrailers、DisableCookies）
sudo krot --create --type awg3 --ip 10.11.0.1/24 --port 51823 --endpoint vpn.example.com

sudo krot --list                         # 所有隧道
sudo krot --list --type awg2 --json      # 过滤器 --type/--interface/--ip/--port，逻辑与
sudo krot --down --interface awg0        # systemctl stop awg-quick@awg0
sudo krot --up --interface awg0          # systemctl start；重复 --up 输出 "already up"，退出码 0
sudo krot --delete --interface awg0      # 如有客户端会要求确认
sudo krot --delete --interface awg0 --yes
```

### 诊断

```sh
krot --doctor   # 无需 root：工具、内核模块、AmneziaWG 版本、目录
```

输出为 `CHECK/STATUS/DETAIL` 表格，状态为 `ok` / `warn` / `fail`。
如果没有 `fail` 项，退出码为 `0`，否则为 `6`。不更改任何内容，也不获取锁。

#### UDP 端口诊断

`krot` 可以检查两台主机之间实际可通的 UDP 端口。当托管服务商或其上游
过滤部分端口（例如由于 DDoS 规则）时，这非常有用。

在服务器上启动 listener：

```sh
# 单个端口
sudo krot --udp-listener --port 51820

# 端口范围（最多 1024 个端口，每个端口一个套接字）
sudo krot --udp-listener --port 51820:51830
```

Listener 响应三种数据报：
- `ping SEQ:N` → `pong SEQ:N`；
- `ECHO <text>` → `<text>`；
- `TIME` → 当前 Unix 时间（毫秒）。

在客户端运行 checker：

```sh
# 检查单个端口
krot --udp-checker --address 203.0.113.1 --port 51820

# 检查端口范围
krot --udp-checker --address 203.0.113.1 --port 51820:51830

# 增加 echo 负载大小（1–1400 字节，默认 64）——有助于找到最佳 MTU
krot --udp-checker --address 203.0.113.1 --port 51820 --echo-size 1400

# 设置 worker 数量并显示进度
krot --udp-checker --address 203.0.113.1 --port 51820:51830 --workers 64 --verbose
```

默认情况下，checker 每个端口发送 3 个每种类型的数据包（`--count`，
1–100），并在 `--timeout`（默认 2s）内等待响应。默认并行检查 32 个端口，
`--workers` 可设置为 1–100。`--verbose` 将端口范围和完成结果输出到 stderr；
在 listener 中则记录客户端地址和请求类型。输出：

```
PORT     STATUS   PING    ECHO    TIME (min/avg/max)   DETAIL
51820    ok       3/3     3/3     12/24/45 ms          -
51821    filtered 0/3     0/3     -                    timeout
```

状态：
- `ok` — 所有 ping 和 echo 数据包都收到了有效响应；
- `filtered` — 至少一个数据包超时（该端口很可能被过滤）；
- `error` — 收到了无效响应。

使用 `--json` 时，输出为对象数组，字段包括 `port`、`status`、
`ping_sent`、`ping_ok`、`echo_sent`、`echo_ok`、`time_sent`、`time_ok`、
`rtt_min_ms`、`rtt_avg_ms`、`rtt_max_ms`、`detail`。

按下 `Ctrl+C` 或收到 `SIGTERM` 时，checker 会以选定格式输出已完整检查的端口结果，
并以退出码 `10` 结束；未完成的端口不会包含在结果中。

`--udp-checker` 无需 root。`--udp-listener` 仅在范围包含 `<1024` 端口
或 `/proc/net/udp` 和 `/proc/net/udp6` 当前用户不可读时需要 root。
Listener 运行直到 `Ctrl+C`/`SIGTERM`。

`--list` 示例：

```
INTERFACE  TYPE  VERSION  ADDRESS                   PORT   STATUS  CLIENTS
awg0       awg2  2.0      10.8.0.1/24,fd00:8::1/64  51820  up      12 (1 locked)
wg0        wg    -        10.9.0.1/24               51821  down    3
```

默认 MTU：`wg` — 1420，`awg`/`awg2` — 1376，`awg3` — 1280。
混淆参数（Jc、S1–S4、H1–H4 等）在安全范围内自动生成并通过验证
（规范第 7 节）。

### 客户端

```sh
# 添加客户端：地址自动分配（第一个空闲的 /32 和 /128）；
# 客户端配置仅打印一次——客户端私钥绝不存储
sudo krot --add-client --interface awg0 --id phone-sergei --allowed-ips 0.0.0.0/0,::/0

# 带 PresharedKey 和终端二维码
sudo krot --add-client --interface awg0 --id laptop --allowed-ips 0.0.0.0/0 --psk --qr

# 额外输出 AmneziaVPN 密钥：配置后的 vpn://… 行
sudo krot --add-client --interface awg0 --id tablet --allowed-ips 0.0.0.0/0,::/0 --amnezia-key

# KeeneticOS CLI 命令：将 Keenetic 路由器连接为客户端（仅限 wg，仅限 IPv4）
sudo krot --add-client --interface wg0 --id keenetic --allowed-ips 0.0.0.0/0 --keenetic

# 指定客户端地址
sudo krot --add-client --interface awg0 --id router --allowed-ips 10.8.0.0/24 --ip 10.8.0.50/32

# 非 krot 创建的配置：显式传递 endpoint（不写入服务器配置）
sudo krot --add-client --interface office --id guest --allowed-ips 0.0.0.0/0 --endpoint 203.0.113.10

# 为单个客户端自定义 DNS（不写入服务器配置）
sudo krot --add-client --interface awg0 --id kid-phone --allowed-ips 0.0.0.0/0 --dns 94.140.14.15

sudo krot --list-client --interface awg0
sudo krot --list-client --interface awg0 --id phone-sergei --json
sudo krot --lock-client   --interface awg0 --id laptop    # 重复执行 — "already locked"，退出码 0
sudo krot --unlock-client --interface awg0 --id laptop
sudo krot --delete-client --interface awg0 --id laptop
```

`--allowed-ips` 设置**客户端**配置中的 `AllowedIPs`（客户端通过隧道
路由的内容）。客户端 DNS：如果给出 `--dns` 则使用；否则使用接口的
`# krot:dns` 元数据（`--create` 时的 `--dns`）；否则使用服务器配置
`[Interface]` 中的 `DNS` 行；如果任何地方都没有 DNS，则客户端配置
不包含 `DNS` 行。

客户端地址从接口子网分配，排除网络地址、广播地址和服务器地址。
也支持点对点子网 `/31` 和 `/127`（RFC 3021、RFC 6164）：两个地址
都可用，因此可以通过 `--create … --ip 10.8.0.0/31,fd00:8::/127`
创建“服务器 + 单客户端”隧道。

请立即保存生成的配置：无法再次获取，只能删除客户端后重新添加。
二维码使用 Unicode 半块字符打印，适用于深色终端背景。

`--amnezia-key` 在配置后打印 AmneziaVPN 密钥——单行 `vpn://…`，
可粘贴到 AmneziaVPN 应用（“插入密钥”），例如通过即时通讯工具发送。
其内容就是同一个客户端配置：`vpn://` + Qt `qCompress` 的 base64url
（不带 `=`）（4 字节大端长度 + zlib）。该密钥适用于所有类型，包括 `wg`。
`--qr` 生成的二维码包含配置文本——AmneziaVPN 和 AmneziaWG 可扫描；
对于 `wg`，WireGuard 也可以扫描。

`--keenetic` 在配置后打印 KeeneticOS CLI 命令块——将 Keenetic 路由器
连接为隧道客户端。该块遵循 KeeneticOS 的 running-config 语法，可直接
粘贴到路由器 CLI（SSH/telnet 或 Web 界面“命令行界面”）。KeeneticOS
支持 AmneziaWG 混淆（ASC 参数）：AWG 1.0 自 KeeneticOS 4.2 起，
AWG 1.5/2.0 自 5.1 起；对于 awg 隧道，该块包含 `wireguard asc …` 行
（CPS 字符串 I1–I5 带引号；通过 Web 界面导入配置文件更可靠）。
不支持 AWG 3.1。仅限 IPv4；如果隧道 MTU 与默认值 1420 不同，会打印
提示。格式见规范第 6.13 节。

`--list-client` 示例：

```
ID            PUBLIC KEY  ADDRESS                    STATUS  LAST HANDSHAKE  RX       TX
phone-sergei  aB3d…Xy9=   10.8.0.2/32,fd00:8::2/128  active  1m ago          12.4MiB  1.1GiB
laptop-old    Qw7e…Zz1=   10.8.0.3/32                locked  -               -        -
```

### 伪装（I1–I5 数据包）

从 1.5 版本开始，AmneziaWG 可以在每次握手前发送最多五个签名数据包
`I1`–`I5`：这些数据包看起来像其他协议（DNS、STUN、NTP、SSDP、DTLS）
的开头。krot 通过 `--mimic <profile>` 标志自行生成此类链：

```sh
# 接口所有客户端的默认配置
krot --create --type awg --ip 10.8.0.1/24 --port 51820 --endpoint vpn.example.com --mimic dns
# 为单个客户端覆盖
krot --add-client --interface awg0 --id phone --allowed-ips 0.0.0.0/0 --mimic stun
```

配置：`dns`（三个 A/AAAA/HTTPS 查询）、`stun`（Binding Request）、
`ntp`、`ssdp`（M-SEARCH）、`dtls`（ClientHello）。链在 `--add-client`
时生成（每个客户端独立）并仅写入客户端配置：I 数据包由握手发起方
发送，服务器不需要它们。客户端需要 AmneziaWG 1.5+——见规范第 6.12 节。

除了内置配置，krot 还接受通过 `--i1` 在 `--create` 时传入的现成
CPS 字符串——例如用于 QUIC：不生成内置 QUIC 配置，因为单个 QUIC
Initial 为 2400–2500 字符，超过 2048 限制（由于
[amneziawg-tools#69](https://github.com/amnezia-vpn/amneziawg-tools/issues/69)）。
现成字符串可从第三方生成器获取（均为 MIT 许可，与 krot 无关联）：

- [payloadGen](https://github.com/Sketchystan1/payloadGen) — 网页：
  在浏览器中打开 `index.html`，选择协议，取第一行（I1）；
- 来自 [awg-multi-script](https://github.com/pumbaX/awg-multi-script)
  的 `cpsgen.py`（`src/py/cpsgen.py`）— payloadGen 的 Python 3
  控制台版本，无需 root。配置：`dns`、`ntp`、`rtp`、`stun`、`sip`、
  `ssdp`、`webrtc`、`quic`、`curl_quic`；第二个参数为域名。

```sh
# 下载生成器并在运行前审查其代码
git clone --depth 1 https://github.com/pumbaX/awg-multi-script
less awg-multi-script/src/py/cpsgen.py

# 仅生成 I1 并检查长度（必须 ≤ 2048 字符）
I1=$(python3 awg-multi-script/src/py/cpsgen.py dns example.com --only-i1)
echo ${#I1}

sudo krot --create --type awg2 --ip 10.8.0.1/24 --port 51820 --endpoint vpn.example.com --i1 "$I1"
```

注意事项：

- `I1` 仅在 `--create` 时设置，写入服务器配置并复制到所有客户端配置。
  krot 不接受 `I2`–`I5`。
- 长度不得超过 2048 字符（见[限制](#限制)），否则退出码 `3`。
  `quic` 和 `curl_quic` 配置的数据包约为 2400 字符，无法容纳。
  较短的配置可用：`rtp`（约 30 字符）、`dns` 和 `ntp`（约 100）、
  `stun`（约 250–350）、`ssdp`（约 400）、`sip`（约 1200）。
- `--type awg` 带 `--i1` 表示 AmneziaWG 1.5，有各自的最低版本要求
  （见 [AmneziaWG 版本](#amneziawg-版本)）。`<rc N>`/`<rd N>` 标签
  （由 `sip` 配置生成）自 0.2.15 起被 amneziawg-go 理解，因此 1.5
  建议使用不含它们的配置。
- 客户端必须支持 AmneziaWG 1.5 或更高版本。
- 使用较长的 `I1` 时，配置可能无法放入二维码：`--qr` 将以退出码 `3`
  退出（客户端已添加且配置已打印）。请通过文件传输配置或使用
  `--amnezia-key`。

### JSON

`--list --json` 输出对象数组，字段包括 `interface`、`type`、
`version`、`addresses[]`、`port`、`status`、`clients_total`、
`clients_locked`。`--list-client --json` 输出对象数组，字段包括 `id`、
`public_key`、`addresses[]`、`status`、`latest_handshake`（unix 时间或
`null`）、`rx_bytes`、`tx_bytes`、`endpoint`（或 `null`）、`created`
（非 krot 管理的 peer 为 `null`）。空结果为 `[]`。

## 退出码

| 代码 | 含义 |
|---|---|
| 0 | 成功 |
| 1 | 内部或意外错误（包括无法将更改应用到运行中的接口——配置已回滚） |
| 2 | CLI 使用错误（标志；有客户端时未使用 `--yes` 且 stdin 非终端的删除操作） |
| 3 | 值验证错误（包括缺少 endpoint、“config too large for QR”） |
| 4 | 未找到对象（接口、客户端） |
| 5 | 冲突：已存在、地址或端口被占用、无空闲地址 |
| 6 | 环境：无 root、缺少工具、AmneziaWG 版本不受支持 |
| 7 | 10 秒内无法获取 `/run/lock/krot.lock` 锁 |
| 10 | 用户取消操作 |

输出到 stdout；错误和警告输出到 stderr，带 `krot:` 前缀。

## 数据存储

| 类型 | 配置 | systemd 单元 | 工具 |
|---|---|---|---|
| `wg` | `/etc/wireguard/<if>.conf` | `wg-quick@<if>` | `wg` |
| `awg`、`awg2`、`awg3` | `/etc/amnezia/amneziawg/<if>.conf` | `awg-quick@<if>` | `awg` |

元数据以 `wg-quick`/`awg-quick` 忽略的注释形式存储：

```ini
# krot:type=awg2
# krot:endpoint=vpn.example.com
# krot:dns=1.1.1.1,9.9.9.9
[Interface]
PrivateKey = ...
Address = 10.8.0.1/24, fd00:8::1/64
ListenPort = 51820
MTU = 1376
Jc = 6
...
# PreUp =
# PostUp =
# PreDown =
# PostDown =

# krot:id=phone-sergei
# krot:created=2026-10-17T01:30:00Z
[Peer]
PublicKey = ...
AllowedIPs = 10.8.0.2/32, fd00:8::2/128

# krot:id=laptop-old
# krot:created=2026-10-01T10:00:00Z
# krot:locked=2026-10-16T12:00:00Z
#~ [Peer]
#~ PublicKey = ...
#~ AllowedIPs = 10.8.0.3/32
```

- 锁定的客户端是以 `#~ ` 前缀注释掉的 `[Peer]` 块。
- 注释掉的 `PreUp`/`PostUp`/`PreDown`/`PostDown` 钩子可以取消注释并
  填写（例如 NAT 规则）——krot 不会修改它们。
- 支持不含 krot 元数据的配置（手动创建）：`--list` 会显示它们，
  `--list-client` 打印其 peer 且 id 为 `-`（`--id` 操作不适用于此类
  peer），任何写入都会逐字节保留所有原始行、注释和顺序。
- 写入是原子的（临时文件 → `fsync` → `rename` → 目录 `fsync`），
  权限为 `0600`。
- 所有修改操作都会对 `/run/lock/krot.lock` 获取排他 `flock`——
  并发运行（例如 Salt 和人工操作）不会丢失更改。
- krot 不创建备份。

## 限制

- NAT、转发、防火墙、sysctl 不在 krot 范围内（请使用
  `PostUp`/`PostDown` 钩子）。
- 不支持安装 WireGuard/AmneziaWG 以及在类型之间迁移隧道。
  伪装签名生成使用内置配置（`--mimic`）；QUIC 配置和自定义签名
  通过 `--i1` 和第三方生成器，见[伪装](#伪装i1i5-数据包)。
- `--i1` 不得超过 2048 字符：由于
  [amneziawg-tools#69](https://github.com/amnezia-vpn/amneziawg-tools/issues/69)，
  过长的 `I1`–`I5`（约 3600+ 字符）会导致 `awg show` 挂起，
  `awg set`/`setconf` 失败。
- 不存储客户端私钥；无法重新签发客户端配置。

## 开发

需要 `go.mod` 中的 Go（当前为 1.27.4）和 Docker 用于集成测试。

```sh
make check        # 所有本地检查：格式、测试、模块、安全和 workflow 检查器
make fuzz-smoke   # 对所有 targets 进行限时 fuzzing
make build        # 静态 bin/krot（CGO_ENABLED=0），版本来自 git describe
make build-arm64
make integration  # 在带 systemd 的特权 Docker 容器中进行集成测试
make package-preflight # 为两种架构构建 DEB/RPM + Ubuntu 24.04/AlmaLinux 9 smoke
```

仓库规则：[`AGENTS.md`](AGENTS.md)——规范优先，先测试后代码，
每个任务一个提交。

- **单元测试**不需要 root，不触碰系统：目录替换为临时目录，
  外部命令（`wg`、`awg`、`systemctl`、`ip` 等）通过带调用记录的
  模拟执行器处理。单独的测试验证客户端操作仅调用 `wg|awg show` 和
  `wg|awg set <if> peer …`。
- **Golden 文件**位于各包的 `testdata/`。生成的输出（`internal/app`、
  `internal/qr`）仅在有意时更新——使用
  `go test ./internal/app ./internal/qr -update` 并在提交中说明；
  其他 golden 文件根据规范手动编写。
- **集成测试**（`test/integration`，构建标签 `integration`）需要 root
  和 systemd，仅在独立的网络命名空间中运行：测试接口单元通过
  `NetworkNamespacePath` drop-in 放入 netns。涵盖所有类型的完整周期、
  客户端操作透明性（连续 ping 无丢失、ifindex 和握手不变）、IPv6、
  手动创建的配置和并发运行。本地通过 `scripts/integration-docker.sh`，
  CI 中使用 `sudo`。
- `scripts/collect-versions.sh` 重新构建真实的 AmneziaWG 版本输出
  用于 golden 测试。
- **CI**（`.github/workflows/ci.yml`）：本地质量检查、带 shuffle、`-race`
  和覆盖率（最低 90%）的单元测试、限时 fuzzing、集成测试、package preflight
  以及 `amd64`/`arm64` 构建。**发布**（`release.yml`，标签 `v*`）：
  二进制文件、`.deb`/`.rpm` 软件包（nfpm、`packaging/nfpm.yaml`）、
  所有操作系统的软件包冒烟测试（`scripts/smoke.sh`）、带 `SHA256SUMS`
  的 GitHub Release。

## 许可证

[Apache License 2.0](LICENSE)。版权所有 2026 Sergey Ovchinnikov。
