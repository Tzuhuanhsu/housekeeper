-- 建立 room order table
CREATE TABLE IF NOT EXISTS data_room_order (
    -- 訂單編號
    OrderId NVARCHAR(20) NOT NULL PRIMARY KEY,
    -- 入住時間
    CheckInDate DATE NOT NULL,
    -- 退房時間
    CheckOutDate DATE NOT NULL,
    -- 人數
    NumberOfPeople INT NOT NULL,
    -- 金額
    Cost Int NOT NULL,
    -- 目前狀態
    OrderStatus int not null,
    -- 付款狀態
    Paid Boolean DEFAULT false,
    -- 額外說明
    RoomExplain NVARCHAR(50)
);
-- 建立 room order table
CREATE TABLE IF NOT EXISTS code_staff_type (
    -- 員工類別代碼
    StaffType int not null Primary key,
    -- 說明
    StaffExplain NVARCHAR(10)
);
-- 建立 room order table
CREATE TABLE IF NOT EXISTS code_order_status (
    -- 訂單狀態
    OrderStatus int not null Primary key,
    -- 說明
    StatusExplain NVARCHAR(20)
);
-- 員工帳號密碼資料表
CREATE TABLE IF NOT EXISTS data_stuff(
    -- 員工帳號
    UserAccount NVARCHAR(20) not null Primary key,
    -- 員工密碼
    UserPassword NVARCHAR(20) not null,
    -- 員工類型
    StaffType int not null
);
-- 建立 訂單與帳號連結
CREATE TABLE IF NOT EXISTS data_order_with_account (
    -- 訂單編號
    OrderId NVARCHAR(20) NOT NULL Primary key,
    -- 員工帳號
    UserAccount NVARCHAR(20) NOT NULL,
    -- EditDate
    EditDate DATETIME DEFAULT(NOW()) NULL,
    -- EditEvent
    EditEvent INT NOT NULL
);