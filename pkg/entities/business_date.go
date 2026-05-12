package entities

/* 營業日期表 */
import "time"

type BusinessDate struct {
	ID           uint       `json:"id" gorm:"primary_key"`                                           // 流水碼
	BusinessDate string     `gorm:"column:business_date;type:varchar(8);index" json:"business_date"` // 營業日
	StartDate    time.Time  `gorm:"column:start_date;"`                                              // 開始日期
	EndDate      *time.Time `gorm:"column:end_date;"`                                                // 結束日期
	Flag         string     `gorm:"column:flag;type:varchar(1);default:0;" json:"flag"`              // 旗標(D.日結 M.月結)
}

func (m *BusinessDate) TableName() string {
	return "business_date"
}
