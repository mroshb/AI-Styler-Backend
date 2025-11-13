# راهنمای استارت ربات در Windows

## روش 1: استفاده از اسکریپت Batch (ساده‌ترین)

### دوبار کلیک کنید روی:
```
start-bot.bat
```

یا در Command Prompt:
```cmd
start-bot.bat
```

## روش 2: استفاده از PowerShell

### در PowerShell:
```powershell
.\start-bot.ps1
```

اگر خطای execution policy گرفتید:
```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
.\start-bot.ps1
```

## روش 3: دستی (بدون اسکریپت)

### در Command Prompt:

```cmd
REM بارگذاری متغیرهای محیطی (دستی)
set TELEGRAM_BOT_TOKEN=8578746464:AAGCVHk0NMvy-TXKwTwplgu2iTJxAd9Hhbg
set BOT_ENV=development
set API_BASE_URL=http://localhost:8080
set POSTGRES_DSN=host=185.202.113.229 port=5432 user=postgres password=A1212A1212a dbname=styler sslmode=disable
set MAX_UPLOAD_SIZE=10MB
set HEALTH_PORT=8081
set RATE_LIMIT_MESSAGES=10
set RATE_LIMIT_CONVERSIONS=5
set RATE_LIMIT_WINDOW=1m
set JWT_SECRET=95286739ac9475a2aac66036e01f18d34f18def61241df5f0aee472dfa3fdbc6c7522fe670226ed1910099bf59ecbedfce465677d15cfcda3558d6e7e9fd2c11

REM استارت ربات
go run cmd/bot/main.go
```

### در PowerShell:

```powershell
# بارگذاری متغیرهای محیطی
$env:TELEGRAM_BOT_TOKEN="8578746464:AAGCVHk0NMvy-TXKwTwplgu2iTJxAd9Hhbg"
$env:BOT_ENV="development"
$env:API_BASE_URL="http://localhost:8080"
$env:POSTGRES_DSN="host=185.202.113.229 port=5432 user=postgres password=A1212A1212a dbname=styler sslmode=disable"
$env:MAX_UPLOAD_SIZE="10MB"
$env:HEALTH_PORT="8081"
$env:RATE_LIMIT_MESSAGES="10"
$env:RATE_LIMIT_CONVERSIONS="5"
$env:RATE_LIMIT_WINDOW="1m"
$env:JWT_SECRET="95286739ac9475a2aac66036e01f18d34f18def61241df5f0aee472dfa3fdbc6c7522fe670226ed1910099bf59ecbedfce465677d15cfcda3558d6e7e9fd2c11"

# استارت ربات
go run cmd/bot/main.go
```

## روش 4: استفاده از .env.bot (پیشنهادی)

### در PowerShell (بهترین روش):

```powershell
# بارگذاری خودکار از .env.bot
Get-Content ".env.bot" | ForEach-Object {
    $line = $_.Trim()
    if ($line -and -not $line.StartsWith("#")) {
        $parts = $line -split "=", 2
        if ($parts.Length -eq 2) {
            $key = $parts[0].Trim()
            $value = $parts[1].Trim()
            [Environment]::SetEnvironmentVariable($key, $value, "Process")
        }
    }
}

# استارت ربات
go run cmd/bot/main.go
```

## لاگ‌های مورد انتظار

بعد از استارت، باید این لاگ‌ها را ببینید:

```
Starting Telegram bot in development mode...
Creating bot with token: 8578746464...
Bot authenticated successfully! Username: @chi_beposham_bot (ID: ...)
Starting bot in polling mode...
Getting updates channel...
✅ Bot is now listening for updates! Send /start to test.
✅ Bot service started successfully!
📱 Send /start to your bot to test it
```

## عیب‌یابی

### مشکل: "TELEGRAM_BOT_TOKEN is required"

**راه حل:**
- بررسی کنید که `.env.bot` موجود است
- یا متغیرهای محیطی را دستی تنظیم کنید (روش 3)

### مشکل: "Failed to initialize database"

**راه حل:**
- دیتابیس باید در دسترس باشد
- `POSTGRES_DSN` را بررسی کنید

### مشکل: Execution Policy در PowerShell

**راه حل:**
```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

## نکات مهم

1. **Backend باید در حال اجرا باشد** - ربات به API backend نیاز دارد
2. **هر دو سرویس می‌توانند همزمان اجرا شوند** - در پنجره‌های جداگانه
3. **Redis اختیاری است** - اگر Redis در دسترس نباشد، ربات کار می‌کند

## دستورات سریع

### Command Prompt:
```cmd
start-bot.bat
```

### PowerShell:
```powershell
.\start-bot.ps1
```

---

**حالا یکی از روش‌ها را امتحان کنید!** 🚀

