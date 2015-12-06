// Copyright 2015 Shannon Wynter. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package steamauth

import (
	"net/url"
	"time"
)

type timeAligner struct {
	aligned        bool
	timeDifference time.Duration
}

// TimeAligner is used to synchronise your local time to the time
// in steamservers so that the SteamGuard codes generated match
// the expectations of Steam
var TimeAligner = &timeAligner{}

// GetSteamTime will returned the synchronised time, calling
// `AlignTime` if required
func (t *timeAligner) GetSteamTime() time.Time {
	if !t.aligned {
		t.AlignTime()
	}

	return time.Now().Add(t.timeDifference)
}

// AlignTime will get the current time from steam and store
// the offset internally for use later
func (t *timeAligner) AlignTime() {
	log("Synchronising time")
	tsr := timeSyncResponse{}
	_, err := SteamWeb().
		Get(APIEndpoints.TwoFactorTimeQuery.String()).
		SetParams(url.Values{"steamid": []string{"0"}}).
		HandleJSON(&tsr).
		Do()

	if err != nil {
		return
	}

	t.timeDifference = tsr.Response.ServerTime.Sub(time.Now())
	logf("Difference between server time and local is %s", t.timeDifference)
	t.aligned = true
}

type timeSyncResponse struct {
	Response struct {
		ServerTime                 timestamp `json:"server_time"`
		SkewTolerence              seconds   `json:"skew_tolerance_seconds"`
		LargeTimeJink              seconds   `json:"large_time_jink"`
		ProbeFrequency             seconds   `json:"probe_frequency_seconds"`
		AdjustedTimeProbeFrequency seconds   `json:"adjusted_time_probe_frequency_seconds"`
		HintProbeFrequency         seconds   `json:"hint_probe_frequency_seconds"`
		SyncTimeout                seconds   `json:"sync_timeout"`
		RetryDelay                 seconds   `json:"try_again_seconds"`
		MaxAttempts                int       `json:"max_attempts"`
	} `json:"response"`
}
