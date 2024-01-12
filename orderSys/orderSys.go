package orderSys

import (
	"encoding/json"
	"fmt"
	keymgrmod "housekeepr/keyMgrMod"
	"housekeepr/loginMod"
	"housekeepr/serviceMod"
	"housekeepr/settingMod"
	"housekeepr/sqlMod"
	"housekeepr/telegramBot"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/olahol/melody.v1"
)

type OrderSys struct {
	orderInfo OrderInfo
}

//訂單資訊
//[orderId]Order data
type OrderInfo map[string]*RoomOrder

type RoomOrder struct {
	OrderId        string             `json:"OrderId"`
	CheckInData    string             `json:"CheckInData"`
	CheckOutData   string             `json:"CheckOutData"`
	NumberOfPeople int                `json:"NumberOfPeople"`
	Cost           int                `json:"Cost"`
	RoomStatus     OrderStatus        `json:"OrderStatus"`
	RoomExplain    string             `json:"RoomExplain"`
	RoomType       keymgrmod.RoomType `json:"RoomType"`
	Paid           bool               `json:"Paid"`
}

type PostSetOrder struct {
	CheckInData    string             `json:"CheckInData"`
	CheckOutData   string             `json:"CheckOutData"`
	NumberOfPeople int                `json:"NumberOfPeople"`
	Cost           int                `json:"Cost"`
	RoomExplain    string             `json:"RoomExplain"`
	RoomType       keymgrmod.RoomType `json:"RoomType"`
	Paid           bool               `json:"Paid"`
	Account        string             `json:"Account"`
	Token          string             `json:"Token"`
}

//訂單狀態
type OrderStatus int

const (
	Unknown OrderStatus = 0
	//預定
	Reserve OrderStatus = 1
	//等待入住
	WaitCheckIn OrderStatus = 2
	//入住
	CheckIn OrderStatus = 3
	//等待清潔
	WaitClear OrderStatus = 4
	//確認清潔
	WaitClearCheck OrderStatus = 5
	//清潔完成
	ClearFinish OrderStatus = 6
	//付款
	Paid OrderStatus = 7
	//刪除訂單
	Delete   OrderStatus = 8
	RoomKeys             = "OrderId, CheckInDate, CheckOutDate, NumberOfPeople, Cost,OrderStatus, RoomExplain, Paid, RoomType"

	ORDER_ID_FORM = "{CheckInTime}-{RoomType}"
)

//指定房間金鑰有效日
const SPECIFIC_KEY_VALIDITY_INTERVAL = 7

//確認訂單內容
func (sys *OrderSys) checkOrder(postOrder *PostSetOrder) (bool, string) {
	baseDateString := "2006-01-02"
	if postOrder.NumberOfPeople <= 0 {
		log.Printf("Order 輸入人數有誤")
		return false, "Order 輸入人數有誤"
	}
	if postOrder.Cost <= 0 {
		log.Printf("Order 輸入金額有誤")
		return false, "Order 輸入金額有誤"
	}
	if postOrder.RoomType == keymgrmod.RoomUnknown {
		log.Printf("訂單房間代號有誤 (%v)", postOrder.RoomType)
		return false, fmt.Sprintf("訂單房間代號有誤 (%v)", postOrder.RoomType)
	}

	checkInDate, err := time.Parse(baseDateString, postOrder.CheckInData)
	if err != nil {
		log.Printf("Order check in time(%s) 時間錯誤", postOrder.CheckInData)
		return false, fmt.Sprintf("Order check in time(%s) 時間錯誤", postOrder.CheckInData)
	}

	checkOutData, err := time.Parse(baseDateString, postOrder.CheckOutData)
	if err != nil {
		log.Printf("Order check out time(%s) 時間錯誤", postOrder.CheckOutData)
		return false, fmt.Sprintf("Order check out time(%s) 時間錯誤", postOrder.CheckOutData)
	}

	for _, order := range sys.orderInfo {
		orderCheckInDate, _ := time.Parse(baseDateString, order.CheckInData)
		orderCheckOutDate, _ := time.Parse(baseDateString, order.CheckOutData)
		if postOrder.RoomType == order.RoomType {
			if orderCheckInDate.Before(checkInDate) && orderCheckOutDate.After(checkInDate) {
				return false, fmt.Sprintf("Check in date (%s~%s) 已經有人預定", orderCheckInDate, orderCheckOutDate)
			}
			if checkInDate.Before(orderCheckInDate) && checkOutData.After(orderCheckInDate) {
				return false, fmt.Sprintf("Check in date (%s~%s) 已經有人預定", orderCheckInDate, orderCheckOutDate)
			}
		}
	}

	if postOrder.RoomExplain == "" {
		postOrder.RoomExplain = "null"
	}
	return true, ""
}

