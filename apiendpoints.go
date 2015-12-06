// Copyright 2015 Shannon Wynter. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package steamauth

import "net/url"

// APIEndpoints is storage for a bunch of common endpoints
var APIEndpoints = struct {
	SteamAPIBase       *url.URL
	CommunityBase      *url.URL
	TwoFactorTimeQuery *url.URL
}{
	SteamAPIBase:       &url.URL{Scheme: "https", Host: "api.steampowered.com"},
	CommunityBase:      &url.URL{Scheme: "https", Host: "steamcommunity.com"},
	TwoFactorTimeQuery: &url.URL{Scheme: "https", Host: "api.steampowered.com", Path: "/ITwoFactorService/QueryTime/v0001"},
}
