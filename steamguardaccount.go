// Copyright 2015 Shannon Wynter. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package steamauth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

var (
	steamGuardCodeTranslations = []byte{50, 51, 52, 53, 54, 55, 56, 57, 66, 67, 68, 70, 71, 72, 74, 75, 77, 78, 80, 81, 82, 84, 86, 87, 88, 89}
	confIDRegex                = regexp.MustCompile(`data-confid="(\d+)"`)
	confKeyRegex               = regexp.MustCompile(`data-key="(\d+)"`)
	confDescRegex              = regexp.MustCompile(`<div>((Confirm|Trade with|Sell -) .+)</div>`)
)

// SteamGuardAccount is a structure to represent an authenticated
// account, you need to save/export this data or you risk losing
// access to your account
type SteamGuardAccount struct {
	SharedSecret   string       `json:"shared_secret"`
	SerialNumber   string       `json:"serial_number"`
	RevocationCode string       `json:"revocation_code"`
	URI            string       `json:"uri"`
	ServerTime     timestamp    `json:"server_time"`
	AccountName    string       `json:"account_name"`
	TokenGID       string       `json:"token_gid"`
	IdentitySecret string       `json:"identity_secret"`
	Secret1        string       `json:"secret_1"`
	Status         int          `json:"status"`
	DeviceID       string       `json:"device_id"`
	FullyEnrolled  bool         `json:"fully_enrolled"`
	Session        *SessionData `json:"session"`
}

// Export the account data as a json string
func (s *SteamGuardAccount) Export() (string, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(b), err
}

// Import account data from a json string
func (s *SteamGuardAccount) Import(jsonStr string) error {
	return json.Unmarshal([]byte(jsonStr), s)
}

// Save account data to an io.Writer as a json string
func (s *SteamGuardAccount) Save(w io.Writer) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(s)
}

// Load account data from an io.Reader json string
func (s *SteamGuardAccount) Load(r io.Reader) error {
	decoder := json.NewDecoder(r)
	return decoder.Decode(s)
}

// DeactivateAuthenticator will disable steamguard for this authenticator
func (s *SteamGuardAccount) DeactivateAuthenticator() bool {
	postData := url.Values{
		"steamid":           []string{s.Session.SteamID.String()},
		"steamguard_scheme": []string{"2"},
		"revocation_code":   []string{s.RevocationCode},
		"access_token":      []string{s.Session.OAuthToken},
	}

	log("Requestiong to remove this authenticator")
	removeResponse := RemoveAuthenticatorResponse{}
	_, err := SteamWeb().
		SetParams(postData).
		Post(APIEndpoints.SteamAPIBase.String() + "/ITwoFactorService/RemoveAuthenticator/v0001").
		HandleJSON(&removeResponse).
		MobileLoginRequest()

	if err != nil {
		logf("unhandled internal error: %s", err)
		return false
	}

	return removeResponse.Response.Success
}

// GenerateSteamGuardCode for the this account at this time
func (s *SteamGuardAccount) GenerateSteamGuardCode() string {
	return s.GenerateSteamGuardCodeForTime(TimeAligner.GetSteamTime())
}

// GenerateSteamGuardCodeForTime for the given time
func (s *SteamGuardAccount) GenerateSteamGuardCodeForTime(atTime time.Time) string {
	if s.SharedSecret == "" {
		return ""
	}

	sharedSecret, err := base64.StdEncoding.DecodeString(s.SharedSecret)
	if err != nil {
		logf("unhandled internal error: %s", err)
		return ""
	}

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, atTime.Unix()/30) // 30 should probably come from timeSyncResponse

	mac := hmac.New(sha1.New, sharedSecret)
	mac.Write(buf.Bytes())
	hashedData := mac.Sum(nil)

	codeBytes := make([]byte, 5)

	b := byte(hashedData[19] & 0xF)
	codePoint := int(hashedData[b]&0x7F)<<24 | int(hashedData[b+1]&0xFF)<<16 | int(hashedData[b+2]&0xFF)<<8 | int(hashedData[b+3]&0xFF)
	translationCount := len(steamGuardCodeTranslations)

	for i := 0; i < 5; i++ {
		codeBytes[i] = steamGuardCodeTranslations[codePoint%translationCount]
		codePoint /= translationCount
	}

	return string(codeBytes)
}

