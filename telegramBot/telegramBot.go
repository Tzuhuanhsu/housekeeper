package telegramBot

import (
	"fmt"
	"housekeepr/loginMod"
	"housekeepr/settingMod"
	"housekeepr/sqlMod"
	"strconv"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type telegramContact struct {
	telegramAccount string
	sysAccount      string
	stuffType       loginMod.StaffType
}
type telegramBot struct {
	bot     *tgbotapi.BotAPI
	contact []telegramContact
}

var (
	telegramBotInstance *telegramBot
	once                = &sync.Once{}
)

//get telegramBot instance
func GetInstance() *telegramBot {
	once.Do(func() {
		telegramBotInstance = new(telegramBot)
		telegramBotInstance.init()

	})
	return telegramBotInstance
}

//init
func (Bot *telegramBot) init() {
	if bot, err := tgbotapi.NewBotAPI(fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.TELEGRAM_BOT_TOKEN))); err != nil {
		fmt.Printf("telegramBot create bot error:(%v)", err)
	} else {
		Bot.bot = bot
	}
	Bot.bot.Debug = false
	query := "SELECT {RowName} FROM {TableName} "
	query = strings.ReplaceAll(query, "{TableName}", fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.DB_TELEGRAM_CONTACT_TAB)))
	query = strings.ReplaceAll(query, "{RowName}", "TelegramAccount, UserAccount, StaffType")
	fmt.Printf("query: %v\n", query)
	rows := sqlMod.GetInstance().Query(query)
	for rows.Next() {
		var contact telegramContact
		rows.Scan(&contact.telegramAccount, &contact.sysAccount, &contact.stuffType)
		Bot.contact = append(Bot.contact, contact)
	}
}

//傳送 telegram bot message
func (Bot *telegramBot) Broadcast(msg string, stuffType loginMod.StaffType) {

	for _, contact := range Bot.contact {
		if contact.stuffType == stuffType {
			msg += ("@" + contact.telegramAccount + " ")
		}
	}
	chatId, _ := strconv.ParseInt(fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.TELEGRAM_BOT_CHAT_ID)), 10, 64)
	telegramMsg := tgbotapi.NewMessage(chatId, msg)
	telegramMsg.ParseMode = tgbotapi.ModeMarkdown
	_, err := Bot.bot.Send(telegramMsg)
	if err == nil {
		fmt.Printf("Send telegram message success \n")
	} else {
		fmt.Printf("Send telegram message error %v \n", err)
	}
}