// 初始化
func (sys *OrderSys) init() {
	TimeFormatForm := "2006-01-02 15:04:05"
	sys.orderInfo = make(OrderInfo)
	now := time.Now()
	firstDateTime := now.AddDate(0, 0, -now.Day()+1)
	firstDateZeroTime := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, firstDateTime.Location())
	lastDateZeroTime := time.Date(now.Year(), 12, 31, 0, 0, 0, 0, firstDateTime.Location())
	sys.queryOrder(firstDateZeroTime.Local().Format(TimeFormatForm), lastDateZeroTime.Local().Format(TimeFormatForm))
}

func (sys *OrderSys) Run() {
	log.Println("order sys running")
	//系統初始化
	sys.init()
	keymgrmod.GetInstance().Init()
	//檢查 token 有效
	checkUserTokeTime := time.NewTicker(time.Minute * 30)
	go sys.checkUserTokenLiveTime(checkUserTokeTime)
	//websocket
	serviceMod.GetInstance().RegisterWebsocket()
	serviceMod.GetInstance().RegisterWebsocketConnect(sys.handleWebsocketConnect)
	serviceMod.GetInstance().RegisterWebsocketMessage(sys.handleWebsocketMessage)
	//register post api
	//Logout
	serviceMod.GetInstance().RegisterPOST(
		fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.SERVICE_POST_LOGOUT)),
		sys.handleLogoutService)
	//Login
	serviceMod.GetInstance().RegisterPOST(
		fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.SERVICE_POST_LOGIN)),
		sys.handleLoginService)
	//Set order
	serviceMod.GetInstance().RegisterPOST(
		fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.SERVICE_POST_SET_DATA)),
		sys.handlePOSTOrderService)

	//取得指定的訂單資料
	serviceMod.GetInstance().RegisterGet(
		fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.SERVICE_GET_GET_DATA)),
		sys.handleGetDataServices)

	//取得目前的房號設定
	serviceMod.GetInstance().RegisterGet(
		fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.SERVICE_GET_GET_ROOM_SETTING)),
		sys.handleGetRoomSetting)
	//取得Uer Type
	serviceMod.GetInstance().RegisterGet(
		fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.SERVICE_GET_GET_USER_TYPE)),
		sys.handleGetUserType)
	//取得大門金鑰資料
	serviceMod.GetInstance().RegisterGet(
		fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.SERVICE_GET_GET_DOOR_KEY_DATA)),
		sys.handleGetDoorKeyData)
	//取得房間金鑰資料
	serviceMod.GetInstance().RegisterGet(
		fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.SERVICE_GET_GET_ROOM_KEY_DATA)),
		sys.handleGetRoomKeyData)

	//新增大門金鑰
	serviceMod.GetInstance().RegisterPOST(
		fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.SERVICE_POST_ADD_DOOR_KEY)),
		sys.handleAddDoorKey)
	//新增房間金鑰
	serviceMod.GetInstance().RegisterPOST(
		fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.SERVICE_POST_ADD_ROOM_KEY)),
		sys.handleAddRoomKey)
	//刪除大門金鑰
	serviceMod.GetInstance().RegisterPOST(
		fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.SERVICE_POST_DELETE_DOOR_KEY)),
		sys.handleDeleteDoorKey)
	// 刪除房間金鑰
	serviceMod.GetInstance().RegisterPOST(
		fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.SERVICE_POST_DELETE_ROOM_KEY)),
		sys.handleDeleteRoomKey)

	// serviceMod.GetInstance().RegisterWebPage(
	// 	fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.SERVICE_CONSOL_WEB_PAGE)),
	// 	fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.SERVICE_CONSOL_WEB_PAGE_DIR)),
	// 	fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.SERVICE_CONSOL_WEB_PAGE_ASSETS)))

	serviceMod.GetInstance().Run()
}

//檢查使用者的 token 存活時間
func (sys *OrderSys) checkUserTokenLiveTime(ticker interface{}) {
	for range ticker.(*time.Ticker).C {
		//檢查 user
		loginMod.GetInstance().CheckUserTokenLive()
	}
}

