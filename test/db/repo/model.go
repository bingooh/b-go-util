package repo

type TemplateModel struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Age       int    `json:"age"`
	IsSuper   bool   `json:"is_super" gorm:"type:tinyint"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}
