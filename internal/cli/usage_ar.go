package cli

const usageTextArabic = `krot — إدارة أنفاق WireGuard وAmneziaWG

الاستخدام:
  krot <الإجراء> [الخيارات]

الإجراءات (إجراء واحد بالضبط):
  --list            سرد الأنفاق
  --create          إنشاء نفق
  --delete          حذف نفق
  --up              بدء واجهة
  --down            إيقاف واجهة
  --add-client      إضافة عميل
  --delete-client   حذف عميل
  --lock-client     قفل عميل
  --unlock-client   إلغاء قفل عميل
  --list-client     سرد العملاء
  --doctor          فحص البيئة (الأدوات، وحدات النواة، إصدارات AmneziaWG)
  --udp-listener    مستمع UDP لتشخيص المنافذ
  --udp-checker     فاحص UDP لتشخيص المنافذ
  --help            عرض هذه المساعدة
  --version         عرض الإصدار والالتزام وتاريخ البناء

الخيارات (--flag value أو --flag=value):
  --type wg|awg|awg2|awg3   نوع البروتوكول (create: مطلوب؛ list: مرشح)
  --interface NAME          اسم الواجهة، حتى 15 حرفًا [a-zA-Z0-9_.=+-]
  --ip CIDR[,CIDR]          create: عناوين الخادم (10.8.0.1/24,fd00:8::1/64)؛
                            add-client: عناوين العميل (/32، /128)؛
                            list: التصفية حسب العنوان
  --port N|N:M              منفذ UDP أو نطاق 1..65535، بحد أقصى 1024 منفذًا
                            (create وudp-listener وudp-checker: مطلوب؛ list: مرشح)
  --endpoint HOST           عنوان الخادم العام لإعدادات العميل
                            (create: مطلوب؛ add-client: تجاوز لعميل محدد)
  --address HOST            عنوان المستمع البعيد لـ --udp-checker (دون منفذ)
  --mtu N                   قيمة MTU من 576 إلى 9000 (الافتراضي: wg 1420، awg/awg2 1376، awg3 1280)
  --dns IP[,IP]             خوادم DNS لإعدادات العميل
                            (create: لجميع العملاء؛ add-client: لهذا العميل فقط)
  --i1 CPS                  حزمة توقيع I1 لـ AmneziaWG (لـ awg* فقط، حتى 2048 حرفًا)
  --mimic PROFILE           توليد سلسلة محاكاة I1-I5 لإعدادات العميل:
                            dns|stun|ntp|ssdp|dtls (لـ awg* فقط؛ لا تُدمج مع --i1)
  --id ID                   معرف العميل [a-zA-Z0-9_.-]{1,64}، فريد داخل الواجهة
  --allowed-ips CIDR[,CIDR] قيمة AllowedIPs في إعداد العميل، مثل 0.0.0.0/0,::/0
  --psk                     توليد PresharedKey للعميل
  --qr                      أيضًا طباعة إعداد العميل كرمز QR
  --amnezia-key             أيضًا طباعة إعداد العميل كمفتاح AmneziaVPN (vpn://...)
  --keenetic                أيضًا طباعة أوامر KeeneticOS CLI لموجه Keenetic
                            (أنفاق wg وawg 1.0/1.5/2.0، IPv4 فقط)
  --json                    إخراج JSON بدلًا من الجدول (list وlist-client وudp-listener وudp-checker)
  --yes                     عدم طلب التأكيد (delete)
  --timeout DURATION        مهلة الحزمة، مثل 2s أو 500ms (الافتراضي: 2s)
  --count N                 حزم من كل نوع لكل منفذ، من 1 إلى 100 (الافتراضي: 3)
  --echo-size N             حجم حمولة echo، من 1 إلى 1400 (الافتراضي: 64، udp-checker فقط)
  --workers N               عمليات فحص المنافذ المتزامنة، من 1 إلى 100 (الافتراضي: 32، udp-checker فقط)
  --verbose                 تقدم التشخيص (udp-listener وudp-checker؛ stderr)

الخيارات حسب الإجراء:
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
  --doctor         (دون خيارات؛ يعمل دون root)
  --udp-listener   --port [--json] [--verbose]
  --udp-checker    --address --port [--timeout] [--count] [--echo-size] [--workers] [--json] [--verbose]

رموز الخروج:
  0 نجاح                    5 تعارض (موجود بالفعل، العنوان أو المنفذ مستخدم)
  1 خطأ داخلي               6 البيئة (ليس root، أدوات مفقودة، إصدار AmneziaWG)
  2 خطأ في الاستخدام        7 تعذر الحصول على القفل
  3 قيمة غير صالحة          10 ألغاه المستخدم
  4 غير موجود

صلاحيات root:
  لا تُطلب لـ --help أو --version أو --doctor أو --udp-checker.
  يتطلب --udp-listener صلاحية root فقط للمنافذ الأقل من 1024 أو عندما
  لا يمكن قراءة /proc/net/udp و/proc/net/udp6.
  تتطلب جميع الإجراءات الأخرى root.
`
