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

	SERVICE_CONSOL_WEB_PAGE        = "application.service.routerPath.console"
	SERVICE_CONSOL_WEB_PAGE_DIR    = "application.service.routerPath.console_dir"
	SERVICE_CONSOL_WEB_PAGE_ASSETS = "application.service.routerPath.console_assets"
	DB_DATA                        = "application.sqlService"
	DB_ORDER_TAB                   = "application.sqlService.order_tab"
	DB_ROOM_TYPE_TAB               = "application.sqlService.room_type_tab"
	DB_USER_TAB                    = "application.sqlService.user_tab"
	DB_USER_EVENT_TAB              = "application.sqlService.user_event_tab"
	DB_TELEGRAM_CONTACT_TAB        = "application.sqlService.telegram_contact_tab"

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
