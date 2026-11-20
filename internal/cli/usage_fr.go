package cli

const usageTextFrench = `krot — gère les tunnels WireGuard et AmneziaWG

Utilisation :
  krot <action> [options]

Actions (exactement une) :
  --list            liste les tunnels
  --create          crée un tunnel
  --delete          supprime un tunnel
  --up              démarre une interface
  --down            arrête une interface
  --add-client      ajoute un client
  --delete-client   supprime un client
  --lock-client     verrouille un client
  --unlock-client   déverrouille un client
  --list-client     liste les clients
  --doctor          vérifie l’environnement (outils, modules noyau, versions AmneziaWG)
  --udp-listener    listener UDP pour le diagnostic des ports
  --udp-checker     checker UDP pour le diagnostic des ports
  --help            affiche cette aide
  --version         affiche la version, le commit et la date de compilation

Options (--flag value ou --flag=value) :
  --type wg|awg|awg2|awg3   type de protocole (create : obligatoire ; list : filtre)
  --interface NAME          nom d’interface, jusqu’à 15 caractères [a-zA-Z0-9_.=+-]
  --ip CIDR[,CIDR]          create : adresses du serveur (10.8.0.1/24,fd00:8::1/64) ;
                            add-client : adresses du client (/32, /128) ;
                            list : filtre par adresse
  --port N|N:M              port UDP ou plage 1..65535, au maximum 1024 ports
                            (create, udp-listener, udp-checker : obligatoire ; list : filtre)
  --endpoint HOST           adresse publique du serveur pour les configurations client
                            (create : obligatoire ; add-client : remplacement par client)
  --address HOST            adresse du listener distant pour --udp-checker (sans port)
  --mtu N                   MTU 576..9000 (défaut : wg 1420, awg/awg2 1376, awg3 1280)
  --dns IP[,IP]             serveurs DNS pour les configurations client
                            (create : tous les clients ; add-client : ce client uniquement)
  --i1 CPS                  paquet de signature I1 AmneziaWG (awg* uniquement, 2048 caractères max.)
  --mimic PROFILE           génère une chaîne de mimétisme I1-I5 pour les configurations client :
                            dns|stun|ntp|ssdp|dtls (awg* uniquement ; pas avec --i1)
  --id ID                   identifiant client [a-zA-Z0-9_.-]{1,64}, unique dans une interface
  --allowed-ips CIDR[,CIDR] AllowedIPs de la configuration client, ex. 0.0.0.0/0,::/0
  --psk                     génère une PresharedKey pour le client
  --qr                      affiche aussi la configuration client sous forme de QR code
  --amnezia-key             affiche aussi la configuration comme clé AmneziaVPN (vpn://...)
  --keenetic                affiche aussi les commandes CLI KeeneticOS pour un routeur Keenetic
                            (tunnels wg et awg 1.0/1.5/2.0, IPv4 uniquement)
  --json                    sortie JSON au lieu d’un tableau (list, list-client, udp-listener, udp-checker)
  --yes                     ne demande pas de confirmation (delete)
  --timeout DURATION        délai d’attente du paquet, ex. 2s ou 500ms (défaut : 2s)
  --count N                 paquets de chaque type par port, 1..100 (défaut : 3)
  --echo-size N             taille du payload echo, 1..1400 (défaut : 64, udp-checker uniquement)
  --workers N               vérifications de ports simultanées, 1..100 (défaut : 32, udp-checker uniquement)
  --verbose                 progression du diagnostic (udp-listener, udp-checker ; stderr)

Options par action :
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
  --doctor         (aucune option ; fonctionne sans root)
  --udp-listener   --port [--timeout] [--count] [--json] [--verbose]
  --udp-checker    --address --port [--timeout] [--count] [--echo-size] [--workers] [--json] [--verbose]

Codes de sortie :
  0 succès                  5 conflit (existe déjà, adresse ou port utilisé)
  1 erreur interne          6 environnement (pas root, outils absents, version AmneziaWG)
  2 erreur d’utilisation    7 impossible d’obtenir le verrouillage
  3 valeur non valide       10 annulé par l’utilisateur
  4 introuvable

Privilèges root :
  Non requis pour --help, --version, --doctor ou --udp-checker.
  --udp-listener requiert root uniquement pour les ports inférieurs à 1024 ou lorsque
  /proc/net/udp et /proc/net/udp6 ne sont pas lisibles.
  Toutes les autres actions requièrent root.
`
