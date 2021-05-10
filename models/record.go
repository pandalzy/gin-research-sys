package models

type Record struct {
	BaseModel
	Title      string `gorm:"size:32,not null" json:"title"`
	ResearchID string `gorm:"size:128;index" json:"researchID"`
	RecordID   string `gorm:"size:128,index" json:"recordID"`
	UserID     int    `json:"-"`
	User       User   `json:"user"`
}

type RecordMgo struct {
	FieldsValue map[string]interface{} `json:"fieldsValue" bson:"fieldsValue"`
}
