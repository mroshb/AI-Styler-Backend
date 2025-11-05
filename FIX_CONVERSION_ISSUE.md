# راهنمای رفع مشکل "Cloth image not found or not accessible"

## مشکل:
خطای `pq: Cloth image not found or not accessible` هنگام ایجاد conversion

## علت:
تابع SQL `create_conversion` در دیتابیس به‌روز نشده است و نمی‌تواند تصاویر متعلق به کاربر را به عنوان cloth image بپذیرد.

## راه حل سریع:

### روش 1: اجرای SQL Script (توصیه می‌شود)

```bash
# اجرای script برای بررسی وضعیت تصاویر و به‌روزرسانی function
psql -h localhost -p 5432 -U YOUR_DB_USER -d styler -f scripts/check_image_and_fix.sql
```

یا اگر متغیرهای محیطی تنظیم شده‌اند:

```bash
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f scripts/check_image_and_fix.sql
```

### روش 2: اجرای مستقیم SQL

```sql
-- به‌روزرسانی function
CREATE OR REPLACE FUNCTION create_conversion(
    p_user_id UUID,
    p_vendor_id UUID,
    p_user_image_id UUID,
    p_cloth_image_id UUID,
    p_conversion_type TEXT DEFAULT 'free',
    p_style_name TEXT DEFAULT NULL
) RETURNS UUID AS $$
DECLARE
    conversion_id UUID;
    owner_type TEXT;
    owner_id UUID;
BEGIN
    -- Determine owner
    IF p_user_id IS NOT NULL THEN
        owner_type := 'user';
        owner_id := p_user_id;
    ELSIF p_vendor_id IS NOT NULL THEN
        owner_type := 'vendor';
        owner_id := p_vendor_id;
    ELSE
        RAISE EXCEPTION 'Either user_id or vendor_id must be provided';
    END IF;
    
    -- Validate images exist and belong to owner
    IF p_user_id IS NOT NULL THEN
        IF NOT EXISTS (
            SELECT 1 FROM images 
            WHERE id = p_user_image_id 
            AND user_id = p_user_id
            AND type IN ('user', 'result')
        ) THEN
            RAISE EXCEPTION 'User image not found or does not belong to user';
        END IF;
    ELSIF p_vendor_id IS NOT NULL THEN
        IF NOT EXISTS (
            SELECT 1 FROM images 
            WHERE id = p_user_image_id 
            AND vendor_id = p_vendor_id
            AND type IN ('vendor', 'result')
        ) THEN
            RAISE EXCEPTION 'Image not found or does not belong to vendor';
        END IF;
    END IF;
    
    -- Validate cloth image (can be public vendor image, public image, or user's own image)
    IF NOT EXISTS (
        SELECT 1 FROM images 
        WHERE id = p_cloth_image_id 
        AND (
            type = 'vendor' 
            OR is_public = true
            OR (p_user_id IS NOT NULL AND user_id = p_user_id AND type = 'user')
        )
    ) THEN
        RAISE EXCEPTION 'Cloth image not found or not accessible';
    END IF;
    
    -- Create conversion record
    INSERT INTO conversions (
        user_id, vendor_id, user_image_id, cloth_image_id, 
        conversion_type, style_name
    )
    VALUES (
        p_user_id, p_vendor_id, p_user_image_id, p_cloth_image_id,
        p_conversion_type, p_style_name
    )
    RETURNING id INTO conversion_id;
    
    -- Record usage history
    INSERT INTO image_usage_history (
        image_id, user_id, vendor_id, conversion_id, action
    )
    VALUES (
        p_user_image_id, p_user_id, p_vendor_id, conversion_id, 'use_in_conversion'
    );
    
    INSERT INTO image_usage_history (
        image_id, user_id, vendor_id, conversion_id, action
    )
    VALUES (
        p_cloth_image_id, p_user_id, p_vendor_id, conversion_id, 'use_in_conversion'
    );
    
    RETURN conversion_id;
END;
$$ LANGUAGE plpgsql;
```

## بررسی وضعیت تصاویر:

اگر می‌خواهید وضعیت تصاویر را بررسی کنید:

```sql
-- بررسی cloth image
SELECT 
    id,
    user_id,
    vendor_id,
    type,
    is_public,
    file_name
FROM images 
WHERE id = 'ab7e7810-c870-47bc-ae44-476bd2d87168';

-- بررسی user image
SELECT 
    id,
    user_id,
    vendor_id,
    type,
    is_public,
    file_name
FROM images 
WHERE id = '93e7c110-35e4-4498-ad22-f3c8171a069d';
```

## اگر تصویر public نیست:

اگر cloth image متعلق به کاربر است اما public نیست، می‌توانید آن را public کنید:

```sql
-- Public کردن تصویر
UPDATE images 
SET is_public = true 
WHERE id = 'ab7e7810-c870-47bc-ae44-476bd2d87168';
```

یا اگر می‌خواهید مطمئن شوید که تصویر متعلق به کاربر است:

```sql
-- بررسی مالکیت تصویر
SELECT 
    id,
    user_id,
    vendor_id,
    type,
    is_public
FROM images 
WHERE id = 'ab7e7810-c870-47bc-ae44-476bd2d87168'
AND (user_id = 'YOUR_USER_ID' OR vendor_id = 'YOUR_USER_ID');
```

## پس از اجرای SQL:

1. برنامه را restart کنید (یا نیازی نیست - function بلافاصله به‌روز می‌شود)
2. دوباره conversion را تست کنید

## نکته مهم:

اگر بعد از اجرای SQL script هنوز خطا می‌گیرید، احتمالاً:
- تصویر با آن ID وجود ندارد
- تصویر متعلق به کاربر دیگری است و public هم نیست
- نوع تصویر ('type') درست نیست

در این صورت، از query های بالا برای بررسی وضعیت استفاده کنید.

