package cli

const usageTextSpanish = `krot — gestiona túneles WireGuard y AmneziaWG

Uso:
  krot <acción> [flags]

Acciones (exactamente una):
  --list            lista los túneles
  --create          crea un túnel
  --delete          elimina un túnel
  --up              inicia una interfaz
  --down            detiene una interfaz
  --add-client      agrega un cliente
  --delete-client   elimina un cliente
  --lock-client     bloquea un cliente
  --unlock-client   desbloquea un cliente
  --list-client     lista los clientes
  --doctor          comprueba el entorno (herramientas, módulos de kernel, versiones de AmneziaWG)
  --udp-listener    listener UDP para diagnosticar puertos
  --udp-checker     checker UDP para diagnosticar puertos
  --help            muestra esta ayuda
  --version         muestra la versión, el commit y la fecha de compilación

Flags (--flag value o --flag=value):
  --type wg|awg|awg2|awg3   tipo de protocolo (create: obligatorio; list: filtro)
  --interface NAME          nombre de interfaz, hasta 15 caracteres [a-zA-Z0-9_.=+-]
  --ip CIDR[,CIDR]          create: direcciones del servidor (10.8.0.1/24,fd00:8::1/64);
                            add-client: direcciones del cliente (/32, /128);
                            list: filtro por dirección
  --port N|N:M              puerto UDP o rango 1..65535, como máximo 1024 puertos
                            (create, udp-listener, udp-checker: obligatorio; list: filtro)
  --endpoint HOST           dirección pública del servidor para configuraciones de cliente
                            (create: obligatorio; add-client: sustitución por cliente)
  --address HOST            dirección del listener remoto para --udp-checker (sin puerto)
  --mtu N                   MTU 576..9000 (predeterminado: wg 1420, awg/awg2 1376, awg3 1280)
  --dns IP[,IP]             servidores DNS para configuraciones de cliente
                            (create: todos los clientes; add-client: solo este cliente)
  --i1 CPS                  paquete de firma I1 de AmneziaWG (solo awg*, hasta 2048 caracteres)
  --mimic PROFILE           genera una cadena de mimicry I1-I5 para configuraciones de cliente:
                            dns|stun|ntp|ssdp|dtls (solo awg*; no con --i1)
  --id ID                   ID de cliente [a-zA-Z0-9_.-]{1,64}, único dentro de una interfaz
  --allowed-ips CIDR[,CIDR] AllowedIPs de la configuración de cliente, p. ej. 0.0.0.0/0,::/0
  --psk                     genera una PresharedKey para el cliente
  --qr                      también imprime la configuración de cliente como código QR
  --amnezia-key             también imprime la configuración como clave AmneziaVPN (vpn://...)
  --keenetic                también imprime comandos CLI de KeeneticOS para un router Keenetic
                            (túneles wg y awg 1.0/1.5/2.0, solo IPv4)
  --json                    salida JSON en lugar de tabla (list, list-client, udp-listener, udp-checker)
  --yes                     no pide confirmación (delete)
  --timeout DURATION        tiempo de espera del paquete, p. ej. 2s o 500ms (predeterminado: 2s)
  --count N                 paquetes de cada tipo por puerto, 1..100 (predeterminado: 3)
  --echo-size N             tamaño del payload echo, 1..1400 (predeterminado: 64, solo udp-checker)
  --workers N               comprobaciones simultáneas de puertos, 1..100 (predeterminado: 32, solo udp-checker)
  --verbose                 progreso de diagnóstico (udp-listener, udp-checker; stderr)

Flags por acción:
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
  --doctor         (sin flags; funciona sin root)
  --udp-listener   --port [--json] [--verbose]
  --udp-checker    --address --port [--timeout] [--count] [--echo-size] [--workers] [--json] [--verbose]

Códigos de salida:
  0 éxito                   5 conflicto (ya existe, dirección o puerto en uso)
  1 error interno           6 entorno (sin root, faltan herramientas, versión de AmneziaWG)
  2 error de uso            7 no se pudo obtener el bloqueo
  3 valor no válido         10 cancelado por el usuario
  4 no encontrado

Privilegios de root:
  No se requieren para --help, --version, --doctor ni --udp-checker.
  --udp-listener requiere root solo para puertos inferiores a 1024 o cuando
  /proc/net/udp y /proc/net/udp6 no se pueden leer.
  Todas las demás acciones requieren root.
`
