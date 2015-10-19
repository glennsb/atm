//ATM - Automatic TempUrl Maker
//A builder of Swift TempURLs
package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo"
	mw "github.com/labstack/echo/middleware"
)

const (
	HOST     = "https://o3.omrf.org"
	DURATION = int64(300)
)

type tmpurl struct {
	Url  string `json:"url"`
	Path string `json:"path"`
}

type urlRequest struct {
	Account   string `json:account`
	Container string `json:container`
	Object    string `json:object`
	Method    string `json:method`
}

func (u *urlRequest) Valid() bool {
	return "" != u.Account &&
		"" != u.Container &&
		"" != u.Object &&
		"" != u.Method
}

func (u *urlRequest) Path() string {
	return fmt.Sprintf("/v1/%s/%s/%s", u.Account, u.Container, u.Object)
}

func (u *urlRequest) signature(expires int64) string {
	h := hmac.New(sha1.New, []byte(Key))
	message := fmt.Sprintf("%s\n%d\n%s", strings.ToUpper(u.Method), expires, u.Path())
	h.Write([]byte(message))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (u *urlRequest) SignedUrl() string {
	expires := time.Now().UTC().Unix() + DURATION
	return fmt.Sprintf("%s%s?temp_url_sig=%s&temp_url_expires=%d", HOST, u.Path(), u.signature(expires), expires)
}

func errMsg(msg string) map[string]string {
	return map[string]string{"error": msg}
}

var Key string

func main() {
	Key = os.Args[1]

	e := echo.New()

	// Middleware
	e.Use(mw.Logger())
	e.Use(mw.Recover())

	e.Post("/urls", createUrl)
	e.Run(":8080")
}

func createUrl(c *echo.Context) error {
	o := &urlRequest{}
	if err := c.Bind(o); nil != err {
		return c.JSON(http.StatusBadRequest, errMsg(err.Error()))
	}

	if !o.Valid() {
		return c.JSON(http.StatusBadRequest, errMsg("Missing account, container, object, or method"))
	}
	u := &tmpurl{
		Url:  o.SignedUrl(),
		Path: o.Path(),
	}

	c.Response().Header().Set("Location", u.Url)
	return c.JSON(http.StatusCreated, u)
}
