-- 建立 訂單與帳號連結
CREATE TABLE IF NOT EXISTS data_stuff_telegram (
    -- 員工Telegram帳號
    TelegramAccount NVARCHAR(30) NOT NULL Primary key,
    -- 員工帳號
    UserAccount NVARCHAR(20) NOT NULL,
    -- 員工類型
    StaffType int not null
);