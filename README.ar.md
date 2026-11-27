# krot

[English](README.md) | [Русский](README.ru.md) | [简体中文](README.zh.md) | [繁體中文](README.zh-Hant.md) | [Español](README.es.md) | [Português](README.pt.md) | [한국어](README.ko.md) | [日本語](README.ja.md) | **العربية** | [Français](README.fr.md) | [Deutsch](README.de.md)

`krot` هي أداة سطر أوامر لإدارة أنفاق **WireGuard** و**AmneziaWG**
(1.x و2.0 و3.1) على خوادم Linux: إنشاء الواجهات وحذفها، وإدارة العملاء (peers)،
وتوليد إعدادات العملاء ورموز QR ومفاتيح AmneziaVPN (`vpn://…`).

يعني اسم **krot** «الخُلد» بالروسية وعدة لغات سلافية أخرى، وهو حيوان يحفر الأنفاق.

- ملف ثنائي ثابت واحد من دون تبعيات (Go، `CGO_ENABLED=0`).
- عمليات العملاء **لا تعيد تشغيل الواجهة أو إعادة إنشائها إطلاقًا**، بل تنفذ فقط
  استدعاءات محددة الهدف `wg|awg set <if> peer …`، ولا تتأثر بقية العملاء.
- ملف إعدادات الواجهة هو المرجع الوحيد المعتمد؛ فلا توجد قواعد بيانات منفصلة،
  وتُحفظ بيانات krot الوصفية في التعليقات، كما تُقرأ ملفات الإعدادات التي لم تنشئها krot وتُحفظ كما هي.
- مخرجات متوقعة: جداول للمستخدمين، و`--json` ورموز خروج لأغراض الأتمتة.
- لا تضبط krot قواعد NAT أو إعادة التوجيه أو الجدار الناري.

المواصفات التقنية الكاملة: [`docs/SPEC.md`](docs/SPEC.md).

## المتطلبات

| | |
|---|---|
| نظام التشغيل | Ubuntu 22.04 / 24.04 / 26.04، وDebian 12 / 13، وAlmaLinux (Rocky، RHEL) 9 / 10 |
| المعماريات | `amd64` و`arm64` |
| لاستخدام `--type wg` | `wireguard-tools` (`wg` و`wg-quick`) ووحدة WireGuard في النواة |
| لاستخدام `--type awg*` | `amneziawg-tools` (`awg` و`awg-quick`) ووحدة AmneziaWG في النواة **أو** `amneziawg-go` |
| مطلوب دائمًا | `iproute2` (`ip`) و`systemd` |
| الصلاحيات | root (باستثناء `--help` و`--version` و`--doctor` و`--udp-checker`؛ ولا يحتاج `--udp-listener` إلى root إلا للمنافذ `<1024` أو عند تعذر قراءة `/proc/net/udp{,6}`) |

لا تثبت krot أيًا من هذه المكونات بنفسها. وإذا كان أحد البرامج المطلوبة مفقودًا،
تنتهي برمز الخروج `6` وتذكر اسم الحزمة التي ينبغي تثبيتها.

### إصدارات AmneziaWG

قبل تنفيذ `--create`، تتحقق krot من أن AmneziaWG المثبتة تدعم النوع المطلوب:

| النوع | amneziawg-tools | وحدة النواة | عائلة netlink | amneziawg-go |
|---|---|---|---|---|
| `awg` (1.0) | 1.0.20240213 | — | ≥ 1 | 0.1.0 |
| `awg` مع `--i1` (1.5) | 1.0.20250901 | 1.0.20251009 | ≥ 2 | 0.2.13 |
| `awg2` (2.0) | 1.0.20250901 | 1.0.20251009 | ≥ 2 | 0.2.15 |
| `awg3` (3.1) | 3.1.20260812 | 3.1.20260812 | ≥ 3 | 3.1.20260812 |

