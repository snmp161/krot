# krot

[English](README.md) | [Русский](README.ru.md) | [简体中文](README.zh.md) | [繁體中文](README.zh-Hant.md) | [Español](README.es.md) | **Português** | [한국어](README.ko.md) | [日本語](README.ja.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [Deutsch](README.de.md)

`krot` é um utilitário de linha de comando para gerenciar túneis
**WireGuard** e **AmneziaWG** (1.x, 2.0, 3.1) em servidores Linux: criar e
excluir interfaces, gerenciar clientes (peers) e gerar configurações de
cliente, códigos QR e chaves AmneziaVPN (`vpn://…`).

O nome **krot** significa "toupeira" em russo e em vários outros idiomas
eslavos — um animal que escava túneis.

- Um único binário estático sem dependências (Go, `CGO_ENABLED=0`).
- As operações com clientes **nunca reiniciam nem recriam a interface**:
  apenas chamadas pontuais a `wg|awg set <if> peer …` — os outros clientes
  não são afetados.
- A configuração da interface é a única fonte da verdade: sem bancos de
  dados separados; os metadados do krot são armazenados em comentários e
  configurações não criadas pelo krot são lidas e preservadas.
- Saída previsível: tabelas para humanos, `--json` e códigos de saída para
  automação.
- O krot não configura NAT, forwarding nem regras de firewall.

Especificação técnica completa: [`docs/SPEC.md`](docs/SPEC.md).

## Requisitos

| | |
|---|---|
| SO | Ubuntu 22.04 / 24.04 / 26.04, Debian 12 / 13, AlmaLinux (Rocky, RHEL) 9 / 10 |
| Arquiteturas | `amd64`, `arm64` |
| Para `--type wg` | `wireguard-tools` (`wg`, `wg-quick`), módulo de kernel WireGuard |
| Para `--type awg*` | `amneziawg-tools` (`awg`, `awg-quick`) e módulo de kernel AmneziaWG **ou** `amneziawg-go` |
| Sempre | `iproute2` (`ip`), `systemd` |
| Privilégios | root (exceto `--help`, `--version`, `--doctor`, `--udp-checker`; `--udp-listener` requer root apenas para portas `<1024` ou quando `/proc/net/udp{,6}` não é legível) |

O krot não instala nada disso por conta própria. Se um programa necessário
estiver faltando, ele sai com código `6` e informa o pacote a ser instalado.

### Versões do AmneziaWG

Antes de `--create`, o krot verifica se o AmneziaWG instalado suporta o tipo
solicitado:

| Tipo | amneziawg-tools | módulo de kernel | família netlink | amneziawg-go |
|---|---|---|---|---|
| `awg` (1.0) | 1.0.20240213 | — | ≥ 1 | 0.1.0 |
| `awg` com `--i1` (1.5) | 1.0.20250901 | 1.0.20251009 | ≥ 2 | 0.2.13 |
| `awg2` (2.0) | 1.0.20250901 | 1.0.20251009 | ≥ 2 | 0.2.15 |
| `awg3` (3.1) | 3.1.20260812 | 3.1.20260812 | ≥ 3 | 3.1.20260812 |

Algumas builds relatam uma versão de preenchimento (por exemplo,
`awg --version` nas ferramentas 1.x quase sempre imprime `1.0.20210914`,
e um módulo compilado com `make` relata `1.0.0`). Tais versões geram um
aviso, não um erro; o código `6` é retornado apenas quando a versão é
detectada de forma confiável e é muito antiga. Se a versão for realmente
muito antiga, a unidade não iniciará e o krot reverterá as alterações.

## Instalação

Instale a partir de pacotes ou binários na página de releases
<https://github.com/snmp161/krot/releases>, e sempre verifique a soma de
verificação.

```sh
VERSION=0.7.3
BASE=https://github.com/snmp161/krot/releases/download/v$VERSION

# baixar o pacote e a lista de somas de verificação
wget "$BASE/krot_${VERSION}_amd64.deb" "$BASE/SHA256SUMS"
# verificar: deve imprimir "krot_..._amd64.deb: OK"
sha256sum --check --ignore-missing SHA256SUMS
# instalar
sudo dpkg -i "krot_${VERSION}_amd64.deb"
```

Sistemas baseados em RPM (AlmaLinux, Rocky, RHEL):

```sh
wget "$BASE/krot-${VERSION}-1.x86_64.rpm" "$BASE/SHA256SUMS"
sha256sum --check --ignore-missing SHA256SUMS
sudo rpm -i "krot-${VERSION}-1.x86_64.rpm"
```

Binário sem pacote: baixe `krot-linux-amd64` (ou `krot-linux-arm64`) e
`SHA256SUMS`, verifique da mesma forma e instale com
`sudo install -m 0755 krot-linux-amd64 /usr/bin/krot`.

Para `arm64`, use os arquivos `*_arm64.deb`, `*.aarch64.rpm` e
`krot-linux-arm64`.

## Uso

```
krot <ação> [flags]
```

Exatamente uma ação por execução; os flags usam apenas a forma longa
`--flag value` ou `--flag=value`. Um flag que não se aplica à ação é um
erro (código `2`). Ajuda completa: `krot --help`.

### Túneis

```sh
# criar AmneziaWG 2.0 (dual-stack); o nome é selecionado automaticamente: awg0, awg1, …
sudo krot --create --type awg2 --ip 10.8.0.1/24,fd00:8::1/64 --port 51820 \
          --endpoint vpn.example.com --dns 1.1.1.1,9.9.9.9

# WireGuard com nome explícito e MTU
sudo krot --create --type wg --interface wg0 --ip 10.9.0.1/24 --port 51821 \
          --endpoint 203.0.113.10 --mtu 1420

# AmneziaWG 1.5 com um pacote de assinatura I1 (o krot valida a sintaxe CPS e o comprimento — até 2048 caracteres)
sudo krot --create --type awg --ip 10.10.0.1/24 --port 51822 --endpoint vpn.example.com \
          --i1 '<b 0xf6ab3267fa><r 16>'

# AmneziaWG 3.1 (Header Protection, RandomTrailers, DisableCookies sempre habilitados)
sudo krot --create --type awg3 --ip 10.11.0.1/24 --port 51823 --endpoint vpn.example.com

sudo krot --list                         # todos os túneis
sudo krot --list --type awg2 --json      # filtros --type/--interface/--ip/--port, com AND
sudo krot --down --interface awg0        # systemctl stop awg-quick@awg0
sudo krot --up --interface awg0          # systemctl start; --up repetido imprime "already up", código 0
sudo krot --delete --interface awg0      # pede confirmação se houver clientes
sudo krot --delete --interface awg0 --yes
```

### Diagnósticos

```sh
krot --doctor   # funciona sem root: ferramentas, módulos de kernel, versões do AmneziaWG, diretórios
```

A saída é uma tabela `CHECK/STATUS/DETAIL` com os status `ok` / `warn` /
`fail`. O código de saída é `0` se não houver nenhum `fail`, caso contrário
`6`. Não altera nada e não obtém nenhum bloqueio.

#### Diagnóstico de portas UDP

O `krot` pode verificar quais portas UDP realmente passam entre dois hosts.
Isso é útil quando um provedor de hospedagem ou seu uplink filtra algumas
portas (por exemplo, devido a regras anti-DDoS).

Inicie o listener no servidor:

```sh
# uma única porta
sudo krot --udp-listener --port 51820

# faixa de portas (até 1024 portas, um socket por porta)
sudo krot --udp-listener --port 51820:51830
```

O listener responde a três tipos de datagramas:
- `ping SEQ:N` → `pong SEQ:N`;
- `ECHO <text>` → `<text>`;
- `TIME` → a hora Unix atual em milissegundos.

Execute o checker no cliente:

```sh
# verificar uma única porta
krot --udp-checker --address 203.0.113.1 --port 51820

# verificar uma faixa
krot --udp-checker --address 203.0.113.1 --port 51820:51830

# aumentar o tamanho do payload echo (1–1400 bytes, padrão 64) — ajuda a encontrar o melhor MTU
krot --udp-checker --address 203.0.113.1 --port 51820 --echo-size 1400

# definir o número de workers e mostrar o progresso
krot --udp-checker --address 203.0.113.1 --port 51820:51830 --workers 64 --verbose
```

Por padrão, o checker envia 3 pacotes de cada tipo por porta (`--count`,
1–100) e aguarda uma resposta dentro de `--timeout` (padrão 2s). Ele verifica
32 portas simultaneamente; `--workers` permite escolher de 1 a 100. `--verbose`
envia a faixa e os resultados das portas para stderr à medida que terminam;
no listener, registra os endereços dos clientes e os tipos de solicitações. Saída:

```
PORT     STATUS   PING    ECHO    TIME (min/avg/max)   DETAIL
51820    ok       3/3     3/3     12/24/45 ms          -
51821    filtered 0/3     0/3     -                    timeout
```

Status:
- `ok` — todos os pacotes ping e echo receberam respostas válidas;
- `filtered` — pelo menos um pacote esgotou o tempo limite (a porta
  provavelmente está filtrada);
- `error` — uma resposta inválida foi recebida.

Com `--json`, a saída é um array de objetos com os campos `port`, `status`,
`ping_sent`, `ping_ok`, `echo_sent`, `echo_ok`, `time_sent`, `time_ok`,
`rtt_min_ms`, `rtt_avg_ms`, `rtt_max_ms`, `detail`.

Com `Ctrl+C`/`SIGTERM`, o checker mostra os resultados das portas totalmente verificadas
no formato selecionado e termina com código `10`; as portas incompletas são omitidas.

`--udp-checker` funciona sem root. `--udp-listener` requer root apenas se
a faixa incluir portas `<1024` ou se `/proc/net/udp` e `/proc/net/udp6`
não forem legíveis pelo usuário atual. O listener é executado até
`Ctrl+C`/`SIGTERM`.

Exemplo de `--list`:

```
INTERFACE  TYPE  VERSION  ADDRESS                   PORT   STATUS  CLIENTS
awg0       awg2  2.0      10.8.0.1/24,fd00:8::1/64  51820  up      12 (1 locked)
wg0        wg    -        10.9.0.1/24               51821  down    3
```

MTU padrão: `wg` — 1420, `awg`/`awg2` — 1376, `awg3` — 1280. Os parâmetros
de ofuscação (Jc, S1–S4, H1–H4, …) são gerados automaticamente em faixas
seguras e validados (seção 7 da especificação).

### Clientes

```sh
# adicionar um cliente: os endereços são atribuídos automaticamente (primeiro /32 e /128 livres);
# a configuração do cliente é impressa UMA única vez — a chave privada do cliente nunca é armazenada
sudo krot --add-client --interface awg0 --id phone-sergei --allowed-ips 0.0.0.0/0,::/0

# com uma PresharedKey e um código QR no terminal
sudo krot --add-client --interface awg0 --id laptop --allowed-ips 0.0.0.0/0 --psk --qr

# imprimir também uma chave AmneziaVPN: uma linha vpn://… após a configuração
sudo krot --add-client --interface awg0 --id tablet --allowed-ips 0.0.0.0/0,::/0 --amnezia-key

# comandos CLI do KeeneticOS: conectar um roteador Keenetic como cliente (somente wg, somente IPv4)
sudo krot --add-client --interface wg0 --id keenetic --allowed-ips 0.0.0.0/0 --keenetic

# endereço de cliente específico
sudo krot --add-client --interface awg0 --id router --allowed-ips 10.8.0.0/24 --ip 10.8.0.50/32

# configuração não criada pelo krot: passe o endpoint explicitamente (não é gravado na configuração do servidor)
sudo krot --add-client --interface office --id guest --allowed-ips 0.0.0.0/0 --endpoint 203.0.113.10

# DNS personalizado para um único cliente (não é gravado na configuração do servidor)
sudo krot --add-client --interface awg0 --id kid-phone --allowed-ips 0.0.0.0/0 --dns 94.140.14.15

sudo krot --list-client --interface awg0
sudo krot --list-client --interface awg0 --id phone-sergei --json
sudo krot --lock-client   --interface awg0 --id laptop    # repetido — "already locked", código 0
sudo krot --unlock-client --interface awg0 --id laptop
sudo krot --delete-client --interface awg0 --id laptop
```

`--allowed-ips` define `AllowedIPs` na configuração do **cliente** (o que o
cliente roteia pelo túnel). DNS do cliente: `--dns` se fornecido; caso
contrário, os metadados `# krot:dns` da interface (`--dns` em `--create`);
caso contrário, as linhas `DNS` do `[Interface]` da configuração do
servidor; se não houver DNS em nenhum lugar, a configuração do cliente não
terá linha `DNS`.

Os endereços dos clientes são alocados a partir da sub-rede da interface,
excluindo o endereço de rede, o endereço de broadcast e o endereço do
servidor. Sub-redes ponto a ponto `/31` e `/127` (RFC 3021, RFC 6164)
também são suportadas: ambos os endereços são utilizáveis, portanto um
túnel "servidor + um cliente" pode ser criado com
`--create … --ip 10.8.0.0/31,fd00:8::/127`.

Salve a configuração gerada imediatamente: ela não pode ser recuperada
novamente; você só pode excluir o cliente e adicioná-lo novamente. Os
códigos QR são impressos com meio blocos Unicode e pressupõem um fundo de
terminal escuro.

`--amnezia-key` imprime uma chave AmneziaVPN após a configuração — uma
única linha `vpn://…` para colar no aplicativo AmneziaVPN ("Inserir chave"),
por exemplo, enviando-a por um mensageiro. Ela contém a mesma configuração
do cliente: `vpn://` + base64url sem `=` do `qCompress` do Qt (comprimento
big-endian de 4 bytes + zlib). A chave funciona para todos os tipos,
incluindo `wg`. O código QR de `--qr` contém o texto da configuração —
AmneziaVPN e AmneziaWG podem escaneá-lo; para `wg`, o WireGuard também pode.

`--keenetic` imprime um bloco de comandos CLI do KeeneticOS após a
configuração — conectando um roteador Keenetic como cliente do túnel. O
bloco segue a sintaxe de running-config do KeeneticOS e pode ser colado na
CLI do roteador (SSH/telnet ou a "Interface de linha de comando" da
interface web) como está. O KeeneticOS suporta ofuscação AmneziaWG
(parâmetros ASC): AWG 1.0 desde o KeeneticOS 4.2, AWG 1.5/2.0 desde 5.1;
para túneis awg o bloco inclui uma linha `wireguard asc …` (as strings CPS
I1–I5 vão entre aspas; importar o arquivo de configuração pela interface
web é mais confiável). AWG 3.1 não é suportado. Somente IPv4; se o MTU do
túnel diferir do padrão 1420, uma nota é impressa. Consulte a seção 6.13 da
especificação para o formato.

Exemplo de `--list-client`:

```
ID            PUBLIC KEY  ADDRESS                    STATUS  LAST HANDSHAKE  RX       TX
phone-sergei  aB3d…Xy9=   10.8.0.2/32,fd00:8::2/128  active  1m ago          12.4MiB  1.1GiB
laptop-old    Qw7e…Zz1=   10.8.0.3/32                locked  -               -        -
```

### Mimetismo (pacotes I1–I5)

A partir da versão 1.5, o AmneziaWG pode enviar até cinco pacotes de
assinatura `I1`–`I5` antes de cada handshake: pacotes que parecem o início
de outro protocolo (DNS, STUN, NTP, SSDP, DTLS). O krot gera essas cadeias
por conta própria por meio do flag `--mimic <profile>`:

```sh
# perfil padrão para todos os clientes de uma interface
krot --create --type awg --ip 10.8.0.1/24 --port 51820 --endpoint vpn.example.com --mimic dns
# substituir para um único cliente
krot --add-client --interface awg0 --id phone --allowed-ips 0.0.0.0/0 --mimic stun
```

Perfis: `dns` (três consultas A/AAAA/HTTPS), `stun` (Binding Request),
`ntp`, `ssdp` (M-SEARCH), `dtls` (ClientHello). A cadeia é gerada em
`--add-client` (cada cliente obtém a sua) e gravada apenas na configuração
do cliente: os pacotes I são enviados pelo iniciador do handshake, portanto
o servidor não precisa deles. O cliente requer AmneziaWG 1.5+ — consulte a
seção 6.12 da especificação.

Além dos perfis integrados, o krot aceita uma string CPS pronta por meio de
`--i1` em `--create` — por exemplo, para QUIC: os perfis QUIC integrados
não são gerados porque um único QUIC Initial tem 2400–2500 caracteres e
excede o limite de 2048 (devido a
[amneziawg-tools#69](https://github.com/amnezia-vpn/amneziawg-tools/issues/69)).
Uma string pronta pode ser obtida de geradores de terceiros (ambos com
licença MIT, não afiliados ao krot):

- [payloadGen](https://github.com/Sketchystan1/payloadGen) — uma página web:
  abra `index.html` em um navegador, selecione um protocolo e pegue a
  primeira linha (I1);
- `cpsgen.py` do [awg-multi-script](https://github.com/pumbaX/awg-multi-script)
  (`src/py/cpsgen.py`) — uma versão de console do payloadGen em Python 3,
  não requer root. Perfis: `dns`, `ntp`, `rtp`, `stun`, `sip`, `ssdp`,
  `webrtc`, `quic`, `curl_quic`; o segundo argumento é o domínio.

```sh
# baixar o gerador e revisar seu código antes de executar
git clone --depth 1 https://github.com/pumbaX/awg-multi-script
less awg-multi-script/src/py/cpsgen.py

# gerar apenas I1 e verificar seu comprimento (deve ser ≤ 2048 caracteres)
I1=$(python3 awg-multi-script/src/py/cpsgen.py dns example.com --only-i1)
echo ${#I1}

sudo krot --create --type awg2 --ip 10.8.0.1/24 --port 51820 --endpoint vpn.example.com --i1 "$I1"
```

Notas:

- `I1` é definido apenas em `--create`, gravado na configuração do servidor
  e copiado para todas as configurações de cliente. `I2`–`I5` não são
  aceitos pelo krot.
- O comprimento não deve exceder 2048 caracteres (consulte
  [Limitações](#limitações)), caso contrário código `3`. Os pacotes dos
  perfis `quic` e `curl_quic` têm cerca de 2400 caracteres e não cabem.
  Perfis mais curtos funcionam: `rtp` (~30 caracteres), `dns` e `ntp`
  (~100), `stun` (~250–350), `ssdp` (~400), `sip` (~1200).
- `--type awg` com `--i1` significa AmneziaWG 1.5 com suas próprias versões
  mínimas (consulte [Versões do AmneziaWG](#versões-do-amneziawg)). As tags
  `<rc N>`/`<rd N>` (produzidas pelo perfil `sip`) são entendidas pelo
  amneziawg-go desde 0.2.15, portanto para 1.5 prefira perfis sem elas.
- O cliente deve suportar AmneziaWG 1.5 ou mais recente.
- Com um `I1` longo, a configuração pode não caber em um código QR:
  `--qr` então sai com código `3` (o cliente já foi adicionado e a
  configuração já foi impressa). Transfira a configuração como arquivo ou
  use `--amnezia-key`.

### JSON

`--list --json` produz um array de objetos com `interface`, `type`,
`version`, `addresses[]`, `port`, `status`, `clients_total`,
`clients_locked`. `--list-client --json` produz um array de objetos com
`id`, `public_key`, `addresses[]`, `status`, `latest_handshake` (hora Unix
ou `null`), `rx_bytes`, `tx_bytes`, `endpoint` (ou `null`), `created` (ou
`null` para peers não gerenciados pelo krot). Um resultado vazio é `[]`.

## Códigos de saída

| Código | Significado |
|---|---|
| 0 | sucesso |
| 1 | erro interno ou inesperado (incluindo falha ao aplicar uma alteração a uma interface em execução — configuração revertida) |
| 2 | erro de uso da CLI (flags; excluir com clientes sem `--yes` quando stdin não é um terminal) |
| 3 | erro de validação de valores (incluindo endpoint ausente, "config too large for QR") |
| 4 | objeto não encontrado (interface, cliente) |
| 5 | conflito: já existe, endereço ou porta em uso, não há endereços livres |
| 6 | ambiente: sem root, ferramentas ausentes, versão do AmneziaWG não suportada |
| 7 | não foi possível adquirir o bloqueio `/run/lock/krot.lock` em 10 segundos |
| 10 | operação cancelada pelo usuário |

A saída vai para stdout; erros e avisos vão para stderr com o prefixo
`krot:`.

## Armazenamento de dados

| Tipo | Configuração | Unidade systemd | Ferramenta |
|---|---|---|---|
| `wg` | `/etc/wireguard/<if>.conf` | `wg-quick@<if>` | `wg` |
| `awg`, `awg2`, `awg3` | `/etc/amnezia/amneziawg/<if>.conf` | `awg-quick@<if>` | `awg` |

Os metadados são armazenados como comentários que `wg-quick`/`awg-quick`
ignoram:

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

- Um cliente bloqueado é um bloco `[Peer]` comentado com o prefixo `#~ `.
- Os hooks comentados `PreUp`/`PostUp`/`PreDown`/`PostDown` podem ser
  descomentados e preenchidos (por exemplo, com regras NAT) — o krot não os
  modifica.
- Configurações sem metadados do krot (criadas manualmente) são suportadas:
  `--list` as mostra, `--list-client` imprime seus peers com id `-`
  (operações por `--id` não se aplicam a tais peers), e qualquer gravação
  preserva todas as linhas originais, comentários e ordenação byte a byte.
- As gravações são atômicas (arquivo temporário → `fsync` → `rename` →
  `fsync` do diretório), permissões `0600`.
- Todas as ações que modificam obtêm um `flock` exclusivo em
  `/run/lock/krot.lock` — execuções concorrentes (por exemplo, Salt e um
  humano) não perdem alterações.
- O krot não faz backups.

## Limitações

- NAT, forwarding, firewall e sysctl estão fora do escopo do krot (use os
  hooks `PostUp`/`PostDown`).
- A instalação do WireGuard/AmneziaWG e a migração de túneis entre tipos
  não são suportadas. A geração de assinaturas de mimetismo usa perfis
  integrados (`--mimic`); perfis QUIC e assinaturas personalizadas passam
  por `--i1` e geradores de terceiros, consulte
  [Mimetismo](#mimetismo-pacotes-i1i5).
- `--i1` não deve exceder 2048 caracteres: devido a
  [amneziawg-tools#69](https://github.com/amnezia-vpn/amneziawg-tools/issues/69),
  `I1`–`I5` longos (aproximadamente 3600+ caracteres) fazem o `awg show`
  travar e o `awg set`/`setconf` falhar.
- As chaves privadas dos clientes não são armazenadas; as configurações de
  cliente não podem ser reemitidas.

## Desenvolvimento

Você precisa do Go do `go.mod` (atualmente 1.27.4) e do Docker para os
testes de integração.

```sh
make check        # todas as verificações locais: formato, testes, módulos, segurança e workflows
make fuzz-smoke   # fuzzing limitado de todos os targets
make build        # bin/krot estático (CGO_ENABLED=0), versão do git describe
make build-arm64
make integration  # testes de integração em um contêiner Docker privilegiado com systemd
make package-preflight # DEB/RPM para ambas arquiteturas + smoke no Ubuntu 24.04/AlmaLinux 9
```

Regras do repositório: [`AGENTS.md`](AGENTS.md) — a especificação primeiro,
os testes antes do código, um commit por tarefa.

- **Os testes unitários** não requerem root e não tocam no sistema: os
  diretórios são substituídos por temporários, os comandos externos (`wg`,
  `awg`, `systemctl`, `ip`, …) são simulados com registro de chamadas.
  Testes separados verificam que as operações com clientes chamam apenas
  `wg|awg show` e `wg|awg set <if> peer …`.
- **Os arquivos golden** ficam no `testdata/` de cada pacote. As saídas
  geradas (`internal/app`, `internal/qr`) são atualizadas apenas
  deliberadamente — com `go test ./internal/app ./internal/qr -update` e
  uma explicação no commit; outros arquivos golden são escritos manualmente
  a partir da especificação.
- **Os testes de integração** (`test/integration`, tag de compilação
  `integration`) requerem root e systemd e são executados apenas em
  namespaces de rede separados: as unidades de interface de teste são
  colocadas em netns por meio do drop-in `NetworkNamespacePath`. Cobrem o
  ciclo completo para todos os tipos, a transparência das operações com
  clientes (ping contínuo sem perda, ifindex e handshake inalterados),
  IPv6, configurações criadas manualmente e execuções concorrentes.
  Localmente via `scripts/integration-docker.sh`, no CI sob `sudo`.
- `scripts/collect-versions.sh` reconstrói saídas reais de versões do
  AmneziaWG para os testes golden.
- **CI** (`.github/workflows/ci.yml`): verificações locais de qualidade, testes
  unitários com shuffle, `-race` e cobertura (mínimo 90%), fuzzing limitado,
  testes de integração, package preflight e builds `amd64`/`arm64`.
  **Release** (`release.yml`, tag `v*`): binários, pacotes `.deb`/`.rpm`
  (nfpm, `packaging/nfpm.yaml`), testes de fumaça de pacotes em todos os
  SOs (`scripts/smoke.sh`), GitHub Release com `SHA256SUMS`.

## Licença

[Apache License 2.0](LICENSE). Copyright 2026 Sergey Ovchinnikov.
