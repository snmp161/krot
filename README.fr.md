# krot

[English](README.md) | [Русский](README.ru.md) | [简体中文](README.zh.md) | [繁體中文](README.zh-Hant.md) | [Español](README.es.md) | [Português](README.pt.md) | [한국어](README.ko.md) | [日本語](README.ja.md) | [العربية](README.ar.md) | **Français** | [Deutsch](README.de.md)

`krot` est un utilitaire en ligne de commande pour gérer des tunnels
**WireGuard** et **AmneziaWG** (1.x, 2.0, 3.1) sur des serveurs Linux : créer et
supprimer des interfaces, gérer les clients (pairs) et générer des configurations
clients, des codes QR et des clés AmneziaVPN (`vpn://…`).

Le nom **krot** signifie « taupe » en russe et dans plusieurs autres langues
slaves : cet animal creuse des tunnels.

- Un binaire statique unique sans dépendances (Go, `CGO_ENABLED=0`).
- Les opérations sur les clients **ne redémarrent ni ne recréent jamais
  l'interface** : seuls des appels ciblés `wg|awg set <if> peer …` sont utilisés,
  sans affecter les autres clients.
- La configuration de l'interface est l'unique référence : aucune base de données
  distincte ; les métadonnées krot sont stockées dans des commentaires, et les
  configurations non créées par krot sont lues et préservées.
- Sortie prévisible : tableaux pour les utilisateurs, `--json` et codes de sortie
  pour l'automatisation.
- krot ne configure ni NAT, ni forwarding, ni règles de pare-feu.

Spécification technique complète : [`docs/SPEC.md`](docs/SPEC.md).

## Prérequis

