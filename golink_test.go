// Copyright 2022 Tailscale Inc & Contributors
// SPDX-License-Identifier: BSD-3-Clause

package golink

import (
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"golang.org/x/net/xsrftoken"
)

func init() {
	// tests always need golink to be run in dev mode
	*dev = ":8080"
}

func TestServeGo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	links := map[string]*Link{
		"who":         {Short: "who", Long: "http://who/"},
		"me":          {Short: "me", Long: "/who/{{.User}}"},
		"invalid-var": {Short: "invalid-var", Long: "/who/{{.Invalid}}"},
	}

	db = NewMockDatabase(ctrl)

	tests := []struct {
		name        string
		link        string
		short       string
		currentUser func(*http.Request) (string, error)
		wantStatus  int
		wantLink    string
		shouldErr   bool
	}{
		{
			name:       "simple link",
			link:       "/who",
			short:      "who",
			wantStatus: http.StatusFound,
			wantLink:   "http://who/",
		},
		{
			name:        "simple link, anonymous request",
			link:        "/who",
			short:       "who",
			currentUser: func(*http.Request) (string, error) { return "", nil },
			wantStatus:  http.StatusFound,
			wantLink:    "http://who/",
		},
		{
			name:       "user link",
			link:       "/me",
			short:      "me",
			wantStatus: http.StatusFound,
			wantLink:   "/who/foo@example.com",
		},
		{
			name:       "unknown link",
			link:       "/does-not-exist",
			short:      "does-not-exist",
			shouldErr:  true,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "unknown variable",
			link:       "/invalid-var",
			short:      "invalid-var",
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:        "user link, anonymous request",
			link:        "/me",
			short:       "me",
			currentUser: func(*http.Request) (string, error) { return "", nil },
			wantStatus:  http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.currentUser != nil {
				oldCurrentUser := currentUser
				currentUser = tt.currentUser
				t.Cleanup(func() {
					currentUser = oldCurrentUser
				})
			}

			r := httptest.NewRequest("GET", tt.link, nil)
			w := httptest.NewRecorder()

			var result interface{}
			var err error
			if tt.shouldErr {
				err = fs.ErrNotExist
			} else {
				result = links[tt.short]
			}

			db.(*MockDatabase).
				EXPECT().
				Load(tt.short).
				Return(result, err)

			serveGo(w, r)

			if w.Code != tt.wantStatus {
				t.Errorf("serveGo(%q) = %d; want %d", tt.link, w.Code, tt.wantStatus)
			}
			if gotLink := w.Header().Get("Location"); gotLink != tt.wantLink {
				t.Errorf("serveGo(%q) = %q; want %q", tt.link, gotLink, tt.wantLink)
			}
		})
	}
}

func TestServeSave(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db = NewMockDatabase(ctrl)

	tests := []struct {
		name              string
		short             string
		long              string
		allowUnknownUsers bool
		currentUser       func(*http.Request) (string, error)
		wantStatus        int
	}{
		{
			name:       "missing short",
			short:      "",
			long:       "http://who/",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing long",
			short:      "",
			long:       "http://who/",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "save simple link",
			short:      "who",
			long:       "http://who/",
			wantStatus: http.StatusOK,
		},
		{
			name:        "disallow editing another's link",
			short:       "who",
			long:        "http://who/",
			currentUser: func(*http.Request) (string, error) { return "bar@example.com", nil },
			wantStatus:  http.StatusForbidden,
		},
		{
			name:        "disallow unknown users",
			short:       "who2",
			long:        "http://who/",
			currentUser: func(*http.Request) (string, error) { return "", errors.New("") },
			wantStatus:  http.StatusInternalServerError,
		},
		// TODO: Uncomment test when the detection of users is properly done
		// {
		// 	name:              "allow unknown users",
		// 	short:             "who2",
		// 	long:              "http://who/",
		// 	allowUnknownUsers: true,
		// 	currentUser:       func(*http.Request) (string, error) { return "", nil },
		// 	wantStatus:        http.StatusOK,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			link := &Link{Owner: "foo@example.com"}
			if tt.currentUser != nil {
				oldCurrentUser := currentUser
				currentUser = tt.currentUser
				t.Cleanup(func() {
					currentUser = oldCurrentUser
				})
			}

			var err error
			if len(tt.short) <= 0 {
				err = fs.ErrNotExist
			}

			if tt.wantStatus == http.StatusOK || tt.wantStatus == http.StatusForbidden {
				db.(*MockDatabase).EXPECT().
					Load(tt.short).
					Return(link, err)
			}

			if tt.wantStatus == http.StatusOK {
				db.(*MockDatabase).EXPECT().
					Save(gomock.Any()).
					AnyTimes().
					Return(nil)
			}

			oldAllowUnknownUsers := *allowUnknownUsers
			*allowUnknownUsers = tt.allowUnknownUsers
			t.Cleanup(func() { *allowUnknownUsers = oldAllowUnknownUsers })

			if tt.allowUnknownUsers {
				oldDev := *dev
				*dev = ""
				t.Cleanup(func() { *dev = oldDev })
			}

			r := httptest.NewRequest("POST", "/", strings.NewReader(url.Values{
				"short": {tt.short},
				"long":  {tt.long},
			}.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			serveSave(w, r)

			if w.Code != tt.wantStatus {
				t.Errorf("serveSave(%q, %q) = %d; want %d", tt.short, tt.long, w.Code, tt.wantStatus)
			}
		})
	}
}

