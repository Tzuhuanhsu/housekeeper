package keymgrmod

import (
	"fmt"
	"housekeepr/loginMod"
	"housekeepr/settingMod"
	"housekeepr/sqlMod"
	"housekeepr/telegramBot"
	"strings"
	"sync"
	"time"
)

type KeyMgrMode struct {
	doorKeyData map[string]DoorKeyData
	roomKeyData map[string]RoomKeyData
	roomSetting map[int]string
}

var (
	keyMgrModeInstance *KeyMgrMode
	once               = &sync.Once{}
)

// 大門金鑰資料
type DoorKeyData struct {
	Id        string
	DoorKey   string
	BeginTime string
	EndTime   string
}

type POSTDoorKeyData struct {
	Id        string `json:"Id"`
	DoorKey   string `json:"DoorKey"`
	BeginTime string `json:"BeginTime"`
	EndTime   string `json:"EndTime"`
	Account   string `json:"Account"`
	Token     string `json:"Token"`
}

// 房間金鑰資料
type RoomKeyData struct {
	Id        string
	RoomKey   string
	BeginTime string
	EndTime   string
	RoomType  int
}

type POSTRoomKeyData struct {
	Id        string `json:"Id"`
	RoomKey   string `json:"RoomKey"`
	BeginTime string `json:"BeginTime"`
	EndTime   string `json:"EndTime"`
	RoomType  int    `json:"RoomType"`
	Account   string `json:"Account"`
	Token     string `json:"Token"`
}

// 房間型別設定
type RoomTypeSetting struct {
	RoomType int
	RoomName string
}

//房間型別
type RoomType int

const (
	RoomUnknown RoomType = iota
	//二樓兩人房
	TwoTwo RoomType = 1
	//二樓四人房
	TwoFour RoomType = 2
	//三樓兩人房(1)
	ThreeTwoOne RoomType = 3
	//三樓兩人房(2)
	ThreeTwoTwo RoomType = 4
)

func GetInstance() *KeyMgrMode {
	once.Do(func() {
		keyMgrModeInstance = new(KeyMgrMode)
		keyMgrModeInstance.doorKeyData = make(map[string]DoorKeyData)
		keyMgrModeInstance.roomKeyData = make(map[string]RoomKeyData)
		keyMgrModeInstance.roomSetting = make(map[int]string)

	})
	return keyMgrModeInstance
}

func (mgr *KeyMgrMode) Init() {
	TimeFormatForm := "2006-01-02"
	now := time.Now()
	mgr.queryRoomType()
	mgr.queryDoorKeyInfo(now.Local().Format(TimeFormatForm))
	mgr.queryRoomKeyInfo(now.Local().Format(TimeFormatForm))
}

//查訊目前的大門金鑰資料
func (mgr *KeyMgrMode) queryDoorKeyInfo(firstDate string) {
	query := "SELECT {RowName} FROM {TableName}"
	query = strings.ReplaceAll(query, "{TableName}", fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.DB_DOOR_KEY_INFO_TAB)))
	query = strings.ReplaceAll(query, "{RowName}", "C_Id, DoorKey, BeginTime, EndTime")
	query = query + fmt.Sprintf(" where EndTime>='%v'", firstDate)
	fmt.Printf("query %v\n", query)
	rows := sqlMod.GetInstance().Query(query)
	for rows.Next() {
		var doorKeyData DoorKeyData
		rows.Scan(&doorKeyData.Id, &doorKeyData.DoorKey, &doorKeyData.BeginTime, &doorKeyData.EndTime)
		mgr.doorKeyData[doorKeyData.Id] = doorKeyData
	}
}