تُبلغ بعض الإصدارات عن رقم إصدار مؤقت؛ فمثلًا يطبع `awg --version` في أدوات 1.x
`1.0.20210914` في معظم الحالات، بينما تُبلغ الوحدة المبنية باستخدام `make` عن `1.0.0`.
تؤدي هذه الإصدارات إلى تحذير لا إلى خطأ؛ ولا يُعاد رمز الخروج `6` إلا عندما يمكن
التحقق بموثوقية من الإصدار ويكون قديمًا أكثر من اللازم. وإذا كان الإصدار قديمًا فعلًا
وتعذر بدء وحدة systemd، فستتراجع krot عن التغييرات.

## التثبيت

ثبّت الحزم أو الملفات التنفيذية من صفحة الإصدارات
<https://github.com/snmp161/krot/releases>، وتحقق دائمًا من المجموع الاختباري.

```sh
VERSION=0.7.4
BASE=https://github.com/snmp161/krot/releases/download/v$VERSION

# تنزيل الحزمة وقائمة المجاميع الاختبارية
wget "$BASE/krot_${VERSION}_amd64.deb" "$BASE/SHA256SUMS"
# التحقق: يجب أن تظهر "krot_..._amd64.deb: OK"
gh attestation verify SHA256SUMS --repo snmp161/krot
sha256sum --check --ignore-missing SHA256SUMS
# التثبيت
sudo dpkg -i "krot_${VERSION}_amd64.deb"
```

الأنظمة المعتمدة على RPM (AlmaLinux وRocky وRHEL):

```sh
wget "$BASE/krot-${VERSION}-1.x86_64.rpm" "$BASE/SHA256SUMS"
gh attestation verify SHA256SUMS --repo snmp161/krot
sha256sum --check --ignore-missing SHA256SUMS
sudo rpm -i "krot-${VERSION}-1.x86_64.rpm"
```

للتثبيت من ملف تنفيذي بلا حزمة: نزّل `krot-linux-amd64` (أو `krot-linux-arm64`)
و`SHA256SUMS`، وتحقق بالطريقة نفسها، ثم نفّذ
`sudo install -m 0755 krot-linux-amd64 /usr/bin/krot`.

```sh
wget "$BASE/LICENSE" "$BASE/THIRD_PARTY_NOTICES"
```

على معمارية `arm64`، استخدم الملفات `*_arm64.deb` و`*.aarch64.rpm` و`krot-linux-arm64`.

## الاستخدام

```
krot <الإجراء> [الخيارات]
```

يجب تحديد إجراء واحد فقط في كل تشغيل. تستخدم الخيارات الصيغة الطويلة فقط:
`--flag value` أو `--flag=value`. ويؤدي استخدام خيار لا ينطبق على الإجراء إلى خطأ
برمز الخروج `2`. لعرض التعليمات الكاملة: `krot --help`.

### الأنفاق

```sh
# إنشاء AmneziaWG 2.0 (ثنائي المكدس)؛ يُختار الاسم تلقائيًا: awg0 ثم awg1 وهكذا
sudo krot --create --type awg2 --ip 10.8.0.1/24,fd00:8::1/64 --port 51820 \
          --endpoint vpn.example.com --dns 1.1.1.1,9.9.9.9

# WireGuard باسم وMTU محددين
sudo krot --create --type wg --interface wg0 --ip 10.9.0.1/24 --port 51821 \
          --endpoint 203.0.113.10 --mtu 1420

# AmneziaWG 1.5 مع حزمة توقيع I1 (تتحقق krot من صياغة CPS وطولها — بحد أقصى 2048 حرفًا)
sudo krot --create --type awg --ip 10.10.0.1/24 --port 51822 --endpoint vpn.example.com \
          --i1 '<b 0xf6ab3267fa><r 16>'

# AmneziaWG 3.1 (تكون Header Protection وRandomTrailers وDisableCookies مفعلة دائمًا)
sudo krot --create --type awg3 --ip 10.11.0.1/24 --port 51823 --endpoint vpn.example.com

sudo krot --list                         # جميع الأنفاق
sudo krot --list --type awg2 --json      # مرشحات --type/--interface/--ip/--port، وتُدمج بمنطق AND
sudo krot --down --interface awg0        # systemctl stop awg-quick@awg0
sudo krot --up --interface awg0          # systemctl start؛ تكرار --up يطبع "already up" برمز 0
sudo krot --delete --interface awg0      # يطلب تأكيدًا إذا كانت هناك عملاء
sudo krot --delete --interface awg0 --yes
```