//Handle websocket Meg(order process)
func (sys *OrderSys) handleWebsocketMessage(s *melody.Session, message []byte) {
	var orderData map[string]string
	fmt.Println("handleWebsocketMessage", string(message))
	websocketMsg := new(serviceMod.WebSocketMessage)
	if err := json.Unmarshal(message, websocketMsg); err != nil {
		s.Write(serviceMod.GetInstance().CreateWebSocketMsg(serviceMod.WEBSOCKET_EVENT_ERROR,
			err).ToByte())
		return
	}
	websocketContentByteData, err := json.Marshal(websocketMsg.Content)
	if err != nil {
		fmt.Println("handleWebsocketMessage error", err)
		s.Write(serviceMod.GetInstance().CreateWebSocketMsg(serviceMod.WEBSOCKET_EVENT_ON_FAIL,
			err).ToByte())
		return
	}
	if err := json.Unmarshal(websocketContentByteData, &orderData); err != nil {
		fmt.Println("handleWebsocketMessage error", err)
		s.Write(serviceMod.GetInstance().CreateWebSocketMsg(serviceMod.WEBSOCKET_EVENT_ON_FAIL,
			err).ToByte())
		return
	}
	orderId := orderData["orderId"]
	account := orderData["account"]
	switch websocketMsg.Event {
	//pay
	case serviceMod.WEBSOCKET_EVENT_CHECK_PAY:
		if loginMod.GetInstance().GetUserStaffType(orderData["account"]) != loginMod.BOSS {
			s.Write(serviceMod.GetInstance().CreateWebSocketMsg(
				serviceMod.WEBSOCKET_EVENT_ON_FAIL, "帳號沒有權限").ToByte())
			return
		}
		if sys.Pay(orderId, account) {
			fmt.Println("handleWebsocketMessage check pay order id=", orderData["orderId"])

			s.Write(serviceMod.GetInstance().CreateWebSocketMsg(
				serviceMod.WEBSOCKET_EVENT_ON_SUCCESS, orderData["orderId"]+" check pay success").ToByte())

		} else {
			s.Write(serviceMod.GetInstance().CreateWebSocketMsg(
				serviceMod.WEBSOCKET_EVENT_ON_FAIL, orderData["orderId"]+" 不存在").ToByte())
		}

	//check clear
	case serviceMod.WEBSOCKET_EVENT_CHECK_CLEAR:
		if sys.checkClear(orderId, account) {
			fmt.Println("handleWebsocketMessage check clear order id=", orderData["orderId"])
			s.Write(serviceMod.GetInstance().CreateWebSocketMsg(
				serviceMod.WEBSOCKET_EVENT_ON_SUCCESS, orderData["orderId"]+" check clear success").ToByte())

		} else {
			s.Write(serviceMod.GetInstance().CreateWebSocketMsg(
				serviceMod.WEBSOCKET_EVENT_ON_FAIL, orderData["orderId"]+" 不存在").ToByte())
		}

	//check out
	case serviceMod.WEBSOCKET_EVENT_CHECK_OUT:

		if loginMod.GetInstance().GetUserStaffType(orderData["account"]) != loginMod.BOSS {
			s.Write(serviceMod.GetInstance().CreateWebSocketMsg(
				serviceMod.WEBSOCKET_EVENT_ON_FAIL, "帳號沒有權限").ToByte())
			return
		}
		if sys.checkOut(orderId, account) {
			fmt.Println("handleWebsocketMessage check out order id=", orderData["orderId"])

			s.Write(serviceMod.GetInstance().CreateWebSocketMsg(
				serviceMod.WEBSOCKET_EVENT_ON_SUCCESS, orderData["orderId"]+" check out success").ToByte())

		} else {
			s.Write(serviceMod.GetInstance().CreateWebSocketMsg(
				serviceMod.WEBSOCKET_EVENT_ON_FAIL, orderData["orderId"]+" 不存在").ToByte())
		}

	//check in
	case serviceMod.WEBSOCKET_EVENT_CHECK_IN:

		if loginMod.GetInstance().GetUserStaffType(orderData["account"]) != loginMod.BOSS {
			s.Write(serviceMod.GetInstance().CreateWebSocketMsg(
				serviceMod.WEBSOCKET_EVENT_ON_FAIL, "帳號沒有權限").ToByte())
			return
		}
		if sys.checkIn(orderId, account) {
			fmt.Println("handleWebsocketMessage check in order id=", orderData["orderId"])
			s.Write(serviceMod.GetInstance().CreateWebSocketMsg(
				serviceMod.WEBSOCKET_EVENT_ON_SUCCESS, orderData["orderId"]+" check in success").ToByte())

		} else {
			s.Write(serviceMod.GetInstance().CreateWebSocketMsg(
				serviceMod.WEBSOCKET_EVENT_ON_FAIL, orderData["orderId"]+" 不存在").ToByte())
		}

		//刪除訂單
	case serviceMod.WEBSOCKET_EVENT_DEL_ORDER:
		if loginMod.GetInstance().GetUserStaffType(orderData["account"]) != loginMod.BOSS {
			s.Write(serviceMod.GetInstance().CreateWebSocketMsg(
				serviceMod.WEBSOCKET_EVENT_ON_FAIL, "帳號沒有權限").ToByte())
			return
		}
		if sys.delOrder(orderId, account) {
			fmt.Println("handleWebsocketMessage check in order id=", orderId)
			s.Write(serviceMod.GetInstance().CreateWebSocketMsg(
				serviceMod.WEBSOCKET_EVENT_ON_SUCCESS, orderId+" 刪除完成").ToByte())

		} else {
			s.Write(serviceMod.GetInstance().CreateWebSocketMsg(
				serviceMod.WEBSOCKET_EVENT_ON_FAIL, orderId+" 不存在").ToByte())
		}
	}
}

