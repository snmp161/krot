# krot

[English](README.md) | [Русский](README.ru.md) | [中文](README.zh.md) | **Español** | [Português](README.pt.md)

`krot` es una utilidad de línea de comandos para administrar túneles
**WireGuard** y **AmneziaWG** (1.x, 2.0, 3.1) en servidores Linux: crear y
eliminar interfaces, administrar clientes (peers) y generar configuraciones
de cliente, códigos QR y claves de AmneziaVPN (`vpn://…`).

El nombre **krot** significa "topo" en ruso y en varios otros idiomas
eslavos — un animal que excava túneles.

- Un solo binario estático sin dependencias (Go, `CGO_ENABLED=0`).
- Las operaciones con clientes **nunca reinician ni recrean la interfaz**:
  solo llamadas puntuales a `wg|awg set <if> peer …` — los demás clientes
  no se ven afectados.
- La configuración de la interfaz es la única fuente de verdad: sin bases
  de datos separadas; los metadatos de krot se almacenan en comentarios y
  las configuraciones no creadas por krot se leen y se conservan.
- Salida predecible: tablas para humanos, `--json` y códigos de salida para
  automatización.
- krot no configura NAT, forwarding ni reglas de firewall.

Especificación técnica completa: [`docs/SPEC.md`](docs/SPEC.md).

## Requisitos

| | |
|---|---|
| SO | Ubuntu 22.04 / 24.04 / 26.04, Debian 12 / 13, AlmaLinux (Rocky, RHEL) 9 / 10 |
| Arquitecturas | `amd64`, `arm64` |
| Para `--type wg` | `wireguard-tools` (`wg`, `wg-quick`), módulo de kernel WireGuard |
| Para `--type awg*` | `amneziawg-tools` (`awg`, `awg-quick`) y módulo de kernel AmneziaWG **o** `amneziawg-go` |
| Siempre | `iproute2` (`ip`), `systemd` |
| Privilegios | root (excepto `--help`, `--version`, `--doctor`, `--udp-checker`; `--udp-listener` requiere root solo para puertos `<1024` o cuando `/proc/net/udp{,6}` no es legible) |

krot no instala nada de esto por sí mismo. Si falta un programa requerido,
sale con código `6` e indica el paquete que hay que instalar.

### Versiones de AmneziaWG

Antes de `--create`, krot verifica que la AmneziaWG instalada soporte el
tipo solicitado:

| Tipo | amneziawg-tools | módulo de kernel | familia netlink | amneziawg-go |
|---|---|---|---|---|
| `awg` (1.0) | 1.0.20240213 | — | ≥ 1 | 0.1.0 |
| `awg` con `--i1` (1.5) | 1.0.20250901 | 1.0.20251009 | ≥ 2 | 0.2.13 |
| `awg2` (2.0) | 1.0.20250901 | 1.0.20251009 | ≥ 2 | 0.2.15 |
| `awg3` (3.1) | 3.1.20260812 | 3.1.20260812 | ≥ 3 | 3.1.20260812 |

Algunas compilaciones reportan una versión de relleno (por ejemplo,
`awg --version` en herramientas 1.x casi siempre imprime `1.0.20210914`,
y un módulo compilado con `make` reporta `1.0.0`). Tales versiones generan
una advertencia, no un error; el código `6` se devuelve solo cuando la
versión se detecta de forma fiable y es demasiado antigua. Si la versión
realmente es demasiado antigua, la unidad no arrancará y krot revertirá
los cambios.

## Instalación

Instala desde paquetes o binarios en la página de releases
<https://github.com/snmp161/krot/releases>, y verifica siempre la suma de
comprobación.

```sh
VERSION=0.1.0
BASE=https://github.com/snmp161/krot/releases/download/v$VERSION

# descargar el paquete y la lista de sumas de comprobación
wget "$BASE/krot_${VERSION}_amd64.deb" "$BASE/SHA256SUMS"
# verificar: debe imprimir "krot_..._amd64.deb: OK"
sha256sum --check --ignore-missing SHA256SUMS
# instalar
sudo dpkg -i "krot_${VERSION}_amd64.deb"
```

Sistemas basados en RPM (AlmaLinux, Rocky, RHEL):

```sh
wget "$BASE/krot-${VERSION}-1.x86_64.rpm" "$BASE/SHA256SUMS"
sha256sum --check --ignore-missing SHA256SUMS
sudo rpm -i "krot-${VERSION}-1.x86_64.rpm"
```