### التشخيص

```sh
krot --doctor   # يعمل بلا root: الأدوات ووحدات النواة وإصدارات AmneziaWG والمجلدات
```

تكون المخرجات جدول `CHECK/STATUS/DETAIL` بالحالات `ok` / `warn` / `fail`.
يكون رمز الخروج `0` إذا لم توجد أي عناصر `fail`، وإلا يكون `6`. لا يغير هذا الإجراء
أي شيء ولا يحصل على قفل.

#### تشخيص منافذ UDP

يمكن لـ`krot` فحص منافذ UDP التي تمر فعليًا بين مضيفين. وهذا مفيد عندما يقوم
مزود الاستضافة أو مزود الوصلة الصاعدة بتصفية بعض المنافذ، مثلًا بسبب قواعد مكافحة DDoS.

شغّل المستمع (listener) على الخادم:

```sh
# منفذ واحد
sudo krot --udp-listener --port 51820

# نطاق منافذ (حتى 1024 منفذ، مع socket واحد لكل منفذ)
sudo krot --udp-listener --port 51820:51830
```

يستجيب المستمع لثلاثة أنواع من مخططات البيانات:
- `ping SEQ:N` → `pong SEQ:N`؛
- `ECHO <text>` → `<text>`؛
- `TIME` → وقت Unix الحالي بالمللي ثانية.

شغّل الفاحص (checker) على العميل:

```sh
# فحص منفذ واحد
krot --udp-checker --address 203.0.113.1 --port 51820

# فحص نطاق من المنافذ
krot --udp-checker --address 203.0.113.1 --port 51820:51830

# زيادة حجم حمولة echo (1–1400 بايت، والافتراضي 64) — مفيد لتحديد أفضل MTU
krot --udp-checker --address 203.0.113.1 --port 51820 --echo-size 1400

# تحديد عدد الـ workers وإظهار التقدم
krot --udp-checker --address 203.0.113.1 --port 51820:51830 --workers 64 --verbose
```

يرسل الفاحص افتراضيًا 3 حزم من كل نوع إلى كل منفذ (`--count`، من 1 إلى 100)،
وينتظر الرد خلال `--timeout` (الافتراضي 2s). ويفحص 32 منفذًا بالتوازي افتراضيًا؛
ويمكن ضبط عدد الـ workers من 1 إلى 100 باستخدام `--workers`. يكتب الخيار `--verbose`
النطاق ونتيجة كل منفذ إلى stderr فور اكتمال فحصه؛ وفي المستمع يسجل عناوين العملاء
وأنواع الطلبات. مثال على المخرجات:

```
PORT     STATUS   PING    ECHO    TIME (min/avg/max)   DETAIL
51820    ok       3/3     3/3     12/24/45 ms          -
51821    filtered 0/3     0/3     -                    timeout
```

الحالات:
- `ok` — وصلت ردود صحيحة لجميع حزم ping وecho؛
- `filtered` — انتهت مهلة حزمة واحدة على الأقل (ومن المرجح أن المنفذ يخضع للتصفية)؛
- `error` — وصل رد غير صالح.

مع `--json`، تكون المخرجات مصفوفة كائنات بالحقول `port` و`status` و`ping_sent`
و`ping_ok` و`echo_sent` و`echo_ok` و`time_sent` و`time_ok` و`rtt_min_ms`
و`rtt_avg_ms` و`rtt_max_ms` و`detail`.

عند الضغط على `Ctrl+C` أو تلقي `SIGTERM`، يطبع الفاحص نتائج المنافذ التي اكتمل
فحصها بالصيغة المختارة وينتهي برمز الخروج `10`؛ ولا تظهر المنافذ غير المكتملة في النتائج.

