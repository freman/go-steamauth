// Copyright 2015 Shannon Wynter. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package steamauth

import (
	"fmt"
	"net/http"
	"net/http/httputil"
)

type logLogger interface {
	Output(calldepth int, s string) error
}

var (
	globalLogger     logLogger
	wantLogRequests  bool
	wantLogResponses bool
	wantLogCookies   bool
)

// SetLogger to be used for logging
func SetLogger(logger logLogger) {
	globalLogger = logger
}

// LogRequests enable/disable
func LogRequests(doit bool) {
	wantLogRequests = doit
}

// LogResponses enable/disable
func LogResponses(doit bool) {
	wantLogResponses = doit
}

// LogCookies enable/disable
func LogCookies(doit bool) {
	wantLogCookies = doit
}

func log(v ...interface{}) {
	if globalLogger != nil {
		globalLogger.Output(2, fmt.Sprint(v...))
	}
}

func logln(v ...interface{}) {
	if globalLogger != nil {
		globalLogger.Output(2, fmt.Sprintln(v...))
	}
}

func logf(format string, v ...interface{}) {
	if globalLogger != nil {
		globalLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

func logRequest(r *http.Request) {
	if wantLogRequests && globalLogger != nil {
		dump, _ := httputil.DumpRequestOut(r, true)
		globalLogger.Output(2, string(dump))
	}
}

func logResponse(r *http.Response) {
	if wantLogResponses && globalLogger != nil {
		dump, _ := httputil.DumpResponse(r, true)
		globalLogger.Output(2, string(dump))
	}
}

func logCookies(s *steamWeb, r *http.Request) {
	if wantLogCookies && globalLogger != nil && s.Jar != nil {
		output := []string{fmt.Sprintf("cookies for %s", r.URL)}
		for _, cookie := range s.Jar.Cookies(r.URL) {
			output = append(output, fmt.Sprintf("%-25s : %s", cookie.Name, cookie.Value))
		}
	}
}
