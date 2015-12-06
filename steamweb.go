// Copyright 2015 Shannon Wynter. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package steamauth

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)
import "net/url"

type responseHandlerFunc func(*http.Response) error

type steamWeb struct {
	*http.Client
	headers http.Header
	params  url.Values
	urlStr  string
	method  string

	oV interface{}
	oF responseHandlerFunc
}

// SteamWeb returns a convenient chainable steamWeb object that allows
// you to perform requests against the steam API with a simple sequence
// of method calls.
func SteamWeb() *steamWeb {
	return &steamWeb{
		Client: &http.Client{},
		headers: http.Header{
			"User-Agent": []string{"Mozilla/5.0 (Linux; U; Android 4.1.1; en-us; Google Nexus 4 - 4.1.1 - API 16 - 768x1280 Build/JRO03S) AppleWebKit/534.30 (KHTML, like Gecko) Version/4.0 Mobile Safari/534.30"},
			"Accept":     []string{"text/javascript, text/html, application/xml, text/xml, */*"},
		},
		params: url.Values{},
		method: "GET",
	}
}

func (s *steamWeb) newRequest() (*http.Request, error) {
	var body io.Reader
	urlStr := s.urlStr

	if len(s.params) > 0 {
		switch s.method {
		case "GET":
			query := s.params.Encode()
			if strings.Contains(urlStr, "?") {
				urlStr = "&" + query
			} else {
				urlStr = "?" + query
			}
		case "POST":
			body = strings.NewReader(s.params.Encode())
			s.headers.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
		}
	}

	req, err := http.NewRequest(s.method, urlStr, body)
	if err != nil {
		return nil, err
	}

	req.Header = s.headers

	return req, nil
}

// SetJar set this requests cookiejar
func (s *steamWeb) SetJar(cookieJar http.CookieJar) *steamWeb {
	s.Jar = cookieJar
	return s
}

// SetHeaders set this requests headers
func (s *steamWeb) SetHeaders(headers http.Header) *steamWeb {
	s.headers = headers
	return s
}

// SetReferrer is a shortcut method to set the referrer header
// in this request, don't use `SetHeaders` after you call this
// method or it'll be overwritten
func (s *steamWeb) SetReferrer(referrer string) *steamWeb {
	s.headers.Set("Referrer", referrer)
	return s
}

// AddHeader add a value to the headers of this request
func (s *steamWeb) AddHeader(name, value string) *steamWeb {
	s.headers.Add(name, value)
	return s
}

// SetHeader set a header for this request
func (s *steamWeb) SetHeader(name, value string) *steamWeb {
	s.headers.Set(name, value)
	return s
}

// SetParams sets the `POST` form params, or adds query
// parameters to a `GET` request
func (s *steamWeb) SetParams(data url.Values) *steamWeb {
	s.params = data
	return s
}

// Post allows you to prepare a post request for a given url
func (s *steamWeb) Post(urlStr string) *steamWeb {
	s.urlStr = urlStr
	s.method = "POST"
	return s
}

// Get allows you to prepare a get request for a given url
func (s *steamWeb) Get(urlStr string) *steamWeb {
	s.urlStr = urlStr
	s.method = "GET"
	return s
}

// HandleJSON will configure the request parse a json response
// into the given `v` interface
func (s *steamWeb) HandleJSON(v interface{}) *steamWeb {
	s.oV = v
	s.oF = s.handleJSON
	return s
}

// Do the request, execute it then do any post processing
func (s *steamWeb) Do() (*http.Response, error) {
	req, err := s.newRequest()
	if err == nil {
		logRequest(req)
		logCookies(s, req)
		resp, err := s.Client.Do(req)
		logResponse(resp)

		// Ouput format the content via the handle function...
		if err == nil && s.oF != nil {
			err = s.oF(resp)
		}

		return resp, err
	}
	return nil, err
}

// MobileLoginRequest is a shotcut method that sets the referrer
// before executing the request
func (s *steamWeb) MobileLoginRequest() (*http.Response, error) {
	return s.
		SetReferrer(APIEndpoints.CommunityBase.String() + "/mobilelogin?oauth_client_id=DE45CD61&oauth_scope=read_profile%20write_profile%20read_client%20write_client").
		Do()
}

func (s *steamWeb) handleJSON(r *http.Response) error {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return errors.New("incorrect content type, expecting application/json")
	}
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	return decoder.Decode(s.oV)
}
