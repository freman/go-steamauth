// Copyright 2015 Shannon Wynter. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package steamauth

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"net/http/cookiejar"
	"net/url"
	"strconv"
)

// AuthenticatorLinker will link this Authenticator to your steam account
// Once successfully linked LinkedAccount contains a SteamGuardAccount
// object and save it for reuse in future requests
type AuthenticatorLinker struct {
	PhoneNumber   string
	DeviceID      string
	LinkedAccount SteamGuardAccount
	Finalized     bool

	session   *SessionData
	cookieJar *cookiejar.Jar
}

// NewAuthenticatorLinker will create an account linker
//
// Pass it the SessionData from a logged in instance of
// a UserLogin structure.
func NewAuthenticatorLinker(session *SessionData) *AuthenticatorLinker {
	cookieJar, _ := cookiejar.New(&cookiejar.Options{})

	session.SetCookies(cookieJar)

	return &AuthenticatorLinker{
		session:   session,
		DeviceID:  generateDeviceID(),
		cookieJar: cookieJar,
	}
}

// AddAuthenticator will configure your account for steamguard,
// create and link an instance of a SteamGuardAccount structure
//
// If everything goes well you will get a LinkResult of
// AwaitingFinalization which means you must collect the code
// sent to the linked phone and submit it with
// `FinalizeAddAuthenticator`
func (al *AuthenticatorLinker) AddAuthenticator() (LinkResult, error) {
	hasPhone := al.hasPhoneAttached()
	if hasPhone && al.PhoneNumber != "" {
		log(MustRemovePhoneNumber)
		return MustRemovePhoneNumber, nil
	}
	if !hasPhone && al.PhoneNumber == "" {
		log(MustProvidePhoneNumber)
		return MustProvidePhoneNumber, nil
	}

	if !hasPhone {
		if !al.addPhoneNumber() {
			log(LinkGeneralFailure)
			return LinkGeneralFailure, nil
		}
	}

	var postData = url.Values{
		"access_token":       []string{al.session.OAuthToken},
		"steamid":            []string{al.session.SteamID.String()},
		"authenticator_type": []string{"1"},
		"device_identifier":  []string{al.DeviceID},
		"sms_phone_id":       []string{"1"},
	}

	logf("Attempting add authenticator for device %s", al.DeviceID)

	addAuthenticatorResponse := AddAuthenticatorResponse{}
	_, err := SteamWeb().
		SetParams(postData).
		Post(APIEndpoints.SteamAPIBase.String() + "/ITwoFactorService/AddAuthenticator/v0001").
		HandleJSON(&addAuthenticatorResponse).
		MobileLoginRequest()

	if err != nil {
		logf("Protocol error: %s", err)
		return LinkGeneralFailure, err
	}

	if addAuthenticatorResponse.Response.Status != 1 {
		logf("Protocol error: Response.Status was %d, expected 1", addAuthenticatorResponse.Response.Status)
		return LinkGeneralFailure, nil
	}

	// SteamGuardAccount?
	al.LinkedAccount = addAuthenticatorResponse.Response
	al.LinkedAccount.Session = al.session
	al.LinkedAccount.DeviceID = al.DeviceID

	log(AwaitingFinalization)
	return AwaitingFinalization, nil
}