يعمل `--udp-checker` بلا root. ولا يحتاج `--udp-listener` إلى root إلا إذا تضمن النطاق
منافذ `<1024` أو تعذر على المستخدم الحالي قراءة `/proc/net/udp` و`/proc/net/udp6`.
ويستمر المستمع في العمل حتى `Ctrl+C`/`SIGTERM`.

مثال على `--list`:

```
INTERFACE  TYPE  VERSION  ADDRESS                   PORT   STATUS  CLIENTS
awg0       awg2  2.0      10.8.0.1/24,fd00:8::1/64  51820  up      12 (1 locked)
wg0        wg    -        10.9.0.1/24               51821  down    3
```

قيم MTU الافتراضية: `wg` — 1420، و`awg`/`awg2` — 1376، و`awg3` — 1280.
تُولّد معاملات التمويه (Jc وS1–S4 وH1–H4 وغيرها) تلقائيًا ضمن نطاقات آمنة
وتُتحقق منها (القسم 7 من المواصفات).

### العملاء

```sh
# إضافة عميل: تُخصص العناوين تلقائيًا (أول /32 و/128 متاحين)؛
# تُطبع إعدادات العميل مرة واحدة فقط — ولا يُحفظ مفتاحه الخاص إطلاقًا
sudo krot --add-client --interface awg0 --id phone-sergei --allowed-ips 0.0.0.0/0,::/0

# مع PresharedKey ورمز QR في الطرفية
sudo krot --add-client --interface awg0 --id laptop --allowed-ips 0.0.0.0/0 --psk --qr

# طباعة مفتاح AmneziaVPN إضافي: سطر vpn://… بعد الإعدادات
sudo krot --add-client --interface awg0 --id tablet --allowed-ips 0.0.0.0/0,::/0 --amnezia-key

# أوامر KeeneticOS CLI: توصيل موجه Keenetic كعميل (wg وawg 1.0/1.5/2.0، وIPv4 فقط)
sudo krot --add-client --interface wg0 --id keenetic --allowed-ips 0.0.0.0/0 --keenetic

# تحديد عنوان معين للعميل
sudo krot --add-client --interface awg0 --id router --allowed-ips 10.8.0.0/24 --ip 10.8.0.50/32

# إعدادات خارجية بلا بيانات krot الوصفية: مرر endpoint صراحةً (لا يُكتب في إعدادات الخادم)
sudo krot --add-client --interface office --id guest --allowed-ips 0.0.0.0/0 --endpoint 203.0.113.10

# DNS مخصص لعميل واحد (لا يُكتب في إعدادات الخادم)
sudo krot --add-client --interface awg0 --id kid-phone --allowed-ips 0.0.0.0/0 --dns 94.140.14.15

sudo krot --list-client --interface awg0
sudo krot --list-client --interface awg0 --id phone-sergei --json
sudo krot --lock-client   --interface awg0 --id laptop    # عند التكرار — "already locked" برمز 0
sudo krot --unlock-client --interface awg0 --id laptop
sudo krot --delete-client --interface awg0 --id laptop
```

يضبط `--allowed-ips` قيمة `AllowedIPs` في إعدادات **العميل**، أي ما يوجهه العميل
عبر النفق. أما DNS العميل، فيُستخدم `--dns` إذا حُدد؛ وإلا تُستخدم البيانات الوصفية
`# krot:dns` للواجهة (`--dns` عند `--create`)؛ وإلا تُستخدم أسطر `DNS` من قسم
`[Interface]` في إعدادات الخادم. وإذا لم يُضبط DNS في أي موضع، فلن تحتوي إعدادات
العميل على سطر `DNS`.

تُخصص عناوين العملاء من شبكة الواجهة الفرعية، مع استبعاد عنوان الشبكة وعنوان
البث وعنوان الخادم. وتدعم krot أيضًا الشبكات الفرعية من نقطة إلى نقطة `/31` و`/127`
(RFC 3021 وRFC 6164): يمكن استخدام كلا العنوانين، لذلك يمكن إنشاء نفق
«خادم + عميل واحد» باستخدام `--create … --ip 10.8.0.0/31,fd00:8::/127`.

