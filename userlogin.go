// Copyright 2015 Shannon Wynter. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package steamauth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"time"
)

// UserLogin lets you log in to steam as a user.
// Once you provide it with a username and a password
// you are free to try to DoLogin().
//
// This can be used on it's own without SteamGuardAccount
// provided you have the ability to access your email
// and parse the tokens
type UserLogin struct {
	Username string
	Password string
	SteamID  SteamID

	RequiresCaptcha bool
	CaptchaGID      CaptchaGID
	CaptchaText     string

	RequiresEmail bool
	EmailDomain   string
	EmailCode     string

	Requires2FA   bool
	TwoFactorCode string

	Session  *SessionData
	LoggedIn bool

	cookieJar *cookiejar.Jar
}

// NewUserLogin allocates and returns a new UserLogin.
func NewUserLogin(username, password string) *UserLogin {
	cookieJar, _ := cookiejar.New(&cookiejar.Options{})
	return &UserLogin{
		cookieJar: cookieJar,
		Session:   &SessionData{},
		Username:  username,
		Password:  password}
}

// DoLogin actually attempt to login.
// Creates a fresh session if required.
// Grabs the RSA public key.
// Encrypts your password.
// Attempts to authenticate.
func (u *UserLogin) DoLogin() (LoginResult, error) {
	postData := url.Values{}
	cookieJar := u.cookieJar

	if len(cookieJar.Cookies(APIEndpoints.CommunityBase)) == 0 {
		log("Creating new 'empty' sesson")
		u.Session.SetCookies(cookieJar)

		SteamWeb().
			SetJar(cookieJar).
			AddHeader("X-Requested-With", "com.valvesoftware.android.steam.community").
			Get(APIEndpoints.CommunityBase.String() + "/login?oauth_client_id=DE45CD61&oauth_scope=read_profile%20write_profile%20read_client%20write_client").
			MobileLoginRequest()
	}

	logf("Retriving RSAKey for %s", u.Username)

	postData.Set("username", u.Username)
	rsaResponse := rsaResponse{}
	_, err := SteamWeb().
		SetJar(cookieJar).
		SetParams(postData).
		Post(APIEndpoints.CommunityBase.String() + "/login/getrsakey").
		HandleJSON(&rsaResponse).
		MobileLoginRequest()

	if err != nil {
		return LoginGeneralFailure, err
	}

	encryptedPassword, err := rsa.EncryptPKCS1v15(rand.Reader, rsaResponse.PublicKey, []byte(u.Password))
	if err != nil {
		return BadRSA, err
	}

	postData.Set("password", base64.StdEncoding.EncodeToString(encryptedPassword))

	postData.Set("twofactorcode", u.TwoFactorCode)

	postData.Set("captchagid", iif(u.RequiresCaptcha, u.CaptchaGID.String(), "-1"))
	postData.Set("captcha_text", iif(u.RequiresCaptcha, u.CaptchaText, ""))

	postData.Set("emailsteamid", iif(u.Requires2FA || u.RequiresEmail, u.SteamID.String(), ""))
	postData.Set("emailauth", iif(u.RequiresEmail, u.EmailCode, ""))

	postData.Set("rsatimestamp", rsaResponse.Timestamp.String())
	postData.Set("remember_login", "false")
	postData.Set("oauth_client_id", "DE45CD61")
	postData.Set("oauth_scope", "read_profile write_profile read_client write_client")
	postData.Set("loginfriendlyname", "#login_emailauth_friendlyname_mobile")
	postData.Set("donotcache", strconv.FormatInt(time.Now().Unix(), 10))

	logf("Attempting to authenticate as %s", u.Username)
	loginResponse := LoginResponse{}
	_, err = SteamWeb().
		SetJar(cookieJar).
		SetParams(postData).
		Post(APIEndpoints.CommunityBase.String() + "/login/dologin").
		HandleJSON(&loginResponse).
		MobileLoginRequest()

	if err != nil {
		logf("Protocol error %s", err)
		return LoginGeneralFailure, err
	}

	if loginResponse.CaptchaNeeded {
		log(NeedCaptcha)
		u.RequiresCaptcha = true
		u.CaptchaGID = loginResponse.CaptchaGID
		return NeedCaptcha, nil
	}

	if loginResponse.EmailAuthNeeded {
		log(NeedEmail)
		u.RequiresEmail = true
		u.SteamID = loginResponse.SteamID
		logf("Have steamid... it is %#v", u.SteamID)
		return NeedEmail, nil
	}

	if loginResponse.TwoFactorNeeded && !loginResponse.Success {
		log(Need2FA)
		u.Requires2FA = true
		return Need2FA, nil
	}

	if !loginResponse.LoginComplete {
		log(BadCredentials)
		if loginResponse.Message != "" {
			log(loginResponse)
			return BadCredentials, errors.New(loginResponse.Message)
		}
		return BadCredentials, nil
	}

	if loginResponse.OAuth == nil || len(loginResponse.OAuth.OAuthToken) == 0 {
		logf("Protocol error: missing oath")
		return LoginGeneralFailure, errors.New("missing oauth")
	}

	var sessionCookie *http.Cookie
	for _, cookie := range cookieJar.Cookies(APIEndpoints.CommunityBase) {
		if cookie.Name == "sessionid" {
			sessionCookie = cookie
			break
		}
	}
	if sessionCookie == nil {
		logf("Protocol error: missing session cookie")
		return LoginGeneralFailure, errors.New("missing session cookie")
	}

	u.Session = &SessionData{
		OAuthToken:       loginResponse.OAuth.OAuthToken,
		SteamID:          loginResponse.OAuth.SteamID,
		SteamLogin:       loginResponse.OAuth.SteamID.String() + "%7C%7C" + loginResponse.OAuth.SteamLogin,
		SteamLoginSecure: loginResponse.OAuth.SteamID.String() + "%7C%7C" + loginResponse.OAuth.SteamLoginSecure,
		WebCookie:        loginResponse.OAuth.Webcookie,
		SessionID:        sessionCookie.Value,
	}

	log(LoginOkay)
	return LoginOkay, nil
}

