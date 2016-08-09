// ATM - Automatic TempUrl Maker
// Copyright (c) 2015 Stuart Glenn
// All rights reserved
// Use of this source code is goverened by a BSD 3-clause license,
// see included LICENSE file for details
// Contains the main logic behind making TempUrls
package atm

import (
	"crypto/hmac"
	"crypto/sha1"
	"fmt"
	"strings"
	"time"
)

type Tmpurl struct {
	Url  string `json:"url"`
	Path string `json:"path"`
}

type UrlRequest struct {
	Account   string `json:account`
	Container string `json:container`
	Object    string `json:object`
	Method    string `json:method`
	Key       string `json:"-"`
	Host      string `json:"-"`
	Duration  int64  `json:"duration"`
}

func (u *UrlRequest) Valid() bool {
	return "" != u.Account &&
		"" != u.Container &&
		"" != u.Object &&
		"" != u.Method &&
		u.Duration > 0
}

func (u *UrlRequest) Path() string {
	return fmt.Sprintf("/v1/%s/%s/%s", u.Account, u.Container, u.Object)
}

func (u *UrlRequest) signature(expires int64) string {
	h := hmac.New(sha1.New, []byte(u.Key))
	message := fmt.Sprintf("%s\n%d\n%s", strings.ToUpper(u.Method), expires, u.Path())
	h.Write([]byte(message))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (u *UrlRequest) SignedUrl() string {
	expires := time.Now().UTC().Unix() + u.Duration
	return fmt.Sprintf("%s%s?temp_url_sig=%s&temp_url_expires=%d", u.Host, u.Path(),
		u.signature(expires), expires)
}

func ErrMsg(msg string) map[string]string {
	return map[string]string{"error": msg}
}
