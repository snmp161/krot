# krot

[English](README.md) | [Русский](README.ru.md) | [简体中文](README.zh.md) | [繁體中文](README.zh-Hant.md) | [Español](README.es.md) | [Português](README.pt.md) | [한국어](README.ko.md) | **日本語** | [العربية](README.ar.md) | [Français](README.fr.md) | [Deutsch](README.de.md)

`krot` は、Linux サーバー上で **WireGuard** および **AmneziaWG**
（1.x、2.0、3.1）トンネルを管理するコマンドラインユーティリティです。
インターフェースの作成・削除、クライアント（peer）の管理、クライアント設定、
QR コード、AmneziaVPN キー（`vpn://…`）の生成を行えます。

**krot** という名前は、ロシア語およびその他のいくつかのスラブ言語で
「モグラ」を意味します。モグラはトンネルを掘る動物です。

- 依存関係のない単一の静的バイナリです（Go、`CGO_ENABLED=0`）。
- クライアント操作では**インターフェースを再起動または再作成することはありません**。
  `wg|awg set <if> peer …` という対象を絞った呼び出しのみを行うため、他のクライアントには影響しません。
- インターフェース設定が唯一の信頼できる情報源です。別途データベースは使用せず、
  krot のメタデータはコメントに保存されます。krot が作成していない設定も読み取り、保持されます。
- 人間向けの表、自動化向けの `--json` と終了コードによる、予測可能な出力です。
- krot は NAT、フォワーディング、ファイアウォールルールを設定しません。

完全な技術仕様: [`docs/SPEC.md`](docs/SPEC.md)。

## 要件

| | |
|---|---|
| OS | Ubuntu 22.04 / 24.04 / 26.04、Debian 12 / 13、AlmaLinux（Rocky、RHEL）9 / 10 |
| アーキテクチャ | `amd64`、`arm64` |
| `--type wg` | `wireguard-tools`（`wg`、`wg-quick`）、WireGuard カーネルモジュール |
| `--type awg*` | `amneziawg-tools`（`awg`、`awg-quick`）および AmneziaWG カーネルモジュール、**または** `amneziawg-go` |
| 常に必要 | `iproute2`（`ip`）、`systemd` |
| 権限 | root（`--help`、`--version`、`--doctor`、`--udp-checker` を除く。`--udp-listener` はポート `<1024` を使用する場合、または `/proc/net/udp{,6}` を読み取れない場合のみ root が必要） |

krot はこれらのコンポーネントを自動的にはインストールしません。必要なプログラムがない場合は、
終了コード `6` で終了し、インストールすべきパッケージ名を表示します。

### AmneziaWG バージョン

`--create` の前に、krot はインストール済みの AmneziaWG が要求されたタイプを
サポートしているか確認します。

| タイプ | amneziawg-tools | カーネルモジュール | netlink family | amneziawg-go |
|---|---|---|---|---|
| `awg`（1.0） | 1.0.20240213 | — | ≥ 1 | 0.1.0 |
| `--i1` 付き `awg`（1.5） | 1.0.20250901 | 1.0.20251009 | ≥ 2 | 0.2.13 |
| `awg2`（2.0） | 1.0.20250901 | 1.0.20251009 | ≥ 2 | 0.2.15 |
| `awg3`（3.1） | 3.1.20260812 | 3.1.20260812 | ≥ 3 | 3.1.20260812 |

一部のビルドではプレースホルダーのバージョンが報告されます。例えば、1.x ツールの
`awg --version` はほぼ常に `1.0.20210914` を表示し、`make` でビルドしたモジュールは
`1.0.0` を報告します。このようなバージョンではエラーではなく警告が出ます。
バージョンを確実に検出でき、かつ古すぎる場合にのみ終了コード `6` を返します。
実際にバージョンが古すぎてユニットを開始できない場合、krot は変更をロールバックします。

## インストール

リリースページ <https://github.com/snmp161/krot/releases> からパッケージまたは
バイナリをインストールし、必ずチェックサムを検証してください。

