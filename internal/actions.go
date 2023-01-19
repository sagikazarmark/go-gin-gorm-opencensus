package internal

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/go-gin-gorm-opencensus/pkg/ocgorm"
	"github.com/jinzhu/gorm"
)

type NewPerson struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func CreatePerson(db *gorm.DB) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		var newPerson NewPerson

		err := c.BindJSON(&newPerson)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, &gin.Error{Err: err})

			return
		}

		person := Person{
			FirstName: newPerson.FirstName,
			LastName:  newPerson.LastName,
		}

		orm := ocgorm.WithContext(c.Request.Context(), db)

		err = orm.Create(&person).Error
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, &gin.Error{Err: err})

			return
		}

		c.JSON(http.StatusOK, person)
	})
}

func Hello(db *gorm.DB) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		firstName := c.Param("firstName")

		person := Person{
			FirstName: firstName,
		}

		orm := ocgorm.WithContext(c.Request.Context(), db)

		err := orm.Where(person).First(&person).Error
		if gorm.IsRecordNotFoundError(err) {
			c.AbortWithStatusJSON(http.StatusNotFound, &gin.Error{Err: err})

			return
		} else if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, &gin.Error{Err: err})

			return
		}

		response := map[string]string{
			"message": fmt.Sprintf("Hello, %s %s!", person.FirstName, person.LastName),
		}

		c.JSON(http.StatusOK, response)
	})
}
