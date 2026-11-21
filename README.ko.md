# krot

[English](README.md) | [Русский](README.ru.md) | [简体中文](README.zh.md) | [繁體中文](README.zh-Hant.md) | [Español](README.es.md) | [Português](README.pt.md) | **한국어** | [日本語](README.ja.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [Deutsch](README.de.md)

`krot`은 Linux 서버에서 **WireGuard** 및 **AmneziaWG**
(1.x, 2.0, 3.1) 터널을 관리하는 명령줄 유틸리티입니다. 인터페이스를 생성하고 삭제하며,
클라이언트(peer)를 관리하고, 클라이언트 구성, QR 코드, AmneziaVPN 키(`vpn://…`)를 생성합니다.

**krot**이라는 이름은 러시아어와 몇몇 다른 슬라브어에서 "두더지"를 뜻합니다.
두더지는 터널을 파는 동물입니다.

- 의존성이 없는 단일 정적 바이너리입니다(Go, `CGO_ENABLED=0`).
- 클라이언트 작업은 **인터페이스를 재시작하거나 재생성하지 않습니다**.
  `wg|awg set <if> peer …` 호출만 대상으로 수행하므로 다른 클라이언트에는 영향이 없습니다.
- 인터페이스 구성 파일을 유일하게 신뢰할 수 있는 정보 소스로 사용합니다. 별도의 데이터베이스가 없으며,
  krot 메타데이터는 주석에 저장됩니다. krot이 생성하지 않은 구성도 읽고 보존합니다.
- 사람이 읽기 위한 표와 자동화를 위한 `--json` 및 종료 코드를 제공하는 예측 가능한 출력입니다.
- krot은 NAT, 포워딩 또는 방화벽 규칙을 구성하지 않습니다.

전체 기술 명세: [`docs/SPEC.md`](docs/SPEC.md).

## 요구 사항

| | |
|---|---|
| OS | Ubuntu 22.04 / 24.04 / 26.04, Debian 12 / 13, AlmaLinux (Rocky, RHEL) 9 / 10 |
| 아키텍처 | `amd64`, `arm64` |
| `--type wg` | `wireguard-tools`(`wg`, `wg-quick`), WireGuard 커널 모듈 |
| `--type awg*` | `amneziawg-tools`(`awg`, `awg-quick`)와 AmneziaWG 커널 모듈 **또는** `amneziawg-go` |
| 항상 필요 | `iproute2`(`ip`), `systemd` |
| 권한 | root(`--help`, `--version`, `--doctor`, `--udp-checker` 제외; `--udp-listener`는 포트 `<1024`를 사용하거나 `/proc/net/udp{,6}`를 읽을 수 없는 경우에만 root 필요) |

krot은 이러한 구성 요소를 직접 설치하지 않습니다. 필요한 프로그램이 없으면
종료 코드 `6`으로 종료하고 설치해야 할 패키지 이름을 표시합니다.

### AmneziaWG 버전

`--create`를 실행하기 전에 krot은 설치된 AmneziaWG가 요청한 유형을 지원하는지 확인합니다.

| 유형 | amneziawg-tools | 커널 모듈 | netlink family | amneziawg-go |
|---|---|---|---|---|
| `awg`(1.0) | 1.0.20240213 | — | ≥ 1 | 0.1.0 |
| `--i1`을 사용하는 `awg`(1.5) | 1.0.20250901 | 1.0.20251009 | ≥ 2 | 0.2.13 |
| `awg2`(2.0) | 1.0.20250901 | 1.0.20251009 | ≥ 2 | 0.2.15 |
| `awg3`(3.1) | 3.1.20260812 | 3.1.20260812 | ≥ 3 | 3.1.20260812 |

일부 빌드는 자리 표시자 버전을 보고합니다. 예를 들어 1.x 도구의 `awg --version`은
거의 항상 `1.0.20210914`를 출력하고, `make`로 빌드한 모듈은 `1.0.0`을 보고합니다.
이러한 버전은 오류가 아닌 경고를 생성합니다. 버전을 신뢰성 있게 감지했고
실제로 너무 오래된 경우에만 종료 코드 `6`을 반환합니다. 실제로 버전이 너무 오래되어
유닛이 시작되지 않으면 krot은 변경 사항을 롤백합니다.

## 설치

릴리스 페이지 <https://github.com/snmp161/krot/releases>에서 패키지 또는 바이너리를
설치하고, 항상 체크섬을 검증하십시오.

```sh
VERSION=0.7.3
BASE=https://github.com/snmp161/krot/releases/download/v$VERSION

# 패키지와 체크섬 목록 다운로드
wget "$BASE/krot_${VERSION}_amd64.deb" "$BASE/SHA256SUMS"
# 검증: "krot_..._amd64.deb: OK"가 출력되어야 함
sha256sum --check --ignore-missing SHA256SUMS
# 설치
sudo dpkg -i "krot_${VERSION}_amd64.deb"
```

RPM 기반 시스템(AlmaLinux, Rocky, RHEL):

```sh
wget "$BASE/krot-${VERSION}-1.x86_64.rpm" "$BASE/SHA256SUMS"
sha256sum --check --ignore-missing SHA256SUMS
sudo rpm -i "krot-${VERSION}-1.x86_64.rpm"
```

패키지 없이 바이너리를 설치하려면 `krot-linux-amd64`(또는 `krot-linux-arm64`)와
`SHA256SUMS`를 다운로드하고 같은 방법으로 검증한 후,
`sudo install -m 0755 krot-linux-amd64 /usr/bin/krot`을 실행하십시오.

`arm64`에서는 `*_arm64.deb`, `*.aarch64.rpm`, `krot-linux-arm64` 파일을 사용하십시오.

## 사용법

```
krot <작업> [플래그]
```

실행할 때마다 정확히 하나의 작업만 지정합니다. 플래그는 `--flag value` 또는
`--flag=value` 형태의 긴 형식만 사용합니다. 해당 작업에 적용되지 않는 플래그는
오류(종료 코드 `2`)입니다. 전체 도움말은 `krot --help`로 확인할 수 있습니다.

### 터널

```sh
# AmneziaWG 2.0 생성(듀얼 스택); 이름은 awg0, awg1, … 순으로 자동 선택됨
sudo krot --create --type awg2 --ip 10.8.0.1/24,fd00:8::1/64 --port 51820 \
          --endpoint vpn.example.com --dns 1.1.1.1,9.9.9.9

# 명시적 이름과 MTU를 사용하는 WireGuard
sudo krot --create --type wg --interface wg0 --ip 10.9.0.1/24 --port 51821 \
          --endpoint 203.0.113.10 --mtu 1420

# I1 시그니처 패킷을 사용하는 AmneziaWG 1.5(krot이 CPS 구문과 길이 검증 — 최대 2048자)
sudo krot --create --type awg --ip 10.10.0.1/24 --port 51822 --endpoint vpn.example.com \
          --i1 '<b 0xf6ab3267fa><r 16>'

# AmneziaWG 3.1(Header Protection, RandomTrailers, DisableCookies는 항상 활성화됨)
sudo krot --create --type awg3 --ip 10.11.0.1/24 --port 51823 --endpoint vpn.example.com

sudo krot --list                         # 모든 터널
sudo krot --list --type awg2 --json      # --type/--interface/--ip/--port 필터, AND로 결합됨
sudo krot --down --interface awg0        # systemctl stop awg-quick@awg0
sudo krot --up --interface awg0          # systemctl start; --up을 반복하면 "already up" 출력, 코드 0
sudo krot --delete --interface awg0      # 클라이언트가 있으면 확인을 요청함
sudo krot --delete --interface awg0 --yes
```

### 진단

```sh
krot --doctor   # root 없이 작동: 도구, 커널 모듈, AmneziaWG 버전, 디렉터리
```

`CHECK/STATUS/DETAIL` 표를 출력하며 상태는 `ok` / `warn` / `fail`입니다.
`fail` 항목이 없으면 종료 코드는 `0`, 있으면 `6`입니다. 아무것도 변경하지 않으며
잠금도 획득하지 않습니다.

#### UDP 포트 진단

`krot`은 두 호스트 사이에서 실제로 통과할 수 있는 UDP 포트를 확인할 수 있습니다.
호스팅 제공업체나 그 업링크가 DDoS 규칙 등으로 인해 일부 포트를 필터링할 때 유용합니다.

서버에서 listener를 시작하십시오:

```sh
# 단일 포트
sudo krot --udp-listener --port 51820

# 포트 범위(최대 1024개 포트, 포트당 소켓 하나)
sudo krot --udp-listener --port 51820:51830
```

Listener는 세 가지 데이터그램에 응답합니다:
- `ping SEQ:N` → `pong SEQ:N`;
- `ECHO <text>` → `<text>`;
- `TIME` → 현재 Unix 시간(밀리초).

클라이언트에서 checker를 실행하십시오:

```sh
# 단일 포트 확인
krot --udp-checker --address 203.0.113.1 --port 51820

# 포트 범위 확인
krot --udp-checker --address 203.0.113.1 --port 51820:51830

# echo 페이로드 크기 늘리기(1–1400바이트, 기본값 64) — 최적 MTU를 찾는 데 유용함
krot --udp-checker --address 203.0.113.1 --port 51820 --echo-size 1400

# 워커 수를 설정하고 진행 상황 표시
krot --udp-checker --address 203.0.113.1 --port 51820:51830 --workers 64 --verbose
```

기본적으로 checker는 포트마다 각 유형의 패킷을 3개씩 보내고(`--count`, 1–100)
`--timeout`(기본값 2s) 이내에 응답을 기다립니다. 기본적으로 32개 포트를 동시에 검사하며,
`--workers`로 1–100개의 워커를 설정할 수 있습니다. `--verbose`는 범위와 각 포트의
검사 완료 결과를 stderr에 기록합니다. listener에서는 클라이언트 주소와 요청 유형을 기록합니다.
출력 예:

```
PORT     STATUS   PING    ECHO    TIME (min/avg/max)   DETAIL
51820    ok       3/3     3/3     12/24/45 ms          -
51821    filtered 0/3     0/3     -                    timeout
```

상태:
- `ok` — 모든 ping 및 echo 패킷이 유효한 응답을 받았습니다.
- `filtered` — 하나 이상의 패킷에 대한 응답이 제한 시간 내에 도착하지 않았습니다(포트가 필터링되었을 가능성이 높습니다).
- `error` — 잘못된 응답을 받았습니다.

`--json`을 사용하면 `port`, `status`, `ping_sent`, `ping_ok`, `echo_sent`, `echo_ok`,
`time_sent`, `time_ok`, `rtt_min_ms`, `rtt_avg_ms`, `rtt_max_ms`, `detail` 필드를 가진
객체 배열을 출력합니다.

`Ctrl+C`/`SIGTERM`을 받으면 checker는 선택한 형식으로 완전히 검사된 포트의 결과를 출력하고
종료 코드 `10`으로 종료합니다. 완료되지 않은 포트는 생략합니다.

`--udp-checker`는 root 없이 작동합니다. `--udp-listener`는 범위에 `<1024` 포트가 포함되거나
현재 사용자가 `/proc/net/udp`와 `/proc/net/udp6`를 읽을 수 없는 경우에만 root가 필요합니다.
Listener는 `Ctrl+C`/`SIGTERM`을 받을 때까지 실행됩니다.

`--list` 예:

```
INTERFACE  TYPE  VERSION  ADDRESS                   PORT   STATUS  CLIENTS
awg0       awg2  2.0      10.8.0.1/24,fd00:8::1/64  51820  up      12 (1 locked)
wg0        wg    -        10.9.0.1/24               51821  down    3
```

기본 MTU: `wg` — 1420, `awg`/`awg2` — 1376, `awg3` — 1280.
난독화 매개변수(Jc, S1–S4, H1–H4 등)는 안전한 범위에서 자동으로 생성되고
검증됩니다(명세 7절).

### 클라이언트

```sh
# 클라이언트 추가: 주소 자동 할당(첫 번째 사용 가능한 /32 및 /128);
# 클라이언트 구성은 한 번만 출력되며 클라이언트 개인 키는 절대 저장되지 않음
sudo krot --add-client --interface awg0 --id phone-sergei --allowed-ips 0.0.0.0/0,::/0

# PresharedKey와 터미널 QR 코드 포함
sudo krot --add-client --interface awg0 --id laptop --allowed-ips 0.0.0.0/0 --psk --qr

# AmneziaVPN 키 추가 출력: 구성 뒤에 vpn://… 줄 출력
sudo krot --add-client --interface awg0 --id tablet --allowed-ips 0.0.0.0/0,::/0 --amnezia-key

# KeeneticOS CLI 명령: Keenetic 라우터를 클라이언트로 연결(wg 전용, IPv4 전용)
sudo krot --add-client --interface wg0 --id keenetic --allowed-ips 0.0.0.0/0 --keenetic

# 특정 클라이언트 주소 지정
sudo krot --add-client --interface awg0 --id router --allowed-ips 10.8.0.0/24 --ip 10.8.0.50/32

# krot 메타데이터가 없는 외부 구성: endpoint를 명시적으로 전달(서버 구성에 기록하지 않음)
sudo krot --add-client --interface office --id guest --allowed-ips 0.0.0.0/0 --endpoint 203.0.113.10

# 단일 클라이언트용 사용자 지정 DNS(서버 구성에 기록하지 않음)
sudo krot --add-client --interface awg0 --id kid-phone --allowed-ips 0.0.0.0/0 --dns 94.140.14.15

sudo krot --list-client --interface awg0
sudo krot --list-client --interface awg0 --id phone-sergei --json
sudo krot --lock-client   --interface awg0 --id laptop    # 반복 실행 — "already locked", 코드 0
sudo krot --unlock-client --interface awg0 --id laptop
sudo krot --delete-client --interface awg0 --id laptop
```

`--allowed-ips`는 **클라이언트** 구성의 `AllowedIPs`를 설정합니다. 즉, 클라이언트가
터널을 통해 라우팅할 대상입니다. 클라이언트 DNS는 `--dns`가 지정되면 그 값을 사용합니다.
지정되지 않으면 인터페이스의 `# krot:dns` 메타데이터(`--create`의 `--dns`)를 사용하고,
그것도 없으면 서버 구성 `[Interface]`의 `DNS` 줄을 사용합니다. 어디에도 DNS가 없으면
클라이언트 구성에 `DNS` 줄이 포함되지 않습니다.

클라이언트 주소는 네트워크 주소, 브로드캐스트 주소, 서버 주소를 제외하고
인터페이스 서브넷에서 할당됩니다. 포인트-투-포인트 서브넷 `/31`과 `/127`
(RFC 3021, RFC 6164)도 지원합니다. 두 주소를 모두 사용할 수 있으므로
`--create … --ip 10.8.0.0/31,fd00:8::/127`로 "서버 + 클라이언트 하나" 터널을 만들 수 있습니다.

생성된 구성은 즉시 저장하십시오. 나중에 다시 가져올 수 없으며, 클라이언트를 삭제하고
다시 추가해야만 합니다. QR 코드는 Unicode 하프 블록으로 출력되며 어두운 터미널 배경을 전제로 합니다.

`--amnezia-key`는 구성 뒤에 AmneziaVPN 키를 한 줄의 `vpn://…` 형태로 출력합니다.
메신저 등으로 전달한 후 AmneziaVPN 앱의 "Insert key"에 붙여 넣을 수 있습니다.
키에는 동일한 클라이언트 구성이 들어 있습니다. `vpn://` + Qt `qCompress`의
base64url(`=` 없음)이며, 형식은 4바이트 빅 엔디언 길이 + zlib입니다.
이 키는 `wg`를 포함한 모든 유형에서 작동합니다. `--qr`의 QR 코드에는 구성 텍스트가
들어 있으므로 AmneziaVPN과 AmneziaWG에서 스캔할 수 있으며, `wg`의 경우 WireGuard에서도 스캔할 수 있습니다.

`--keenetic`은 구성 뒤에 Keenetic 라우터를 터널 클라이언트로 연결하기 위한
KeeneticOS CLI 명령 블록을 출력합니다. 이 블록은 KeeneticOS running-config 구문을 따르므로
라우터 CLI(SSH/telnet 또는 웹 인터페이스의 "Command-line interface")에 그대로 붙여 넣을 수 있습니다.
KeeneticOS는 AmneziaWG 난독화(ASC 매개변수)를 지원합니다. AWG 1.0은 KeeneticOS 4.2부터,
AWG 1.5/2.0은 5.1부터 지원합니다. awg 터널의 블록에는 `wireguard asc …` 줄이 포함됩니다
(CPS 문자열 I1–I5는 따옴표로 묶이며, 웹 UI에서 구성 파일을 가져오는 방식이 더 안정적입니다).
AWG 3.1은 지원되지 않습니다. IPv4만 지원하며, 터널 MTU가 기본값 1420과 다르면 안내가 출력됩니다.
형식은 명세 6.13절을 참조하십시오.

`--list-client` 예:

```
ID            PUBLIC KEY  ADDRESS                    STATUS  LAST HANDSHAKE  RX       TX
phone-sergei  aB3d…Xy9=   10.8.0.2/32,fd00:8::2/128  active  1m ago          12.4MiB  1.1GiB
laptop-old    Qw7e…Zz1=   10.8.0.3/32                locked  -               -        -
```

### 미미크리(I1–I5 패킷)

버전 1.5부터 AmneziaWG는 각 핸드셰이크 전에 최대 다섯 개의 시그니처 패킷
`I1`–`I5`를 보낼 수 있습니다. 이 패킷은 다른 프로토콜(DNS, STUN, NTP, SSDP, DTLS)의
시작 부분처럼 보입니다. krot은 `--mimic <profile>` 플래그를 사용해 이러한 체인을 직접 생성합니다.

```sh
# 인터페이스의 모든 클라이언트에 사용할 기본 프로필
krot --create --type awg --ip 10.8.0.1/24 --port 51820 --endpoint vpn.example.com --mimic dns
# 단일 클라이언트에서 재정의
krot --add-client --interface awg0 --id phone --allowed-ips 0.0.0.0/0 --mimic stun
```

프로필: `dns`(A/AAAA/HTTPS 쿼리 3개), `stun`(Binding Request), `ntp`,
`ssdp`(M-SEARCH), `dtls`(ClientHello). 체인은 `--add-client` 시 생성되며
클라이언트마다 고유하고 클라이언트 구성에만 기록됩니다. I 패킷은 핸드셰이크 시작자가
내므로 서버에는 필요하지 않습니다. 클라이언트에는 AmneziaWG 1.5 이상이 필요합니다.
명세 6.12절을 참조하십시오.

krot은 내장 프로필 외에도 `--create`에서 `--i1`을 통해 미리 만들어진 CPS 문자열을 받을 수 있습니다.
예를 들어 QUIC이 가능하지만, 내장 QUIC 프로필은 생성하지 않습니다. 단일 QUIC Initial이
2400–2500자로 2048자 제한을 초과하기 때문입니다
([amneziawg-tools#69](https://github.com/amnezia-vpn/amneziawg-tools/issues/69)).
미리 만들어진 문자열은 다음 서드파티 생성기에서 얻을 수 있습니다. 두 프로젝트 모두 MIT 라이선스이며
krot과 관련이 없습니다.

- [payloadGen](https://github.com/Sketchystan1/payloadGen) — 웹 페이지입니다.
  브라우저에서 `index.html`을 열고 프로토콜을 선택한 후 첫 번째 줄(I1)을 가져옵니다.
- [awg-multi-script](https://github.com/pumbaX/awg-multi-script)의
  `cpsgen.py`(`src/py/cpsgen.py`) — payloadGen의 Python 3 콘솔 버전으로 root가 필요 없습니다.
  프로필: `dns`, `ntp`, `rtp`, `stun`, `sip`, `ssdp`, `webrtc`, `quic`, `curl_quic`.
  두 번째 인수는 도메인입니다.

```sh
# 생성기를 다운로드하고 실행하기 전에 코드를 검토함
git clone --depth 1 https://github.com/pumbaX/awg-multi-script
less awg-multi-script/src/py/cpsgen.py

# I1만 생성하고 길이 확인(2048자 이하여야 함)
I1=$(python3 awg-multi-script/src/py/cpsgen.py dns example.com --only-i1)
echo ${#I1}

sudo krot --create --type awg2 --ip 10.8.0.1/24 --port 51820 --endpoint vpn.example.com --i1 "$I1"
```

참고 사항:

- `I1`은 `--create`에서만 설정되며 서버 구성에 기록되고 모든 클라이언트 구성에 복사됩니다.
  krot은 `I2`–`I5`를 받지 않습니다.
- 길이는 2048자를 초과할 수 없습니다([제한 사항](#제한-사항) 참조). 초과하면 종료 코드 `3`입니다.
  `quic` 및 `curl_quic` 프로필 패킷은 약 2400자이므로 들어가지 않습니다.
  더 짧은 프로필은 사용할 수 있습니다. `rtp`(약 30자), `dns` 및 `ntp`(약 100자),
  `stun`(약 250–350자), `ssdp`(약 400자), `sip`(약 1200자).
- `--i1`과 함께 사용하는 `--type awg`는 AmneziaWG 1.5를 의미하며 별도의 최소 버전이 있습니다
  ([AmneziaWG 버전](#amneziawg-버전) 참조). `sip` 프로필이 생성하는 `<rc N>`/`<rd N>` 태그는
  amneziawg-go 0.2.15부터 지원되므로 1.5에서는 이 태그가 없는 프로필을 권장합니다.
- 클라이언트는 AmneziaWG 1.5 이상을 지원해야 합니다.
- `I1`이 길면 구성이 QR 코드에 들어가지 않을 수 있습니다. 이 경우 `--qr`은 종료 코드 `3`으로
  종료합니다(클라이언트는 이미 추가되었고 구성도 출력된 상태입니다).
  구성을 파일로 전달하거나 `--amnezia-key`를 사용하십시오.

### JSON

`--list --json`은 `interface`, `type`, `version`, `addresses[]`, `port`, `status`,
`clients_total`, `clients_locked` 필드를 가진 객체 배열을 출력합니다.
`--list-client --json`은 `id`, `public_key`, `addresses[]`, `status`,
`latest_handshake`(Unix 시간 또는 `null`), `rx_bytes`, `tx_bytes`, `endpoint`(또는 `null`),
`created`(krot이 관리하지 않는 peer는 `null`) 필드를 가진 객체 배열을 출력합니다.
결과가 비어 있으면 `[]`입니다.

## 종료 코드

| 코드 | 의미 |
|---|---|
| 0 | 성공 |
| 1 | 내부 또는 예기치 않은 오류(실행 중인 인터페이스에 변경을 적용하지 못한 경우 포함 — 구성 롤백됨) |
| 2 | CLI 사용 오류(플래그; stdin이 터미널이 아닐 때 클라이언트가 있는 터널을 `--yes` 없이 삭제하는 경우) |
| 3 | 값 검증 오류(endpoint 누락, "config too large for QR" 등) |
| 4 | 객체를 찾을 수 없음(인터페이스, 클라이언트) |
| 5 | 충돌: 이미 존재함, 주소 또는 포트 사용 중, 사용 가능한 주소 없음 |
| 6 | 환경: root 없음, 도구 누락, 지원되지 않는 AmneziaWG 버전 |
| 7 | 10초 이내에 `/run/lock/krot.lock` 잠금을 획득하지 못함 |
| 10 | 사용자가 작업을 취소함 |

일반 출력은 stdout으로, 오류와 경고는 `krot:` 접두사와 함께 stderr로 출력됩니다.

## 데이터 저장

| 유형 | 구성 | systemd 유닛 | 도구 |
|---|---|---|---|
| `wg` | `/etc/wireguard/<if>.conf` | `wg-quick@<if>` | `wg` |
| `awg`, `awg2`, `awg3` | `/etc/amnezia/amneziawg/<if>.conf` | `awg-quick@<if>` | `awg` |

메타데이터는 `wg-quick`/`awg-quick`이 무시하는 주석으로 저장됩니다:

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

- 잠긴 클라이언트는 `#~ ` 접두사로 주석 처리된 `[Peer]` 블록입니다.
- 주석 처리된 `PreUp`/`PostUp`/`PreDown`/`PostDown` 훅은 주석을 해제하고 내용을 채울 수 있습니다
  (예: NAT 규칙). krot은 이를 수정하지 않습니다.
- krot 메타데이터가 없는 구성(수동 생성)도 지원됩니다. `--list`는 이를 표시하고,
  `--list-client`는 peer를 id `-`와 함께 출력합니다. 이러한 peer에는 `--id` 작업을 적용할 수 없습니다.
  쓰기 작업은 원래 줄, 주석, 순서를 바이트 단위로 그대로 보존합니다.
- 쓰기는 원자적입니다(임시 파일 → `fsync` → `rename` → 디렉터리 `fsync`). 권한은 `0600`입니다.
- 모든 변경 작업은 `/run/lock/krot.lock`에 배타적 `flock`을 획득합니다.
  동시 실행(예: Salt와 사람)에서도 변경 사항이 유실되지 않습니다.
- krot은 백업을 만들지 않습니다.

## 제한 사항

- NAT, 포워딩, 방화벽, sysctl은 krot의 범위 밖입니다. `PostUp`/`PostDown` 훅을 사용하십시오.
- WireGuard/AmneziaWG 설치와 터널 유형 간 마이그레이션은 지원되지 않습니다.
  미미크리 시그니처 생성에는 내장 프로필(`--mimic`)을 사용합니다. QUIC 프로필과 사용자 지정
  시그니처는 `--i1` 및 서드파티 생성기를 사용합니다. [미미크리](#미미크리i1i5-패킷)를 참조하십시오.
- `--i1`은 2048자를 초과할 수 없습니다.
  [amneziawg-tools#69](https://github.com/amnezia-vpn/amneziawg-tools/issues/69) 때문에
  긴 `I1`–`I5`(약 3600자 이상)는 `awg show`를 멈추게 하고 `awg set`/`setconf`를 실패하게 합니다.
- 클라이언트 개인 키는 저장되지 않으며, 클라이언트 구성을 다시 발급할 수 없습니다.

## 개발

`go.mod`에 지정된 Go(현재 1.27.4)와 통합 테스트용 Docker가 필요합니다.

```sh
make check        # 모든 로컬 검사: 형식, 테스트, 모듈, 보안 및 workflow 린터
make fuzz-smoke   # 모든 targets의 시간 제한 fuzzing
make build        # 정적 bin/krot(CGO_ENABLED=0), 버전은 git describe에서 가져옴
make build-arm64
make integration  # systemd를 실행하는 특권 Docker 컨테이너에서 통합 테스트 실행
make package-preflight # 양쪽 아키텍처용 DEB/RPM + Ubuntu 24.04/AlmaLinux 9 smoke
```

리포지터리 규칙: [`AGENTS.md`](AGENTS.md) — 명세 우선, 코드보다 테스트 먼저,
작업당 커밋 하나.

- **단위 테스트**는 root가 필요 없고 시스템을 변경하지 않습니다. 디렉터리는 임시 디렉터리로
  대체하고 외부 명령(`wg`, `awg`, `systemctl`, `ip` 등)은 호출을 기록하는 가짜 실행기로 처리합니다.
  별도의 테스트는 클라이언트 작업이 `wg|awg show`와 `wg|awg set <if> peer …`만 호출하는지 검증합니다.
- **골든 파일**은 각 패키지의 `testdata/`에 있습니다. 생성된 출력(`internal/app`, `internal/qr`)은
  `go test ./internal/app ./internal/qr -update`로만 의도적으로 업데이트하고 커밋에 이유를 기록합니다.
  다른 골든 파일은 명세에 따라 수동으로 작성합니다.
- **통합 테스트**(`test/integration`, 빌드 태그 `integration`)에는 root와 systemd가 필요하며,
  별도의 네트워크 네임스페이스에서만 실행됩니다. 테스트 인터페이스 유닛은
  `NetworkNamespacePath` drop-in을 통해 netns에 배치됩니다. 모든 유형의 전체 수명 주기,
  클라이언트 작업의 투명성(손실 없는 연속 ping, 변경되지 않는 ifindex와 핸드셰이크), IPv6,
  외부 구성, 동시 실행을 검증합니다. 로컬에서는 `scripts/integration-docker.sh`를 사용하고,
  CI에서는 `sudo`로 실행합니다.
- `scripts/collect-versions.sh`는 골든 테스트를 위해 실제 AmneziaWG 버전 출력을 다시 수집합니다.
- **CI**(`.github/workflows/ci.yml`): 로컬 품질 검사, shuffle·`-race` 및 최소 90%
  커버리지를 적용한 단위 테스트, 시간 제한 fuzzing, 통합 테스트, package preflight,
  `amd64`/`arm64` 빌드.
  **릴리스**(`release.yml`, 태그 `v*`): 바이너리, `.deb`/`.rpm` 패키지(nfpm, `packaging/nfpm.yaml`),
  모든 OS에서 패키지 스모크 테스트(`scripts/smoke.sh`), `SHA256SUMS`가 포함된 GitHub Release.

## 라이선스

[Apache License 2.0](LICENSE). Copyright 2026 Sergey Ovchinnikov.