احفظ الإعدادات المولدة فورًا؛ فلا يمكن استرجاعها لاحقًا، ولا يمكنك إلا حذف العميل
وإضافته من جديد. تُطبع رموز QR باستخدام كتل Unicode النصفية وتفترض خلفية طرفية داكنة.

يطبع `--amnezia-key` مفتاح AmneziaVPN بعد الإعدادات في سطر واحد `vpn://…`
يمكن لصقه في تطبيق AmneziaVPN عبر خيار «Insert key»، بعد إرساله مثلًا عبر أحد
تطبيقات المراسلة. يحتوي المفتاح على إعدادات العميل نفسها: `vpn://` + ترميز base64url
من دون `=` لناتج دالة `qCompress` في Qt (طول من 4 بايتات بترتيب big-endian + zlib).
يعمل المفتاح مع جميع الأنواع بما فيها `wg`. ويحتوي رمز QR الناتج عن `--qr` على نص
الإعدادات؛ فيمكن لتطبيقي AmneziaVPN وAmneziaWG مسحه، وكذلك WireGuard في حالة `wg`.

يطبع `--keenetic` بعد الإعدادات مجموعة أوامر KeeneticOS CLI لتوصيل موجه Keenetic
كعميل للنفق. تتبع المجموعة صياغة running-config في KeeneticOS، ويمكن لصقها كما هي
في CLI الخاص بالموجه (SSH/telnet أو «Command-line interface» في واجهة الويب).
تدعم KeeneticOS تمويه AmneziaWG (معاملات ASC): يدعم الإصدار KeeneticOS 4.2 وما بعده
AWG 1.0، ويدعم الإصدار 5.1 وما بعده AWG 1.5/2.0. وتتضمن مجموعة أوامر أنفاق awg سطرًا
`wireguard asc …` (تُوضع سلاسل CPS من I1 إلى I5 بين علامتي اقتباس؛ ويُعد استيراد
ملف الإعدادات عبر واجهة الويب أكثر موثوقية). لا يُدعم AWG 3.1. ويدعم هذا الخيار IPv4 فقط؛
وإذا اختلف MTU للنفق عن القيمة الافتراضية 1420، فتُطبع ملاحظة. راجع القسم 6.13
من المواصفات للتعرف إلى الصيغة.

مثال على `--list-client`:

```
ID            PUBLIC KEY  ADDRESS                    STATUS  LAST HANDSHAKE  RX       TX
phone-sergei  aB3d…Xy9=   10.8.0.2/32,fd00:8::2/128  active  1m ago          12.4MiB  1.1GiB
laptop-old    Qw7e…Zz1=   10.8.0.3/32                locked  -               -        -
```

### التمويه (حزم I1–I5)

بدءًا من الإصدار 1.5، يمكن لـ AmneziaWG إرسال ما يصل إلى خمس حزم توقيع `I1`–`I5`
قبل كل مصافحة؛ وهي حزم تبدو كأنها بداية بروتوكول آخر (DNS أو STUN أو NTP أو SSDP أو DTLS).
تولّد krot هذه السلاسل بنفسها باستخدام الخيار `--mimic <profile>`:

```sh
# ملف التعريف الافتراضي لجميع عملاء الواجهة
krot --create --type awg --ip 10.8.0.1/24 --port 51820 --endpoint vpn.example.com --mimic dns
# تجاوز ملف التعريف لعميل واحد
krot --add-client --interface awg0 --id phone --allowed-ips 0.0.0.0/0 --mimic stun
```

ملفات التعريف المتاحة: `dns` (ثلاثة استعلامات A/AAAA/HTTPS)، و`stun` (Binding Request)،
و`ntp`، و`ssdp` (M-SEARCH)، و`dtls` (ClientHello). تُولّد السلسلة عند `--add-client`
(ويحصل كل عميل على سلسلة خاصة به) ولا تُكتب إلا في إعدادات العميل؛ لأن حزم I يرسلها
طرف بدء المصافحة، ولا يحتاج الخادم إليها. يتطلب العميل AmneziaWG 1.5 أو أحدث؛
راجع القسم 6.12 من المواصفات.