```sh
VERSION=0.7.3
BASE=https://github.com/snmp161/krot/releases/download/v$VERSION

# パッケージとチェックサム一覧をダウンロード
wget "$BASE/krot_${VERSION}_amd64.deb" "$BASE/SHA256SUMS"
# 検証: "krot_..._amd64.deb: OK" と表示されること
gh attestation verify SHA256SUMS --repo snmp161/krot
sha256sum --check --ignore-missing SHA256SUMS
# インストール
sudo dpkg -i "krot_${VERSION}_amd64.deb"
```

RPM ベースのシステム（AlmaLinux、Rocky、RHEL）:

```sh
wget "$BASE/krot-${VERSION}-1.x86_64.rpm" "$BASE/SHA256SUMS"
gh attestation verify SHA256SUMS --repo snmp161/krot
sha256sum --check --ignore-missing SHA256SUMS
sudo rpm -i "krot-${VERSION}-1.x86_64.rpm"
```

パッケージを使わずにバイナリをインストールする場合は、`krot-linux-amd64`
（または `krot-linux-arm64`）と `SHA256SUMS` をダウンロードし、同じ方法で検証してから
`sudo install -m 0755 krot-linux-amd64 /usr/bin/krot` を実行してください。

```sh
wget "$BASE/LICENSE" "$BASE/THIRD_PARTY_NOTICES"
```

`arm64` では、`*_arm64.deb`、`*.aarch64.rpm`、`krot-linux-arm64` の各ファイルを使用します。

## 使用方法

```
krot <アクション> [フラグ]
```

1 回の実行で指定できるアクションは 1 つだけです。フラグは `--flag value` または
`--flag=value` のロング形式のみを使用します。そのアクションに適用できないフラグは
エラー（終了コード `2`）になります。完全なヘルプは `krot --help` で表示できます。

### トンネル

```sh
# AmneziaWG 2.0 を作成（デュアルスタック）。名前は awg0、awg1、… の順に自動選択
sudo krot --create --type awg2 --ip 10.8.0.1/24,fd00:8::1/64 --port 51820 \
          --endpoint vpn.example.com --dns 1.1.1.1,9.9.9.9

# 名前と MTU を明示した WireGuard
sudo krot --create --type wg --interface wg0 --ip 10.9.0.1/24 --port 51821 \
          --endpoint 203.0.113.10 --mtu 1420

# I1 シグネチャパケット付き AmneziaWG 1.5（krot が CPS 構文と長さを検証。最大 2048 文字）
sudo krot --create --type awg --ip 10.10.0.1/24 --port 51822 --endpoint vpn.example.com \
          --i1 '<b 0xf6ab3267fa><r 16>'

# AmneziaWG 3.1（Header Protection、RandomTrailers、DisableCookies は常に有効）
sudo krot --create --type awg3 --ip 10.11.0.1/24 --port 51823 --endpoint vpn.example.com

sudo krot --list                         # すべてのトンネル
sudo krot --list --type awg2 --json      # --type/--interface/--ip/--port フィルター。AND 条件で結合
sudo krot --down --interface awg0        # systemctl stop awg-quick@awg0
sudo krot --up --interface awg0          # systemctl start。--up の再実行では "already up" を表示し、コード 0
sudo krot --delete --interface awg0      # クライアントが存在する場合は確認を要求
sudo krot --delete --interface awg0 --yes
```

### 診断

```sh
krot --doctor   # root 不要: ツール、カーネルモジュール、AmneziaWG バージョン、ディレクトリ
```

`CHECK/STATUS/DETAIL` の表を出力し、ステータスは `ok` / `warn` / `fail` です。
`fail` 項目がなければ終了コードは `0`、あれば `6` です。システムを変更せず、
ロックも取得しません。

#### UDP ポート診断

`krot` は、2 台のホスト間で実際に通過できる UDP ポートを確認できます。
ホスティング事業者やそのアップリンクが DDoS 対策ルールなどによって
一部のポートをフィルタリングしている場合に便利です。