//Check in
func (sys *OrderSys) checkIn(orderId, account string) bool {
	if sys.orderInfo[orderId] != nil {
		roomSetting := keymgrmod.GetInstance().GetRoomSetting()
		sys.orderInfo[orderId].RoomStatus = CheckIn
		sys.updateOrderToDB(sys.orderInfo[orderId])
		sys.updateUserEditEvent(orderId, account, CheckIn)
		//send Telegram Bot msg
		telegramBot.GetInstance().Broadcast(fmt.Sprintf("OrderId:%v \nRoom:%v \n已經入住",
			orderId, roomSetting[int(sys.orderInfo[orderId].RoomType)]),
			loginMod.BOSS)
		//broadcast all user refresh data
		serviceMod.GetInstance().BroadcastWebsocketMsg(
			serviceMod.GetInstance().CreateWebSocketMsg(serviceMod.WEBSOCKET_EVENT_UPDATE, sys.orderInfo))
		return true
	} else {
		return false
	}
}

//Check out
func (sys *OrderSys) checkOut(orderId, account string) bool {
	if sys.orderInfo[orderId] != nil {
		roomSetting := keymgrmod.GetInstance().GetRoomSetting()
		sys.orderInfo[orderId].RoomStatus = WaitClear
		sys.updateUserEditEvent(orderId, account, WaitClear)
		sys.updateOrderToDB(sys.orderInfo[orderId])
		//send Telegram Bot msg
		telegramBot.GetInstance().Broadcast(fmt.Sprintf("OrderId:%v \nRoom:%v \n已經退房",
			orderId, roomSetting[int(sys.orderInfo[orderId].RoomType)]), loginMod.JANITOR)
		//broadcast all user refresh data
		serviceMod.GetInstance().BroadcastWebsocketMsg(
			serviceMod.GetInstance().CreateWebSocketMsg(serviceMod.WEBSOCKET_EVENT_UPDATE, sys.orderInfo))
		return true
	} else {
		return false
	}
}

//確認清潔
func (sys *OrderSys) checkClear(orderId, account string) bool {
	if sys.orderInfo[orderId] != nil {
		roomSetting := keymgrmod.GetInstance().GetRoomSetting()
		sys.orderInfo[orderId].RoomStatus = ClearFinish
		sys.updateUserEditEvent(orderId, account, ClearFinish)
		sys.updateOrderToDB(sys.orderInfo[orderId])
		//send Telegram Bot msg
		telegramBot.GetInstance().Broadcast(fmt.Sprintf("OrderId:%v \nRoom:%v \n檢查清潔完成",
			orderId, roomSetting[int(sys.orderInfo[orderId].RoomType)]), loginMod.BOSS)
		//broadcast all user refresh data
		serviceMod.GetInstance().BroadcastWebsocketMsg(
			serviceMod.GetInstance().CreateWebSocketMsg(serviceMod.WEBSOCKET_EVENT_UPDATE, sys.orderInfo))
		return true
	} else {
		return false
	}
}

//付款
func (sys *OrderSys) Pay(orderId, account string) bool {
	if sys.orderInfo[orderId] != nil {
		roomSetting := keymgrmod.GetInstance().GetRoomSetting()
		sys.orderInfo[orderId].Paid = true
		sys.updateOrderToDB(sys.orderInfo[orderId])
		sys.updateUserEditEvent(orderId, account, Paid)
		//send Telegram Bot msg
		telegramBot.GetInstance().Broadcast(fmt.Sprintf("OrderId:%v \nRoom:%v \n付款完成",
			orderId, roomSetting[int(sys.orderInfo[orderId].RoomType)]), loginMod.BOSS)
		//broadcast all user refresh data
		serviceMod.GetInstance().BroadcastWebsocketMsg(
			serviceMod.GetInstance().CreateWebSocketMsg(serviceMod.WEBSOCKET_EVENT_UPDATE, sys.orderInfo))
		return true
	} else {
		return false
	}
}