Binario sin paquete: descarga `krot-linux-amd64` (o `krot-linux-arm64`) y
`SHA256SUMS`, verifica de la misma forma y luego instala con
`sudo install -m 0755 krot-linux-amd64 /usr/bin/krot`.

Para `arm64`, usa los archivos `*_arm64.deb`, `*.aarch64.rpm` y
`krot-linux-arm64`.

## Uso

```
krot <acción> [flags]
```

Exactamente una acción por ejecución; los flags usan solo la forma larga
`--flag value` o `--flag=value`. Un flag que no aplica a la acción es un
error (código `2`). Ayuda completa: `krot --help`.

### Túneles

```sh
# crear AmneziaWG 2.0 (dual-stack); el nombre se selecciona automáticamente: awg0, awg1, …
sudo krot --create --type awg2 --ip 10.8.0.1/24,fd00:8::1/64 --port 51820 \
          --endpoint vpn.example.com --dns 1.1.1.1,9.9.9.9

# WireGuard con nombre explícito y MTU
sudo krot --create --type wg --interface wg0 --ip 10.9.0.1/24 --port 51821 \
          --endpoint 203.0.113.10 --mtu 1420

# AmneziaWG 1.5 con un paquete de firma I1 (krot valida la sintaxis CPS y la longitud — hasta 2048 caracteres)
sudo krot --create --type awg --ip 10.10.0.1/24 --port 51822 --endpoint vpn.example.com \
          --i1 '<b 0xf6ab3267fa><r 16>'

# AmneziaWG 3.1 (Header Protection, RandomTrailers, DisableCookies siempre habilitados)
sudo krot --create --type awg3 --ip 10.11.0.1/24 --port 51823 --endpoint vpn.example.com

sudo krot --list                         # todos los túneles
sudo krot --list --type awg2 --json      # filtros --type/--interface/--ip/--port, con AND
sudo krot --down --interface awg0        # systemctl stop awg-quick@awg0
sudo krot --up --interface awg0          # systemctl start; --up repetido imprime "already up", código 0
sudo krot --delete --interface awg0      # pide confirmación si hay clientes
sudo krot --delete --interface awg0 --yes
```

### Diagnósticos

```sh
krot --doctor   # funciona sin root: herramientas, módulos de kernel, versiones de AmneziaWG, directorios
```

La salida es una tabla `CHECK/STATUS/DETAIL` con estados `ok` / `warn` /
`fail`. El código de salida es `0` si no hay ningún `fail`, de lo contrario
`6`. No cambia nada y no toma ningún bloqueo.

#### Diagnóstico de puertos UDP

`krot` puede comprobar qué puertos UDP realmente pasan entre dos hosts.
Esto es útil cuando un proveedor de hosting o su uplink filtra algunos
puertos (por ejemplo, debido a reglas anti-DDoS).

Inicia el listener en el servidor:

```sh
# un solo puerto
sudo krot --udp-listener --port 51820

# rango de puertos (hasta 1024 puertos, un socket por puerto)
sudo krot --udp-listener --port 51820:51830
```

El listener responde a tres tipos de datagramas:
- `ping SEQ:N` → `pong SEQ:N`;
- `ECHO <text>` → `<text>`;
- `TIME` → la hora Unix actual en milisegundos.

Ejecuta el checker en el cliente:

```sh
# comprobar un solo puerto
krot --udp-checker --address 203.0.113.1 --port 51820

# comprobar un rango
krot --udp-checker --address 203.0.113.1 --port 51820:51830

# aumentar el tamaño del payload echo (1–1400 bytes, por defecto 64) — ayuda a encontrar el mejor MTU
krot --udp-checker --address 203.0.113.1 --port 51820 --echo-size 1400

# definir el número de workers y mostrar el progreso
krot --udp-checker --address 203.0.113.1 --port 51820:51830 --workers 64 --verbose
```

Por defecto, el checker envía 3 paquetes de cada tipo por puerto
(`--count`, 1–100) y espera una respuesta dentro de `--timeout`
(por defecto 2s). Comprueba 32 puertos simultáneamente; `--workers` permite
elegir entre 1 y 100. `--verbose` envía el rango y los resultados a stderr
según terminan; en el listener registra las direcciones de clientes y los
tipos de solicitudes. Salida:

