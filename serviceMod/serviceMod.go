package serviceMod

import (
	"encoding/json"
	"fmt"
	"housekeepr/settingMod"
	"log"
	"net/http"
	"sync"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gopkg.in/olahol/melody.v1"
)

type ServiceCode int
type websocketSubEvent string

type Service struct {
	ginServer       *gin.Engine
	melodyWebsocket *melody.Melody
}

type WebSocketMessage struct {
	Event   websocketSubEvent `json:"event"`
	Content interface{}       `json:"content"`
}

func (m *WebSocketMessage) ToByte() []byte {
	result, _ := json.Marshal(m)
	return result
}

var (
	serviceInstance *Service
	once            = &sync.Once{}
)

const (
	WEBSOCKET_EVENT_ORDER_INFO         websocketSubEvent = "OrderInfo"
	WEBSOCKET_EVENT_CHECK_IN           websocketSubEvent = "CheckIn"
	WEBSOCKET_EVENT_CHECK_OUT          websocketSubEvent = "CheckOut"
	WEBSOCKET_EVENT_CHECK_CLEAR        websocketSubEvent = "CheckClear"
	WEBSOCKET_EVENT_CHECK_PAY          websocketSubEvent = "CheckPay"
	WEBSOCKET_EVENT_DEL_ORDER          websocketSubEvent = "DelOrder"
	WEBSOCKET_EVENT_ON_SUCCESS         websocketSubEvent = "OnSuccess"
	WEBSOCKET_EVENT_ON_FAIL            websocketSubEvent = "OnFail"
	WEBSOCKET_EVENT_UPDATE             websocketSubEvent = "OrderInfoUpdate"
	WEBSOCKET_EVENT_ERROR              websocketSubEvent = "Error"
	POST_KEY_CHECK_IN_TIME                               = "checkInDate"
	POST_KEY_CHECK_OUT_TIME                              = "checkOutData"
	POST_KEY_NUMBER_OF_PEOPLE                            = "numberOfPeople"
	RESPONSE_SUCCESS                   ServiceCode       = 200
	RESPONSE_POST_DATA_FORM_ERR        ServiceCode       = 401
	RESPONSE_POST_ORDER_NOT_EXIST      ServiceCode       = 402
	RESPONSE_POST_GET_USER_TOKEN_FAIL  ServiceCode       = 403
	RESPONSE_POST_GET_USER_Login_FAIL  ServiceCode       = 405
	RESPONSE_WEBSOCKET_VERY_TOKEN_FAIL ServiceCode       = 406
	RESPONSE_GET_TOKEN_FAIL            ServiceCode       = 407
	RESPONSE_POST_PERMISSION_FAIL      ServiceCode       = 408
)

//get setting instance
func GetInstance() *Service {
	once.Do(func() {
		serviceInstance = new(Service)
		serviceInstance.init()
	})
	return serviceInstance
}

func (s *Service) init() {
	log.Println("service init--------------------")
	ginServer := gin.Default()
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"http://127.0.0.1:5500", "http://192.168.0.90:9999", "http://192.168.0.90:5500"}
	ginServer.Use(cors.New(corsConfig))
	s.ginServer = ginServer

}

func (s *Service) Run() {
	s.ginServer.Run(fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.PORT)))
}

//websocket
func (s *Service) RegisterWebsocket() {
	s.melodyWebsocket = melody.New()
	s.RegisterGet("/ws", func(c *gin.Context) {
		s.melodyWebsocket.HandleRequest(c.Writer, c.Request)
	})
}

func (s *Service) BroadcastWebsocketMsg(msg *WebSocketMessage) {
	s.melodyWebsocket.Broadcast(msg.ToByte())
}

func (s *Service) CreateWebSocketMsg(event websocketSubEvent, content interface{}) *WebSocketMessage {
	return &WebSocketMessage{
		Event:   event,
		Content: content,
	}
}

func (s *Service) RegisterWebsocketConnect(callback func(*melody.Session)) {
	s.melodyWebsocket.HandleConnect(callback)
}

func (s *Service) RegisterWebsocketMessage(callback func(*melody.Session, []byte)) {
	s.melodyWebsocket.HandleMessage(callback)
}

//register POST
func (s *Service) RegisterPOST(routerPath string, callback func(c *gin.Context)) {
	s.ginServer.POST(routerPath, callback)
}

//register GET
func (s *Service) RegisterGet(routerPath string, callback func(c *gin.Context)) {
	s.ginServer.GET(routerPath, callback)
}

func (s *Service) RegisterWebPage(routerPath, htmlPath, assetsPath string) {
	s.ginServer.LoadHTMLGlob(htmlPath)
	s.ginServer.Static("/assets", assetsPath)
	s.ginServer.GET(routerPath, func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})
}