//刪除訂單
func (sys *OrderSys) delOrder(orderId, account string) bool {
	if sys.orderInfo[orderId] != nil {
		delete(sys.orderInfo, orderId)
		sys.deleteOrderToDB(orderId)
		sys.updateUserEditEvent(orderId, account, Delete)
		//send Telegram Bot msg
		telegramBot.GetInstance().Broadcast(fmt.Sprintf("OrderId:%v \nAccount:%v \n刪除完成",
			orderId, account), loginMod.BOSS)
		//broadcast all user refresh data
		serviceMod.GetInstance().BroadcastWebsocketMsg(
			serviceMod.GetInstance().CreateWebSocketMsg(serviceMod.WEBSOCKET_EVENT_UPDATE, sys.orderInfo))
		return true
	} else {
		return false
	}

}
func (sys *OrderSys) handleWebsocketConnect(s *melody.Session) {
	fmt.Println("handleWebsocketConnect")
	token := s.Request.URL.Query().Get("token")
	account := s.Request.URL.Query().Get("account")
	//檢查 account and token 是否存在
	if msg, success := loginMod.GetInstance().CheckUserToken(account, token); !success {
		//token 錯誤
		s.CloseWithMsg([]byte(msg))
		return
	}
	//send order data to client
	loginMod.GetInstance().SetWebSocketSession(account, s)
	s.Write(serviceMod.GetInstance().CreateWebSocketMsg(serviceMod.WEBSOCKET_EVENT_ORDER_INFO, sys.orderInfo).ToByte())

}

//get data service handle
func (sys *OrderSys) handleGetDataServices(c *gin.Context) {
	c.JSON(200, sys.orderInfo)
}

func (sys *OrderSys) handleGetRoomSetting(c *gin.Context) {
	token := c.Query("token")
	account := c.Query("account")
	fmt.Printf("token: %v\n", token)
	fmt.Printf("account: %v\n", account)
	if msg, ok := loginMod.GetInstance().CheckUserToken(account, token); ok {
		c.JSON(200, keymgrmod.GetInstance().GetRoomSetting())
	} else {
		c.JSON(int(serviceMod.RESPONSE_GET_TOKEN_FAIL), msg)
	}
}

func (sys *OrderSys) handleGetUserType(c *gin.Context) {
	token := c.Query("token")
	account := c.Query("account")
	fmt.Printf("token: %v\n", token)
	fmt.Printf("account: %v\n", account)
	if msg, ok := loginMod.GetInstance().CheckUserToken(account, token); ok {
		c.JSON(200, loginMod.GetInstance().GetUserStaffType(account))
	} else {
		c.JSON(int(serviceMod.RESPONSE_GET_TOKEN_FAIL), msg)
	}
}

//取得大門金鑰資料
func (sys *OrderSys) handleGetDoorKeyData(c *gin.Context) {
	token := c.Query("token")
	account := c.Query("account")
	fmt.Printf("token: %v\n", token)
	fmt.Printf("account: %v\n", account)
	if msg, ok := loginMod.GetInstance().CheckUserToken(account, token); ok {
		c.JSON(200, keymgrmod.GetInstance().GetDoorKeyData())
	} else {
		c.JSON(int(serviceMod.RESPONSE_GET_TOKEN_FAIL), msg)
	}
}

//取得房間金鑰資料
func (sys *OrderSys) handleGetRoomKeyData(c *gin.Context) {
	token := c.Query("token")
	account := c.Query("account")
	fmt.Printf("token: %v\n", token)
	fmt.Printf("account: %v\n", account)
	if msg, ok := loginMod.GetInstance().CheckUserToken(account, token); ok {
		c.JSON(200, keymgrmod.GetInstance().GetRoomKeyData())
	} else {
		c.JSON(int(serviceMod.RESPONSE_GET_TOKEN_FAIL), msg)
	}
}

//Login service
func (sys *OrderSys) handleLoginService(c *gin.Context) {
	json := make(map[string]interface{})
	c.BindJSON(&json)
	if json["Account"] != nil && json["Password"] != nil {
		account := fmt.Sprintf("%v", json["Account"])
		password := fmt.Sprintf("%v", json["Password"])
		if loginMod.GetInstance().DoLogin(account, password) {
			log.Println(account, "Login success")
			if token, ok := loginMod.GetInstance().GetUserToken(account); ok {
				c.JSON(int(serviceMod.RESPONSE_SUCCESS), token)
			} else {
				c.JSON(int(serviceMod.RESPONSE_POST_GET_USER_TOKEN_FAIL), "Get token fail")
			}
		} else {
			c.JSON(int(serviceMod.RESPONSE_POST_GET_USER_Login_FAIL), "account or pwd error")
		}
	} else {
		c.JSON(int(serviceMod.RESPONSE_POST_GET_USER_Login_FAIL), "account or pwd empty")
	}
}

