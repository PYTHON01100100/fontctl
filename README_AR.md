# fontctl — مدير الخطوط في لينكس

> أداة CLI وTUI مكتوبة بلغة Go باستخدام Bubble Tea لإدارة الخطوط، وتجربتها بالعربية والإنجليزية، وضبط الخطوط حسب اللغة والطرفية وسطح المكتب.

**الحالة: إصدار ألفا v0.1 (شفرة مصدرية).** يتضمن محركًا حقيقيًا لاكتشاف الخطوط، وواجهة طرفية، وإعدادات Fontconfig، ودعم Kitty وAlacritty وGNOME، واستيراد الخطوط من ملفات محلية أو روابط HTTPS مباشرة. لم يتم بناء واجهة Bubble Tea كاملة في بيئة التجهيز لعدم توفر اتصال لتنزيل الاعتماديات؛ راجع تعليمات البناء.

[README in English](README.md) · [AGENDA](AGENDA.md) · [Architecture](docs/ARCHITECTURE.md)

## لماذا fontctl؟

بدلًا من تعديل إعدادات كل طرفية وكل برنامج يدويًا، يمكنك استعراض الخطوط المثبتة، وتحديد تفضيلات العربية أو اليابانية أو الإنجليزية، وتطبيقها مع إمكانية التراجع عن آخر تغيير.

لا نعتمد على مدير حزم توزيعة معيّنة؛ يستخدم البرنامج **Fontconfig** ومجلدات XDG القياسية. لذلك يعمل المحرك الأساسي على Fedora وUbuntu وArch وغيرها **عندما تتوفر أدوات Fontconfig**، لكن التكامل مع سطح المكتب يعتمد على البيئة المستخدمة (GNOME حاليًا، وKDE لاحقًا).

## التثبيت والبناء

تحتاج إلى Go 1.25 أو أحدث، وحزمة `fontconfig`، وإنترنت عند تنزيل اعتماديات Go أول مرة.

```bash
# مثال Fedora
sudo dnf install fontconfig
# مثال Ubuntu / Debian
sudo apt install fontconfig
# مثال Arch
sudo pacman -S fontconfig

# من داخل مجلد المشروع بعد فك ZIP
go mod tidy
go test ./...
go build -o fontctl ./cmd/fontctl
./fontctl
```

لا يحتاج البرنامج نفسه إلى `sudo`. أوامر `sudo` أعلاه لتثبيت الحزمة الناقصة في النظام فقط.

## تشغيل الواجهة

```bash
./fontctl
```

المفاتيح المهمة: الأسهم لاختيار الخط، `Tab` لتغيير الهدف، `l` للغة، `/` للبحث، `Enter` لحفظ اختيار الخط (دون تطبيق)، `p` لتبديل جملة المعاينة العربية، `i` لإضافة ملف خط من جهازك، `u` لوضع رابط HTTPS مباشر، `d` للمعاينة دون تطبيق، `a` للتطبيق مع التأكيد والنسخ الاحتياطية، `r` للاسترجاع، `q` للخروج.

## إضافة الخطوط: طريقتان

### 1) تنزيل الخط يدويًا من الإنترنت ثم استيراده

يمكن تنزيل ZIP أو TTF من موقع رسمي أو GitHub Releases، ثم:

```bash
./fontctl import ~/Downloads/CascadiaCode.zip
./fontctl import ~/Downloads/Example-Regular.ttf
```

هذه الطريقة مفيدة عند تنزيل الملف من المتصفح كما في الطريقة التي استخدمتها سابقًا.

### 2) تنزيله مباشرة من داخل fontctl

انسخ **رابط الملف المباشر** وليس صفحة GitHub العامة:

```bash
./fontctl install --url 'https://example.org/fonts/Example.zip'
```

يُسمح بروابط HTTPS فقط، ويُتحقق مبدئيًا من ترويسات الخطوط وأحجام الملفات. الخطوط المثبتة تحفظ في مجلد المستخدم مثل `~/.local/share/fonts/fontctl` ويُحدّث Fontconfig إن كانت أداة `fc-cache` متاحة.

**كتالوج للبحث عن خطوط الإنترنت ومعاينة الخط الحقيقي قبل تنزيله** ضمن خطة التطوير المستقبلية في [AGENDA.md](AGENDA.md)، وليس ميزة جاهزة في v0.1.

## تخصيص الخطوط

```bash
./fontctl fonts --lang ar
./fontctl set lang ar --family 'Noto Sans Arabic' --kind sans
./fontctl set terminal kitty --font 'Cascadia Code NF' --size 12
./fontctl set terminal alacritty --font 'Cascadia Code NF' --size 12
./fontctl set desktop gnome --interface 'Noto Sans'

./fontctl plan
./fontctl apply --dry-run
./fontctl apply
./fontctl restore latest
```

لا يُطبق اختيار `set` فورًا؛ يجب مراجعة الخطة ثم استخدام `apply`.

## جمل المعاينة

بالإنجليزية تظهر الجمل الثلاث معًا:

```text
Hello World My friend
one cli to rule them all
I am the king of the world
```

بالعربية تتغير الجملة في كل مرة بالترتيب، ثم تعود للأولى:

```text
صِفْ خَلْقَ خَوْدٍ كَمِثْلِ الشَّمْسِ إِذْ بَزَغَتْ يَحْظَى ضَجِيعٌ بِهَا نَجْلَاءَ مِعْطَارِ
بَلاغَةُ الحَرْفِ تُغْنِي عَنِ الكَلامِ
العَرَبِيَّةُ: أَصَالَةُ المَاضِي وَأُفُقُ المُسْتَقْبَلِ
```

```bash
./fontctl preview --lang ar
./fontctl preview --lang ar
./fontctl preview --lang ar
```

**تنبيه تقني:** معاينة TUI حاليًا نصية، ويظهر الخط المستخدم فعلًا في الطرفية نفسها، ولا يبدّل شكل الخط قبل تثبيته. دعم العربية من حيث اتجاه الكتابة والتشكيل يختلف من طرفية لأخرى. وكذلك قواعد Fontconfig لا تتحكم في كل التطبيقات والطرفيات تلقائيًا.

## دعم التوزيعات والأسطح

الوظائف الأساسية مستقلة عن التوزيعة، بينما تدعم النسخة الأولية GNOME وKitty وAlacritty فقط. دعم KDE وGNOME Terminal وKonsole وWezTerm وFoot يأتي لاحقًا.

## الأمان

لا تُعدّل ملفات النظام أو تثبت الخطوط بصلاحيات Root. الاستيراد يقبل TTF وOTF وTTC وZIP فقط، ويستثني WOFF2 المخصص للويب. يجب التأكد من موثوقية الخط وترخيصه، فالفحص الحالي أولي وليس مضادًا للبرمجيات الضارة.

حقوق مصدر البرنامج وفق رخصة [MIT](LICENSE)، لكن لكل خط خارجي شروط ترخيص منفصلة.