| | |
|---|---|
| OS | Ubuntu 22.04 / 24.04 / 26.04, Debian 12 / 13, AlmaLinux (Rocky, RHEL) 9 / 10 |
| Architectures | `amd64`, `arm64` |
| Pour `--type wg` | `wireguard-tools` (`wg`, `wg-quick`), module noyau WireGuard |
| Pour `--type awg*` | `amneziawg-tools` (`awg`, `awg-quick`) et module noyau AmneziaWG **ou** `amneziawg-go` |
| Toujours | `iproute2` (`ip`), `systemd` |
| Privilèges | root (sauf `--help`, `--version`, `--doctor`, `--udp-checker` ; `--udp-listener` ne nécessite root que pour les ports `<1024` ou si `/proc/net/udp{,6}` n'est pas lisible) |

krot n'installe aucun de ces composants. Si un programme requis est absent,
il quitte avec le code `6` et indique le paquet à installer.

### Versions d'AmneziaWG

Avant `--create`, krot vérifie que la version d'AmneziaWG installée prend en
charge le type demandé :

| Type | amneziawg-tools | module noyau | famille netlink | amneziawg-go |
|---|---|---|---|---|
| `awg` (1.0) | 1.0.20240213 | — | ≥ 1 | 0.1.0 |
| `awg` avec `--i1` (1.5) | 1.0.20250901 | 1.0.20251009 | ≥ 2 | 0.2.13 |
| `awg2` (2.0) | 1.0.20250901 | 1.0.20251009 | ≥ 2 | 0.2.15 |
| `awg3` (3.1) | 3.1.20260812 | 3.1.20260812 | ≥ 3 | 3.1.20260812 |

Certaines compilations signalent une version factice (par exemple, `awg --version`
sur les outils 1.x affiche presque toujours `1.0.20210914`, et un module compilé
avec `make` indique `1.0.0`). Ces versions produisent un avertissement, pas une
erreur ; le code `6` n'est retourné que si la version est détectée de manière
fiable et qu'elle est trop ancienne. Si la version est réellement trop ancienne,
le démarrage de l'unité échoue et krot annule les modifications.

## Installation

Effectuez l'installation depuis les paquets ou les binaires de la page des releases
<https://github.com/snmp161/krot/releases>, et vérifiez toujours la somme de contrôle.

`gh attestation verify` nécessite GitHub CLI 2.49 ou une version ultérieure. Si votre
distribution fournit une version plus ancienne (par exemple, Ubuntu 24.04), installez
un `gh` récent depuis les [releases officielles de GitHub CLI](https://github.com/cli/cli/releases).

```sh
VERSION=0.7.4
BASE=https://github.com/snmp161/krot/releases/download/v$VERSION

# télécharger le paquet et la liste des sommes de contrôle
wget "$BASE/krot_${VERSION}_amd64.deb" "$BASE/SHA256SUMS"
# vérifier : doit afficher "krot_..._amd64.deb: OK"
gh attestation verify SHA256SUMS --repo snmp161/krot
sha256sum --check --ignore-missing SHA256SUMS
# installer
sudo dpkg -i "krot_${VERSION}_amd64.deb"
```

Systèmes utilisant RPM (AlmaLinux, Rocky, RHEL) :

```sh
wget "$BASE/krot-${VERSION}-1.x86_64.rpm" "$BASE/SHA256SUMS"
gh attestation verify SHA256SUMS --repo snmp161/krot
sha256sum --check --ignore-missing SHA256SUMS
sudo rpm -i "krot-${VERSION}-1.x86_64.rpm"
```

Binaire sans paquet : téléchargez `krot-linux-amd64` (ou `krot-linux-arm64`)
et `SHA256SUMS`, effectuez la même vérification, puis installez avec
`sudo install -m 0755 krot-linux-amd64 /usr/bin/krot`.

```sh
wget "$BASE/LICENSE" "$BASE/THIRD_PARTY_NOTICES"
```

Pour `arm64`, utilisez les fichiers `*_arm64.deb`, `*.aarch64.rpm` et `krot-linux-arm64`.

## Utilisation

```
krot <action> [flags]
```

Une seule action par exécution ; les options utilisent uniquement la forme longue
`--flag value` ou `--flag=value`. Une option inapplicable à l'action constitue une
erreur (code `2`). Aide complète : `krot --help`.

### Tunnels

```sh
# créer AmneziaWG 2.0 (dual-stack) ; nom choisi automatiquement : awg0, awg1, …
sudo krot --create --type awg2 --ip 10.8.0.1/24,fd00:8::1/64 --port 51820 \
          --endpoint vpn.example.com --dns 1.1.1.1,9.9.9.9

# WireGuard avec un nom et un MTU explicites
sudo krot --create --type wg --interface wg0 --ip 10.9.0.1/24 --port 51821 \
          --endpoint 203.0.113.10 --mtu 1420

# AmneziaWG 1.5 avec un paquet de signature I1 (krot valide la syntaxe CPS et la longueur, jusqu'à 2048 caractères)
sudo krot --create --type awg --ip 10.10.0.1/24 --port 51822 --endpoint vpn.example.com \
          --i1 '<b 0xf6ab3267fa><r 16>'

# AmneziaWG 3.1 (Header Protection, RandomTrailers, DisableCookies sont toujours activés)
sudo krot --create --type awg3 --ip 10.11.0.1/24 --port 51823 --endpoint vpn.example.com

sudo krot --list                         # tous les tunnels
sudo krot --list --type awg2 --json      # filtres --type/--interface/--ip/--port, combinés par ET
sudo krot --down --interface awg0        # systemctl stop awg-quick@awg0
sudo krot --up --interface awg0          # systemctl start ; un --up répété affiche "already up", code 0
sudo krot --delete --interface awg0      # demande une confirmation si des clients existent
sudo krot --delete --interface awg0 --yes
```

### Diagnostics

```sh
krot --doctor   # fonctionne sans root : outils, modules noyau, versions d'AmneziaWG, répertoires
```

La sortie est un tableau `CHECK/STATUS/DETAIL` avec les statuts `ok` / `warn` /
`fail`. Le code de sortie est `0` s'il n'y a aucune entrée `fail`, sinon `6`.
Cette action ne modifie rien et ne prend aucun verrou.

#### Diagnostic des ports UDP

`krot` peut vérifier quels ports UDP sont réellement accessibles entre deux hôtes.
Cela est utile lorsqu'un hébergeur ou sa liaison montante filtre certains ports,
par exemple en raison de règles anti-DDoS.

Démarrez le listener sur le serveur :

```sh
# port unique
sudo krot --udp-listener --port 51820

# plage de ports (1024 ports maximum, un socket par port)
sudo krot --udp-listener --port 51820:51830
```

Le listener répond à trois types de datagrammes :
- `ping SEQ:N` → `pong SEQ:N` ;
- `ECHO <text>` → `<text>` ;
- `TIME` → heure Unix actuelle en millisecondes.

Le listener est un réflecteur UDP ouvert sur toutes les interfaces. Exécutez-le
uniquement pendant le diagnostic et arrêtez-le ensuite avec `Ctrl+C`/`SIGTERM`.

Exécutez le checker sur le client :

```sh
# vérifier un seul port
krot --udp-checker --address 203.0.113.1 --port 51820

# vérifier une plage
krot --udp-checker --address 203.0.113.1 --port 51820:51830

# augmenter la taille des données echo (1–1400 octets, 64 par défaut) — aide à déterminer le meilleur MTU
krot --udp-checker --address 203.0.113.1 --port 51820 --echo-size 1400

# définir le nombre de workers et afficher la progression
krot --udp-checker --address 203.0.113.1 --port 51820:51830 --workers 64 --verbose
```

Par défaut, le checker envoie 3 paquets de chaque type par port (`--count`,
1–100) et attend une réponse pendant `--timeout` (2s par défaut). Il vérifie
32 ports en parallèle ; utilisez `--workers` pour définir 1–100 workers.
`--verbose` écrit la plage et les résultats de chaque port sur stderr dès qu'ils
sont disponibles ; sur le listener, il journalise les adresses des clients et les
types de requêtes. Sortie :

```
PORT     STATUS   PING    ECHO    TIME (min/avg/max)   DETAIL
51820    ok       3/3     3/3     12/24/45 ms          -
51821    filtered 0/3     0/3     -                    timeout
```

Statuts :
- `ok` — tous les paquets ping et echo ont reçu des réponses valides ;
- `filtered` — au moins un paquet a dépassé le délai (le port est probablement filtré) ;
- `error` — une réponse invalide a été reçue.

Avec `--json`, la sortie est un tableau d'objets comportant les champs `port`,
`status`, `ping_sent`, `ping_ok`, `echo_sent`, `echo_ok`, `time_sent`, `time_ok`,
`rtt_min_ms`, `rtt_avg_ms`, `rtt_max_ms`, `detail`.

Avec `Ctrl+C`/`SIGTERM`, le checker affiche les résultats des ports entièrement
vérifiés dans le format choisi et quitte avec le code `10` ; les ports non terminés
sont omis.

`--udp-checker` fonctionne sans root. `--udp-listener` ne nécessite root que si
la plage comprend des ports `<1024` ou si `/proc/net/udp` et `/proc/net/udp6`
ne sont pas lisibles par l'utilisateur courant. Le listener fonctionne jusqu'à
`Ctrl+C`/`SIGTERM`.

Exemple de `--list` :

```
INTERFACE  TYPE  VERSION  ADDRESS                   PORT   STATUS  CLIENTS
awg0       awg2  2.0      10.8.0.1/24,fd00:8::1/64  51820  up      12 (1 locked)
wg0        wg    -        10.9.0.1/24               51821  down    3
```

MTU par défaut : `wg` — 1420, `awg`/`awg2` — 1376, `awg3` — 1280. Les paramètres
d'obfuscation (Jc, S1–S4, H1–H4, …) sont générés automatiquement dans des plages
sûres et validés (section 7 de la spécification).

### Clients

```sh
# ajouter un client : les adresses sont attribuées automatiquement (premiers /32 et /128 libres) ;
# la configuration client n'est affichée qu'UNE SEULE FOIS — la clé privée du client n'est jamais stockée
sudo krot --add-client --interface awg0 --id phone-sergei --allowed-ips 0.0.0.0/0,::/0

# avec une PresharedKey et un code QR dans le terminal
sudo krot --add-client --interface awg0 --id laptop --allowed-ips 0.0.0.0/0 --psk --qr

# afficher aussi une clé AmneziaVPN : une ligne vpn://… après la configuration
sudo krot --add-client --interface awg0 --id tablet --allowed-ips 0.0.0.0/0,::/0 --amnezia-key

# commandes CLI KeeneticOS : connecter un routeur Keenetic comme client (wg et awg 1.0/1.5/2.0, IPv4 uniquement)
sudo krot --add-client --interface wg0 --id keenetic --allowed-ips 0.0.0.0/0 --keenetic

# adresse client spécifique
sudo krot --add-client --interface awg0 --id router --allowed-ips 10.8.0.0/24 --ip 10.8.0.50/32

# configuration externe sans métadonnées krot : passer l'endpoint explicitement (non écrit dans la configuration serveur)
sudo krot --add-client --interface office --id guest --allowed-ips 0.0.0.0/0 --endpoint 203.0.113.10

# DNS personnalisé pour un seul client (non écrit dans la configuration serveur)
sudo krot --add-client --interface awg0 --id kid-phone --allowed-ips 0.0.0.0/0 --dns 94.140.14.15

sudo krot --list-client --interface awg0
sudo krot --list-client --interface awg0 --id phone-sergei --json
sudo krot --lock-client   --interface awg0 --id laptop    # répété — "already locked", code 0
sudo krot --unlock-client --interface awg0 --id laptop
sudo krot --delete-client --interface awg0 --id laptop
```

`--allowed-ips` définit `AllowedIPs` dans la configuration **client** (ce que le
client route via le tunnel). DNS du client : `--dns` s'il est fourni ; sinon les
métadonnées `# krot:dns` de l'interface (`--dns` lors de `--create`) ; sinon les
lignes `DNS` de la section `[Interface]` de la configuration serveur ; si aucun DNS
n'est défini, la configuration client ne contient pas de ligne `DNS`.

Les adresses des clients sont attribuées dans le sous-réseau de l'interface, à
l'exclusion de l'adresse du réseau, de l'adresse broadcast et de l'adresse du
serveur. Les sous-réseaux point à point `/31` et `/127` (RFC 3021, RFC 6164) sont
également pris en charge : les deux adresses sont utilisables, ce qui permet de
créer un tunnel « serveur + un client » avec
`--create … --ip 10.8.0.0/31,fd00:8::/127`.

Enregistrez immédiatement la configuration générée : elle ne peut pas être
réaffichée ; vous pouvez seulement supprimer le client puis l'ajouter à nouveau.
Les codes QR sont affichés avec des demi-blocs Unicode et supposent un terminal
à l'arrière-plan sombre.

`--amnezia-key` affiche une clé AmneziaVPN après la configuration : une seule ligne
`vpn://…` à coller dans l'application AmneziaVPN (« Insert key »), par exemple en
l'envoyant via une messagerie. Elle contient la même configuration client :
`vpn://` + base64url sans `=` du résultat de `qCompress` de Qt (longueur sur
4 octets big-endian + zlib). La clé fonctionne pour tous les types, y compris `wg`.
Le code QR de `--qr` contient le texte de la configuration — AmneziaVPN et
AmneziaWG peuvent le scanner ; pour `wg`, WireGuard le peut également.

`--keenetic` affiche un bloc de commandes CLI KeeneticOS après la configuration
pour connecter un routeur Keenetic comme client du tunnel. Le bloc respecte la
syntaxe running-config de KeeneticOS et peut être collé tel quel dans le CLI du
routeur (SSH/telnet ou l'interface web « Command-line interface »). KeeneticOS
prend en charge l'obfuscation AmneziaWG (paramètres ASC) : AWG 1.0 depuis
KeeneticOS 4.2, AWG 1.5/2.0 depuis 5.1 ; pour les tunnels awg, le bloc comprend
une ligne `wireguard asc …` (les chaînes CPS I1–I5 sont entre guillemets ;
l'importation du fichier de configuration via l'interface web est plus fiable).
AWG 3.1 n'est pas pris en charge. IPv4 uniquement ; si le MTU du tunnel diffère
de la valeur par défaut 1420, une remarque est affichée. Voir la section 6.13 de
la spécification pour le format.

Exemple de `--list-client` :

```
ID            PUBLIC KEY  ADDRESS                    STATUS  LAST HANDSHAKE  RX       TX
phone-sergei  aB3d…Xy9=   10.8.0.2/32,fd00:8::2/128  active  1m ago          12.4MiB  1.1GiB
laptop-old    Qw7e…Zz1=   10.8.0.3/32                locked  -               -        -
```

### Mimétisme (paquets I1–I5)

Depuis la version 1.5, AmneziaWG peut envoyer jusqu'à cinq paquets de signature
`I1`–`I5` avant chaque handshake : des paquets qui ressemblent au début d'un autre
protocole (DNS, STUN, NTP, SSDP, DTLS). krot génère lui-même ces chaînes avec le
option `--mimic <profile>` :

```sh
# profil par défaut pour tous les clients d'une interface
krot --create --type awg --ip 10.8.0.1/24 --port 51820 --endpoint vpn.example.com --mimic dns
# remplacement pour un seul client
krot --add-client --interface awg0 --id phone --allowed-ips 0.0.0.0/0 --mimic stun
```

Profils : `dns` (trois requêtes A/AAAA/HTTPS), `stun` (Binding Request), `ntp`,
`ssdp` (M-SEARCH), `dtls` (ClientHello). La chaîne est générée lors de
`--add-client` (chaque client reçoit la sienne) et écrite uniquement dans la
configuration client : les paquets I sont envoyés par l'initiateur du handshake,
le serveur n'en a donc pas besoin. Le client nécessite AmneziaWG 1.5+ — voir la
section 6.12 de la spécification.

Outre les profils intégrés, krot accepte une chaîne CPS prête à l'emploi via `--i1`
lors de `--create`, par exemple pour QUIC. Les profils QUIC intégrés ne sont pas
générés, car un seul QUIC Initial contient 2400–2500 caractères et dépasse la
limite de 2048 (en raison de
[amneziawg-tools#69](https://github.com/amnezia-vpn/amneziawg-tools/issues/69)).
Une chaîne prête à l'emploi peut être obtenue auprès de générateurs tiers
(tous deux sous licence MIT et non affiliés à krot) :

- [payloadGen](https://github.com/Sketchystan1/payloadGen) — une page web : ouvrez
  `index.html` dans un navigateur, sélectionnez un protocole et prenez la première ligne (I1) ;
- `cpsgen.py` de [awg-multi-script](https://github.com/pumbaX/awg-multi-script)
  (`src/py/cpsgen.py`) — version console de payloadGen en Python 3, sans root.
  Profils : `dns`, `ntp`, `rtp`, `stun`, `sip`, `ssdp`, `webrtc`, `quic`,
  `curl_quic` ; le second argument est le domaine.

```sh
# télécharger le générateur et examiner son code avant de l'exécuter
git clone --depth 1 https://github.com/pumbaX/awg-multi-script
less awg-multi-script/src/py/cpsgen.py

# générer uniquement I1 et vérifier sa longueur (2048 caractères maximum)
I1=$(python3 awg-multi-script/src/py/cpsgen.py dns example.com --only-i1)
echo ${#I1}

sudo krot --create --type awg2 --ip 10.8.0.1/24 --port 51820 --endpoint vpn.example.com --i1 "$I1"
```

Remarques :

- `I1` est défini uniquement lors de `--create`, écrit dans la configuration du
  serveur et copié dans toutes les configurations clients. krot n'accepte pas
  `I2`–`I5`.
- La longueur ne doit pas dépasser 2048 caractères (voir [Limites](#limites)),
  sinon le code est `3`. Les paquets des profils `quic` et `curl_quic` font environ
  2400 caractères et ne conviennent pas. Des profils plus courts fonctionnent :
  `rtp` (~30 caractères), `dns` et `ntp` (~100), `stun` (~250–350), `ssdp` (~400),
  `sip` (~1200).
- `--type awg` avec `--i1` signifie AmneziaWG 1.5 avec ses propres versions
  minimales (voir [Versions d'AmneziaWG](#versions-damneziawg)). Les tags
  `<rc N>`/`<rd N>` (produits par le profil `sip`) sont reconnus par amneziawg-go
  depuis 0.2.15 ; pour 1.5, privilégiez donc les profils qui ne les utilisent pas.
- Le client doit prendre en charge AmneziaWG 1.5 ou une version plus récente.
- Avec un `I1` long, la configuration peut ne pas tenir dans un code QR : `--qr`
  quitte alors avec le code `3` (le client est déjà ajouté et la configuration est
  affichée). Transférez la configuration sous forme de fichier ou utilisez
  `--amnezia-key`.

### JSON

`--list --json` produit un tableau d'objets avec `interface`, `type`, `version`,
`addresses[]`, `port`, `status`, `clients_total`, `clients_locked`.
`--list-client --json` produit un tableau d'objets avec `id`, `public_key`,
`addresses[]`, `status`, `latest_handshake` (heure Unix ou `null`), `rx_bytes`,
`tx_bytes`, `endpoint` (ou `null`), `created` (ou `null` pour les pairs non gérés par krot).
Un résultat vide est `[]`.

## Codes de sortie

| Code | Signification |
|---|---|
| 0 | succès |
| 1 | erreur interne ou inattendue (y compris l'échec d'application d'une modification à une interface active — configuration restaurée) |
| 2 | erreur d'utilisation du CLI (options ; suppression avec des clients sans `--yes` lorsque stdin n'est pas un terminal) |
| 3 | erreur de validation d'une valeur (y compris endpoint absent, « config too large for QR ») |
| 4 | objet introuvable (interface, client) |
| 5 | conflit : existe déjà, adresse ou port occupé, aucune adresse libre |
| 6 | environnement : absence de root, outils manquants, version d'AmneziaWG non prise en charge |
| 7 | impossible d'obtenir le verrou `/run/lock/krot.lock` dans les 10 secondes |
| 10 | opération annulée par l'utilisateur |

La sortie est écrite sur stdout ; les erreurs et avertissements sont écrits sur
stderr avec le préfixe `krot:`.

## Stockage des données

| Type | Configuration | Unité systemd | Outil |
|---|---|---|---|
| `wg` | `/etc/wireguard/<if>.conf` | `wg-quick@<if>` | `wg` |
| `awg`, `awg2`, `awg3` | `/etc/amnezia/amneziawg/<if>.conf` | `awg-quick@<if>` | `awg` |

Les métadonnées sont stockées dans des commentaires que `wg-quick`/`awg-quick`
ignorent :

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

- Un client verrouillé est un bloc `[Peer]` commenté avec le préfixe `#~ `.
- Les hooks `PreUp`/`PostUp`/`PreDown`/`PostDown` commentés peuvent être
  décommentés et complétés (par exemple, avec des règles NAT) — krot ne les modifie pas.
- Les configurations sans métadonnées krot (créées manuellement) sont prises en
  charge : `--list` les affiche, `--list-client` affiche leurs pairs avec l'id `-`
  (les opérations par `--id` ne s'appliquent pas à ces pairs), et toute écriture
  préserve les lignes, commentaires et l'ordre d'origine, octet par octet.
- Les écritures sont atomiques (fichier temporaire → `fsync` → `rename` → `fsync`
  du répertoire), avec les permissions `0600`.
- Toutes les actions de modification prennent un `flock` exclusif sur
  `/run/lock/krot.lock` — les exécutions concurrentes (par exemple, Salt et un
  utilisateur) ne perdent pas de modifications.
- krot ne crée pas de sauvegardes.

## Limites

- NAT, forwarding, pare-feu et sysctl sont hors du périmètre de krot (utilisez les
  hooks `PostUp`/`PostDown`).
- L'installation de WireGuard/AmneziaWG et la migration des tunnels entre types ne
  sont pas prises en charge. La génération de signatures de mimétisme utilise les
  profils intégrés (`--mimic`) ; les profils QUIC et les signatures personnalisées
  passent par `--i1` et des générateurs tiers, voir [Mimétisme](#mimétisme-paquets-i1i5).
- `--i1` ne doit pas dépasser 2048 caractères : en raison de
  [amneziawg-tools#69](https://github.com/amnezia-vpn/amneziawg-tools/issues/69),
  des `I1`–`I5` longs (environ 3600 caractères ou plus) bloquent `awg show` et
  font échouer `awg set`/`setconf`.
- Les clés privées des clients ne sont pas stockées ; les configurations clients
  ne peuvent pas être régénérées.

## Développement

Vous avez besoin de Go depuis `go.mod` (actuellement 1.27.4) et de Docker pour
les tests d'intégration.

```sh
make check        # toutes les vérifications locales : format, tests, modules, sécurité et workflows
make fuzz-smoke   # fuzzing limité de toutes les targets
make build        # bin/krot statique (CGO_ENABLED=0), version depuis git describe
make build-arm64
make integration  # tests d'intégration dans un conteneur Docker privilégié avec systemd
make changelog      # régénérer le changelog à partir des notes de version
make repro-check    # vérifier des builds reproductibles octet par octet
make package-preflight # DEB/RPM pour les deux architectures + smoke Ubuntu 24.04/AlmaLinux 9
```

Règles du dépôt : [`AGENTS.md`](AGENTS.md) — spécification d'abord, tests avant
le code, un commit par tâche.

- Les **tests unitaires** ne nécessitent pas root et ne modifient pas le système :
  les répertoires sont remplacés par des répertoires temporaires, les commandes
  externes (`wg`, `awg`, `systemctl`, `ip`, …) sont simulées avec enregistrement
  des appels. Des tests distincts vérifient que les opérations sur les clients
  appellent uniquement `wg|awg show` et `wg|awg set <if> peer …`.
- Les **fichiers golden** se trouvent dans `testdata/` des paquets. Les sorties
  générées (`internal/app`, `internal/qr`) ne sont mises à jour que délibérément —
  avec `go test ./internal/app ./internal/qr -update` et une explication dans le
  commit ; les autres fichiers golden sont écrits manuellement d'après la spécification.
- Les **tests d'intégration** (`test/integration`, tag de build `integration`)
  nécessitent root et systemd et s'exécutent uniquement dans des espaces de noms
  réseau distincts : les unités des interfaces de test sont placées dans le netns via un
  drop-in `NetworkNamespacePath`. Ils couvrent le cycle complet de tous les types,
  la transparence des opérations sur les clients (ping continu sans perte, ifindex
  et handshake inchangés), IPv6, les configurations externes et les exécutions
  concurrentes. Localement via `scripts/integration-docker.sh`, en CI sous `sudo`.
- `scripts/collect-versions.sh` reconstruit les sorties réelles des versions
  d'AmneziaWG pour les tests golden.
- **CI** (`.github/workflows/ci.yml`) : contrôles locaux de qualité, tests unitaires
  avec shuffle, `-race` et couverture (minimum 90 %), fuzzing limité, tests
  d'intégration, package preflight et compilations `amd64`/`arm64`. **Release** (`release.yml`, tag `v*`) :
  binaires, paquets `.deb`/`.rpm` (nfpm, `packaging/nfpm.yaml`), smoke tests des
  paquets sur tous les OS (`scripts/smoke.sh`), GitHub Release avec `SHA256SUMS`.

## Licence

[Apache License 2.0](LICENSE). Copyright 2026 Sergey Ovchinnikov.
