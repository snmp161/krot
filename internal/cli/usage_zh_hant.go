package cli

const usageTextChineseTraditional = `krot — 管理 WireGuard 和 AmneziaWG 隧道

用法：
  krot <操作> [選項]

操作（必須且只能指定一個）：
  --list            列出隧道
  --create          創建隧道
  --delete          刪除隧道
  --up              啟動接口
  --down            停止接口
  --add-client      添加客戶端
  --delete-client   刪除客戶端
  --lock-client     鎖定客戶端
  --unlock-client   解鎖客戶端
  --list-client     列出客戶端
  --doctor          檢查環境（工具、內核模塊、AmneziaWG 版本）
  --udp-listener    用於端口診斷的 UDP 監聽器
  --udp-checker     用於端口診斷的 UDP 檢查器
  --help            顯示此幫助
  --version         顯示版本、提交和構建日期

選項（--flag value 或 --flag=value）：
  --type wg|awg|awg2|awg3   協議類型（create：必需；list：過濾器）
  --interface NAME          接口名稱，最多 15 個字符 [a-zA-Z0-9_.=+-]
  --ip CIDR[,CIDR]          create：服務器地址（10.8.0.1/24,fd00:8::1/64）；
                            add-client：客戶端地址（/32、/128）；
                            list：按地址過濾
  --port N|N:M              UDP 端口或範圍 1..65535，最多 1024 個端口
                            （create、udp-listener、udp-checker：必需；list：過濾器）
  --endpoint HOST           客戶端配置使用的服務器公共地址
                            （create：必需；add-client：覆蓋單個客戶端的設置）
  --address HOST            --udp-checker 的遠程監聽器地址（不含端口）
  --mtu N                   MTU 576..9000（默認：wg 1420，awg/awg2 1376，awg3 1280）
  --dns IP[,IP]             客戶端配置的 DNS 服務器
                            （create：所有客戶端；add-client：僅此客戶端）
  --i1 CPS                  AmneziaWG I1 簽名數據包（僅限 awg*，最多 2048 個字符）
  --mimic PROFILE           為客戶端配置生成 I1-I5 偽裝鏈：
                            dns|stun|ntp|ssdp|dtls（僅限 awg*；不能與 --i1 同用）
  --id ID                   客戶端 ID [a-zA-Z0-9_.-]{1,64}，在接口內唯一
  --allowed-ips CIDR[,CIDR] 客戶端配置的 AllowedIPs，例如 0.0.0.0/0,::/0
  --psk                     為客戶端生成 PresharedKey
  --qr                      同時將客戶端配置輸出為 QR 碼
  --amnezia-key             同時將客戶端配置輸出為 AmneziaVPN 密鑰（vpn://...）
  --keenetic                同時輸出 Keenetic 路由器的 KeeneticOS CLI 命令
                            （wg 和 awg 1.0/1.5/2.0 隧道，僅限 IPv4）
  --json                    使用 JSON 代替表格（list、list-client、udp-listener、udp-checker）
  --yes                     不要求確認（delete）
  --timeout DURATION        數據包超時時間，例如 2s 或 500ms（默認：2s）
  --count N                 每個端口每種類型發送的數據包數，1..100（默認：3）
  --echo-size N             echo 負載大小，1..1400（默認：64，僅限 udp-checker）
  --workers N               同時檢查的端口數，1..100（默認：32，僅限 udp-checker）
  --verbose                 診斷進度（udp-listener、udp-checker；stderr）

各操作的選項：
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
  --doctor         （無選項；無需 root 即可運行）
  --udp-listener   --port [--timeout] [--count] [--json] [--verbose]
  --udp-checker    --address --port [--timeout] [--count] [--echo-size] [--workers] [--json] [--verbose]

退出代碼：
  0 成功                    5 衝突（已存在、地址或端口被佔用）
  1 內部錯誤                6 環境（無 root、缺少工具、AmneziaWG 版本）
  2 用法錯誤                7 無法獲取鎖
  3 無效值                  10 用戶取消
  4 未找到

root 權限：
  --help、--version、--doctor 和 --udp-checker 不需要 root。
  --udp-listener 僅在端口低於 1024 或無法讀取
  /proc/net/udp 和 /proc/net/udp6 時需要 root。
  所有其他操作均需要 root。
`
