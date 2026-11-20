package cli

const usageTextChineseSimplified = `krot — 管理 WireGuard 和 AmneziaWG 隧道

用法：
  krot <操作> [选项]

操作（必须且只能指定一个）：
  --list            列出隧道
  --create          创建隧道
  --delete          删除隧道
  --up              启动接口
  --down            停止接口
  --add-client      添加客户端
  --delete-client   删除客户端
  --lock-client     锁定客户端
  --unlock-client   解锁客户端
  --list-client     列出客户端
  --doctor          检查环境（工具、内核模块、AmneziaWG 版本）
  --udp-listener    用于端口诊断的 UDP 监听器
  --udp-checker     用于端口诊断的 UDP 检查器
  --help            显示此帮助
  --version         显示版本、提交和构建日期

选项（--flag value 或 --flag=value）：
  --type wg|awg|awg2|awg3   协议类型（create：必需；list：过滤器）
  --interface NAME          接口名称，最多 15 个字符 [a-zA-Z0-9_.=+-]
  --ip CIDR[,CIDR]          create：服务器地址（10.8.0.1/24,fd00:8::1/64）；
                            add-client：客户端地址（/32、/128）；
                            list：按地址过滤
  --port N|N:M              UDP 端口或范围 1..65535，最多 1024 个端口
                            （create、udp-listener、udp-checker：必需；list：过滤器）
  --endpoint HOST           客户端配置使用的服务器公共地址
                            （create：必需；add-client：覆盖单个客户端的设置）
  --address HOST            --udp-checker 的远程监听器地址（不含端口）
  --mtu N                   MTU 576..9000（默认：wg 1420，awg/awg2 1376，awg3 1280）
  --dns IP[,IP]             客户端配置的 DNS 服务器
                            （create：所有客户端；add-client：仅此客户端）
  --i1 CPS                  AmneziaWG I1 签名数据包（仅限 awg*，最多 2048 个字符）
  --mimic PROFILE           为客户端配置生成 I1-I5 伪装链：
                            dns|stun|ntp|ssdp|dtls（仅限 awg*；不能与 --i1 同用）
  --id ID                   客户端 ID [a-zA-Z0-9_.-]{1,64}，在接口内唯一
  --allowed-ips CIDR[,CIDR] 客户端配置的 AllowedIPs，例如 0.0.0.0/0,::/0
  --psk                     为客户端生成 PresharedKey
  --qr                      同时将客户端配置输出为 QR 码
  --amnezia-key             同时将客户端配置输出为 AmneziaVPN 密钥（vpn://...）
  --keenetic                同时输出 Keenetic 路由器的 KeeneticOS CLI 命令
                            （wg 和 awg 1.0/1.5/2.0 隧道，仅限 IPv4）
  --json                    使用 JSON 代替表格（list、list-client、udp-listener、udp-checker）
  --yes                     不要求确认（delete）
  --timeout DURATION        数据包超时时间，例如 2s 或 500ms（默认：2s）
  --count N                 每个端口每种类型发送的数据包数，1..100（默认：3）
  --echo-size N             echo 负载大小，1..1400（默认：64，仅限 udp-checker）
  --workers N               同时检查的端口数，1..100（默认：32，仅限 udp-checker）
  --verbose                 诊断进度（udp-listener、udp-checker；stderr）

各操作的选项：
  --list           [--type] [--interface] [--ip] [--port] [--json]
  --create         --type --ip --port --endpoint [--interface] [--mtu] [--dns] [--i1] [--mimic]
  --delete         --interface [--yes]
  --up, --down     --interface
  --add-client     --interface --id --allowed-ips [--ip] [--endpoint] [--psk] [--qr]
                   [--dns] [--amnezia-key] [--mimic] [--keenetic]
  --delete-client  --interface --id
  --lock-client    --interface --id
  --unlock-client  --interface --id
  --list-client    --interface [--id] [--json]
  --doctor         （无选项；无需 root 即可运行）
  --udp-listener   --port [--timeout] [--count] [--json] [--verbose]
  --udp-checker    --address --port [--timeout] [--count] [--echo-size] [--workers] [--json] [--verbose]

退出代码：
  0 成功                    5 冲突（已存在、地址或端口被占用）
  1 内部错误                6 环境（无 root、缺少工具、AmneziaWG 版本）
  2 用法错误                7 无法获取锁
  3 无效值                  10 用户取消
  4 未找到

root 权限：
  --help、--version、--doctor 和 --udp-checker 不需要 root。
  --udp-listener 仅在端口低于 1024 或无法读取
  /proc/net/udp 和 /proc/net/udp6 时需要 root。
  所有其他操作均需要 root。
`
