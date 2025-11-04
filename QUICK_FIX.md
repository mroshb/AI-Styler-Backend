# راه حل سریع برای رفع خطای worker_jobs

## مشکل:
خطای `pq: relation "worker_jobs" does not exist` در worker service

## راه حل 1: اجرای Migration (توصیه می‌شود)

```bash
go run scripts/migrate/main.go up
```

## راه حل 2: اجرای Script مستقیم

```bash
# با استفاده از متغیرهای محیطی
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f scripts/create_worker_table.sql

# یا مستقیماً
psql -h localhost -p 5432 -U postgres -d styler -f scripts/create_worker_table.sql
```

## بررسی وضعیت Migration ها

```bash
go run scripts/migrate/main.go status
```

## توجه:
پس از رفع مشکل، worker service به صورت خودکار شروع به کار می‌کند و خطاها متوقف می‌شوند.
