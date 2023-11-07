package sqlMod

import (
	"database/sql"
	"fmt"
	"housekeepr/settingMod"
	"sync"

	_ "github.com/go-sql-driver/mysql"

	"github.com/mitchellh/mapstructure"
)

type SqlMod struct {
	userData sqlUserData
}

type sqlUserData struct {
	USER     string `mapstructure:"user"`
	PWD      string `mapstructure:"pwd"`
	NETWORK  string `mapstructure:"network"`
	HOST     string `mapstructure:"host"`
	PORT     int    `mapstructure:"port"`
	DATABASE string `mapstructure:"database"`
}

var (
	SqlModInstance *SqlMod
	once           = &sync.Once{}
)

func GetInstance() *SqlMod {
	once.Do(func() {
		SqlModInstance = new(SqlMod)
	})
	return SqlModInstance
}

func init() {
	GetInstance().initSqlMod()
}

func (s *SqlMod) initSqlMod() {
	//取得 sql server connect data
	sqlServerData := settingMod.GetInstance().GetVal(settingMod.DB_DATA)
	mapstructure.Decode(sqlServerData, &GetInstance().userData)

}

func (s *SqlMod) connectDB() (bool, *sql.DB) {
	conn := fmt.Sprintf("%s:%s@%s(%s:%d)/%s", GetInstance().userData.USER,
		GetInstance().userData.PWD,
		GetInstance().userData.NETWORK,
		GetInstance().userData.HOST,
		GetInstance().userData.PORT,
		GetInstance().userData.DATABASE)
	fmt.Println("Connect conn:", conn)
	db, err := sql.Open("mysql", conn)

	if err != nil {
		fmt.Println("開啟 MySQL 連線發生錯誤，原因為：", err)
		return false, nil
	}
	if err := db.Ping(); err != nil {
		fmt.Println("資料庫連線錯誤，原因為：", err.Error())
		return false, nil
	}
	fmt.Println("DB connect success")
	return true, db
}

func (s *SqlMod) Test() {
	fmt.Print("sql Mode test")
}

type RoomOrder struct {
	OrderId        string `json:"OrderId"`
	CheckInData    string `json:"CheckInData"`
	CheckOutData   string `json:"CheckOutData"`
	NumberOfPeople int    `json:"NumberOfPeople"`
	Cost           int    `json:"Cost"`
	RoomExplain    string `json:"RoomExplain"`
	Paid           bool   `json:"Paid"`
}

//查詢 sql 語法
func (s *SqlMod) Query(querySql string) *sql.Rows {

	if success, db := GetInstance().connectDB(); success {
		defer db.Close()
		rows, err := db.Query(querySql)
		if err != nil {
			fmt.Printf("Query fail reason :%v", err)
			return nil
		}
		return rows
	}
	return nil
}

//執行 sql 語法
func (s *SqlMod) Exec(exec string) {
	if success, db := GetInstance().connectDB(); success {
		defer db.Close()
		_, err := db.Exec(exec)
		if err != nil {
			fmt.Printf("exec fail reason :%v", err)
			return

		}
		fmt.Println("exec success")
	}
}