```
PORT     STATUS   PING    ECHO    TIME (min/avg/max)   DETAIL
51820    ok       3/3     3/3     12/24/45 ms          -
51821    filtered 0/3     0/3     -                    timeout
```

Estados:
- `ok` — todos los paquetes ping y echo recibieron respuestas válidas;
- `filtered` — al menos un paquete agotó el tiempo de espera (el puerto
  probablemente está filtrado);
- `error` — se recibió una respuesta no válida.

Con `--json`, la salida es un array de objetos con los campos `port`,
`status`, `ping_sent`, `ping_ok`, `echo_sent`, `echo_ok`, `time_sent`,
`time_ok`, `rtt_min_ms`, `rtt_avg_ms`, `rtt_max_ms`, `detail`.

Con `Ctrl+C`/`SIGTERM`, el checker muestra los resultados de los puertos completamente
comprobados en el formato seleccionado y termina con código `10`; omite los puertos sin terminar.

`--udp-checker` funciona sin root. `--udp-listener` requiere root solo si
el rango incluye puertos `<1024` o `/proc/net/udp` y `/proc/net/udp6` no
son legibles por el usuario actual. El listener se ejecuta hasta
`Ctrl+C`/`SIGTERM`.

Ejemplo de `--list`:

```
INTERFACE  TYPE  VERSION  ADDRESS                   PORT   STATUS  CLIENTS
awg0       awg2  2.0      10.8.0.1/24,fd00:8::1/64  51820  up      12 (1 locked)
wg0        wg    -        10.9.0.1/24               51821  down    3
```

MTU por defecto: `wg` — 1420, `awg`/`awg2` — 1376, `awg3` — 1280. Los
parámetros de ofuscación (Jc, S1–S4, H1–H4, …) se generan automáticamente
en rangos seguros y se validan (sección 7 de la especificación).

### Clientes

```sh
# añadir un cliente: las direcciones se asignan automáticamente (primera /32 y /128 libres);
# la configuración del cliente se imprime UNA sola vez — la clave privada del cliente nunca se almacena
sudo krot --add-client --interface awg0 --id phone-sergei --allowed-ips 0.0.0.0/0,::/0

# con una PresharedKey y un código QR en la terminal
sudo krot --add-client --interface awg0 --id laptop --allowed-ips 0.0.0.0/0 --psk --qr

# imprimir además una clave de AmneziaVPN: una línea vpn://… después de la configuración
sudo krot --add-client --interface awg0 --id tablet --allowed-ips 0.0.0.0/0,::/0 --amnezia-key

# comandos CLI de KeeneticOS: conectar un router Keenetic como cliente (solo wg, solo IPv4)
sudo krot --add-client --interface wg0 --id keenetic --allowed-ips 0.0.0.0/0 --keenetic

# dirección de cliente específica
sudo krot --add-client --interface awg0 --id router --allowed-ips 10.8.0.0/24 --ip 10.8.0.50/32

# configuración no creada por krot: pasar el endpoint explícitamente (no se escribe en la configuración del servidor)
sudo krot --add-client --interface office --id guest --allowed-ips 0.0.0.0/0 --endpoint 203.0.113.10

# DNS personalizado para un solo cliente (no se escribe en la configuración del servidor)
sudo krot --add-client --interface awg0 --id kid-phone --allowed-ips 0.0.0.0/0 --dns 94.140.14.15

sudo krot --list-client --interface awg0
sudo krot --list-client --interface awg0 --id phone-sergei --json
sudo krot --lock-client   --interface awg0 --id laptop    # repetido — "already locked", código 0
sudo krot --unlock-client --interface awg0 --id laptop
sudo krot --delete-client --interface awg0 --id laptop
```

`--allowed-ips` establece `AllowedIPs` en la configuración del **cliente**
(lo que el cliente enruta a través del túnel). DNS del cliente: `--dns` si
se indica; de lo contrario, los metadatos `# krot:dns` de la interfaz
(`--dns` en `--create`); de lo contrario, las líneas `DNS` del `[Interface]`
de la configuración del servidor; si no hay DNS en ninguna parte, la
configuración del cliente no tiene línea `DNS`.

