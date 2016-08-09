package atm

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/parnurzeal/gorequest"
)

type AtmClient struct {
	ApiKey    string
	ApiSecret string
	AtmHost   string
}

func (c *AtmClient) RequestTempUrl(method, account, container, object string,
	duration int64) (string, error) {

	uri := "/v1/urls"
	request := UrlRequest{
		Account:   account,
		Container: container,
		Object:    object,
		Method:    strings.ToUpper(method),
		Duration:  duration,
	}
	json, err := json.Marshal(request)
	if nil != err {
		return "", err
	}
	hopts := NewHmacOpts(func(s string) (string, error) { return "", nil }, nil)
	auth := AuthorizorForRequest(hopts, "POST", uri)
	auth.ApiKey = c.ApiKey
	auth.Md5 = md5Of(json)
	auth.Type = gorequest.Types["json"]
	auth.Xtime = time.Now().UTC().Format(time.RFC3339)
	auth.Nonce = fmt.Sprintf("%d", time.Now().UnixNano())

	api := gorequest.New()

	api = api.Post(c.AtmHost+uri).
		Type("json").
		Timeout(5*time.Second).
		Set(XTIME, auth.Xtime).
		Set(CONTENT_MD5, auth.Md5).
		Set(XNONCE, auth.Nonce).
		Set(API_KEY, auth.ApiKey).
		Set("Authorization", fmt.Sprintf("%s %s:%s", hopts.AuthPrefix, c.ApiKey,
			auth.SignatureWith(c.ApiSecret)))
	api.BounceToRawString = true
	resp, body, errs := api.Send(string(json)).End()
	if nil != errs || len(errs) > 0 {
		return "", errs[0]
	}
	if http.StatusCreated == resp.StatusCode {
		return resp.Header.Get("Location"), nil
	}
	return "", errors.New(body)
}
