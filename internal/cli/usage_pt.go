package cli

const usageTextPortuguese = `krot — gerencia túneis WireGuard e AmneziaWG

Uso:
  krot <ação> [flags]

Ações (exatamente uma):
  --list            lista os túneis
  --create          cria um túnel
  --delete          exclui um túnel
  --up              inicia uma interface
  --down            para uma interface
  --add-client      adiciona um cliente
  --delete-client   exclui um cliente
  --lock-client     bloqueia um cliente
  --unlock-client   desbloqueia um cliente
  --list-client     lista os clientes
  --doctor          verifica o ambiente (ferramentas, módulos de kernel, versões do AmneziaWG)
  --udp-listener    listener UDP para diagnóstico de portas
  --udp-checker     checker UDP para diagnóstico de portas
  --help            mostra esta ajuda
  --version         mostra versão, commit e data de compilação

Flags (--flag value ou --flag=value):
  --type wg|awg|awg2|awg3   tipo de protocolo (create: obrigatório; list: filtro)
  --interface NAME          nome da interface, até 15 caracteres [a-zA-Z0-9_.=+-]
  --ip CIDR[,CIDR]          create: endereços do servidor (10.8.0.1/24,fd00:8::1/64);
                            add-client: endereços do cliente (/32, /128);
                            list: filtro por endereço
  --port N|N:M              porta UDP ou faixa 1..65535, no máximo 1024 portas
                            (create, udp-listener, udp-checker: obrigatório; list: filtro)
  --endpoint HOST           endereço público do servidor para configurações de cliente
                            (create: obrigatório; add-client: substituição por cliente)
  --address HOST            endereço do listener remoto para --udp-checker (sem porta)
  --mtu N                   MTU 576..9000 (padrão: wg 1420, awg/awg2 1376, awg3 1280)
  --dns IP[,IP]             servidores DNS para configurações de cliente
                            (create: todos os clientes; add-client: somente este cliente)
  --i1 CPS                  pacote de assinatura I1 do AmneziaWG (somente awg*, até 2048 caracteres)
  --mimic PROFILE           gera uma cadeia de mimetismo I1-I5 para configurações de cliente:
                            dns|stun|ntp|ssdp|dtls (somente awg*; não com --i1)
  --id ID                   ID do cliente [a-zA-Z0-9_.-]{1,64}, único dentro de uma interface
  --allowed-ips CIDR[,CIDR] AllowedIPs da configuração do cliente, ex.: 0.0.0.0/0,::/0
  --psk                     gera uma PresharedKey para o cliente
  --qr                      também imprime a configuração do cliente como código QR
  --amnezia-key             também imprime a configuração como chave AmneziaVPN (vpn://...)
  --keenetic                também imprime comandos CLI do KeeneticOS para um roteador Keenetic
                            (túneis wg e awg 1.0/1.5/2.0, somente IPv4)
  --json                    saída JSON em vez de tabela (list, list-client, udp-listener, udp-checker)
  --yes                     não pede confirmação (delete)
  --timeout DURATION        tempo limite do pacote, ex.: 2s ou 500ms (padrão: 2s)
  --count N                 pacotes de cada tipo por porta, 1..100 (padrão: 3)
  --echo-size N             tamanho do payload echo, 1..1400 (padrão: 64, somente udp-checker)
  --workers N               verificações simultâneas de portas, 1..100 (padrão: 32, somente udp-checker)
  --verbose                 progresso do diagnóstico (udp-listener, udp-checker; stderr)

Flags por ação:
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
  --doctor         (sem flags; funciona sem root)
  --udp-listener   --port [--json] [--verbose]
  --udp-checker    --address --port [--timeout] [--count] [--echo-size] [--workers] [--json] [--verbose]

Códigos de saída:
  0 sucesso                 5 conflito (já existe, endereço ou porta em uso)
  1 erro interno            6 ambiente (sem root, ferramentas ausentes, versão do AmneziaWG)
  2 erro de uso             7 não foi possível obter o bloqueio
  3 valor inválido          10 cancelado pelo usuário
  4 não encontrado

Privilégios de root:
  Não são necessários para --help, --version, --doctor ou --udp-checker.
  --udp-listener requer root somente para portas abaixo de 1024 ou quando
  /proc/net/udp e /proc/net/udp6 não podem ser lidos.
  Todas as outras ações requerem root.
`