Las direcciones de cliente se asignan desde la subred de la interfaz,
excluyendo la dirección de red, la dirección de broadcast y la dirección
del servidor. También se admiten subredes punto a punto `/31` y `/127`
(RFC 3021, RFC 6164): ambas direcciones son utilizables, por lo que un
túnel "servidor + un cliente" se puede crear con
`--create … --ip 10.8.0.0/31,fd00:8::/127`.

Guarda la configuración generada inmediatamente: no se puede recuperar de
nuevo; solo puedes eliminar el cliente y añadirlo otra vez. Los códigos QR
se imprimen con medios bloques Unicode y asumen un fondo de terminal oscuro.

`--amnezia-key` imprime una clave de AmneziaVPN después de la configuración
— una sola línea `vpn://…` para pegar en la aplicación AmneziaVPN
("Insertar clave"), por ejemplo, enviándola a través de un mensajero.
Contiene la misma configuración del cliente: `vpn://` + base64url sin `=`
del `qCompress` de Qt (longitud big-endian de 4 bytes + zlib). La clave
funciona para todos los tipos, incluido `wg`. El código QR de `--qr`
contiene el texto de la configuración — AmneziaVPN y AmneziaWG pueden
escanearlo; para `wg`, WireGuard también puede.

`--keenetic` imprime un bloque de comandos CLI de KeeneticOS después de la
configuración — conectando un router Keenetic como cliente del túnel. El
bloque sigue la sintaxis de running-config de KeeneticOS y se puede pegar
en la CLI del router (SSH/telnet o la "Interfaz de línea de comandos" de la
interfaz web) tal cual. KeeneticOS soporta la ofuscación de AmneziaWG
(parámetros ASC): AWG 1.0 desde KeeneticOS 4.2, AWG 1.5/2.0 desde 5.1;
para túneles awg el bloque incluye una línea `wireguard asc …` (las cadenas
CPS I1–I5 van entre comillas; importar el archivo de configuración a través
de la interfaz web es más fiable). AWG 3.1 no está soportado. Solo IPv4; si
el MTU del túnel difiere del valor por defecto 1420, se imprime una nota.
Consulta la sección 6.13 de la especificación para el formato.

Ejemplo de `--list-client`:

```
ID            PUBLIC KEY  ADDRESS                    STATUS  LAST HANDSHAKE  RX       TX
phone-sergei  aB3d…Xy9=   10.8.0.2/32,fd00:8::2/128  active  1m ago          12.4MiB  1.1GiB
laptop-old    Qw7e…Zz1=   10.8.0.3/32                locked  -               -        -
```

### Mimetismo (paquetes I1–I5)

A partir de la versión 1.5, AmneziaWG puede enviar hasta cinco paquetes de
firma `I1`–`I5` antes de cada handshake: paquetes que parecen el inicio de
otro protocolo (DNS, STUN, NTP, SSDP, DTLS). krot genera tales cadenas por
sí mismo mediante el flag `--mimic <profile>`:

```sh
# perfil por defecto para todos los clientes de una interfaz
krot --create --type awg --ip 10.8.0.1/24 --port 51820 --endpoint vpn.example.com --mimic dns
# anular para un solo cliente
krot --add-client --interface awg0 --id phone --allowed-ips 0.0.0.0/0 --mimic stun
```

Perfiles: `dns` (tres consultas A/AAAA/HTTPS), `stun` (Binding Request),
`ntp`, `ssdp` (M-SEARCH), `dtls` (ClientHello). La cadena se genera en
`--add-client` (cada cliente obtiene la suya) y se escribe solo en la
configuración del cliente: los paquetes I los envía el iniciador del
handshake, por lo que el servidor no los necesita. El cliente requiere
AmneziaWG 1.5+ — consulta la sección 6.12 de la especificación.