//handle logout service
func (sys *OrderSys) handleLogoutService(c *gin.Context) {
	json := make(map[string]interface{})
	c.BindJSON(&json)
	token := fmt.Sprintf("%v", json["Token"])
	account := fmt.Sprintf("%v", json["Account"])
	fmt.Printf("account:%v logout\n", account)
	if msg, ok := loginMod.GetInstance().CheckUserToken(string(account), token); ok {
		loginMod.GetInstance().DoLogout(account)
		c.JSON(200, nil)
	} else {
		c.JSON(int(serviceMod.RESPONSE_GET_TOKEN_FAIL), msg)
	}
}

// 處理新增大門金鑰
func (sys *OrderSys) handleAddDoorKey(c *gin.Context) {
	var postData keymgrmod.POSTDoorKeyData
	// 檢查資料
	if err := c.Bind(&postData); err != nil {
		log.Println("orderSys handleAddDoorKey error", err)
		c.JSON(int(serviceMod.RESPONSE_POST_DATA_FORM_ERR), "Request data bind err")
		return
	}
	// 檢查權限
	if loginMod.GetInstance().GetUserStaffType(postData.Account) > loginMod.JANITOR ||
		loginMod.GetInstance().GetUserStaffType(postData.Account) == loginMod.UNKNOWN {
		c.JSON(int(serviceMod.RESPONSE_POST_PERMISSION_FAIL), "帳號沒有權限")
		return
	}
	// 檢查 token 是否有效
	if msg, success := loginMod.GetInstance().CheckUserToken(postData.Account, postData.Token); !success {
		c.JSON(int(serviceMod.RESPONSE_POST_PERMISSION_FAIL), msg)
		return
	}
	// 塞入資料
	if msg, success := keymgrmod.GetInstance().AddDoorKey(postData); success {
		c.JSON(int(serviceMod.RESPONSE_SUCCESS), "新增大門金鑰 Success")
	} else {
		c.JSON(int(serviceMod.RESPONSE_POST_PERMISSION_FAIL), msg)
	}
}

// 處理新增房間金鑰
func (sys *OrderSys) handleAddRoomKey(c *gin.Context) {
	var postData keymgrmod.POSTRoomKeyData
	// 檢查資料
	if err := c.Bind(&postData); err != nil {
		log.Println("orderSys handleAddRoomKey error", err)
		c.JSON(int(serviceMod.RESPONSE_POST_DATA_FORM_ERR), "Request data bind err")
		return
	}
	// 檢查權限
	if loginMod.GetInstance().GetUserStaffType(postData.Account) > loginMod.JANITOR ||
		loginMod.GetInstance().GetUserStaffType(postData.Account) == loginMod.UNKNOWN {
		c.JSON(int(serviceMod.RESPONSE_POST_PERMISSION_FAIL), "帳號沒有權限")
		return
	}
	// 檢查 token 是否有效
	if msg, success := loginMod.GetInstance().CheckUserToken(postData.Account, postData.Token); !success {
		c.JSON(int(serviceMod.RESPONSE_POST_PERMISSION_FAIL), msg)
		return
	}
	// 塞入資料
	if msg, success := keymgrmod.GetInstance().AddRoomKey(postData); success {
		c.JSON(int(serviceMod.RESPONSE_SUCCESS), "新增房間金鑰 Success")
	} else {
		c.JSON(int(serviceMod.RESPONSE_POST_PERMISSION_FAIL), msg)
	}
}

// 處理刪除房間金鑰
func (sys *OrderSys) handleDeleteRoomKey(c *gin.Context) {
	var postData keymgrmod.POSTRoomKeyData
	// 檢查資料
	if err := c.Bind(&postData); err != nil {
		log.Println("orderSys handleDeleteRoomKey error", err)
		c.JSON(int(serviceMod.RESPONSE_POST_DATA_FORM_ERR), "Request data bind err")
		return
	}
	// 檢查權限
	if loginMod.GetInstance().GetUserStaffType(postData.Account) > loginMod.JANITOR ||
		loginMod.GetInstance().GetUserStaffType(postData.Account) == loginMod.UNKNOWN {
		c.JSON(int(serviceMod.RESPONSE_POST_PERMISSION_FAIL), "帳號沒有權限")
		return
	}
	// 檢查 token 是否有效
	if msg, success := loginMod.GetInstance().CheckUserToken(postData.Account, postData.Token); !success {
		c.JSON(int(serviceMod.RESPONSE_POST_PERMISSION_FAIL), msg)
		return
	}
	// 刪除房間資料
	if keymgrmod.GetInstance().DeleteRoomKey(postData.Id) {
		c.JSON(int(serviceMod.RESPONSE_SUCCESS), "刪除房間金鑰 Success")
	}
}

