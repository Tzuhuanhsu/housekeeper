package loginMod

import (
	"fmt"
	"housekeepr/settingMod"
	"housekeepr/sqlMod"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt"
	"gopkg.in/olahol/melody.v1"
)

type LoginMod struct {
	accountWithData accountWithInfoMap
}

type accountWithInfoMap map[string]userStruct
type StaffType int
type userStruct struct {
	account   string
	password  string
	staffType StaffType
	token     string
	session   *melody.Session
	time      time.Time
}

var (
	loginModInstance *LoginMod
	once             = &sync.Once{}
)

const (
	UNKNOWN        StaffType = -1
	BOSS           StaffType = 0
	JANITOR        StaffType = 2
	TOKEN_TIME_OUT           = time.Minute * 10
)

//get LoginMod instance
func GetInstance() *LoginMod {
	once.Do(func() {
		loginModInstance = new(LoginMod)
		loginModInstance.accountWithData = make(accountWithInfoMap, 0)
	})
	return loginModInstance
}

//建立 token
func (loginMod *LoginMod) CreateToken(account, password string) string {
	now := time.Now()
	key := []byte(fmt.Sprintf("%v%v%v", account, password, now.UnixNano()))
	t := jwt.New(jwt.SigningMethodHS256)

	s, err := t.SignedString(key)
	if err != nil {
		fmt.Println("loginMod CreateToken fail error:", err)
		return ""
	}
	return s
}

//取得帳號權限類別
func (loginMod *LoginMod) GetUserStaffType(account string) StaffType {
	user, ok := loginMod.accountWithData[account]
	if !ok {
		return UNKNOWN
	}
	fmt.Println("帳號權限", user.staffType)
	return user.staffType
}

//登入
func (loginMod *LoginMod) DoLogin(account, password string) bool {
	fmt.Println("Do Login", account, password)
	user := &userStruct{
		account:  account,
		password: password}
	isSuccess := loginMod.queryUser(account, password, user)
	if isSuccess {
		if existUser, exist := loginMod.accountWithData[account]; exist {
			//強制斷線已經存在的使用者
			fmt.Println("重複登入，剔除前一位使用者", existUser.session)
			if existUser.session != nil {
				existUser.session.CloseWithMsg([]byte("帳號重複登入"))
			}
		}
		token := loginMod.CreateToken(account, password)
		user.token = token
		user.time = time.Now()
		loginMod.accountWithData[account] = *user
	}
	fmt.Println("Do Login", isSuccess)
	return isSuccess
}

//登出
func (loginMod *LoginMod) DoLogout(account string) {
	if existUser, exist := loginMod.accountWithData[account]; exist {
		if !existUser.session.IsClosed() {
			existUser.session.Close()
		}
		delete(loginMod.accountWithData, account)
	}
}

//取得 user token
func (loginMod *LoginMod) GetUserToken(account string) (string, bool) {
	user, ok := loginMod.accountWithData[account]
	return user.token, ok
}

//檢查token
func (loginMod *LoginMod) CheckUserToken(account, token string) (string, bool) {
	user, ok := loginMod.accountWithData[account]
	if ok {
		if strings.Compare(strconv.Quote(user.token), token) == 0 {
			return "success", true
		} else {
			fmt.Println("server token with token not match")
			return "server token with token not match", false
		}
	} else {
		return "server token not exist", false
	}
}

//set websocket session to user
func (loginMod *LoginMod) SetWebSocketSession(account string, session *melody.Session) {
	fmt.Printf("SetWebSocketSession account:%v ,session:%v \n", account, session)
	user, ok := loginModInstance.accountWithData[account]
	if ok {
		user.session = session
		loginMod.accountWithData[account] = user
	} else {
		fmt.Println("無效的使用者")
	}
}

func (loginMod *LoginMod) CheckUserTokenLive() {
	now := time.Now()
	for _, user := range loginModInstance.accountWithData {
		subTime := now.Sub(user.time)
		if subTime >= TOKEN_TIME_OUT {
			fmt.Printf("User:(%v) 登入時效超過，自動登出", user.account)
			user.session.CloseWithMsg([]byte("登入時效超過，自動登出"))
			loginModInstance.DoLogout(user.account)
		}
	}
}

//db查詢使用者
func (loginMod *LoginMod) queryUser(account, password string, user *userStruct) bool {

	query := "SELECT {RowName} FROM {TableName} where UserAccount='%v' and  UserPassword='%v' "
	query = strings.ReplaceAll(query, "{TableName}", fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.DB_USER_TAB)))
	query = strings.ReplaceAll(query, "{RowName}", "UserAccount, UserPassword, StaffType")
	query = fmt.Sprintf(query, account, password)
	fmt.Printf("query: %v\n", query)
	rows := sqlMod.GetInstance().Query(query)
	for rows.Next() {
		if err := rows.Scan(&user.account, &user.password, &user.staffType); err != nil {
			fmt.Printf("映射使用者失敗，原因為：%v\n", err)
		}
	}
	if user.account != "" && user.password != "" {
		fmt.Println("查詢使用者成功", *user)
		return true
	}
	fmt.Println("查詢使用者失敗(無效使者)")
	return false
}
