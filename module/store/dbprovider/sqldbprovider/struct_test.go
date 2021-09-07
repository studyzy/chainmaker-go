package sqldbprovider

type User struct {
	ID   string // 字段名 `ID` 将被作为默认的主键名
	age int64
}

func (User) TableName() string {
	return "test_user"
}