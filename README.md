# VOID — Synthetic Reality Generator

**A synthetic data, environment & behavior simulation platform for software testing, load testing, QA, security testing, performance benchmarking, distributed-systems testing, API testing, database testing and chaos testing.**

Engine in Go · Control-plane UI in TypeScript + React + Next.js · English / فارسی / 中文

---

## Table of Contents
- [English](#english)
- [فارسی](#فارسی)
- [中文](#中文)

---
<a id="english"></a>
## English

### 1. What VOID is

VOID lets you build a **Synthetic Universe**: a large, controllable, reproducible, simulated world made of Entities (Users, Customers, Employees, Devices, Servers, Services, Databases, Transactions, Sessions, Orders, Payments, Sensors, Vehicles, Network Nodes, and more), the relationships between them, the behaviors they follow over time, and the events, traffic, and failures that flow through that world. It's built for people who need realistic, large-scale, repeatable synthetic conditions to test *other* systems against — not for producing a handful of one-off test files.

The **Go engine** owns every heavy operation: entity generation, event simulation, behavior execution, scenario/timeline orchestration, network & chaos simulation, transaction processing, load generation, metrics, snapshotting and export — all built on goroutines, channels and worker pools so large simulations never block. The **TypeScript / React / Next.js interface** is purely for visualization, interaction and project control; it never does simulation work itself.

### 2. Feature highlights

- **Entity Designer** — typed fields (string, integer, float, boolean, datetime, UUID, enum, array, JSON, binary, custom), per-field generators (random, sequential, weighted random, name/email/phone/address, date/time, regex pattern, statistical distribution, dependent/derived, custom function), and relationships (1:1, 1:N, N:N, hierarchical).
- **Relationship-aware Synthetic Data Generator** — new entities are wired to *existing* related entities (an Order really does point at a real Customer) instead of generating orphaned foreign keys, with full seed-based reproducibility (uniform, normal, log-normal, Poisson, exponential, Pareto, weighted distributions).
- **Behavior Engine** — state-machine / behavior-graph definitions (Event, Condition, Probability, Action, Delay, State Change, API Call, DB Operation, Loop nodes) drive how entities act over time (login → browse → purchase → logout, retry-on-error, etc.).
- **Event Simulation Engine** — a backpressure-aware, worker-pool-based event bus capable of very high throughput without blocking the UI.
- **Scenario Builder / Timeline** — schedule spawns, behavior attachment, one-off events, load-generation windows, chaos faults, waits, snapshots and network changes on a single reproducible timeline, at real-time, accelerated, slowed, paused or fully deterministic simulated time.
- **Traffic & Load Generator** — configurable virtual users, request rate, ramp-up/down, live RPS / latency (P50/P95/P99) / error-rate metrics. **It only ever sends traffic to a target you explicitly mark as authorized** — it is a load-testing tool, not an attack tool, and has no spoofing/amplification features.
- **Network Topology Simulator & Chaos Engine** — purely synthetic nodes/links with configurable latency, jitter, packet loss and bandwidth; controlled fault injection (service failure, timeout, high latency, DB unavailable, queue backlog, resource pressure, packet loss, partial failure) that acts only on that synthetic topology, plus cascade-impact calculation across a service dependency graph.
- **Transaction & Business Rules Engine** — synthetic payments/orders/reservations with a small rule evaluator (`if balance < amount then reject`, `if stock == 0 then cancel`, ...).
- **Database Simulation Layer** — batched, parallel seeding into a built-in in-memory store, with a driver-agnostic interface so you can plug in a real `database/sql` driver (PostgreSQL, MySQL, SQLite) for your own environment.
- **Time Simulation Engine** — real-time, accelerated, slowed, paused or fully deterministic simulated clocks (e.g. compress 30 simulated days into a few real minutes).
- **Snapshot & Replay + Simulation Diff** — snapshot a Universe's state to disk, resume from it later, and compare two runs' metrics to see exactly what changed.
- **Metrics & Observability** — an in-memory counters/gauges collector with a Prometheus-compatible text endpoint, latency P50/P95/P99 tracking, and a live Dashboard (counters, line charts, bar charts) in the UI.
- **Scheduler with user-defined Business Hours** — enter your own opening/closing windows for every day of the week (any timezone, multiple windows per day, overnight windows supported); VOID computes live whether you're open right now and exactly how long remains until the next change, and can gate scheduled/recurring Simulation Runs to only fire inside those hours. Nothing about the schedule is hard-coded — you enter every window yourself, in the Scheduler panel or via the API/CLI.
- **AI Simulation Assistant / Scenario Copilot** — turns a plain-language description ("a SaaS with 1 million users, 15% concurrently active, traffic peaking at 18:00") into a first-draft Entity Schema + Scenario for you to review in a Preview before running; it only ever produces configuration, never executes anything itself, and ships with a dependency-free pattern-based interpreter (swap in a real LLM backend via the `ai.Backend` interface if you want deeper language understanding).
- **Plugin system** — Go interfaces for custom Field Generators, Entity Type templates, Behavior templates, Connectors and Exporters, registered in-process (no unsafe dynamic loading).
- **Security** — HMAC-signed token authentication, role-based access control (admin / engineer / viewer), per-subject rate limiting, and an append-only audit log, all implemented with the Go standard library only.
- **Export** — JSON, JSONL, CSV, YAML, XML, SQL insert statements, and a simple length-prefixed binary container, streamed rather than buffered so very large datasets export cleanly.
- **Template Library** — starter domains: E-Commerce, Banking, IoT, SaaS, Social Network, Logistics, Smart City, FinTech, Microservices, Gaming, Telemetry.
- **UI** — a Windows-11-flavoured Fluent shell (Universe Explorer, Simulation Canvas, Inspector, Timeline, Console — resizable and collapsible), five themes (Light, Dark, Windows 11 Default, Red, Blue), a Ctrl+K command palette, and full English / Persian (true right-to-left) / Chinese localisation.

### 3. Architecture

```
backend/                          Go engine — zero external dependencies
  cmd/api/                        REST + WebSocket API server entry point
  cmd/cli/                        Terminal control surface
  internal/
    entity/       generator/      Schema, Field, Relationship, Collection · data generation
    randomx/                      Seeded reproducible randomness + statistical distributions
    event/                        Event model + worker-pool Event Bus
    behavior/                     State-machine / behavior-graph runner
    scenario/                     Timeline / Action model
    simulation/                   The Universe + Engine that ties everything together
    network/       chaos/         Synthetic topology · controlled fault injection
    transaction/                  Business rules + ledger + transaction processor
    database/                     DB simulation / seeding layer (pluggable driver)
    loadgen/                      Authorized-only HTTP load generator
    metrics/       storage/       Counters/gauges/Prometheus · snapshot persistence
    export/                       JSON/JSONL/CSV/YAML/XML/SQL/binary exporters
    scheduler/                    Business hours + scheduled/recurring runs
    security/                     JWT-style auth, RBAC, rate limiting, audit log
    plugin/        ai/            Extension registry · AI Simulation Assistant
    wsutil/        api/           Minimal WebSocket server · REST+WS router & handlers
  configs/                        Example scenario / schema / behavior / business-hours files

frontend/                         TypeScript + React + Next.js control-plane UI
  app/                            Next.js App Router entry (layout, page, global styles)
  components/
    layout/                      AppShell, Universe Explorer, Canvas, Inspector, Timeline, Console
    entity/        behavior/     Entity Designer · Behavior Editor
    dashboard/     scheduler/    Live Dashboard (recharts) · Business Hours panel
    command/       theme/        Ctrl+K Command Palette · Theme Provider
  lib/
    api.ts          types.ts     REST client · shared TypeScript types mirroring the backend
    store.ts                     App-wide state (zustand)
    i18n/                        English / Persian (RTL) / Chinese dictionaries + provider
  styles/themes.css               5-theme design-token system
```

### 4. Prerequisites

| Tool | Minimum version | Used for |
|---|---|---|
| Go | 1.22+ | Backend engine, API server, CLI |
| Node.js | 18.18+ (20 LTS recommended) | Frontend UI |
| npm | 9+ | Frontend package management |

The Go backend has **zero external dependencies** — it only imports the Go standard library, so `go build` works with no network access at all.

### 5. Installation & running

> These steps assume you've already extracted the project archive to a folder on your machine — start from step 1 inside that folder.

**Backend (Go engine + API server + CLI):**

```bash
cd backend

# Build the API server and the CLI
go build -o void-api ./cmd/api
go build -o void-cli ./cmd/cli

# Run the test suite (determinism, event bus, scheduler, export, ...)
go test ./...

# Start the API server (defaults to :8080)
export VOID_JWT_SECRET="choose-a-long-random-secret"
export VOID_API_ADDR=":8080"
./void-api
```

**Frontend (Next.js UI):**

```bash
cd frontend

# Install dependencies
npm install

# Point the UI at your API server (defaults to http://localhost:8080)
cp .env.example .env.local
# edit .env.local if your API server isn't on localhost:8080

# Development server
npm run dev

# ...or a production build
npm run build
npm run start
```

Open the printed local URL, click **Connect** in the top bar (any username, pick a role), then create a Universe from the Universe Explorer panel.

### 6. CLI quick reference

```bash
# Run a full scenario from a config file and export the results
./void-cli run --config configs/ecommerce-scenario.yaml --wait 10 --out-dir ./out --format json

# Generate a standalone synthetic dataset from just a schema file
./void-cli generate --schema configs/schema-user.example.json --count 100000 --seed 42 --out users.csv --format csv

# Check business-hours status from a saved schedule file
./void-cli scheduler status --hours configs/business-hours.example.json
```

Scenario/schema config files may be JSON or YAML (`configs/` has one example of each pattern) and are versioned (`version: 1`) so the format can evolve without breaking old files.

### 7. Security notes

- The Load Generator refuses to run unless the target is explicitly marked `authorized: true` — it is built for testing systems you own or are authorized to test, and has no spoofing, amplification or unauthorized-targeting capability.
- The Chaos Engine and Network Simulator only ever mutate the synthetic `network.Topology` object inside a Universe — never a real host, process or network interface.
- Set `VOID_JWT_SECRET` to a real secret before exposing the API server beyond localhost; every write endpoint requires a valid role (admin/engineer/viewer via RBAC).

### 8. Honest scope notes

This is a genuine, working platform (every subsystem listed above actually runs — the whole thing was built and smoke-tested end-to-end), not a mockup. A few pieces are deliberately a solid *foundation* rather than a maximal implementation, so you know what you're getting: the Database Simulation Layer ships with an in-memory store and a driver-agnostic interface (bring your own `database/sql` driver for a real Postgres/MySQL/SQLite target); the AI Assistant's built-in interpreter is fast pattern-matching rather than a full LLM (plug in a real backend via the `ai.Backend` interface for deeper natural-language understanding); the Plugin system is an in-process Go interface registry rather than dynamic `.so` loading (safer, and sufficient for most extension needs); and panel **docking/floating** in the UI is a natural next step beyond the resize/collapse that's implemented today.

---
<a id="فارسی"></a>
<div dir="rtl" align="right">

## فارسی

### ۱. VOID چیست

VOID به شما امکان می‌دهد یک **جهان مصنوعی (Synthetic Universe)** بسازید: دنیایی بزرگ، قابل کنترل، تکرارپذیر و شبیه‌سازی‌شده که از موجودیت‌ها (کاربر، مشتری، کارمند، دستگاه، سرور، سرویس، پایگاه‌داده، تراکنش، نشست، سفارش، پرداخت، حسگر، خودرو، گره شبکه و موارد دیگر)، روابط میان آن‌ها، رفتارهایی که در طول زمان دنبال می‌کنند، و رویدادها، ترافیک و خطاهایی که در آن جهان جریان دارند تشکیل شده است. این ابزار برای کسانی ساخته شده که به شرایط مصنوعی واقعی، بزرگ‌مقیاس و تکرارپذیر برای آزمودن سایر سیستم‌ها نیاز دارند — نه برای تولید چند فایل تست ساده و یک‌بارمصرف.

**هسته Go** مسئول تمام عملیات سنگین است: تولید موجودیت، شبیه‌سازی رویداد، اجرای رفتار، هماهنگ‌سازی سناریو و خط زمان، شبیه‌سازی شبکه و آشوب، پردازش تراکنش، تولید بار، متریک‌ها، عکس‌فوری و خروجی‌گیری — همگی بر پایه goroutine، channel و worker pool تا شبیه‌سازی‌های بزرگ هرگز رابط کاربری را مسدود نکنند. **رابط TypeScript / React / Next.js** صرفاً برای نمایش، تعامل و کنترل پروژه است و خودش هیچ پردازش شبیه‌سازی انجام نمی‌دهد.

### ۲. ویژگی‌های کلیدی

- **طراح موجودیت** — فیلدهای تایپ‌دار (رشته، عدد صحیح، اعشاری، بولی، تاریخ‌زمان، UUID، enum، آرایه، JSON، باینری، سفارشی)، مولدهای مختلف برای هر فیلد (تصادفی، ترتیبی، تصادفی وزن‌دار، نام/ایمیل/تلفن/آدرس، تاریخ/زمان، الگوی regex، توزیع آماری، وابسته/مشتق‌شده، تابع سفارشی) و روابط (یک‌به‌یک، یک‌به‌چند، چندبه‌چند، سلسله‌مراتبی).
- **مولد داده مصنوعی رابطه‌آگاه** — موجودیت‌های جدید واقعاً به موجودیت‌های مرتبط *موجود* متصل می‌شوند (یک سفارش واقعاً به یک مشتری واقعی اشاره می‌کند) به‌جای تولید کلید خارجی یتیم، با تکرارپذیری کامل بر پایه seed (توزیع‌های یکنواخت، نرمال، لگ‌نرمال، پواسون، نمایی، پارتو، وزن‌دار).
- **موتور رفتار** — تعاریف ماشین‌حالت / گراف رفتار (گره‌های رویداد، شرط، احتمال، عملیات، تأخیر، تغییر حالت، فراخوانی API، عملیات پایگاه‌داده، حلقه) نحوه رفتار موجودیت‌ها را در طول زمان هدایت می‌کنند.
- **موتور شبیه‌سازی رویداد** — یک event bus مبتنی بر worker pool و آگاه از فشار برگشتی (backpressure) که توان عملیاتی بسیار بالایی دارد بدون آنکه رابط کاربری را مسدود کند.
- **سازنده سناریو / خط زمان** — تولید موجودیت، اتصال رفتار، رویدادهای مجزا، بازه‌های تولید بار، خطاهای آشوب، مکث، عکس‌فوری و تغییرات شبکه را روی یک خط زمان واحد و تکرارپذیر زمان‌بندی کنید، در زمان واقعی، شتاب‌یافته، کندشده، مکث‌شده یا کاملاً قطعی.
- **تولیدکننده ترافیک و بار** — کاربران مجازی قابل‌تنظیم، نرخ درخواست، افزایش/کاهش تدریجی، متریک‌های زنده RPS / تأخیر (P50/P95/P99) / نرخ خطا. **این ابزار فقط به هدفی که صراحتاً authorized علامت‌گذاری کرده‌اید ترافیک ارسال می‌کند** — این یک ابزار تست بار است، نه ابزار حمله، و هیچ قابلیت جعل یا تقویت ترافیک ندارد.
- **شبیه‌ساز توپولوژی شبکه و موتور آشوب** — گره‌ها و لینک‌های کاملاً مصنوعی با تأخیر، jitter، افت بسته و پهنای‌باند قابل‌تنظیم؛ تزریق خطای کنترل‌شده (خرابی سرویس، timeout، تأخیر بالا، عدم دسترسی پایگاه‌داده، انباشت صف، فشار منابع، افت بسته، خرابی جزئی) که فقط روی همان توپولوژی مصنوعی اثر می‌گذارد، به‌همراه محاسبه اثر آبشاری در گراف وابستگی سرویس‌ها.
- **موتور تراکنش و قوانین کسب‌وکار** — پرداخت/سفارش/رزرو مصنوعی همراه با یک ارزیاب قانون ساده (`اگر موجودی کمتر از مبلغ بود رد کن`، `اگر موجودی صفر بود لغو کن` و مشابه آن).
- **لایه شبیه‌سازی پایگاه‌داده** — بارگذاری دسته‌ای و موازی در یک ذخیره‌ساز درون‌حافظه‌ای داخلی، همراه با رابطی مستقل از درایور تا بتوانید درایور واقعی `database/sql` (PostgreSQL، MySQL، SQLite) خودتان را متصل کنید.
- **موتور شبیه‌سازی زمان** — ساعت شبیه‌سازی به‌صورت زمان واقعی، شتاب‌یافته، کندشده، مکث‌شده یا کاملاً قطعی (مثلاً فشرده‌سازی ۳۰ روز شبیه‌سازی‌شده در چند دقیقه واقعی).
- **عکس‌فوری و بازپخش + مقایسه شبیه‌سازی** — وضعیت یک جهان را روی دیسک ذخیره کنید، بعداً از همان‌جا ادامه دهید، و متریک‌های دو اجرا را برای دیدن دقیق تفاوت‌ها مقایسه کنید.
- **متریک و قابلیت مشاهده** — یک جمع‌آورنده شمارنده/گیج درون‌حافظه‌ای با یک نقطه پایانی متنی سازگار با Prometheus، ردیابی تأخیر P50/P95/P99، و یک داشبورد زنده (شمارنده، نمودار خطی، نمودار میله‌ای) در رابط کاربری.
- **زمان‌بند با ساعات کاری تعریف‌شده توسط کاربر** — بازه‌های باز/بسته شدن خودتان را برای هر روز هفته وارد کنید (هر منطقه زمانی، چند بازه در هر روز، پشتیبانی از بازه‌های شبانه)؛ VOID به‌طور زنده محاسبه می‌کند که آیا اکنون باز هستید یا خیر و دقیقاً چقدر تا تغییر بعدی زمان باقی مانده، و می‌تواند اجرای شبیه‌سازی‌های زمان‌بندی‌شده/تکرارشونده را فقط در همان ساعات فعال کند. هیچ‌چیز از پیش در برنامه ثابت نشده — شما هر بازه را خودتان در پنل زمان‌بند یا از طریق API/CLI وارد می‌کنید.
- **دستیار هوش مصنوعی شبیه‌سازی / Scenario Copilot** — یک توضیح به زبان ساده (مثلاً «یک SaaS با ۱ میلیون کاربر، ۱۵٪ به‌طور همزمان فعال، اوج ترافیک ساعت ۱۸») را به یک پیش‌نویس اولیه از Schema موجودیت + سناریو تبدیل می‌کند تا پیش از اجرا در یک Preview بررسی کنید؛ این دستیار فقط پیکربندی تولید می‌کند و هرگز خودش چیزی را اجرا نمی‌کند، و به‌طور پیش‌فرض با یک مفسر الگو-محور بدون وابستگی بیرونی عرضه می‌شود (در صورت نیاز به درک زبانی عمیق‌تر می‌توانید یک backend واقعی LLM را از طریق رابط `ai.Backend` متصل کنید).
- **سیستم پلاگین** — رابط‌های Go برای مولدهای فیلد سفارشی، الگوهای نوع موجودیت، الگوهای رفتار، Connector ها و Exporter ها، ثبت‌شده درون همان پردازه (بدون بارگذاری پویای ناامن).
- **امنیت** — احراز هویت مبتنی بر توکن امضاشده با HMAC، کنترل دسترسی مبتنی بر نقش (admin / engineer / viewer)، محدودسازی نرخ درخواست به ازای هر کاربر، و یک لاگ ممیزی فقط‌الحاقی، همگی صرفاً با کتابخانه استاندارد Go پیاده‌سازی شده‌اند.
- **خروجی‌گیری** — JSON، JSONL، CSV، YAML، XML، دستورات INSERT در SQL، و یک قالب باینری ساده با طول از پیش مشخص، به‌صورت جریانی (streaming) نه بافرشده، تا مجموعه‌داده‌های بسیار بزرگ به‌درستی خروجی گرفته شوند.
- **کتابخانه قالب** — حوزه‌های آماده: فروشگاه اینترنتی، بانکداری، اینترنت اشیا، SaaS، شبکه اجتماعی، لجستیک، شهر هوشمند، فین‌تک، ریزسرویس‌ها، بازی، تله‌متری.
- **رابط کاربری** — پوسته‌ای با الهام از طراحی Fluent ویندوز ۱۱ (کاوشگر جهان، بوم شبیه‌سازی، بازرس، خط زمان، کنسول — همگی قابل تغییر اندازه و جمع‌شدن)، پنج تم (روشن، تاریک، پیش‌فرض ویندوز ۱۱، قرمز، آبی)، پالت دستور با Ctrl+K، و بومی‌سازی کامل انگلیسی / فارسی (راست‌به‌چپ واقعی) / چینی.

### ۳. معماری

پوشه `backend/` هسته Go (بدون هیچ وابستگی بیرونی) شامل بسته‌های `entity`، `generator`، `randomx`، `event`، `behavior`، `scenario`، `simulation`، `network`، `chaos`، `transaction`، `database`، `loadgen`، `metrics`، `storage`، `export`، `scheduler`، `security`، `plugin`، `ai`، `wsutil` و `api` است، به‌همراه نقطه ورود `cmd/api` (سرور REST + WebSocket) و `cmd/cli` (خط فرمان). پوشه `frontend/` رابط کاربری Next.js را شامل می‌شود: `app/` (مسیر ورود Next.js)، `components/layout` (پوسته اصلی، کاوشگر جهان، بوم، بازرس، خط زمان، کنسول)، `components/entity` و `components/behavior` (طراح موجودیت و ویرایشگر رفتار)، `components/dashboard` و `components/scheduler` (داشبورد زنده و پنل ساعات کاری)، `components/command` و `components/theme`، و `lib/` (کلاینت API، انواع مشترک، فروشگاه وضعیت، بومی‌سازی).

### ۴. پیش‌نیازها

Go نسخه ۱٫۲۲ یا بالاتر برای هسته و سرور API و CLI، و Node.js نسخه ۱۸٫۱۸ یا بالاتر (نسخه ۲۰ LTS توصیه می‌شود) به‌همراه npm نسخه ۹ یا بالاتر برای رابط کاربری. هسته Go هیچ وابستگی بیرونی ندارد — فقط از کتابخانه استاندارد Go استفاده می‌کند، بنابراین `go build` حتی بدون دسترسی به اینترنت هم کار می‌کند.

### ۵. نصب و اجرا

> این مراحل فرض می‌کنند که بایگانی (zip) پروژه را از قبل در پوشه‌ای روی سیستم خود استخراج کرده‌اید — از همان پوشه شروع کنید.

**بخش سرور (هسته Go + سرور API + CLI):**

```bash
cd backend

# ساخت سرور API و CLI
go build -o void-api ./cmd/api
go build -o void-cli ./cmd/cli

# اجرای مجموعه تست‌ها
go test ./...

# اجرای سرور API (به‌طور پیش‌فرض روی پورت ۸۰۸۰)
export VOID_JWT_SECRET="یک-رمز-طولانی-و-تصادفی-انتخاب-کنید"
export VOID_API_ADDR=":8080"
./void-api
```

**بخش رابط کاربری (Next.js):**

```bash
cd frontend

# نصب وابستگی‌ها
npm install

# تنظیم آدرس سرور API (پیش‌فرض http://localhost:8080)
cp .env.example .env.local
# در صورت نیاز، آدرس داخل .env.local را ویرایش کنید

# اجرای حالت توسعه
npm run dev

# یا ساخت نسخه تولید
npm run build
npm run start
```

آدرس محلی نمایش‌داده‌شده را باز کنید، روی دکمه **اتصال (Connect)** در نوار بالا کلیک کنید (هر نام کاربری، با انتخاب یک نقش)، سپس از پنل کاوشگر جهان یک Universe جدید بسازید.

### ۶. مرجع سریع CLI

```bash
# اجرای یک سناریوی کامل از فایل پیکربندی و خروجی‌گیری از نتایج
./void-cli run --config configs/ecommerce-scenario.yaml --wait 10 --out-dir ./out --format json

# تولید یک مجموعه‌داده مصنوعی مستقل فقط از یک فایل schema
./void-cli generate --schema configs/schema-user.example.json --count 100000 --seed 42 --out users.csv --format csv

# بررسی وضعیت ساعات کاری از یک فایل زمان‌بندی ذخیره‌شده
./void-cli scheduler status --hours configs/business-hours.example.json
```

### ۷. نکات امنیتی

تولیدکننده بار تا زمانی‌که هدف صراحتاً با `authorized: true` علامت‌گذاری نشده باشد اجرا نمی‌شود — این ابزار برای تست سیستم‌هایی ساخته شده که مالک آن‌ها هستید یا مجاز به تست آن‌ها هستید، و هیچ قابلیت جعل، تقویت یا هدف‌گیری غیرمجاز ندارد. موتور آشوب و شبیه‌ساز شبکه فقط و فقط شیء `network.Topology` مصنوعی درون یک Universe را تغییر می‌دهند — هرگز یک میزبان، پردازه یا رابط شبکه واقعی را. پیش از در دسترس قرار دادن سرور API فراتر از localhost، حتماً `VOID_JWT_SECRET` را به یک رمز واقعی تنظیم کنید؛ هر نقطه پایانی نوشتنی به یک نقش معتبر (admin/engineer/viewer از طریق RBAC) نیاز دارد.

### ۸. یادداشت صادقانه درباره محدوده کار

این یک پلتفرم واقعی و کارکردی است (تمام زیرسیستم‌های ذکرشده در بالا واقعاً اجرا می‌شوند — کل مجموعه ساخته و به‌صورت سرتاسری آزمایش شده است)، نه یک نمونه ظاهری. چند بخش عمداً به‌صورت یک **پایه محکم** پیاده‌سازی شده‌اند نه پیاده‌سازی حداکثری، تا دقیقاً بدانید چه چیزی دریافت می‌کنید: لایه شبیه‌سازی پایگاه‌داده با یک ذخیره‌ساز درون‌حافظه‌ای و یک رابط مستقل از درایور عرضه می‌شود (درایور واقعی `database/sql` خودتان را برای Postgres/MySQL/SQLite متصل کنید)؛ مفسر داخلی دستیار هوش مصنوعی یک تطبیق‌دهنده الگوی سریع است نه یک LLM کامل (برای درک زبانی عمیق‌تر یک backend واقعی را از طریق رابط `ai.Backend` متصل کنید)؛ سیستم پلاگین یک رجیستری رابط Go درون‌پردازه‌ای است نه بارگذاری پویای `.so.` (ایمن‌تر و برای اغلب نیازهای توسعه کافی است)؛ و قابلیت **docking/floating** پنل‌ها در رابط کاربری گام طبیعی بعدی فراتر از تغییر اندازه/جمع‌شدنی است که هم‌اکنون پیاده‌سازی شده.

</div>

---
<a id="中文"></a>
## 中文

### 1. VOID 是什么

VOID 让你构建一个**合成宇宙（Synthetic Universe）**：一个庞大、可控、可复现、可仿真的世界,由实体(用户、客户、员工、设备、服务器、服务、数据库、交易、会话、订单、支付、传感器、车辆、网络节点等)、它们之间的关系、它们随时间遵循的行为，以及在这个世界中流动的事件、流量和故障组成。它专为需要真实、大规模、可重复的合成条件来测试*其他*系统的人而设计——而不是用来生成几个一次性的测试文件。

**Go 引擎**负责所有繁重的操作：实体生成、事件仿真、行为执行、场景/时间线编排、网络与混沌仿真、交易处理、负载生成、指标采集、快照与导出——全部基于 goroutine、channel 和 worker pool 构建，确保大规模仿真永远不会阻塞界面。**TypeScript / React / Next.js 界面**纯粹用于可视化、交互和项目控制，它本身从不执行任何仿真运算。

### 2. 主要特性

- **实体设计器** — 类型化字段（字符串、整数、浮点数、布尔值、日期时间、UUID、枚举、数组、JSON、二进制、自定义），每个字段可选生成器（随机、顺序、加权随机、姓名/邮箱/电话/地址、日期/时间、正则模式、统计分布、依赖/派生、自定义函数），以及关系（一对一、一对多、多对多、层级关系）。
- **关系感知的合成数据生成器** — 新实体会真实关联到*已存在*的相关实体（订单确实指向一个真实存在的客户），而不是生成孤立的外键，并具备完整的基于种子的可复现性（均匀、正态、对数正态、泊松、指数、帕累托、加权分布）。
- **行为引擎** — 状态机/行为图定义(事件、条件、概率、动作、延迟、状态变更、API 调用、数据库操作、循环节点)驱动实体随时间的行为方式(登录 → 浏览 → 购买 → 登出、出错重试等)。
- **事件仿真引擎** — 基于工作池、具备背压感知能力的事件总线，可实现极高吞吐量而不阻塞界面。
- **场景构建器 / 时间线** — 在单一可复现的时间线上调度实体生成、行为绑定、单次事件、负载生成窗口、混沌故障、等待、快照与网络变更，支持实时、加速、减速、暂停或完全确定性的仿真时间。
- **流量与负载生成器** — 可配置虚拟用户数、请求速率、爬升/回落曲线，实时 RPS / 延迟 (P50/P95/P99) / 错误率指标。**它只会向你明确标记为已授权的目标发送流量**——这是一个负载测试工具，而非攻击工具，不具备任何伪造或流量放大能力。
- **网络拓扑模拟器与混沌引擎** — 纯合成的节点/链路，可配置延迟、抖动、丢包率与带宽；受控的故障注入(服务故障、超时、高延迟、数据库不可用、队列积压、资源压力、丢包、部分故障)仅作用于该合成拓扑本身，并可计算服务依赖图中的级联影响。
- **交易与业务规则引擎** — 合成的支付/订单/预订操作，配有一个简洁的规则求值器（如"若余额小于金额则拒绝"、"若库存为零则取消"等）。
- **数据库仿真层** — 批量并行写入内置的内存存储，并提供与具体驱动无关的接口，方便你接入自己的 `database/sql` 驱动（PostgreSQL、MySQL、SQLite）。
- **时间仿真引擎** — 支持实时、加速、减速、暂停或完全确定性的仿真时钟(例如将 30 天的仿真时间压缩到几分钟的真实时间内完成)。
- **快照与回放 + 仿真差异对比** — 将某一时刻的宇宙状态保存到磁盘，之后可从该状态继续，并对比两次运行的指标以精确查看发生了哪些变化。
- **指标与可观测性** — 内存中的计数器/仪表采集器，提供与 Prometheus 兼容的文本端点，支持 P50/P95/P99 延迟跟踪，并在界面中提供实时仪表盘(计数器、折线图、柱状图)。
- **带用户自定义营业时间的调度器** — 为一周中的每一天输入你自己的开始/结束时间段（支持任意时区、每天多个时间段、支持跨夜时间段）；VOID 会实时计算你当前是否处于营业状态，以及距离下一次状态变化确切还剩多长时间，并可将定时/周期性仿真运行限制在这些时间段内执行。时间表中没有任何内容是硬编码的——每一个时间段都由你自己在调度器面板或通过 API/CLI 输入。
- **AI 仿真助手 / 场景副驾驶** — 将一段自然语言描述（例如"一个拥有 100 万用户的 SaaS 产品，15% 同时在线，流量在 18 点达到峰值"）转换为一份初步的实体 Schema + 场景草稿，供你在运行前的预览中审阅；该助手只生成配置，从不自行执行任何操作，默认内置一个无需外部依赖、基于模式匹配的解释器（如需更深入的语言理解能力，可通过 `ai.Backend` 接口接入真实的大语言模型后端）。
- **插件系统** — 提供用于自定义字段生成器、实体类型模板、行为模板、连接器与导出器的 Go 接口，在进程内注册（不使用不安全的动态加载）。
- **安全性** — 基于 HMAC 签名的令牌认证、基于角色的访问控制（admin / engineer / viewer）、按用户的速率限制，以及一份仅追加写入的审计日志，全部仅使用 Go 标准库实现。
- **导出** — 支持 JSON、JSONL、CSV、YAML、XML、SQL 插入语句，以及一种简单的带长度前缀的二进制容器格式，采用流式写出而非全量缓冲，确保超大数据集也能顺利导出。
- **模板库** — 现成的领域模板：电子商务、银行业务、物联网、SaaS、社交网络、物流、智慧城市、金融科技、微服务、游戏、遥测。
- **界面** — 具有 Windows 11 Fluent 设计风格的外壳（宇宙浏览器、仿真画布、检查器、时间线、控制台——均可调整大小与折叠）、五种主题（浅色、深色、Windows 11 默认、红色、蓝色）、Ctrl+K 命令面板，以及完整的英文 / 波斯语（真正的从右到左）/ 中文本地化支持。

### 3. 架构

`backend/` 目录是 Go 引擎（零外部依赖），包含 `entity`、`generator`、`randomx`、`event`、`behavior`、`scenario`、`simulation`、`network`、`chaos`、`transaction`、`database`、`loadgen`、`metrics`、`storage`、`export`、`scheduler`、`security`、`plugin`、`ai`、`wsutil` 与 `api` 等包，以及入口点 `cmd/api`（REST + WebSocket 服务器）与 `cmd/cli`（命令行工具）。`frontend/` 目录包含 Next.js 界面：`app/`（Next.js 入口）、`components/layout`（主外壳、宇宙浏览器、画布、检查器、时间线、控制台）、`components/entity` 与 `components/behavior`（实体设计器与行为编辑器）、`components/dashboard` 与 `components/scheduler`（实时仪表盘与营业时间面板）、`components/command` 与 `components/theme`，以及 `lib/`（API 客户端、共享类型、状态存储、本地化）。

### 4. 前置条件

后端引擎、API 服务器与 CLI 需要 Go 1.22 或更高版本；前端界面需要 Node.js 18.18 或更高版本（建议使用 20 LTS）及 npm 9 或更高版本。Go 后端**没有任何外部依赖**——仅使用 Go 标准库，因此即使完全离线也可以执行 `go build`。

### 5. 安装与运行

> 以下步骤假设你已经将项目压缩包解压到本地某个文件夹——请从该文件夹内开始执行第 1 步。

**后端（Go 引擎 + API 服务器 + CLI）：**

```bash
cd backend

# 构建 API 服务器与 CLI
go build -o void-api ./cmd/api
go build -o void-cli ./cmd/cli

# 运行测试套件
go test ./...

# 启动 API 服务器（默认监听 :8080）
export VOID_JWT_SECRET="请设置一个足够长的随机密钥"
export VOID_API_ADDR=":8080"
./void-api
```

**前端（Next.js 界面）：**

```bash
cd frontend

# 安装依赖
npm install

# 配置 API 服务器地址（默认为 http://localhost:8080）
cp .env.example .env.local
# 如果你的 API 服务器不在 localhost:8080，请编辑 .env.local

# 启动开发服务器
npm run dev

# 或者构建生产版本
npm run build
npm run start
```

打开终端输出的本地地址，点击顶部栏的 **连接（Connect）** 按钮（任意用户名，选择一个角色），然后在宇宙浏览器面板中创建一个新的 Universe。

### 6. CLI 快速参考

```bash
# 从配置文件运行完整场景并导出结果
./void-cli run --config configs/ecommerce-scenario.yaml --wait 10 --out-dir ./out --format json

# 仅根据 schema 文件生成一份独立的合成数据集
./void-cli generate --schema configs/schema-user.example.json --count 100000 --seed 42 --out users.csv --format csv

# 根据已保存的时间表文件检查当前营业时间状态
./void-cli scheduler status --hours configs/business-hours.example.json
```

### 7. 安全说明

负载生成器在目标未被明确标记为 `authorized: true` 之前拒绝运行——它是为测试你拥有或已获授权测试的系统而构建的，不具备任何伪造、流量放大或未授权目标定位的能力。混沌引擎与网络模拟器只会修改 Universe 内部那个合成的 `network.Topology` 对象——绝不会触及真实的主机、进程或网络接口。在将 API 服务器暴露到 localhost 之外之前，请务必将 `VOID_JWT_SECRET` 设置为一个真正的密钥；每一个写入类端点都需要通过 RBAC 验证的有效角色（admin/engineer/viewer）。

### 8. 关于实现范围的诚实说明

这是一个真实可运行的平台（上面列出的每一个子系统都确实能运行——整个项目已经过端到端的构建与冒烟测试），而不是一个空壳演示。其中有几个部分是有意做成一个扎实的**基础实现**，而非追求最大化的完整实现，以便你清楚自己得到的是什么：数据库仿真层默认提供一个内存存储，并附带与具体驱动无关的接口（你可以接入自己的 `database/sql` 驱动，用于真正的 Postgres/MySQL/SQLite）；AI 助手内置的解释器是一个快速的模式匹配实现，而非完整的大语言模型（如需更深入的自然语言理解，可通过 `ai.Backend` 接口接入真实后端）；插件系统是进程内的 Go 接口注册表，而非动态 `.so` 加载（更安全，且足以满足大多数扩展需求）；而界面中面板的**停靠/浮动（docking/floating）**功能，是在当前已实现的调整大小/折叠功能基础上自然的下一步扩展方向。