// 查詢房間鑰匙資料
func (mgr *KeyMgrMode) queryRoomKeyInfo(firstDate string) {
	query := "SELECT {RowName} FROM {TableName}"
	query = strings.ReplaceAll(query, "{TableName}", fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.DB_ROOM_KEY_INFO_TAB)))
	query = strings.ReplaceAll(query, "{RowName}", "C_Id, RoomKey, BeginTime, EndTime, RoomType")
	query = query + fmt.Sprintf(" where EndTime>='%v'", firstDate)
	fmt.Printf("query %v\n", query)
	rows := sqlMod.GetInstance().Query(query)
	for rows.Next() {
		var roomKeyData RoomKeyData
		rows.Scan(&roomKeyData.Id, &roomKeyData.RoomKey, &roomKeyData.BeginTime, &roomKeyData.EndTime, &roomKeyData.RoomType)
		mgr.roomKeyData[roomKeyData.Id] = roomKeyData
	}
}

//查詢房號設定
func (mgr *KeyMgrMode) queryRoomType() {
	query := "SELECT {RowName} FROM {TableName} "
	query = strings.ReplaceAll(query, "{TableName}", fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.DB_ROOM_TYPE_TAB)))
	query = strings.ReplaceAll(query, "{RowName}", "RoomType, RoomName")
	fmt.Printf("query: %v\n", query)
	rows := sqlMod.GetInstance().Query(query)
	for rows.Next() {
		var roomSetting RoomTypeSetting
		rows.Scan(&roomSetting.RoomType, &roomSetting.RoomName)
		mgr.roomSetting[roomSetting.RoomType] = roomSetting.RoomName
	}
}

//取得房間金鑰資料
func (mgr *KeyMgrMode) GetRoomKeyData() map[string]RoomKeyData {
	return mgr.roomKeyData
}

//取得大門金鑰資料
func (mgr *KeyMgrMode) GetDoorKeyData() map[string]DoorKeyData {
	return mgr.doorKeyData
}

//取得房間設定
func (mgr *KeyMgrMode) GetRoomSetting() map[int]string {
	return mgr.roomSetting
}

//刪除大門金鑰
func (mgr *KeyMgrMode) DeleteDoorKey(id string) bool {
	_, ok := mgr.doorKeyData[id]
	if ok {
		telegramBot.GetInstance().Broadcast(fmt.Sprintf("刪除大門金鑰ID:%v,Key:%v)",
			id, mgr.doorKeyData[id].DoorKey), loginMod.BOSS)
		mgr.deleteDoorKeyToDB(id)
		delete(mgr.doorKeyData, id)
		return true
	} else {
		return false
	}
}

//新增大門金鑰
func (mgr *KeyMgrMode) AddDoorKey(data POSTDoorKeyData) (string, bool) {
	_, exist := mgr.doorKeyData[data.Id]
	if exist {
		fmt.Printf("Add door key %v exist", data.Id)
		return fmt.Sprintf("Key:%v exist", data.Id), false
	} else {
		var doorKeyData DoorKeyData
		doorKeyData.Id = data.Id
		doorKeyData.DoorKey = data.DoorKey
		doorKeyData.BeginTime = data.BeginTime
		doorKeyData.EndTime = data.EndTime
		mgr.doorKeyData[data.Id] = doorKeyData
		mgr.insertDoorKeyToDB(data.Id)
		telegramBot.GetInstance().Broadcast(fmt.Sprintf("設定大門金鑰ID:%v,Key:%v,(%v ~ %v)",
			data.Id, data.DoorKey, data.BeginTime, data.EndTime), loginMod.BOSS)
		return "Success", true
	}
}

// 刪除房間金鑰
func (mgr *KeyMgrMode) DeleteRoomKey(id string) bool {
	_, ok := mgr.roomKeyData[id]
	if ok {
		telegramBot.GetInstance().Broadcast(fmt.Sprintf("刪除房間(%v)金鑰ID:%v,Key:%v)",
			mgr.roomSetting[mgr.roomKeyData[id].RoomType],
			id,
			mgr.roomKeyData[id].RoomKey),
			loginMod.BOSS)
		mgr.deleteRoomKeyToDB(id)
		delete(mgr.roomKeyData, id)
		return true
	} else {
		return false
	}
}

