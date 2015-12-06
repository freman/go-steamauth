// Copyright 2015 Shannon Wynter. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package steamauth

import (
	"encoding/json"
	"testing"
)

func TestOAuthUnmarshal(t *testing.T) {
	inputCases := []string{
		`{"steamid":"76561198263585543","oauth_token":"TestOAuthToken","wgtoken":"TestSteamLogin","wgtoken_secure":"TestSteamLoginSecure","webcookie":"TestWebcookie"}`,
		`"{\"steamid\":\"76561198263585543\",\"oauth_token\":\"TestOAuthToken\",\"wgtoken\":\"TestSteamLogin\",\"wgtoken_secure\":\"TestSteamLoginSecure\",\"webcookie\":\"TestWebcookie\"}"`,
	}
	expectOutput := OAuth{
		SteamID:          SteamID(76561198263585543),
		OAuthToken:       "TestOAuthToken",
		SteamLogin:       "TestSteamLogin",
		SteamLoginSecure: "TestSteamLoginSecure",
		Webcookie:        "TestWebcookie",
	}

	for _, inputCase := range inputCases {
		output := OAuth{}
		err := json.Unmarshal([]byte(inputCase), &output)
		if err != nil {
			t.Fatal(err)
		}
		if output != expectOutput {
			t.Fatal("Didn't parse properly")
		}
	}
}