Además de los perfiles integrados, krot acepta una cadena CPS lista mediante
`--i1` en `--create` — por ejemplo, para QUIC: los perfiles QUIC integrados
no se generan porque un solo QUIC Initial tiene 2400–2500 caracteres y
excede el límite de 2048 (debido a
[amneziawg-tools#69](https://github.com/amnezia-vpn/amneziawg-tools/issues/69)).
Una cadena lista se puede obtener de generadores de terceros (ambos con
licencia MIT, no afiliados a krot):

- [payloadGen](https://github.com/Sketchystan1/payloadGen) — una página web:
  abre `index.html` en un navegador, selecciona un protocolo y toma la
  primera línea (I1);
- `cpsgen.py` de [awg-multi-script](https://github.com/pumbaX/awg-multi-script)
  (`src/py/cpsgen.py`) — una versión de consola de payloadGen en Python 3,
  no requiere root. Perfiles: `dns`, `ntp`, `rtp`, `stun`, `sip`, `ssdp`,
  `webrtc`, `quic`, `curl_quic`; el segundo argumento es el dominio.

```sh
# descargar el generador y revisar su código antes de ejecutarlo
git clone --depth 1 https://github.com/pumbaX/awg-multi-script
less awg-multi-script/src/py/cpsgen.py

# generar solo I1 y comprobar su longitud (debe ser ≤ 2048 caracteres)
I1=$(python3 awg-multi-script/src/py/cpsgen.py dns example.com --only-i1)
echo ${#I1}

sudo krot --create --type awg2 --ip 10.8.0.1/24 --port 51820 --endpoint vpn.example.com --i1 "$I1"
```

Notas:

- `I1` se establece solo en `--create`, se escribe en la configuración del
  servidor y se copia en todas las configuraciones de cliente. `I2`–`I5`
  no son aceptados por krot.
- La longitud no debe exceder los 2048 caracteres (consulta
  [Limitaciones](#limitaciones)), de lo contrario código `3`. Los paquetes
  de los perfiles `quic` y `curl_quic` tienen unos 2400 caracteres y no
  caben. Los perfiles más cortos funcionan: `rtp` (~30 caracteres), `dns`
  y `ntp` (~100), `stun` (~250–350), `ssdp` (~400), `sip` (~1200).
- `--type awg` con `--i1` significa AmneziaWG 1.5 con sus propias versiones
  mínimas (consulta [Versiones de AmneziaWG](#versiones-de-amneziawg)).
  Las etiquetas `<rc N>`/`<rd N>` (producidas por el perfil `sip`) son
  entendidas por amneziawg-go desde 0.2.15, por lo que para 1.5 es mejor
  usar perfiles sin ellas.
- El cliente debe soportar AmneziaWG 1.5 o más reciente.
- Con un `I1` largo, la configuración puede no caber en un código QR:
  `--qr` entonces sale con código `3` (el cliente ya está añadido y la
  configuración ya está impresa). Transfiere la configuración como archivo
  o usa `--amnezia-key`.

### JSON

`--list --json` produce un array de objetos con `interface`, `type`,
`version`, `addresses[]`, `port`, `status`, `clients_total`,
`clients_locked`. `--list-client --json` produce un array de objetos con
`id`, `public_key`, `addresses[]`, `status`, `latest_handshake` (hora Unix
o `null`), `rx_bytes`, `tx_bytes`, `endpoint` (o `null`), `created` (o
`null` para peers no administrados por krot). Un resultado vacío es `[]`.

## Códigos de salida

| Código | Significado |
|---|---|
| 0 | éxito |
| 1 | error interno o inesperado (incluido el fallo al aplicar un cambio a una interfaz en ejecución — configuración revertida) |
| 2 | error de uso de CLI (flags; eliminar con clientes sin `--yes` cuando stdin no es una terminal) |
| 3 | error de validación de valores (incluido endpoint ausente, "config too large for QR") |
| 4 | objeto no encontrado (interfaz, cliente) |
| 5 | conflicto: ya existe, dirección o puerto en uso, no hay direcciones libres |
| 6 | entorno: sin root, faltan herramientas, versión de AmneziaWG no soportada |
| 7 | no se pudo adquirir el bloqueo `/run/lock/krot.lock` en 10 segundos |
| 10 | operación cancelada por el usuario |

La salida va a stdout; los errores y advertencias van a stderr con el
prefijo `krot:`.

## Almacenamiento de datos

| Tipo | Configuración | Unidad systemd | Herramienta |
|---|---|---|---|
| `wg` | `/etc/wireguard/<if>.conf` | `wg-quick@<if>` | `wg` |
| `awg`, `awg2`, `awg3` | `/etc/amnezia/amneziawg/<if>.conf` | `awg-quick@<if>` | `awg` |

Los metadatos se almacenan como comentarios que `wg-quick`/`awg-quick`
ignoran:

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

- Un cliente bloqueado es un bloque `[Peer]` comentado con el prefijo `#~ `.
- Los hooks comentados `PreUp`/`PostUp`/`PreDown`/`PostDown` se pueden
  descomentar y rellenar (por ejemplo, con reglas NAT) — krot no los toca.
- Se admiten configuraciones sin metadatos de krot (creadas manualmente):
  `--list` las muestra, `--list-client` imprime sus peers con id `-` (las
  operaciones por `--id` no se aplican a tales peers), y cualquier escritura
  conserva todas las líneas originales, comentarios y orden byte a byte.
- Las escrituras son atómicas (archivo temporal → `fsync` → `rename` →
  `fsync` del directorio), permisos `0600`.
- Todas las acciones que modifican toman un `flock` exclusivo en
  `/run/lock/krot.lock` — las ejecuciones concurrentes (por ejemplo, Salt y
  un humano) no pierden cambios.
- krot no hace copias de seguridad.

## Limitaciones

- NAT, forwarding, firewall y sysctl están fuera del alcance de krot (usa
  los hooks `PostUp`/`PostDown`).
- No se admite la instalación de WireGuard/AmneziaWG ni la migración de
  túneles entre tipos. La generación de firmas de mimetismo usa perfiles
  integrados (`--mimic`); los perfiles QUIC y las firmas personalizadas
  pasan por `--i1` y generadores de terceros, consulta
  [Mimetismo](#mimetismo-paquetes-i1i5).
- `--i1` no debe exceder los 2048 caracteres: debido a
  [amneziawg-tools#69](https://github.com/amnezia-vpn/amneziawg-tools/issues/69),
  los `I1`–`I5` largos (aproximadamente 3600+ caracteres) hacen que
  `awg show` se cuelgue y que `awg set`/`setconf` fallen.
- Las claves privadas de los clientes no se almacenan; las configuraciones
  de cliente no se pueden reemitir.

## Desarrollo

Necesitas Go de `go.mod` (actualmente 1.27.2) y Docker para las pruebas de
integración.

```sh
make check        # gofmt, go vet (incluyendo la etiqueta integration), go test -race ./..., gosec, staticcheck
make build        # bin/krot estático (CGO_ENABLED=0), versión de git describe
make build-arm64
make integration  # pruebas de integración en un contenedor Docker privilegiado con systemd
```

Reglas del repositorio: [`AGENTS.md`](AGENTS.md) — la especificación primero,
las pruebas antes que el código, un commit por tarea.

- **Las pruebas unitarias** no requieren root y no tocan el sistema: los
  directorios se reemplazan por temporales, los comandos externos (`wg`,
  `awg`, `systemctl`, `ip`, …) se simulan con registro de llamadas. Pruebas
  separadas verifican que las operaciones con clientes llamen solo a
  `wg|awg show` y `wg|awg set <if> peer …`.
- **Los archivos golden** residen en el `testdata/` de cada paquete. Las
  salidas generadas (`internal/app`, `internal/qr`) se actualizan solo
  deliberadamente — con `go test ./internal/app ./internal/qr -update` y
  una explicación en el commit; otros archivos golden se escriben
  manualmente a partir de la especificación.
- **Las pruebas de integración** (`test/integration`, etiqueta de
  compilación `integration`) requieren root y systemd y se ejecutan solo en
  espacios de nombres de red separados: las unidades de interfaz de prueba
  se colocan en netns mediante el drop-in `NetworkNamespacePath`. Cubren el
  ciclo completo para todos los tipos, la transparencia de las operaciones
  con clientes (ping continuo sin pérdida, ifindex y handshake sin cambios),
  IPv6, configuraciones creadas manualmente y ejecuciones concurrentes.
  Localmente mediante `scripts/integration-docker.sh`, en CI bajo `sudo`.
- `scripts/collect-versions.sh` reconstruye salidas reales de versiones de
  AmneziaWG para las pruebas golden.
- **CI** (`.github/workflows/ci.yml`): gofmt, vet, pruebas unitarias con
  `-race` y cobertura, auditoría de seguridad `govulncheck` + `gosec`,
  `staticcheck`, pruebas de integración, compilaciones `amd64`/`arm64`.
  **Release** (`release.yml`, etiqueta `v*`): binarios, paquetes
  `.deb`/`.rpm` (nfpm, `packaging/nfpm.yaml`), pruebas de humo de paquetes
  en todos los SO (`scripts/smoke.sh`), GitHub Release con `SHA256SUMS`.

## Licencia

[Apache License 2.0](LICENSE). Copyright 2026 Sergey Ovchinnikov.