サーバーでリスナーを起動します:

```sh
# 単一ポート
sudo krot --udp-listener --port 51820

# ポート範囲（最大 1024 ポート、ポートごとに 1 ソケット）
sudo krot --udp-listener --port 51820:51830
```

リスナー は 3 種類のデータグラムに応答します:
- `ping SEQ:N` → `pong SEQ:N`;
- `ECHO <text>` → `<text>`;
- `TIME` → 現在の Unix 時刻（ミリ秒）。

クライアントでチェッカーを実行します:

```sh
# 単一ポートを確認
krot --udp-checker --address 203.0.113.1 --port 51820

# ポート範囲を確認
krot --udp-checker --address 203.0.113.1 --port 51820:51830

# echo ペイロードサイズを増やす（1–1400 バイト、デフォルト 64）— 最適な MTU の確認に有用
krot --udp-checker --address 203.0.113.1 --port 51820 --echo-size 1400

# ワーカー数を設定して進捗を表示
krot --udp-checker --address 203.0.113.1 --port 51820:51830 --workers 64 --verbose
```

デフォルトでは、チェッカーはポートごとに各タイプのパケットを 3 個ずつ送信し
（`--count`、1–100）、`--timeout`（デフォルト 2s）内に応答を待ちます。
デフォルトで 32 ポートを並行して確認し、`--workers` で 1–100 個のワーカー を設定できます。
`--verbose` を指定すると、範囲と各ポートの完了結果を stderr に記録します。
リスナーではクライアントアドレスとリクエストタイプを記録します。出力例:

```
PORT     STATUS   PING    ECHO    TIME (min/avg/max)   DETAIL
51820    ok       3/3     3/3     12/24/45 ms          -
51821    filtered 0/3     0/3     -                    timeout
```

ステータス:
- `ok` — すべての ping および echo パケットで有効な応答を受信しました。
- `filtered` — 少なくとも 1 つのパケットがタイムアウトしました（ポートがフィルタリングされている可能性があります）。
- `error` — 無効な応答を受信しました。

`--json` を使用すると、`port`、`status`、`ping_sent`、`ping_ok`、`echo_sent`、`echo_ok`、
`time_sent`、`time_ok`、`rtt_min_ms`、`rtt_avg_ms`、`rtt_max_ms`、`detail` の各フィールドを持つ
オブジェクトの配列を出力します。

`Ctrl+C`/`SIGTERM` を受け取ると、チェッカーはチェックが完了したポートの結果を選択された形式で出力し、
終了コード `10` で終了します。未完了のポートは省略されます。

`--udp-checker` は root 不要で動作します。`--udp-listener` は、範囲に `<1024` のポートが含まれる場合、
または現在のユーザーが `/proc/net/udp` と `/proc/net/udp6` を読み取れない場合のみ root が必要です。
リスナー は `Ctrl+C`/`SIGTERM` まで実行を続けます。

`--list` の例:

```
INTERFACE  TYPE  VERSION  ADDRESS                   PORT   STATUS  CLIENTS
awg0       awg2  2.0      10.8.0.1/24,fd00:8::1/64  51820  up      12 (1 locked)
wg0        wg    -        10.9.0.1/24               51821  down    3
```

デフォルト MTU: `wg` — 1420、`awg`/`awg2` — 1376、`awg3` — 1280。
難読化パラメーター（Jc、S1–S4、H1–H4 など）は安全な範囲で自動生成され、
検証されます（仕様の第 7 節）。

### クライアント

