// Copyright 2015 Shannon Wynter. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package steamauth

import (
	"encoding/json"
	"testing"
)

func TestSteamID(t *testing.T) {
	jsondata := `{"steam_id": "76561198263585543"}`
	jsonout := struct {
		SteamID SteamID `json:"steam_id"`
	}{}
	err := json.Unmarshal([]byte(jsondata), &jsonout)
	if err != nil {
		t.Error(err)
	}

	if jsonout.SteamID != SteamID(76561198263585543) {
		t.Error("mismatched")
	}
}

func TestCaptchaGID(t *testing.T) {
	cases := []struct {
		input  string
		result CaptchaGID
		output string
	}{
		{`-1`, CaptchaGID("-1"), "-1"},
		{`"756965923036385917"`, CaptchaGID("756965923036385917"), "756965923036385917"},
	}

	for _, test := range cases {
		result := CaptchaGID("")
		err := json.Unmarshal([]byte(test.input), &result)
		if err != nil {
			t.Error(err)
		}
		if result != test.result {
			t.Errorf("mismatched `%#v` <> `%#v`", result, test.result)
		}
		if result.String() != test.output {
			t.Errorf("string mismatched `%s` <> `%s`", result.String(), test.output)
		}
	}
}
