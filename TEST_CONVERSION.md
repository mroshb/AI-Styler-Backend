# راهنمای تست Conversion Endpoint

این فایل راهنمای تست کردن endpoint `/api/convert` است.

## پیش‌نیازها

1. سرور باید در حال اجرا باشد (`localhost:8080`)
2. نیاز به access token دارید (از طریق login)
3. دو عکس باید قبلاً آپلود شده باشند:
   - `user_image_id`: 93e7c110-35e4-4498-ad22-f3c8171a069d
   - `cloth_image_id`: ab7e7810-c870-47bc-ae44-476bd2d87168

## روش 1: استفاده از اسکریپت

```bash
# 1. لاگین کنید و access token بگیرید
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"phone":"YOUR_PHONE","password":"YOUR_PASSWORD"}'

# 2. access token را در environment variable قرار دهید
export ACCESS_TOKEN="your_access_token_here"

# 3. اسکریپت تست را اجرا کنید
./test_conversion_simple.sh
```

## روش 2: استفاده از curl مستقیم

```bash
# 1. ایجاد conversion
curl -X POST http://localhost:8080/api/convert \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_image_id": "93e7c110-35e4-4498-ad22-f3c8171a069d",
    "cloth_image_id": "ab7e7810-c870-47bc-ae44-476bd2d87168",
    "style_name": "vintage"
  }'

# 2. بررسی وضعیت conversion (بعد از چند ثانیه)
curl -X GET http://localhost:8080/api/conversion/{CONVERSION_ID}/status \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"

# 3. دریافت جزئیات کامل conversion
curl -X GET http://localhost:8080/api/conversion/{CONVERSION_ID} \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

## بررسی Logs

برای بررسی اینکه آیا worker در حال پردازش است، لاگ‌های سرور را چک کنید:

```bash
# اگر سرور را با go run اجرا کرده‌اید، لاگ‌ها در terminal نمایش داده می‌شوند
# دنبال این پیام‌ها باشید:
# - "Processing job ... of type image_conversion"
# - "Starting image conversion for job ..."
# - "Downloading user image from ..."
# - "Downloading cloth image from ..."
# - "Calling Gemini API for image conversion..."
# - "Gemini API conversion successful"
```

## بررسی مشکلات احتمالی

### 1. خطای Authentication
- مطمئن شوید access token معتبر است
- دوباره login کنید و token جدید بگیرید

### 2. خطای Image Not Found
- مطمئن شوید که image IDs موجود هستند
- ابتدا تصاویر را آپلود کنید

### 3. خطای Quota Exceeded
- سهمیه conversion شما تمام شده
- باید plan خود را upgrade کنید

### 4. Worker در حال پردازش نیست
- بررسی کنید که worker service در حال اجرا است
- لاگ‌های سرور را چک کنید
- بررسی کنید که `worker_jobs` table وجود دارد

### 5. خطای Gemini API
- بررسی کنید که `GEMINI_API_KEY` درست تنظیم شده
- بررسی کنید که `GEMINI_MODEL` صحیح است
- لاگ‌های API را چک کنید

## فیلدهای Request

```json
{
  "user_image_id": "UUID of user image",
  "cloth_image_id": "UUID of cloth image",
  "style_name": "optional style name (e.g., 'vintage', 'casual', 'formal')"
}
```

## Response موفق (201 Created)

```json
{
  "data": {
    "id": "conversion-uuid",
    "userId": "user-uuid",
    "userImageId": "user-image-uuid",
    "clothImageId": "cloth-image-uuid",
    "status": "pending",
    "createdAt": "2025-11-05T...",
    "updatedAt": "2025-11-05T..."
  }
}
```

## مراحل پردازش

1. **pending**: Conversion ایجاد شده و در صف انتظار است
2. **processing**: Worker در حال پردازش است
3. **completed**: پردازش با موفقیت انجام شد
4. **failed**: پردازش با خطا مواجه شد

## نکات مهم

- پس از ایجاد conversion، worker به صورت خودکار آن را پردازش می‌کند
- پردازش ممکن است چند ثانیه تا چند دقیقه طول بکشد
- برای بررسی وضعیت، از endpoint `/api/conversion/{id}/status` استفاده کنید
- نتیجه conversion در فیلد `resultImageId` ذخیره می‌شود