func TestServeDelete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db = NewMockDatabase(ctrl)
	links := map[string]*Link{
		"a":   {Short: "a", Owner: "a@example.com"},
		"foo": {Short: "foo", Owner: "foo@example.com"},
	}

	xsrf := func(short string) string {
		return xsrftoken.Generate(xsrfKey, "foo@example.com", short)
	}

	tests := []struct {
		name        string
		short       string
		xsrf        string
		currentUser func(*http.Request) (string, error)
		wantStatus  int
	}{
		{
			name:       "missing short",
			short:      "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "non-existant link",
			short:      "does-not-exist",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "unowned link",
			short:      "a",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "invalid xsrf",
			short:      "foo",
			xsrf:       xsrf("invalid"),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "valid xsrf",
			short:      "foo",
			xsrf:       xsrf("foo"),
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.currentUser != nil {
				oldCurrentUser := currentUser
				currentUser = tt.currentUser
				t.Cleanup(func() {
					currentUser = oldCurrentUser
				})
			}

			var err error = nil
			if len(tt.short) > 0 {
				link := links[tt.short]
				if link == nil {
					err = fs.ErrNotExist
				}

				db.(*MockDatabase).EXPECT().
					Load(tt.short).
					Return(link, err)
			}

			if tt.wantStatus == http.StatusOK {
				db.(*MockDatabase).EXPECT().
					Delete(tt.short).
					Return(nil)
				db.(*MockDatabase).EXPECT().
					DeleteStats(tt.short).
					Return(nil)
			}

			r := httptest.NewRequest("POST", "/.delete/"+tt.short, strings.NewReader(url.Values{
				"xsrf": {tt.xsrf},
			}.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			serveDelete(w, r)

			if w.Code != tt.wantStatus {
				t.Errorf("serveDelete(%q) = %d; want %d", tt.short, w.Code, tt.wantStatus)
			}
		})
	}
}

