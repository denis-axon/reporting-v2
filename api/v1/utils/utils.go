package utils

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
)

type Response struct {
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

func ReturnErrorCode(c *gin.Context, statusCode int, err error) {
	c.JSON(statusCode, Response{Error: err.Error()})
}

func ReturnError(c *gin.Context, err error) {
	if errors.Is(err, gocql.ErrNotFound) {
		ReturnErrorCode(c, http.StatusNotFound, err)
	} else {
		ReturnErrorCode(c, http.StatusInternalServerError, err)
	}
}

func ReturnSuccessMessage(c *gin.Context, msg string) {
	c.JSON(
		http.StatusOK,
		Response{
			Data: map[string]string{
				"message": msg,
			},
		})
}

func ReturnSuccessData(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Data: data})
}

func UnmarshalJsonRequest(c *gin.Context, dest interface{}) error {
	defer c.Request.Body.Close()

	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()
	err := dec.Decode(dest)
	if err != nil {
		return err
	}
	return nil
}
