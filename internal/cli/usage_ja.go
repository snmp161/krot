package cli

const usageTextJapanese = `krot — WireGuard と AmneziaWG のトンネルを管理

使用方法:
  krot <アクション> [フラグ]

アクション（いずれか 1 つ）:
  --list            トンネルを一覧表示
  --create          トンネルを作成
  --delete          トンネルを削除
  --up              インターフェースを起動
  --down            インターフェースを停止
  --add-client      クライアントを追加
  --delete-client   クライアントを削除
  --lock-client     クライアントをロック
  --unlock-client   クライアントのロックを解除
  --list-client     クライアントを一覧表示
  --doctor          環境をチェック（ツール、カーネルモジュール、AmneziaWG バージョン）
  --udp-listener    ポート診断用 UDP リスナー
  --udp-checker     ポート診断用 UDP チェッカー
  --help            このヘルプを表示
  --version         バージョン、コミット、ビルド日時を表示

フラグ（--flag value または --flag=value）:
  --type wg|awg|awg2|awg3   プロトコルタイプ（create: 必須、list: フィルター）
  --interface NAME          インターフェース名、最大 15 文字 [a-zA-Z0-9_.=+-]
  --ip CIDR[,CIDR]          create: サーバーアドレス（10.8.0.1/24,fd00:8::1/64）;
                            add-client: クライアントアドレス（/32、/128）;
                            list: アドレスフィルター
  --port N|N:M              UDP ポートまたは範囲 1..65535、最大 1024 ポート
                            （create、udp-listener、udp-checker: 必須、list: フィルター）
  --endpoint HOST           クライアント設定に使用するサーバーの公開アドレス
                            （create: 必須、add-client: クライアントごとの上書き）
  --address HOST            --udp-checker のリモートリスナーアドレス（ポートなし）
  --mtu N                   MTU 576..9000（デフォルト: wg 1420、awg/awg2 1376、awg3 1280）
  --dns IP[,IP]             クライアント設定の DNS サーバー
                            （create: 全クライアント、add-client: このクライアントのみ）
  --i1 CPS                  AmneziaWG I1 シグネチャパケット（awg* のみ、最大 2048 文字）
  --mimic PROFILE           クライアント設定用の I1-I5 ミミクリーチェーンを生成:
                            dns|stun|ntp|ssdp|dtls（awg* のみ、--i1 とは併用不可）
  --id ID                   クライアント ID [a-zA-Z0-9_.-]{1,64}、インターフェース内で一意
  --allowed-ips CIDR[,CIDR] クライアント設定の AllowedIPs（例: 0.0.0.0/0,::/0）
  --psk                     クライアントの PresharedKey を生成
  --qr                      クライアント設定を QR コードとしても出力
  --amnezia-key             クライアント設定を AmneziaVPN キー（vpn://...）としても出力
  --keenetic                Keenetic ルーター用の KeeneticOS CLI コマンドも出力
                            （wg および awg 1.0/1.5/2.0 トンネル、IPv4 のみ）
  --json                    テーブルではなく JSON で出力（list、list-client、udp-listener、udp-checker）
  --yes                     確認を省略（delete）
  --timeout DURATION        パケットのタイムアウト（例: 2s や 500ms、デフォルト: 2s）
  --count N                 ポートごとの各タイプのパケット数、1..100（デフォルト: 3）
  --echo-size N             echo ペイロードサイズ、1..1400（デフォルト: 64、udp-checker のみ）
  --workers N               同時ポートチェック数、1..100（デフォルト: 32、udp-checker のみ）
  --verbose                 診断の進行状況（udp-listener、udp-checker、stderr）

アクション別フラグ:
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
  --doctor         （フラグなし、root なしで動作）
  --udp-listener   --port [--timeout] [--count] [--json] [--verbose]
  --udp-checker    --address --port [--timeout] [--count] [--echo-size] [--workers] [--json] [--verbose]

終了コード:
  0 成功                    5 競合（すでに存在、アドレスまたはポート使用中）
  1 内部エラー              6 環境（root ではない、ツール不足、AmneziaWG バージョン）
  2 使用法エラー            7 ロックを取得できない
  3 無効な値                10 ユーザーによるキャンセル
  4 見つからない

root 権限:
  --help、--version、--doctor、--udp-checker には不要です。
  --udp-listener は 1024 未満のポート、または /proc/net/udp と
  /proc/net/udp6 が読み取れない場合にのみ root が必要です。
  その他のすべてのアクションには root が必要です。
`