// CaptchaURL returns a fully qualified URL to a given captcha GID
func (u *UserLogin) CaptchaURL() string {
	return u.CaptchaGID.URL()
}

// LoginResponse represents the response sent from Steam servers.
type LoginResponse struct {
	Success         bool       `json:"success"`
	LoginComplete   bool       `json:"login_complete"`
	OAuth           *OAuth     `json:"oauth"`
	CaptchaNeeded   bool       `json:"captcha_needed"`
	CaptchaGID      CaptchaGID `json:"captcha_gid"`
	SteamID         SteamID    `json:"emailsteamid"`
	EmailAuthNeeded bool       `json:"emailauth_needed"`
	TwoFactorNeeded bool       `json:"requires_twofactor"`
	Message         string     `json:"message"`
}

// OAuth represents the embedded oauth object sent in the LoginResponse.
// This is actually sent as a string JSON blob but you don't have to
// deal with that, it's all handled for you.
type OAuth struct {
	SteamID          SteamID `json:"steamid"`
	OAuthToken       string  `json:"oauth_token"`
	SteamLogin       string  `json:"wgtoken"`
	SteamLoginSecure string  `json:"wgtoken_secure"`
	Webcookie        string  `json:"webcookie"`
}

func (o *OAuth) UnmarshalJSON(b []byte) (err error) {
	type localOAuth OAuth
	var (
		oauthObject = localOAuth{}
		jsonString  = ""
	)
	err = json.Unmarshal(b, &jsonString)
	if te, ok := err.(*json.UnmarshalTypeError); ok && te.Value == "object" {
		err = json.Unmarshal(b, &oauthObject)
	} else {
		err = json.Unmarshal([]byte(jsonString), &oauthObject)
	}

	if err == nil {
		*o = OAuth(oauthObject)
	}
	return
}

type rsaResponse struct {
	Success   bool `json:"success"`
	PublicKey *rsa.PublicKey
	Timestamp timestamp `json:"timestamp"`
	TokenGID  string    `json:"token_gid"`
	SteamID   SteamID   `json:"steamid"`
}

func (r *rsaResponse) UnmarshalJSON(b []byte) error {
	type localRSAResponse rsaResponse
	localData := struct {
		localRSAResponse
		Modulus  string `json:"publickey_mod"`
		Exponent string `json:"publickey_exp"`
	}{}

	err := json.Unmarshal(b, &localData)
	if err != nil {
		return err
	}
	*r = rsaResponse(localData.localRSAResponse)

	// Absolutly no point progressing beyond this point if there is no success
	if !r.Success {
		return nil
	}

	exponent, err := strconv.ParseInt(localData.Exponent, 16, 0)
	modulus := big.Int{}
	if _, ok := modulus.SetString(localData.Modulus, 16); !ok {
		return errors.New("invalid modulus")
	}

	r.PublicKey = &rsa.PublicKey{
		N: &modulus,
		E: int(exponent),
	}

	return nil
}

// LoginResult is the type of login result.
// Should probably be replaced with a bunch of errors?
type LoginResult int

// Various LoginResults.
// You can call .String() to get a human representation.
// LoginGeneralFailure is more protocol level and usually has an err also.
const (
	LoginOkay LoginResult = iota
	LoginGeneralFailure
	BadRSA
	BadCredentials
	NeedCaptcha
	Need2FA
	NeedEmail
)

var loginResponses = []string{
	LoginOkay:           "ok",
	LoginGeneralFailure: "general failure",
	BadRSA:              "bad rsa",
	BadCredentials:      "bad credentials",
	NeedCaptcha:         "need captcha",
	Need2FA:             "need two factor authentication",
	NeedEmail:           "need email verification",
}

func (l LoginResult) String() string {
	return loginResponses[l]
}
