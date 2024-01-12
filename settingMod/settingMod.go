package settingMod

import (
	"fmt"
	"sync"

	"github.com/spf13/viper"
)

type Setting struct {
	isReady bool
}

const (
	SETTING_PATH = "./assets"
	SETTING_TYPE = "yaml"
	SETTING_FILE = "setting"
	//取得參數
	APP  = "application"
	PORT = "application.port"
	//POST Service
	SERVICE_POST_SET_DATA = "application.service.routerPath.setData"
	SERVICE_POST_LOGIN    = "application.service.routerPath.login"
	SERVICE_POST_LOGOUT   = "application.service.routerPath.logout"
	//GET Services
	SERVICE_GET_GET_DATA = "application.service.routerPath.getData"
	//GET Services
	SERVICE_GET_GET_ROOM_SETTING = "application.service.routerPath.getRoomSetting"
	//Get User Type
	SERVICE_GET_GET_USER_TYPE = "application.service.routerPath.getUserType"
	//Get Door Key Data
	SERVICE_GET_GET_DOOR_KEY_DATA = "application.service.routerPath.getDoorKeyData"
	//Get Room key data
	SERVICE_GET_GET_ROOM_KEY_DATA = "application.service.routerPath.getRoomKeyData"
	SERVICE_POST_ADD_ROOM_KEY     = "application.service.routerPath.addRoomKey"
	SERVICE_POST_ADD_DOOR_KEY     = "application.service.routerPath.addDoorKey"
	SERVICE_POST_DELETE_ROOM_KEY  = "application.service.routerPath.deleteRoomKey"
	SERVICE_POST_DELETE_DOOR_KEY  = "application.service.routerPath.deleteDoorKey"

	SERVICE_CONSOL_WEB_PAGE        = "application.service.routerPath.console"
	SERVICE_CONSOL_WEB_PAGE_DIR    = "application.service.routerPath.console_dir"
	SERVICE_CONSOL_WEB_PAGE_ASSETS = "application.service.routerPath.console_assets"
	DB_DATA                        = "application.sqlService"
	DB_ORDER_TAB                   = "application.sqlService.order_tab"
	DB_ROOM_TYPE_TAB               = "application.sqlService.room_type_tab"
	DB_USER_TAB                    = "application.sqlService.user_tab"
	DB_USER_EVENT_TAB              = "application.sqlService.user_event_tab"
	DB_TELEGRAM_CONTACT_TAB        = "application.sqlService.telegram_contact_tab"
	DB_DOOR_KEY_INFO_TAB           = "application.sqlService.door_key_info_tab"
	DB_ROOM_KEY_INFO_TAB           = "application.sqlService.data_room_key_info"

	TELEGRAM_BOT_TOKEN   = "application.telegramBot.token"
	TELEGRAM_BOT_CHAT_ID = "application.telegramBot.chatId"
)

var (
	SettingInstance *Setting
	once            = &sync.Once{}
)

// class 初始化
func init() {
	GetInstance().init()
}

//get setting instance
func GetInstance() *Setting {
	once.Do(func() {
		SettingInstance = new(Setting)
	})
	return SettingInstance
}

//setting init
func (s *Setting) init() {
	//檢查設定檔案是否已經載入
	if s.isReady {
		fmt.Println("setting 重複初始化")
		return
	}
	fmt.Println("setting 初始化")
	viper.SetConfigName(SETTING_FILE)
	viper.SetConfigType(SETTING_TYPE)
	viper.AddConfigPath(SETTING_PATH)
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println("setting err:", err)
		return
	}
	s.isReady = true
}

//get setting value
func (s *Setting) GetVal(key string) interface{} {
	if !s.isReady {
		fmt.Println("Setting unready")
		return nil
	}
	return viper.Get(key)
}