// FinalizeAddAuthenticator does pretty much what it says
// once you've sucessfully called this method you need
// to save the instance of `SteamGuardAccount` stored in
// `LinkedAccount` or you risk losing access to your
// steam account
func (al *AuthenticatorLinker) FinalizeAddAuthenticator(smsCode string) (FinalizeResult, error) {
	smsCodeGood := false

	var postData = url.Values{
		"steamid":            []string{al.session.SteamID.String()},
		"access_token":       []string{al.session.OAuthToken},
		"authenticator_code": []string{""},
		"activation_code":    []string{smsCode},
	}

	for tries := 0; tries <= 30; {
		postData.Set("authenticator_code", iif(tries == 0, "", al.LinkedAccount.GenerateSteamGuardCode()))
		postData.Set("authenticator_time", strconv.FormatInt(TimeAligner.GetSteamTime().Unix(), 10))

		if smsCodeGood {
			postData.Set("activation_code", "")
		}

		logf("Attempting finalize authentication, attempt %d of 30", tries+1)

		finalizeResponse := FinalizeAuthenticatorResponse{}
		_, err := SteamWeb().
			Post(APIEndpoints.SteamAPIBase.String() + "/ITwoFactorService/FinalizeAddAuthenticator/v0001").
			SetParams(postData).
			HandleJSON(&finalizeResponse).
			MobileLoginRequest()

		if err != nil {
			logf("Protocol error: %s", err)
			return FinalizeGeneralFailure, err
		}

		if finalizeResponse.Response.Status == 89 {
			log(BadSMSCode)
			return BadSMSCode, nil
		}

		if finalizeResponse.Response.Status == 88 && tries >= 30 {
			log(UnableToGenerateCorrectCodes)
			return UnableToGenerateCorrectCodes, nil
		}

		if !finalizeResponse.Response.Success {
			log("Protocol error: Response.Success == false")
			return FinalizeGeneralFailure, nil
		}

		if finalizeResponse.Response.WantMore {
			log("Steam wants more")
			smsCodeGood = true
			tries++
			continue
		}

		al.LinkedAccount.FullyEnrolled = true
		log(Success)
		return Success, nil
	}

	return FinalizeGeneralFailure, nil
}

func (al *AuthenticatorLinker) addPhoneNumber() bool {
	logf("Add phone number %s", al.PhoneNumber)
	addPhoneResponse := AddPhoneResponse{}
	_, err := SteamWeb().
		SetJar(al.cookieJar).
		Get(APIEndpoints.CommunityBase.String() + "/steamguard/phoneajax?op=add_phone_number&arg=" + url.QueryEscape(al.PhoneNumber)).
		HandleJSON(&addPhoneResponse).
		Do()

	if err != nil {
		log("Internal Error: ", err)
		return false
	}

	return addPhoneResponse.Success
}

func (al *AuthenticatorLinker) hasPhoneAttached() bool {
	logf("has phone attached?")
	hasPhoneResponse := HasPhoneResponse{}
	_, err := SteamWeb().
		SetJar(al.cookieJar).
		Get(APIEndpoints.CommunityBase.String() + "/steamguard/phoneajax?op=has_phone&arg=").
		HandleJSON(hasPhoneResponse).
		MobileLoginRequest()

	if err != nil {
		return false
	}

	return hasPhoneResponse.HasPhone
}

func generateDeviceID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	sum := sha1.Sum(buf)
	return "android:" + hex.EncodeToString(sum[:])
}

type LinkResult int

const (
	MustProvidePhoneNumber LinkResult = iota // No phone number on the account
	MustRemovePhoneNumber                    // A phone number is already on the account
	AwaitingFinalization                     // Must provide an SMS code
	LinkGeneralFailure                       // General failure (really now!)
)

var linkResults = []string{
	MustProvidePhoneNumber: "must provide phone number",
	MustRemovePhoneNumber:  "must remove phone number",
	AwaitingFinalization:   "awaiting finalization",
	LinkGeneralFailure:     "general failure",
}

func (l LinkResult) String() string {
	return linkResults[l]
}

type FinalizeResult int

const (
	BadSMSCode FinalizeResult = iota // Bad code was given
	UnableToGenerateCorrectCodes
	Success
	FinalizeGeneralFailure
)

var finalizeResults = []string{
	BadSMSCode:                   "bad sms code",
	UnableToGenerateCorrectCodes: "unable to generate correct codes",
	Success:                "success",
	FinalizeGeneralFailure: "general failure",
}

func (f FinalizeResult) String() string {
	return finalizeResults[f]
}

type AddAuthenticatorResponse struct {
	Response SteamGuardAccount `json:"response"`
}

type FinalizeAuthenticatorResponse struct {
	Response struct {
		Status     int       `json:"status"`
		ServerTime timestamp `json:"server_time"`
		WantMore   bool      `json:"want_more"`
		Success    bool      `json:"success"`
	} `json:"response"`
}

type HasPhoneResponse struct {
	HasPhone bool `json:"has_phone"`
}

type AddPhoneResponse struct {
	Success bool `json:"success"`
}
