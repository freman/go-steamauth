// Copyright 2015 Shannon Wynter. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package steamauth

import (
	"strconv"
	"time"
)

// SteamID is usually returned and used as a string, this just insures consistant handling
type SteamID uint64

func (s *SteamID) String() string {
	if uint64(*s) > 0 {
		return strconv.FormatUint(uint64(*s), 10)
	}
	return ""
}

func (s *SteamID) MarshalJSON() ([]byte, error) {
	return []byte("\"" + s.String() + "\""), nil
}

func (s *SteamID) UnmarshalJSON(b []byte) error {
	if b[0] == '"' {
		b = b[1 : len(b)-1]
	}
	v, err := strconv.ParseUint(string(b), 10, 64)

	*s = SteamID(v)
	return err
}

// CaptchaGID can be returned as a string, or an integeral -1, constancy is awsome
type CaptchaGID string

func (c *CaptchaGID) UnmarshalJSON(b []byte) error {
	if b[0] == '"' {
		*c = CaptchaGID(b[1 : len(b)-1])
		return nil
	}
	*c = CaptchaGID(b)
	return nil
}

func (c *CaptchaGID) String() string {
	return string(*c)
}

// URL returns a fully qualified URL (string) poiting to the captcha image
func (c *CaptchaGID) URL() string {
	if *c == "" || *c == "-1" {
		return ""
	}
	return APIEndpoints.CommunityBase.String() + "/public/captcha.php?gid=" + c.String()
}

type seconds struct {
	time.Duration
}

func (s *seconds) UnmarshalJSON(b []byte) error {
	i, err := unmarshalStringyInt(b)
	if err != nil {
		return err
	}
	s.Duration = time.Duration(i) * time.Second
	return nil
}

type timestamp struct {
	time.Time
}

func (t *timestamp) UnmarshalJSON(b []byte) error {
	s, err := unmarshalStringyValue(b)
	if err != nil {
		return err
	}
	t.Time, err = time.Parse(time.RFC3339, s)
	if _, ok := err.(*time.ParseError); ok {
		// ok, that didn't work lets try parsing an int
		var i int64
		if i, err = strconv.ParseInt(s, 10, 64); err != nil {
			return err
		}
		t.Time = time.Unix(i, 0)
	}
	return err
}

func (t *timestamp) String() string {
	return strconv.FormatInt(t.Unix(), 10)
}