func TestExpandLink(t *testing.T) {
	tests := []struct {
		name      string    // test name
		long      string    // long URL for golink
		now       time.Time // current time
		user      string    // current user resolving link
		remainder string    // remainder of URL path after golink name
		wantErr   bool      // whether we expect an error
		want      string    // expected redirect URL
	}{
		{
			name: "dont-mangle-escapes",
			long: "http://host.com/foo%2f/bar",
			want: "http://host.com/foo%2f/bar",
		},
		{
			name:      "dont-mangle-escapes-and-remainder",
			long:      "http://host.com/foo%2f/bar",
			remainder: "extra",
			want:      "http://host.com/foo%2f/bar/extra",
		},
		{
			name:      "remainder-insert-slash",
			long:      "http://host.com/foo",
			remainder: "extra",
			want:      "http://host.com/foo/extra",
		},
		{
			name:      "remainder-long-as-trailing-slash",
			long:      "http://host.com/foo/",
			remainder: "extra",
			want:      "http://host.com/foo/extra",
		},
		{
			name: "var-expansions-time",
			long: `https://roamresearch.com/#/app/ts-corp/page/{{.Now.Format "01-02-2006"}}`,
			want: "https://roamresearch.com/#/app/ts-corp/page/06-02-2022",
			now:  time.Date(2022, 06, 02, 1, 2, 3, 4, time.UTC),
		},
		{
			name: "var-expansions-user",
			long: `http://host.com/{{.User}}`,
			user: "foo@example.com",
			want: "http://host.com/foo@example.com",
		},
		{
			name:    "var-expansions-no-user",
			long:    `http://host.com/{{.User}}`,
			wantErr: true,
		},
		{
			name:    "unknown-field",
			long:    `http://host.com/{{.Foo}}`,
			wantErr: true,
		},
		{
			name: "template-no-path",
			long: "https://calendar.google.com/{{with .Path}}calendar/embed?mode=week&src={{.}}@tailscale.com{{end}}",
			want: "https://calendar.google.com/",
		},
		{
			name:      "template-with-path",
			long:      "https://calendar.google.com/{{with .Path}}calendar/embed?mode=week&src={{.}}@tailscale.com{{end}}",
			remainder: "amelie",
			want:      "https://calendar.google.com/calendar/embed?mode=week&src=amelie@tailscale.com",
		},
		{
			name:      "template-with-pathescape-func",
			long:      "http://host.com/{{PathEscape .Path}}",
			remainder: "a/b",
			want:      "http://host.com/a%2Fb",
		},
		{
			name:      "template-with-queryescape-func",
			long:      "http://host.com/{{QueryEscape .Path}}",
			remainder: "a+b",
			want:      "http://host.com/a%2Bb",
		},
		{
			name:      "template-with-trimsuffix-func",
			long:      `http://host.com/{{TrimSuffix .Path "/"}}`,
			remainder: "a/",
			want:      "http://host.com/a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := expandLink(tt.long, expandEnv{Now: tt.now, Path: tt.remainder, user: tt.user})
			if (err != nil) != tt.wantErr {
				t.Fatalf("expandLink(%q) returned error %v; want %v", tt.long, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("expandLink(%q) = %q; want %q", tt.long, got, tt.want)
			}
		})
	}
}

func TestResolveLink(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db = NewMockDatabase(ctrl)
	links := map[string]*Link{
		"meet": {Short: "meet", Long: "https://meet.google.com/lookup/"},
		"cs":   {Short: "cs", Long: "http://codesearch/{{with .Path}}search?q={{.}}{{end}}"},
		"m":    {Short: "m", Long: "http://go/meet"},
		"chat": {Short: "chat", Long: "/meet"},
	}

	tests := []struct {
		link  string
		want  string
		short string
	}{
		{
			link:  "meet",
			short: "meet",
			want:  "https://meet.google.com/lookup/",
		},
		{
			link:  "meet/foo",
			short: "meet",
			want:  "https://meet.google.com/lookup/foo",
		},
		{
			link:  "go/meet/foo",
			short: "meet",
			want:  "https://meet.google.com/lookup/foo",
		},
		{
			link:  "http://go/meet/foo",
			short: "meet",
			want:  "https://meet.google.com/lookup/foo",
		},
		{
			// if absolute URL provided, host doesn't actually matter
			link:  "http://mygo/meet/foo",
			short: "meet",
			want:  "https://meet.google.com/lookup/foo",
		},
		{
			link:  "cs",
			short: "cs",
			want:  "http://codesearch/",
		},
		{
			link:  "cs/term",
			short: "cs",
			want:  "http://codesearch/search?q=term",
		},
		{
			// aliased go links with hostname
			link:  "m/foo",
			short: "m",
			want:  "https://meet.google.com/lookup/foo",
		},
		// {
		// 	// aliased go links without hostname
		// 	link:  "chat/foo",
		// 	short: "chat",
		// 	want:  "https://meet.google.com/lookup/foo",
		// },
	}
	for _, tt := range tests {
		name := "golink " + tt.link
		t.Run(name, func(t *testing.T) {
			db.(*MockDatabase).EXPECT().
				Load(tt.short).
				Return(links[tt.short], nil).
				AnyTimes()

			got, err := resolveLink(tt.link)
			if err != nil {
				t.Error(err)
			}
			if got != tt.want {
				t.Errorf("ResolveLink(%q) = %q; want %q", tt.link, got, tt.want)
			}
		})
	}
}