//新增房間金鑰
func (mgr *KeyMgrMode) AddRoomKey(data POSTRoomKeyData) (string, bool) {
	_, exist := mgr.roomKeyData[data.Id]
	if exist {
		fmt.Printf("Add room key %v exist", data.Id)
		return fmt.Sprintf("Key:%v exist", data.Id), false
	} else {
		var roomKeyData RoomKeyData
		roomKeyData.Id = data.Id
		roomKeyData.RoomKey = data.RoomKey
		roomKeyData.BeginTime = data.BeginTime
		roomKeyData.EndTime = data.EndTime
		roomKeyData.RoomType = data.RoomType
		mgr.roomKeyData[data.Id] = roomKeyData
		mgr.insertRoomKeyToDB(data.Id)
		telegramBot.GetInstance().Broadcast(fmt.Sprintf("設定房間(%v)金鑰ID:%v,Key:%v,(%v ~ %v)",
			mgr.roomSetting[data.RoomType],
			data.Id, data.RoomKey,
			data.BeginTime,
			data.EndTime),
			loginMod.BOSS)
		return "Success", true
	}
}

//刪除DB的大門金鑰
func (mgr *KeyMgrMode) deleteDoorKeyToDB(id string) {
	delete := "DELETE FROM {TableName} WHERE C_Id='%v'"
	delete = strings.ReplaceAll(delete, "{TableName}", fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.DB_DOOR_KEY_INFO_TAB)))
	delete = fmt.Sprintf(delete, id)

	fmt.Printf("deleteDoorKeyToDB: %v\n", delete)
	sqlMod.GetInstance().Exec(delete)
}

// 新增大門金鑰到DB
func (mgr *KeyMgrMode) insertDoorKeyToDB(id string) {
	doorKeyData, found := mgr.doorKeyData[id]
	if found {
		insert := "INSERT INTO {TableName} ({RowName}) VALUES({Values})"
		insert = strings.ReplaceAll(insert, "{TableName}", fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.DB_DOOR_KEY_INFO_TAB)))
		insert = strings.ReplaceAll(insert, "{RowName}", "C_id, DoorKey, BeginTime, EndTime")
		insert = strings.ReplaceAll(insert, "{Values}", fmt.Sprintf("'%v', '%v', '%v', '%v'",
			doorKeyData.Id,
			doorKeyData.DoorKey,
			doorKeyData.BeginTime,
			doorKeyData.EndTime))

		fmt.Println("sql:", insert)
		sqlMod.GetInstance().Exec(insert)
	}

}

//刪除DB的房間金鑰
func (mgr *KeyMgrMode) deleteRoomKeyToDB(id string) {
	delete := "DELETE FROM {TableName} WHERE C_Id='%v'"
	delete = strings.ReplaceAll(delete, "{TableName}", fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.DB_ROOM_KEY_INFO_TAB)))
	delete = fmt.Sprintf(delete, id)

	fmt.Printf("deleteRoomKeyToDB: %v\n", delete)

	sqlMod.GetInstance().Exec(delete)
}

// 新增房間金鑰到DB
func (mgr *KeyMgrMode) insertRoomKeyToDB(id string) {
	roomKeyData, found := mgr.roomKeyData[id]
	if found {
		insert := "INSERT INTO {TableName} ({RowName}) VALUES({Values})"
		insert = strings.ReplaceAll(insert, "{TableName}", fmt.Sprintf("%v", settingMod.GetInstance().GetVal(settingMod.DB_ROOM_KEY_INFO_TAB)))
		insert = strings.ReplaceAll(insert, "{RowName}", "C_id, RoomKey, BeginTime, EndTime, RoomType")
		insert = strings.ReplaceAll(insert, "{Values}", fmt.Sprintf("'%v', '%v', '%v', '%v', %v",
			roomKeyData.Id,
			roomKeyData.RoomKey,
			roomKeyData.BeginTime,
			roomKeyData.EndTime,
			roomKeyData.RoomType))

		fmt.Println("sql:", insert)
		sqlMod.GetInstance().Exec(insert)
	}

}