```sh
# クライアントを追加: アドレスは自動割り当て（最初に空いている /32 と /128）;
# クライアント設定は一度だけ出力され、クライアントの秘密鍵は保存されません
sudo krot --add-client --interface awg0 --id phone-sergei --allowed-ips 0.0.0.0/0,::/0

# PresharedKey と端末上の QR コード付き
sudo krot --add-client --interface awg0 --id laptop --allowed-ips 0.0.0.0/0 --psk --qr

# AmneziaVPN キーも出力: 設定の後に vpn://… の行を表示
sudo krot --add-client --interface awg0 --id tablet --allowed-ips 0.0.0.0/0,::/0 --amnezia-key

# KeeneticOS CLI コマンド: Keenetic ルーターをクライアントとして接続（wg のみ、IPv4 のみ）
sudo krot --add-client --interface wg0 --id keenetic --allowed-ips 0.0.0.0/0 --keenetic

# クライアントアドレスを明示
sudo krot --add-client --interface awg0 --id router --allowed-ips 10.8.0.0/24 --ip 10.8.0.50/32

# krot メタデータがなく外部で作成された設定: endpoint を明示（サーバー設定には書き込みません）
sudo krot --add-client --interface office --id guest --allowed-ips 0.0.0.0/0 --endpoint 203.0.113.10

# 単一クライアント用のカスタム DNS（サーバー設定には書き込みません）
sudo krot --add-client --interface awg0 --id kid-phone --allowed-ips 0.0.0.0/0 --dns 94.140.14.15

sudo krot --list-client --interface awg0
sudo krot --list-client --interface awg0 --id phone-sergei --json
sudo krot --lock-client   --interface awg0 --id laptop    # 再実行時は "already locked"、コード 0
sudo krot --unlock-client --interface awg0 --id laptop
sudo krot --delete-client --interface awg0 --id laptop
```

`--allowed-ips` は**クライアント**設定の `AllowedIPs`、つまりクライアントがトンネル経由で
ルーティングする対象を設定します。クライアント DNS には、`--dns` が指定されていればその値を使用します。
指定がなければインターフェースの `# krot:dns` メタデータ（`--create` の `--dns`）を使用し、
それもなければサーバー設定の `[Interface]` にある `DNS` 行を使用します。
どこにも DNS がない場合、クライアント設定に `DNS` 行は含まれません。

クライアントアドレスは、ネットワークアドレス、ブロードキャストアドレス、サーバーアドレスを除き、
インターフェースのサブネットから割り当てられます。ポイントツーポイントサブネット `/31` と `/127`
（RFC 3021、RFC 6164）もサポートされます。両方のアドレスを使用できるため、
`--create … --ip 10.8.0.0/31,fd00:8::/127` で「サーバー + クライアント 1 台」のトンネルを作成できます。

生成された設定はすぐに保存してください。後から再取得することはできず、クライアントを削除して
追加し直す必要があります。QR コードは Unicode のハーフブロックで表示され、
暗い端末背景を前提としています。

`--amnezia-key` は設定の後に AmneziaVPN キーを `vpn://…` の 1 行として出力します。
メッセンジャーなどで送信し、AmneziaVPN アプリの「Insert key」に貼り付けられます。
キーには同じクライアント設定が含まれています。形式は `vpn://` + Qt `qCompress` の
base64url（`=` なし）で、4 バイトのビッグエンディアン長 + zlib です。
このキーは `wg` を含むすべてのタイプで機能します。`--qr` の QR コードには設定テキストが含まれ、
AmneziaVPN と AmneziaWG でスキャンできます。`wg` の場合は WireGuard でもスキャンできます。

`--keenetic` は設定の後に、Keenetic ルーターをトンネルクライアントとして接続するための
KeeneticOS CLI コマンドブロックを出力します。このブロックは KeeneticOS の running-config 構文に従い、
ルーターの CLI（SSH/telnet または Web インターフェースの「Command-line interface」）に
そのまま貼り付けられます。KeeneticOS は AmneziaWG の難読化（ASC パラメーター）をサポートします。
AWG 1.0 は KeeneticOS 4.2 以降、AWG 1.5/2.0 は 5.1 以降でサポートされます。
awg トンネルのブロックには `wireguard asc …` 行が含まれます
（CPS 文字列 I1–I5 は引用符で囲まれます。Web UI から設定ファイルをインポートする方が確実です）。
AWG 3.1 はサポートされません。IPv4 のみです。トンネル MTU がデフォルトの 1420 と異なる場合は
注記が表示されます。形式については仕様の第 6.13 節を参照してください。

