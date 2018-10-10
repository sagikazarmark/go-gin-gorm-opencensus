package internal

import (
	"time"

	"github.com/jinzhu/gorm"
)

type Person struct {
	ID        uint      `gorm:"primary_key" json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func Fixtures(db *gorm.DB) error {
	person := Person{
		FirstName: "John",
		LastName:  "Doe",
	}

	err := db.Where(person).First(&person).Error
	if gorm.IsRecordNotFoundError(err) {
		err = db.Create(&person).Error
	}
	if err != nil {
		return err
	}

	return nil
}
