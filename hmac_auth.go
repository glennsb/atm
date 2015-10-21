package atm

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo"
)

const AUTH_SEP = ":"
const XTIME = "X-Timestamp"
const CONTENT_TYPE = "Content-Type"
const CONTENT_MD5 = "Content-MD5"
const XNONC = "X-Nonce"
const API_KEY = "api-key"

type KeyFinder func(string) (string, error)

type HmacOpts struct {
	AuthPrefix   string
	Expiration   time.Duration
	SecretKeyFor KeyFinder
}

func NewHmacOpts(f KeyFinder) *HmacOpts {
	o := &HmacOpts{
		AuthPrefix:   "ATM_Auth",
		Expiration:   5 * time.Minute,
		SecretKeyFor: f,
	}
	return o
}

func HMACAuth(o *HmacOpts) echo.HandlerFunc {
	return func(c *echo.Context) error {
		auth, err := newAuth(c.Request(), o)
		if nil != err {
			return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
		}
		if err := auth.Authentic(c.Request()); nil != err {
			return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
		}
		c.Set(API_KEY, auth.ApiKey)
		return nil
	}
}

type Authorizor struct {
	ApiKey    string
	Signature string
	Md5       string
	Type      string
	Nonce     string
	Xtime     string
	Method    string
	Uri       string
	Timestamp time.Time
	Opts      *HmacOpts
}

type hmacError struct {
	msg string
}

func (e hmacError) Error() string {
	return e.msg
}

func newAuth(r *http.Request, o *HmacOpts) (*Authorizor, error) {
	h := r.Header
	if len(h.Get(echo.Authorization)) <= 0 {
		return nil, hmacError{"Missing required Authorization header"}
	}
	a := &Authorizor{
		Opts:   o,
		Method: strings.ToUpper(r.Method),
		Uri:    r.URL.RequestURI(),
	}
	err := a.authorizorFromAuthHeader(h.Get(echo.Authorization))
	if nil != err {
		return nil, err
	}
	err = a.extractRequiredHeaders(&h)
	if nil != err {
		return nil, err
	}
	return a, nil
}

func (a *Authorizor) Authentic(r *http.Request) error {
	if err := a.invalidTime(); nil != err {
		return err
	}
	body := readerToByte(r.Body)
	r.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	contentHash := md5Of(body)
	if a.Md5 != contentHash {
		return hmacError{fmt.Sprintf("Content MD5s do not match:%s", contentHash)}
	}
	secret, err := a.Opts.SecretKeyFor(a.ApiKey)
	if nil != err {
		return err
	}
	if "" == secret {
		return hmacError{fmt.Sprintf("No secret key for %s", a.ApiKey)}
	}
	generatedSig := a.signatureWith(secret)
	if "" == generatedSig {
		return hmacError{"Unable to generate hmac signature"}
	}
	if hmac.Equal([]byte(generatedSig), []byte(a.Signature)) {
		return nil
	}
	return hmacError{"HMAC Signature mismatch"}
}

func md5Of(b []byte) string {
	h := md5.New()
	h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}

func readerToByte(s io.Reader) []byte {
	b, _ := ioutil.ReadAll(s)
	return b
}

func (a *Authorizor) invalidTime() error {
	age := time.Since(a.Timestamp)
	if age < 0 {
		age *= -1
	}
	if age > a.Opts.Expiration {
		return hmacError{"Timestamp out of range"}
	}
	return nil
}

func (a *Authorizor) signatureWith(key string) string {
	c := a.SigningString()
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(c))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

//Build the content for signging & sign it with the key
//method\nURI\nMD5\ntype\ntimestamp\nnonce
func (a *Authorizor) SigningString() string {
	var b bytes.Buffer
	b.WriteString(a.Method)
	b.WriteString("\n")
	b.WriteString(a.Uri)
	b.WriteString("\n")
	b.WriteString(a.Md5)
	b.WriteString("\n")
	b.WriteString(a.Type)
	b.WriteString("\n")
	b.WriteString(a.Xtime)
	b.WriteString("\n")
	b.WriteString(a.Nonce)
	b.WriteString("\n")
	b.WriteString(a.ApiKey)
	b.WriteString("\n")

	return b.String()
}

// We need x-timestamp, content-md5, content-type, x-nonce
func (a *Authorizor) extractRequiredHeaders(h *http.Header) error {
	a.Md5 = h.Get(CONTENT_MD5)
	if "" == a.Md5 {
		return hmacError{fmt.Sprintf("Missing required %s header", CONTENT_MD5)}
	}
	a.Type = h.Get(CONTENT_TYPE)
	if "" == a.Type {
		return hmacError{fmt.Sprintf("Missing required %s header", CONTENT_TYPE)}
	}
	a.Nonce = h.Get(XNONC)
	if "" == a.Nonce {
		return hmacError{fmt.Sprintf("Missing required %s header", XNONC)}
	}
	return a.parseTimestamp(h.Get(XTIME))
}

func (a *Authorizor) parseTimestamp(t string) error {
	if "" == t {
		return hmacError{fmt.Sprintf("Missing required %s header", XTIME)}
	}
	a.Xtime = t
	ts, err := time.Parse(time.RFC3339, t)
	if nil != err {
		return hmacError{fmt.Sprintf("Invalid timestamp: %s", err.Error())}
	}
	a.Timestamp = ts
	return nil
}

func (a *Authorizor) authorizorFromAuthHeader(h string) error {
	l := len(a.Opts.AuthPrefix)
	if len(h)-l-1 <= 0 || h[:l] != a.Opts.AuthPrefix {
		return hmacError{fmt.Sprintf("Authorization header invalid format: Missing %s prefix", a.Opts.AuthPrefix)}
	}
	parts := strings.SplitN(strings.Trim(h[l+1:], " "), AUTH_SEP, 2)
	if len(parts) != 2 {
		return hmacError{"Authorization header invalid format: Missing ApiKey:Signature"}
	}
	a.ApiKey = parts[0]
	if "" == a.ApiKey {
		return hmacError{"Authorization header invalid format: Missing ApiKey"}
	}
	a.Signature = parts[1]
	if "" == a.Signature {
		return hmacError{"Authorization header invalid format: Missing Signature"}
	}
	return nil
}
