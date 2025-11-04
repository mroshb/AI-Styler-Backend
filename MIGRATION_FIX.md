# 🔧 راهنمای رفع مشکل worker_jobs

## مشکل فعلی
خطای `pq: relation "worker_jobs" does not exist` در worker service مشاهده می‌شود.

## ✅ راه حل‌های موجود

### روش 1: استفاده از اسکریپت خودکار (ساده‌ترین روش)

```bash
./scripts/create_worker_table.sh
```

این اسکریپت به صورت خودکار:
- فایل `.env` را می‌خواند
- به دیتابیس متصل می‌شود
- جدول `worker_jobs` را ایجاد می‌کند

### روش 2: استفاده از Migration Tool

```bash
# بررسی وضعیت migration ها
go run scripts/migrate/main.go status

# اجرای همه migration ها
go run scripts/migrate/main.go up
```

### روش 3: اجرای مستقیم SQL

```bash
# با استفاده از متغیرهای محیطی
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f scripts/create_worker_table.sql

# یا مستقیماً
psql -h localhost -p 5432 -U postgres -d styler -f scripts/create_worker_table.sql
```

## 📝 نکات مهم

1. **پس از ایجاد جدول**: سرویس worker به صورت خودکار شروع به کار می‌کند
2. **Error Handling**: Worker service حالا خطای "table does not exist" را gracefully handle می‌کند و هر 30 ثانیه یکبار پیام راهنما نمایش می‌دهد
3. **Route های Admin و Notification**: مشکل route های تکراری (`/api/admin/admin/...`) برطرف شده است

## ✅ تغییرات اعمال شده

- ✅ Route های admin و notification درست شدند
- ✅ Worker service error handling بهتر شد
- ✅ Migration script بهبود یافت
- ✅ Script سریع برای ایجاد worker_jobs table اضافه شد

## 🔍 بررسی وضعیت

```bash
# بررسی اینکه جدول ایجاد شده
psql -d styler -c "\d worker_jobs"

# بررسی وضعیت migration ها
go run scripts/migrate/main.go status
```