`--list-client` の例:

```
ID            PUBLIC KEY  ADDRESS                    STATUS  LAST HANDSHAKE  RX       TX
phone-sergei  aB3d…Xy9=   10.8.0.2/32,fd00:8::2/128  active  1m ago          12.4MiB  1.1GiB
laptop-old    Qw7e…Zz1=   10.8.0.3/32                locked  -               -        -
```

### ミミクリー（I1–I5 パケット）

バージョン 1.5 以降、AmneziaWG は各ハンドシェイクの前に最大 5 つのシグネチャパケット
`I1`–`I5` を送信できます。これらのパケットは、別のプロトコル（DNS、STUN、NTP、SSDP、DTLS）の
先頭のように見えます。krot は `--mimic <profile>` フラグを使って、このようなチェーンを自動生成します。

```sh
# インターフェースの全クライアントに使用するデフォルトプロファイル
krot --create --type awg --ip 10.8.0.1/24 --port 51820 --endpoint vpn.example.com --mimic dns
# 単一クライアントで上書き
krot --add-client --interface awg0 --id phone --allowed-ips 0.0.0.0/0 --mimic stun
```

プロファイル: `dns`（A/AAAA/HTTPS クエリ 3 個）、`stun`（Binding Request）、`ntp`、
`ssdp`（M-SEARCH）、`dtls`（ClientHello）。チェーンは `--add-client` 時にクライアントごとに生成され、
クライアント設定にのみ書き込まれます。I パケットを送信するのはハンドシェイクの開始側なので、
サーバーには不要です。クライアントには AmneziaWG 1.5 以降が必要です。
仕様の第 6.12 節を参照してください。