func (s *SteamGuardAccount) FetchConfirmations() []*Confirmation {
	urlStr := s.generateConfirmationURL("conf")
	cookieJar, _ := cookiejar.New(&cookiejar.Options{})
	s.Session.SetCookies(cookieJar)

	resp, _ := SteamWeb().
		SetJar(cookieJar).
		Get(urlStr).
		Do()

	defer resp.Body.Close()
	response, _ := ioutil.ReadAll(resp.Body)

	// Here the regex dragons are unleashed on the world
	if !(confIDRegex.Match(response) && confKeyRegex.Match(response) && confDescRegex.Match(response)) {
		return nil
	}

	confIDs := confIDRegex.FindAllSubmatch(response, -1)
	confKeys := confKeyRegex.FindAllSubmatch(response, -1)
	confDescs := confDescRegex.FindAllSubmatch(response, -1)

	ret := make([]*Confirmation, len(confIDs))

	for i := range confIDs {
		ret[i] = &Confirmation{
			ConfirmationDescription: string(confDescs[i][1]),
			ConfirmationID:          string(confIDs[i][1]),
			ConfirmationKey:         string(confKeys[i][1]),
		}
	}

	return ret
}

func (s *SteamGuardAccount) AcceptConfirmation(conf *Confirmation) bool {
	return s.sendConfirmationAjax(conf, "allow")
}

func (s *SteamGuardAccount) RejectConfirmation(conf *Confirmation) bool {
	return s.sendConfirmationAjax(conf, "cancel")
}

func (s *SteamGuardAccount) sendConfirmationAjax(conf *Confirmation, op string) bool {
	urlStr := APIEndpoints.CommunityBase.String() + "/mobileconf/ajaxop"
	query := s.generateConfirmationQueryParams(op)
	query.Set("op", op)
	query.Set("cid", conf.ConfirmationID)
	query.Set("ck", conf.ConfirmationKey)

	confResponse := SendConfirmationResponse{}
	cookieJar, _ := cookiejar.New(&cookiejar.Options{})
	s.Session.SetCookies(cookieJar)

	logf("requesting to %s confirmation ajax for %s", op, conf.ConfirmationID)
	_, err := SteamWeb().
		SetJar(cookieJar).
		SetParams(query).
		Get(urlStr).
		HandleJSON(&confResponse).
		Do()

	if err != nil {
		logf("unhandled internal error: %s", err)
		return false
	}

	return confResponse.Success
}

func (s *SteamGuardAccount) generateConfirmationURL(tag string) string {
	endpoint := APIEndpoints.CommunityBase.String() + "/mobileconf/conf?"
	query := s.generateConfirmationQueryParams(tag)
	return endpoint + query.Encode()
}

func (s *SteamGuardAccount) generateConfirmationQueryParams(tag string) url.Values {
	atTime := TimeAligner.GetSteamTime()
	return url.Values{
		"p":   []string{s.DeviceID},
		"a":   []string{s.Session.SteamID.String()},
		"k":   []string{s.generateConfirmationHashForTime(atTime, tag)},
		"t":   []string{strconv.FormatInt(atTime.Unix(), 10)},
		"m":   []string{"android"},
		"tag": []string{tag},
	}
}

func (s *SteamGuardAccount) generateConfirmationHashForTime(atTime time.Time, tag string) string {
	decode, _ := base64.StdEncoding.DecodeString(s.IdentitySecret)

	tagLen := len(tag)
	if tagLen > 32 {
		tagLen = 32
	}

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, atTime.Unix())
	binary.Write(buf, binary.BigEndian, tag[0:tagLen])

	mac := hmac.New(sha1.New, decode)
	mac.Write(buf.Bytes())
	hashedData := mac.Sum(nil)

	encoded := base64.StdEncoding.EncodeToString(hashedData)

	return url.QueryEscape(encoded)
}

// RemoveAuthenticatorResponse contains the response to the request to remove the authenticator
type RemoveAuthenticatorResponse struct {
	Response struct {
		Success bool `json:"success"`
	} `json:"response"`
}

// SendConfirmationResponse contains the response to confirmation requests
type SendConfirmationResponse struct {
	Success bool `json:"success"`
}