إلى جانب ملفات التعريف المدمجة، تقبل krot سلسلة CPS جاهزة عبر `--i1` عند `--create`،
ومنها مثلًا سلاسل QUIC. لا تولّد krot ملفات تعريف QUIC مدمجة، لأن حزمة QUIC Initial الواحدة
تبلغ 2400–2500 حرف وتتجاوز حد 2048 حرفًا بسبب
[amneziawg-tools#69](https://github.com/amnezia-vpn/amneziawg-tools/issues/69).
يمكن الحصول على السلاسل الجاهزة من مولدات خارجية (كلاهما مرخصان بموجب MIT ولا يتبعان krot):

- [payloadGen](https://github.com/Sketchystan1/payloadGen) — صفحة ويب: افتح
  `index.html` في المتصفح واختر البروتوكول وخذ السطر الأول (I1)؛
- `cpsgen.py` من [awg-multi-script](https://github.com/pumbaX/awg-multi-script)
  (`src/py/cpsgen.py`) — نسخة طرفية من payloadGen مكتوبة بـ Python 3 ولا تحتاج root.
  ملفات التعريف المتاحة: `dns` و`ntp` و`rtp` و`stun` و`sip` و`ssdp` و`webrtc` و`quic`
  و`curl_quic`؛ والمعامل الثاني هو اسم النطاق.

```sh
# نزّل المولد وراجع شيفرته قبل تشغيله
git clone --depth 1 https://github.com/pumbaX/awg-multi-script
less awg-multi-script/src/py/cpsgen.py

# ولّد I1 فقط وتحقق من طوله (يجب ألا يتجاوز 2048 حرفًا)
I1=$(python3 awg-multi-script/src/py/cpsgen.py dns example.com --only-i1)
echo ${#I1}

sudo krot --create --type awg2 --ip 10.8.0.1/24 --port 51820 --endpoint vpn.example.com --i1 "$I1"
```

ملاحظات:

- لا يمكن ضبط `I1` إلا عند `--create`؛ ويُكتب في إعدادات الخادم ويُنسخ إلى جميع
  إعدادات العملاء. لا تقبل krot القيم `I2`–`I5`.
- يجب ألا يتجاوز الطول 2048 حرفًا (راجع [القيود](#القيود))، وإلا يكون رمز الخروج `3`.
  تبلغ حزم ملفي `quic` و`curl_quic` نحو 2400 حرف ولن تتسع. أما ملفات التعريف الأقصر فتعمل:
  `rtp` (نحو 30 حرفًا)، و`dns` و`ntp` (نحو 100)، و`stun` (نحو 250–350)،
  و`ssdp` (نحو 400)، و`sip` (نحو 1200).
- يعني استخدام `--type awg` مع `--i1` استخدام AmneziaWG 1.5 بمتطلبات الحد الأدنى
  الخاصة بها (راجع [إصدارات AmneziaWG](#إصدارات-amneziawg)). تدعم amneziawg-go
  الوسمين `<rc N>`/`<rd N>` (اللذين ينتج عنهما ملف `sip`) بدءًا من الإصدار 0.2.15؛
  لذلك يُفضل مع 1.5 استخدام ملفات تعريف لا تتضمنهما.
- يجب أن يدعم العميل AmneziaWG 1.5 أو أحدث.
- عندما يكون `I1` طويلًا، قد لا تتسع الإعدادات في رمز QR؛ وعندها ينتهي `--qr` برمز
  الخروج `3` (بعد أن يكون العميل قد أُضيف والإعدادات قد طُبعت). انقل الإعدادات في ملف
  أو استخدم `--amnezia-key`.

### JSON

يخرج `--list --json` مصفوفة كائنات بالحقول `interface` و`type` و`version`
و`addresses[]` و`port` و`status` و`clients_total` و`clients_locked`.
ويخرج `--list-client --json` مصفوفة كائنات بالحقول `id` و`public_key` و`addresses[]`
و`status` و`latest_handshake` (وقت Unix أو `null`) و`rx_bytes` و`tx_bytes`
و`endpoint` (أو `null`) و`created` (أو `null` للأقران التي لا تديرها krot).
تكون النتيجة الفارغة `[]`.

## رموز الخروج

| الرمز | المعنى |
|---|---|
| 0 | نجاح |
| 1 | خطأ داخلي أو غير متوقع (بما في ذلك تعذر تطبيق تغيير على واجهة قيد التشغيل — تُسترجع الإعدادات السابقة) |
| 2 | خطأ في استخدام CLI (الخيارات؛ أو الحذف مع وجود عملاء دون `--yes` عندما لا يكون stdin طرفية) |
| 3 | خطأ في التحقق من قيمة (بما في ذلك غياب endpoint أو «config too large for QR») |
| 4 | لم يُعثر على الكائن (الواجهة أو العميل) |
| 5 | تعارض: الكائن موجود بالفعل، أو العنوان أو المنفذ مستخدم، أو لا توجد عناوين متاحة |
| 6 | البيئة: لا صلاحية root، أو أدوات مفقودة، أو إصدار AmneziaWG غير مدعوم |
| 7 | تعذر الحصول على قفل `/run/lock/krot.lock` خلال 10 ثوانٍ |
| 10 | ألغى المستخدم العملية |

تُكتب المخرجات العادية إلى stdout؛ وتُكتب الأخطاء والتحذيرات إلى stderr مع البادئة `krot:`.

## تخزين البيانات

| النوع | ملف الإعدادات | وحدة systemd | الأداة |
|---|---|---|---|
| `wg` | `/etc/wireguard/<if>.conf` | `wg-quick@<if>` | `wg` |
| `awg` و`awg2` و`awg3` | `/etc/amnezia/amneziawg/<if>.conf` | `awg-quick@<if>` | `awg` |

تُحفظ البيانات الوصفية كتعليقات يتجاهلها `wg-quick`/`awg-quick`:

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

- العميل المقفل هو كتلة `[Peer]` مُعلّقة بالبادئة `#~ `.
- يمكن إلغاء تعليق عناصر `PreUp`/`PostUp`/`PreDown`/`PostDown` وتعبئتها
  (مثلًا بقواعد NAT)، ولن تعدلها krot.
- تُدعم ملفات الإعدادات التي لا تحتوي بيانات krot الوصفية (أي المنشأة يدويًا):
  يعرضها `--list`، ويطبع `--list-client` الأقران الخاصة بها بمعرّف `-` (ولا تنطبق
  عمليات `--id` عليها)، ويحافظ أي إجراء كتابة على جميع الأسطر والتعليقات والترتيب
  الأصلي حرفيًا وبايتًا ببايت.
- عمليات الكتابة ذرّية (ملف مؤقت → `fsync` → `rename` → `fsync` للمجلد)،
  والأذونات `0600`.
- تحصل جميع الإجراءات المعدّلة على قفل `flock` حصري على `/run/lock/krot.lock`،
  فلا تضيع التغييرات عند التشغيل المتزامن (مثل Salt وتدخل بشري).
- لا تنشئ krot نسخًا احتياطية.

## القيود

- تعد NAT وإعادة التوجيه والجدار الناري وsysctl خارج نطاق krot؛ استخدم عناصر
  `PostUp`/`PostDown`.
- لا يُدعم تثبيت WireGuard/AmneziaWG أو ترحيل الأنفاق بين الأنواع. يستخدم توليد
  توقيعات التمويه ملفات مدمجة (`--mimic`)؛ أما ملفات QUIC والتوقيعات المخصصة
  فتمر عبر `--i1` والمولدات الخارجية، راجع [التمويه](#التمويه-حزم-i1i5).
- يجب ألا يتجاوز `--i1` عدد 2048 حرفًا؛ فبسبب
  [amneziawg-tools#69](https://github.com/amnezia-vpn/amneziawg-tools/issues/69)،
  تتسبب القيم الطويلة `I1`–`I5` (نحو 3600 حرف فأكثر) في تعليق `awg show`
  وفشل `awg set`/`setconf`.
- لا تُحفظ المفاتيح الخاصة للعملاء، ولا يمكن إعادة إصدار إعدادات العملاء.

## التطوير

تحتاج إلى إصدار Go المحدد في `go.mod` (حاليًا 1.27.4) وإلى Docker لاختبارات التكامل.

```sh
make check        # جميع الفحوص المحلية: التنسيق والاختبارات والوحدات والأمان ومتحققات workflow
make fuzz-smoke   # fuzzing محدود الوقت لجميع targets
make build        # ملف bin/krot ثابت (CGO_ENABLED=0)، والإصدار من git describe
make build-arm64
make integration  # اختبارات التكامل داخل حاوية Docker ذات صلاحيات مميزة وتعمل بـ systemd
make changelog      # إعادة توليد changelog من ملاحظات الإصدار
make repro-check    # التحقق من تطابق البناء بايتًا ببايت
make package-preflight # حزم DEB/RPM لكلتا البنيتين + smoke على Ubuntu 24.04/AlmaLinux 9
```

قواعد المستودع: [`AGENTS.md`](AGENTS.md) — المواصفات أولًا، والاختبارات قبل الشيفرة،
و`commit` واحد لكل مهمة.

- **اختبارات الوحدة** لا تحتاج root ولا تعدل النظام: تُستبدل المجلدات بمجلدات مؤقتة،
  وتُنفذ الأوامر الخارجية (`wg` و`awg` و`systemctl` و`ip` وغيرها) عبر منفذ وهمي
  يسجل الاستدعاءات. وتتحقق اختبارات منفصلة من أن عمليات العملاء لا تستدعي سوى
  `wg|awg show` و`wg|awg set <if> peer …`.
- توجد **الملفات المرجعية (golden files)** في `testdata/` داخل كل حزمة. لا تُحدّث
  المخرجات المولدة (`internal/app` و`internal/qr`) إلا عن قصد باستخدام
  `go test ./internal/app ./internal/qr -update` مع شرح السبب في الـ commit؛
  أما بقية الملفات المرجعية فتُكتب يدويًا وفق المواصفات.
- تحتاج **اختبارات التكامل** (`test/integration`، وسم البناء `integration`) إلى root
  وsystemd، ولا تعمل إلا داخل مساحات أسماء شبكية منفصلة: تُوضع وحدات واجهات الاختبار
  في netns عبر drop-in باسم `NetworkNamespacePath`. وتغطي دورة الحياة الكاملة لجميع
  الأنواع، وشفافية عمليات العملاء (ping متواصل بلا فقدان، وثبات ifindex والمصافحة)،
  وIPv6، والإعدادات الخارجية، والتشغيل المتزامن. تعمل محليًا عبر
  `scripts/integration-docker.sh`، وفي CI باستخدام `sudo`.
- يعيد `scripts/collect-versions.sh` جمع مخرجات إصدارات AmneziaWG الحقيقية
  للاختبارات المرجعية.
- **CI** (`.github/workflows/ci.yml`): فحوص الجودة المحلية، واختبارات الوحدة مع
  shuffle و`-race` وتغطية لا تقل عن 90%، وfuzzing محدود الوقت، واختبارات التكامل،
  وpackage preflight، وبناء `amd64`/`arm64`. **الإصدار** (`release.yml`، الوسم `v*`): الملفات التنفيذية وحزم
  `.deb`/`.rpm` (nfpm و`packaging/nfpm.yaml`)، واختبارات smoke للحزم على جميع
  أنظمة التشغيل (`scripts/smoke.sh`)، وGitHub Release مع `SHA256SUMS`.

## الترخيص

[Apache License 2.0](LICENSE). Copyright 2026 Sergey Ovchinnikov.
