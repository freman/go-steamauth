// Copyright 2015 Shannon Wynter. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package steamauth

import "net/http"

type SessionData struct {
	SessionID        string
	SteamLogin       string
	SteamLoginSecure string
	WebCookie        string
	OAuthToken       string
	SteamID          SteamID
}

func (s *SessionData) SetCookies(jar http.CookieJar) {
	jar.(http.CookieJar).SetCookies(APIEndpoints.CommunityBase, []*http.Cookie{
		&http.Cookie{Name: "mobileClientVersion", Value: "0 (2.1.3)", Path: "/", Domain: ".steamcommunity.com"},
		&http.Cookie{Name: "mobileClient", Value: "android", Path: "/", Domain: ".steamcommunity.com"},

		&http.Cookie{Name: "steamid", Value: s.SteamID.String(), Path: "/", Domain: ".steamcommunity.com"},
		&http.Cookie{Name: "steamLogin", Value: s.SteamLogin, Path: "/", Domain: ".steamcommunity.com", HttpOnly: true},

		&http.Cookie{Name: "steamLoginSecure", Value: s.SteamLoginSecure, Path: "/", Domain: ".steamcommunity.com", HttpOnly: true, Secure: true},

		&http.Cookie{Name: "steam_language", Value: "english", Path: "/", Domain: ".steamcommunity.com"},
		&http.Cookie{Name: "dob", Value: "", Path: "/", Domain: ".steamcommunity.com"},
	})
}
