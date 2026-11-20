package cli

const usageTextKorean = `krot — WireGuard 및 AmneziaWG 터널 관리

사용법:
  krot <작업> [플래그]

작업(정확히 하나):
  --list            터널을 나열합니다
  --create          터널을 생성합니다
  --delete          터널을 삭제합니다
  --up              인터페이스를 시작합니다
  --down            인터페이스를 중지합니다
  --add-client      클라이언트를 추가합니다
  --delete-client   클라이언트를 삭제합니다
  --lock-client     클라이언트를 잠급니다
  --unlock-client   클라이언트 잠금을 해제합니다
  --list-client     클라이언트를 나열합니다
  --doctor          환경을 검사합니다(도구, 커널 모듈, AmneziaWG 버전)
  --udp-listener    포트 진단용 UDP 리스너
  --udp-checker     포트 진단용 UDP 체커
  --help            이 도움말을 표시합니다
  --version         버전, 커밋 및 빌드 날짜를 표시합니다

플래그(--flag value 또는 --flag=value):
  --type wg|awg|awg2|awg3   프로토콜 유형(create: 필수; list: 필터)
  --interface NAME          인터페이스 이름, 최대 15자 [a-zA-Z0-9_.=+-]
  --ip CIDR[,CIDR]          create: 서버 주소(10.8.0.1/24,fd00:8::1/64);
                            add-client: 클라이언트 주소(/32, /128);
                            list: 주소 필터
  --port N|N:M              UDP 포트 또는 범위 1..65535, 최대 1024개 포트
                            (create, udp-listener, udp-checker: 필수; list: 필터)
  --endpoint HOST           클라이언트 구성에 사용할 서버 공개 주소
                            (create: 필수; add-client: 클라이언트별 재정의)
  --address HOST            --udp-checker의 원격 리스너 주소(포트 제외)
  --mtu N                   MTU 576..9000(기본값: wg 1420, awg/awg2 1376, awg3 1280)
  --dns IP[,IP]             클라이언트 구성의 DNS 서버
                            (create: 모든 클라이언트; add-client: 이 클라이언트만)
  --i1 CPS                  AmneziaWG I1 시그니처 패킷(awg* 전용, 최대 2048자)
  --mimic PROFILE           클라이언트 구성용 I1-I5 미믹 체인을 생성합니다:
                            dns|stun|ntp|ssdp|dtls(awg* 전용; --i1과 함께 사용 불가)
  --id ID                   클라이언트 ID [a-zA-Z0-9_.-]{1,64}, 인터페이스 내에서 고유해야 합니다
  --allowed-ips CIDR[,CIDR] 클라이언트 구성의 AllowedIPs, 예: 0.0.0.0/0,::/0
  --psk                     클라이언트의 PresharedKey를 생성합니다
  --qr                      클라이언트 구성을 QR 코드로도 출력합니다
  --amnezia-key             클라이언트 구성을 AmneziaVPN 키(vpn://...)로도 출력합니다
  --keenetic                Keenetic 라우터용 KeeneticOS CLI 명령도 출력합니다
                            (wg 및 awg 1.0/1.5/2.0 터널, IPv4 전용)
  --json                    테이블 대신 JSON 출력(list, list-client, udp-listener, udp-checker)
  --yes                     확인하지 않습니다(delete)
  --timeout DURATION        패킷 시간 제한, 예: 2s 또는 500ms(기본값: 2s)
  --count N                 포트당 각 유형의 패킷 수, 1..100(기본값: 3)
  --echo-size N             echo 페이로드 크기, 1..1400(기본값: 64, udp-checker 전용)
  --workers N               동시 포트 검사 수, 1..100(기본값: 32, udp-checker 전용)
  --verbose                 진단 진행 상황(udp-listener, udp-checker; stderr)

작업별 플래그:
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
  --doctor         (플래그 없음; root 없이 작동)
  --udp-listener   --port [--timeout] [--count] [--json] [--verbose]
  --udp-checker    --address --port [--timeout] [--count] [--echo-size] [--workers] [--json] [--verbose]

종료 코드:
  0 성공                    5 충돌(이미 존재하거나 주소 또는 포트 사용 중)
  1 내부 오류               6 환경(root 아님, 도구 없음, AmneziaWG 버전)
  2 사용법 오류             7 잠금을 획득할 수 없음
  3 잘못된 값               10 사용자가 취소함
  4 찾을 수 없음

root 권한:
  --help, --version, --doctor, --udp-checker에는 필요하지 않습니다.
  --udp-listener는 1024 미만 포트이거나 /proc/net/udp와 /proc/net/udp6를
  읽을 수 없을 때만 root가 필요합니다.
  다른 모든 작업에는 root가 필요합니다.
`
