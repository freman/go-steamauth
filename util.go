// Copyright 2015 Shannon Wynter. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package steamauth

import (
	"encoding/json"
	"strconv"
)

// God I'm lazy
func unmarshalStringyValue(b []byte) (s string, err error) {
	err = json.Unmarshal(b, &s)
	return
}

// see previous statement x2
func unmarshalStringyInt(b []byte) (i int64, err error) {
	s := ""
	s, err = unmarshalStringyValue(b)
	if err != nil {
		return
	}
	i, err = strconv.ParseInt(s, 10, 64)
	return
}

// you get the point ;)
func iif(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