// 處理刪除大門金鑰
func (sys *OrderSys) handleDeleteDoorKey(c *gin.Context) {
	var postData keymgrmod.POSTRoomKeyData
	// 檢查資料
	if err := c.Bind(&postData); err != nil {
		log.Println("orderSys handleDeleteDoorKey error", err)
		c.JSON(int(serviceMod.RESPONSE_POST_DATA_FORM_ERR), "Request data bind err")
		return
	}
	// 檢查權限
	if loginMod.GetInstance().GetUserStaffType(postData.Account) > loginMod.JANITOR ||
		loginMod.GetInstance().GetUserStaffType(postData.Account) == loginMod.UNKNOWN {
		c.JSON(int(serviceMod.RESPONSE_POST_PERMISSION_FAIL), "帳號沒有權限")
		return
	}
	// 檢查 token 是否有效
	if msg, success := loginMod.GetInstance().CheckUserToken(postData.Account, postData.Token); !success {
		c.JSON(int(serviceMod.RESPONSE_POST_PERMISSION_FAIL), msg)
		return
	}
	// 刪除大門資料
	if keymgrmod.GetInstance().DeleteDoorKey(postData.Id) {
		c.JSON(int(serviceMod.RESPONSE_SUCCESS), "刪除大門金鑰 Success")
	}
}

//set data service handle
func (sys *OrderSys) handlePOSTOrderService(c *gin.Context) {
	var postOrderDate PostSetOrder
	if err := c.Bind(&postOrderDate); err != nil {
		log.Println("orderSys handleSetDataService error", err)
		c.JSON(int(serviceMod.RESPONSE_POST_DATA_FORM_ERR), "Request data bind err")
		return
	}
	fmt.Printf("postOrderDate: %v\n", postOrderDate)
	if success, errMsg := sys.checkOrder(&postOrderDate); !success {
		log.Println("orderSys handleSetDataService error", errMsg)
		c.JSON(int(serviceMod.RESPONSE_POST_DATA_FORM_ERR), errMsg)
		return

	}
	if loginMod.GetInstance().GetUserStaffType(postOrderDate.Account) != loginMod.BOSS {
		c.JSON(int(serviceMod.RESPONSE_POST_PERMISSION_FAIL), "帳號沒有權限")
		return
	}

	if msg, success := loginMod.GetInstance().CheckUserToken(postOrderDate.Account, postOrderDate.Token); !success {
		c.JSON(int(serviceMod.RESPONSE_POST_PERMISSION_FAIL), msg)
		return
	}

	orderId := sys.createOrderId(&postOrderDate)
	fmt.Printf("orderId: %v\n", orderId)
	if _, ok := sys.orderInfo[orderId]; !ok {
		roomSetting := keymgrmod.GetInstance().GetRoomSetting()
		order := RoomOrder{
			OrderId:        orderId,
			CheckInData:    postOrderDate.CheckInData,
			CheckOutData:   postOrderDate.CheckOutData,
			NumberOfPeople: postOrderDate.NumberOfPeople,
			Cost:           postOrderDate.Cost,
			RoomStatus:     Reserve,
			RoomExplain:    postOrderDate.RoomExplain,
			RoomType:       postOrderDate.RoomType,
			Paid:           postOrderDate.Paid,
		}
		fmt.Printf("order: %v\n", order)
		sys.orderInfo[order.OrderId] = &order
		sys.insertOrderToDB(&order)
		sys.insertUserEditEvent(orderId, postOrderDate.Account, Reserve)
		//broadcast all user refresh data
		serviceMod.GetInstance().BroadcastWebsocketMsg(
			serviceMod.GetInstance().CreateWebSocketMsg(serviceMod.WEBSOCKET_EVENT_UPDATE, sys.orderInfo))
		//send Telegram Bot msg
		telegramBot.GetInstance().Broadcast(fmt.Sprintf("OrderId:%v \nRoom:%v \n入住時間:%v ~ %v \n新增訂單 @IgsRichardRd4",
			order.OrderId,
			roomSetting[int(order.RoomType)],
			order.CheckInData,
			order.CheckOutData), loginMod.BOSS)
		c.JSON(int(serviceMod.RESPONSE_SUCCESS), "新增訂單成功")
	}
}

