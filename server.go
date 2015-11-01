package atm

import (
	"log"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo"
	mw "github.com/labstack/echo/middleware"
)

const (
	HOST     = "https://o3.omrf.org"
	DURATION = 5 * time.Minute
)

type Server struct {
	Ds               *Datastore
	Object_host      string
	Default_duration int64
	Nonces           NonceChecker
}

func (a *Server) Run() {
	e := echo.New()

	// Middleware
	e.Use(mw.Logger())
	e.Use(mw.Recover())
	auth_opts := NewHmacOpts(a.Ds.ApiKeySecret, a.Nonces)
	e.Use(HMACAuth(auth_opts))

	v1 := e.Group("/v1")
	v1.Post("/urls", a.createUrl)
	v1.Put("/keys/:name", a.setKey)
	v1.Delete("/keys/:name", a.removeKey)

	e.Run(":8080")
}

type keyRequest struct {
	Key string `json:key`
}

func (s *Server) removeKey(c *echo.Context) error {
	a, err := s.Ds.Account(c.Param("name"))
	if nil != err || a.Id == "" {
		return c.JSON(http.StatusGone, ErrMsg(http.StatusText(http.StatusNotFound)))
	}
	if c.Get(API_KEY) != a.Id {
		return c.JSON(http.StatusForbidden, ErrMsg("Not authorized for this account"))
	}
	s.Ds.RemoveSigningKeyForAccount(a.Id)
	return c.JSON(http.StatusNoContent, a)
}

func (s *Server) setKey(c *echo.Context) error {
	k := &keyRequest{}
	if err := c.Bind(k); nil != err {
		return c.JSON(http.StatusBadRequest, ErrMsg(err.Error()))
	}
	a, err := s.Ds.Account(c.Param("name"))
	if nil != err || a.Id == "" {
		return c.JSON(http.StatusGone, ErrMsg(http.StatusText(http.StatusNotFound)))
	}
	if c.Get(API_KEY) != a.Id {
		return c.JSON(http.StatusForbidden, ErrMsg("Not authorized for this account"))
	}
	s.Ds.AddSigningKeyForAccount(k.Key, a.Id)
	return c.JSON(http.StatusOK, a)
}

func (s *Server) createUrl(c *echo.Context) error {
	o := &UrlRequest{Host: s.Object_host, Duration: s.Default_duration}
	if err := c.Bind(o); nil != err {
		return c.JSON(http.StatusBadRequest, ErrMsg(err.Error()))
	}

	if !o.Valid() {
		return c.JSON(http.StatusBadRequest, ErrMsg("Missing account, container, object, or method"))
	}

	duration := int64(0)
	var err error
	requestorId, ok := c.Get(API_KEY).(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, ErrMsg("Failed getting requesting id"))
	}
	o.Key, duration, err = s.Ds.KeyForRequest(o, requestorId)
	if nil != err {
		log.Printf("keyForRequest: %v, %s. Error: %s", o, "", err.Error())
		return c.JSON(http.StatusInternalServerError, ErrMsg("Trouble checking authorization"))
	}
	if "" == o.Key {
		return c.JSON(http.StatusForbidden, ErrMsg("Not authorized for this resource"))
	}
	if duration > 0 && duration > o.Duration {
		o.Duration = duration
	}

	u := &Tmpurl{
		Url:  o.SignedUrl(),
		Path: o.Path(),
	}

	c.Response().Header().Set("Location", u.Url)
	return c.JSON(http.StatusCreated, u)
}
