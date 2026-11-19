package cli

const usageTextRussian = `krot — управление туннелями WireGuard и AmneziaWG

Использование:
  krot <действие> [флаги]

Действия (ровно одно):
  --list            показать список туннелей
  --create          создать туннель
  --delete          удалить туннель
  --up              запустить интерфейс
  --down            остановить интерфейс
  --add-client      добавить клиента
  --delete-client   удалить клиента
  --lock-client     заблокировать клиента
  --unlock-client   разблокировать клиента
  --list-client     показать список клиентов
  --doctor          проверить окружение (утилиты, модули ядра, версии AmneziaWG)
  --udp-listener    UDP-сервер для диагностики портов
  --udp-checker     UDP-клиент для диагностики портов
  --help            показать эту справку
  --version         показать версию, коммит и дату сборки

Флаги (--flag value или --flag=value):
  --type wg|awg|awg2|awg3   тип протокола (create: обязателен; list: фильтр)
  --interface NAME          имя интерфейса, до 15 символов [a-zA-Z0-9_.=+-]
  --ip CIDR[,CIDR]          create: адреса сервера (10.8.0.1/24,fd00:8::1/64);
                            add-client: адреса клиента (/32, /128);
                            list: фильтр по адресу
  --port N|N:M              UDP-порт или диапазон 1..65535, не более 1024 портов
                            (create, udp-listener, udp-checker: обязателен; list: фильтр)
  --endpoint HOST           публичный адрес сервера для клиентских конфигов
                            (create: обязателен; add-client: переопределение для клиента)
  --address HOST            адрес удалённого UDP-сервера для --udp-checker (без порта)
  --mtu N                   MTU 576..9000 (по умолчанию: wg 1420, awg/awg2 1376, awg3 1280)
  --dns IP[,IP]             DNS-серверы для клиентских конфигов
                            (create: для всех клиентов; add-client: только для этого клиента)
  --i1 CPS                  пакет-сигнатура I1 AmneziaWG (только awg*, не более 2048 символов)
  --mimic PROFILE           создать цепочку мимикрии I1-I5 для клиентских конфигов:
                            dns|stun|ntp|ssdp|dtls (только awg*; несовместимо с --i1)
  --id ID                   идентификатор клиента [a-zA-Z0-9_.-]{1,64}, уникальный в интерфейсе
  --allowed-ips CIDR[,CIDR] AllowedIPs клиентского конфига, например 0.0.0.0/0,::/0
  --psk                     сгенерировать PresharedKey для клиента
  --qr                      также вывести клиентский конфиг как QR-код
  --amnezia-key             также вывести клиентский конфиг как ключ AmneziaVPN (vpn://...)
  --keenetic                также вывести команды CLI KeeneticOS для роутера Keenetic
                            (туннели wg и awg 1.0/1.5/2.0, только IPv4)
  --json                    вывод JSON вместо таблицы (list, list-client, udp-listener, udp-checker)
  --yes                     не запрашивать подтверждение (delete)
  --timeout DURATION        таймаут ожидания пакета, например 2s или 500ms (по умолчанию 2s)
  --count N                 пакетов каждого типа на порт, 1..100 (по умолчанию 3)
  --echo-size N             размер данных echo-запроса, 1..1400 (по умолчанию 64, только udp-checker)
  --workers N               одновременных проверок портов, 1..100 (по умолчанию 32, только udp-checker)
  --verbose                 диагностика выполнения (udp-listener, udp-checker; stderr)

Флаги для действий:
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
  --doctor         (без флагов; работает без root)
  --udp-listener   --port [--timeout] [--count] [--json] [--verbose]
  --udp-checker    --address --port [--timeout] [--count] [--echo-size] [--workers] [--json] [--verbose]

Коды выхода:
  0 успех                   5 конфликт (уже существует, адрес или порт занят)
  1 внутренняя ошибка       6 окружение (нет root, утилит, нужной версии AmneziaWG)
  2 ошибка использования    7 не удалось получить блокировку
  3 недопустимое значение   10 отменено пользователем
  4 не найдено

Права root:
  Не требуются для --help, --version, --doctor и --udp-checker.
  --udp-listener требует root только для портов ниже 1024 или когда
  /proc/net/udp и /proc/net/udp6 недоступны для чтения.
  Все остальные действия требуют root.
`