//建立訂單編號
func (sys *OrderSys) createOrderId(postOrder *PostSetOrder) string {
	orderId := ORDER_ID_FORM
	orderId = strings.ReplaceAll(orderId, "{CheckInTime}", postOrder.CheckInData)
	orderId = strings.ReplaceAll(orderId, "{RoomType}", fmt.Sprintf("%v", postOrder.RoomType))
	return orderId
}

//查詢訂單
func (sys *OrderSys) queryOrder(firstDate, lastDate string) {
	query := "SELECT {RowName} FROM {TableName} "
	query = strings.ReplaceAll(query, "{TableName}", fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.DB_ORDER_TAB)))
	query = strings.ReplaceAll(query, "{RowName}", RoomKeys)
	query = query + fmt.Sprintf("where CheckInDate>='%v' and CheckInDate<='%v'", firstDate, lastDate)

	fmt.Printf("query: %v\n", query)
	rows := sqlMod.GetInstance().Query(query)
	for rows.Next() {
		var order RoomOrder
		rows.Scan(&order.OrderId,
			&order.CheckInData,
			&order.CheckOutData,
			&order.NumberOfPeople,
			&order.Cost,
			&order.RoomStatus,
			&order.RoomExplain,
			&order.Paid,
			&order.RoomType)
		sys.orderInfo[order.OrderId] = &order
	}
}

// 新增訂單to DB
func (sys *OrderSys) insertOrderToDB(order *RoomOrder) {
	insert := "INSERT INTO {TableName} ({RowName}) VALUES({Values})"
	insert = strings.ReplaceAll(insert, "{TableName}", fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.DB_ORDER_TAB)))
	insert = strings.ReplaceAll(insert, "{RowName}", RoomKeys)
	insert = strings.ReplaceAll(insert, "{Values}", fmt.Sprintf("'%v','%v','%v',%v,%v,%v,'%v',%v, %v",
		order.OrderId,
		order.CheckInData,
		order.CheckOutData,
		order.NumberOfPeople,
		order.Cost,
		order.RoomStatus,
		order.RoomExplain,
		order.Paid,
		order.RoomType))
	fmt.Println("sql:", insert)

	sqlMod.GetInstance().Exec(insert)
}

//更新使用者訂單記錄
func (sys *OrderSys) updateUserEditEvent(orderId, account string, event OrderStatus) {
	now := time.Now()
	update := "UPDATE {TableName} SET EditEvent=%v,EditDate='%v', UserAccount='%v' WHERE OrderId='%v'"
	update = strings.ReplaceAll(update, "{TableName}", fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.DB_USER_EVENT_TAB)))
	update = fmt.Sprintf(update,
		event,
		now.Format("2006-01-02 15:04:05"),
		account,
		orderId)
	fmt.Printf("update: %v\n", update)

	sqlMod.GetInstance().Exec(update)
}

//新增使用者訂單記錄 For 追蹤訂單
func (sys *OrderSys) insertUserEditEvent(orderId, account string, event OrderStatus) {
	insert := "INSERT INTO {TableName} ({RowName}) VALUES({Values})"
	insert = strings.ReplaceAll(insert, "{TableName}", fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.DB_USER_EVENT_TAB)))
	insert = strings.ReplaceAll(insert, "{RowName}", "OrderId, UserAccount, EditEvent")
	insert = strings.ReplaceAll(insert, "{Values}", fmt.Sprintf("'%v','%v',%v", orderId, account, event))
	fmt.Println("sql:", insert)
	sqlMod.GetInstance().Exec(insert)
}

//更新DB訂單資料
func (sys *OrderSys) updateOrderToDB(order *RoomOrder) {
	update := "UPDATE {TableName} SET OrderId='%v', CheckInDate='%v', CheckOutDate='%v', NumberOfPeople=%v, Cost=%v,OrderStatus=%v, RoomExplain='%v', Paid=%v WHERE OrderId='%v'"
	update = strings.ReplaceAll(update, "{TableName}", fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.DB_ORDER_TAB)))
	update = fmt.Sprintf(update, order.OrderId,
		order.CheckInData,
		order.CheckOutData,
		order.NumberOfPeople,
		order.Cost,
		order.RoomStatus,
		order.RoomExplain,
		order.Paid,
		order.OrderId)

	fmt.Printf("update: %v\n", update)

	sqlMod.GetInstance().Exec(update)
}

func (sys *OrderSys) deleteOrderToDB(orderId string) {
	delete := "DELETE FROM {TableName} WHERE OrderId='%v'"
	delete = strings.ReplaceAll(delete, "{TableName}", fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.DB_ORDER_TAB)))
	delete = fmt.Sprintf(delete, orderId)

	fmt.Printf("deleteOrderToDB: %v\n", delete)

	sqlMod.GetInstance().Exec(delete)
}
