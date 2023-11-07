CREATE VIEW order_option_with_account AS
SELECT a.OrderId,
    a.UserAccount,
    a.EditEvent,
    b.StatusExplain,
    a.EditDate
FROM data_order_with_account AS A
    LEFT JOIN code_order_status AS B ON A.EditEvent = B.OrderStatus