krot は組み込みプロファイルに加えて、`--create` の `--i1` で用意された CPS 文字列も受け付けます。
例えば QUIC 用の文字列を指定できますが、組み込み QUIC プロファイルは生成されません。
単一の QUIC Initial が 2400–2500 文字となり、2048 文字の制限を超えるためです
（[amneziawg-tools#69](https://github.com/amnezia-vpn/amneziawg-tools/issues/69)）。
用意された文字列は、以下のサードパーティー製ジェネレーターで取得できます。
どちらも MIT ライセンスで、krot とは関係ありません。

- [payloadGen](https://github.com/Sketchystan1/payloadGen) — Web ページです。
  ブラウザーで `index.html` を開き、プロトコルを選択して最初の行（I1）を取得します。
- [awg-multi-script](https://github.com/pumbaX/awg-multi-script) の
  `cpsgen.py`（`src/py/cpsgen.py`）— payloadGen の Python 3 コンソール版で、root は不要です。
  プロファイル: `dns`、`ntp`、`rtp`、`stun`、`sip`、`ssdp`、`webrtc`、`quic`、`curl_quic`。
  第 2 引数はドメインです。

```sh
# ジェネレーターをダウンロードし、実行前にコードを確認
git clone --depth 1 https://github.com/pumbaX/awg-multi-script
less awg-multi-script/src/py/cpsgen.py

# I1 のみを生成して長さを確認（2048 文字以下であること）
I1=$(python3 awg-multi-script/src/py/cpsgen.py dns example.com --only-i1)
echo ${#I1}

sudo krot --create --type awg2 --ip 10.8.0.1/24 --port 51820 --endpoint vpn.example.com --i1 "$I1"
```

注意事項:

- `I1` は `--create` でのみ設定でき、サーバー設定に書き込まれ、すべてのクライアント設定にコピーされます。
  krot は `I2`–`I5` を受け付けません。
- 長さは 2048 文字以下である必要があります（[制限事項](#制限事項)参照）。超えると終了コード `3` です。
  `quic` と `curl_quic` プロファイルのパケットは約 2400 文字なので収まりません。
  短いプロファイルは使用できます: `rtp`（約 30 文字）、`dns` と `ntp`（約 100 文字）、
  `stun`（約 250–350 文字）、`ssdp`（約 400 文字）、`sip`（約 1200 文字）。
- `--i1` 付きの `--type awg` は AmneziaWG 1.5 を意味し、独自の最小バージョン要件があります
  （[AmneziaWG バージョン](#amneziawg-バージョン)参照）。`sip` プロファイルが生成する
  `<rc N>`/`<rd N>` タグは amneziawg-go 0.2.15 以降で認識されるため、
  1.5 ではこれらを含まないプロファイルを推奨します。
- クライアントは AmneziaWG 1.5 以降をサポートしている必要があります。
- `I1` が長い場合、設定が QR コードに収まらないことがあります。この場合 `--qr` は終了コード `3` で
  終了します（クライアントはすでに追加され、設定も出力されています）。
  設定をファイルとして転送するか、`--amnezia-key` を使用してください。

### JSON

`--list --json` は、`interface`、`type`、`version`、`addresses[]`、`port`、`status`、
`clients_total`、`clients_locked` の各フィールドを持つオブジェクトの配列を出力します。
`--list-client --json` は、`id`、`public_key`、`addresses[]`、`status`、
`latest_handshake`（Unix 時刻または `null`）、`rx_bytes`、`tx_bytes`、`endpoint`（または `null`）、
`created`（krot が管理していない peer では `null`）の各フィールドを持つオブジェクトの配列を出力します。
空の結果は `[]` です。

## 終了コード

| コード | 意味 |
|---|---|
| 0 | 成功 |
| 1 | 内部エラーまたは予期しないエラー（実行中のインターフェースへの変更適用失敗を含む — 設定はロールバックされます） |
| 2 | CLI の使用エラー（フラグ、stdin が端末でない場合にクライアントのあるトンネルを `--yes` なしで削除する操作など） |
| 3 | 値の検証エラー（endpoint の不足、「config too large for QR」など） |
| 4 | オブジェクトが見つかりません（インターフェース、クライアント） |
| 5 | 競合: すでに存在する、アドレスまたはポートが使用中、空きアドレスがない |
| 6 | 環境: root 権限がない、ツールがない、サポートされない AmneziaWG バージョン |
| 7 | 10 秒以内に `/run/lock/krot.lock` のロックを取得できませんでした |
| 10 | ユーザーが操作をキャンセルしました |

通常の出力は stdout に、エラーと警告は `krot:` プレフィックス付きで stderr に出力されます。

## データの保存

| タイプ | 設定 | systemd ユニット | ツール |
|---|---|---|---|
| `wg` | `/etc/wireguard/<if>.conf` | `wg-quick@<if>` | `wg` |
| `awg`、`awg2`、`awg3` | `/etc/amnezia/amneziawg/<if>.conf` | `awg-quick@<if>` | `awg` |

メタデータは、`wg-quick`/`awg-quick` が無視するコメントとして保存されます:

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

- ロックされたクライアントは、`#~ ` プレフィックスでコメントアウトされた `[Peer]` ブロックです。
- コメントアウトされた `PreUp`/`PostUp`/`PreDown`/`PostDown` フックは、コメントを外して内容を設定できます
  （例: NAT ルール）。krot はこれらを変更しません。
- krot メタデータのない設定（手動で作成）もサポートされます。`--list` はこれらを表示し、
  `--list-client` は peer を id `-` で出力します。このような peer に `--id` を使った操作は適用できません。
  書き込み時には、元の行、コメント、順序をすべてバイト単位で保持します。
- 書き込みはアトミックです（一時ファイル → `fsync` → `rename` → ディレクトリの `fsync`）。
  パーミッションは `0600` です。
- すべての変更操作は `/run/lock/krot.lock` に対して排他的な `flock` を取得します。
  同時実行（例: Salt と人手の操作）でも変更が失われません。
- krot はバックアップを作成しません。

## 制限事項

- NAT、フォワーディング、ファイアウォール、sysctl は krot の対象外です。
  `PostUp`/`PostDown` フックを使用してください。
- WireGuard/AmneziaWG のインストールとトンネルタイプ間の移行はサポートされません。
  ミミクリーのシグネチャ生成には組み込みプロファイル（`--mimic`）を使用します。
  QUIC プロファイルとカスタムシグネチャには `--i1` とサードパーティー製ジェネレーターを使用します。
  [ミミクリー](#ミミクリーi1i5-パケット)を参照してください。
- `--i1` は 2048 文字を超えてはいけません。
  [amneziawg-tools#69](https://github.com/amnezia-vpn/amneziawg-tools/issues/69) により、
  長い `I1`–`I5`（おおよそ 3600 文字以上）は `awg show` をハングさせ、
  `awg set`/`setconf` を失敗させます。
- クライアントの秘密鍵は保存されず、クライアント設定を再発行することもできません。

## 開発

`go.mod` で指定された Go（現在は 1.27.4）と、統合テスト用の Docker が必要です。

```sh
make check        # すべてのローカルチェック: 書式、テスト、モジュール、セキュリティ、workflow リンター
make fuzz-smoke   # 全 targets の時間制限付き fuzzing
make build        # 静的 bin/krot（CGO_ENABLED=0）。バージョンは git describe から取得
make build-arm64
make integration  # systemd を備えた特権 Docker コンテナで統合テストを実行
make changelog      # リリースノートから changelog を再生成
make repro-check    # バイト単位で再現可能なビルドを検証
make package-preflight # 両アーキテクチャの DEB/RPM + Ubuntu 24.04/AlmaLinux 9 smoke
```

リポジトリのルール: [`AGENTS.md`](AGENTS.md) — 仕様優先、コードより先にテスト、
1 タスクにつき 1 コミット。

- **単体テスト**は root を必要とせず、システムを変更しません。ディレクトリは一時ディレクトリに置き換え、
  外部コマンド（`wg`、`awg`、`systemctl`、`ip` など）は呼び出しを記録するフェイクで処理します。
  クライアント操作が `wg|awg show` と `wg|awg set <if> peer …` のみを呼び出すことを
  個別のテストで検証します。
- **ゴールデンファイル**は各パッケージの `testdata/` にあります。生成された出力（`internal/app`、`internal/qr`）は、
  `go test ./internal/app ./internal/qr -update` を使って意図的に更新する場合のみ変更し、
  コミットで理由を説明します。その他の ゴールデン ファイルは仕様に基づいて手動で作成します。
- **統合テスト**（`test/integration`、ビルドタグ `integration`）には root と systemd が必要で、
  分離されたネットワークネームスペース内でのみ実行されます。テスト用インターフェースユニットは
  `NetworkNamespacePath` drop-in によって netns に配置されます。すべてのタイプの完全なライフサイクル、
  クライアント操作の透過性（パケットロスのない継続的な ping、変化しない ifindex とハンドシェイク）、IPv6、
  外部で作成された設定、同時実行をカバーします。ローカルでは `scripts/integration-docker.sh`、CI では `sudo` で実行します。
- `scripts/collect-versions.sh` は ゴールデン テスト用に実際の AmneziaWG バージョン出力を再収集します。
- **CI**（`.github/workflows/ci.yml`）: ローカル品質チェック、shuffle・`-race`・
  最低 90% のカバレッジ付き単体テスト、時間制限付き fuzzing、統合テスト、
  package preflight、`amd64`/`arm64` ビルド。
  **リリース**（`release.yml`、タグ `v*`）: バイナリ、`.deb`/`.rpm` パッケージ
  （nfpm、`packaging/nfpm.yaml`）、すべての OS でのパッケージスモークテスト（`scripts/smoke.sh`）、
  `SHA256SUMS` 付き GitHub Release。

## ライセンス

[Apache License 2.0](LICENSE)。Copyright 2026 Sergey Ovchinnikov。
