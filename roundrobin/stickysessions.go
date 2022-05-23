package roundrobin

import (
	"net/http"
	"net/url"
	"time"
)

// CookieOptions has all the options one would like to set on the affinity cookie
type CookieOptions struct {
	HTTPOnly bool
	Secure   bool

	Path    string
	Domain  string
	Expires time.Time

	MaxAge   int
	SameSite http.SameSite
}

// StickySession is a mixin for load balancers that implements layer 7 (http cookie) session affinity
type StickySession struct {
	cookieName string
	options    CookieOptions
}

// NewStickySession creates a new StickySession
func NewStickySession(cookieName string) *StickySession {
	return &StickySession{cookieName: cookieName}
}

// NewStickySessionWithOptions creates a new StickySession whilst allowing for options to
// shape its affinity cookie such as "httpOnly" or "secure"
func NewStickySessionWithOptions(cookieName string, options CookieOptions) *StickySession {
	return &StickySession{cookieName: cookieName, options: options}
}

func (s *StickySession) Confirmed(req *http.Request) bool {
	appSessionId := "app_session_id"
	if req.Header.Get(appSessionId) != "" {
		return true
	}
	if req.URL.Query().Get(appSessionId) != "" {
		return true;
	}
	if req.Method == http.MethodGet {
		if _, err := req.Cookie(appSessionId); err == nil {
			return true
		}
	}
	return false
}

// GetBackend returns the backend URL stored in the sticky cookie, iff the backend is still in the valid list of servers.
func (s *StickySession) GetBackend(req *http.Request, servers []*url.URL) (*url.URL, bool, error) {
	var serverURL *url.URL
	var serverURLerr error

	header := req.Header.Get(s.cookieName)
	if header != "" {
		serverURL, serverURLerr = url.Parse(header)
	}

	if serverURL == nil && serverURLerr == nil {
		query := req.URL.Query().Get(s.cookieName)
		if query != "" {
			serverURL, serverURLerr = url.Parse(query)
		}
	}

	if serverURL == nil && serverURLerr == nil {
		cookie, err := req.Cookie(s.cookieName)
		switch err {
		case nil:
		case http.ErrNoCookie:
		default:
			serverURLerr = err
		}
		if err == nil {
			serverURL, serverURLerr = url.Parse(cookie.Value)
		}
	}

	if serverURLerr != nil {
		return nil, false, serverURLerr
	}

	if serverURL == nil {
		return nil, false, nil
	}

	if s.isBackendAlive(serverURL, servers) {
		return serverURL, true, nil
	}
	return nil, false, nil
}

// StickBackend creates and sets the cookie
func (s *StickySession) StickBackend(backend *url.URL, w *http.ResponseWriter) {
	(*w).Header().Set(s.cookieName, backend.String())
	//opt := s.options
	//
	//cp := "/"
	//if opt.Path != "" {
	//	cp = opt.Path
	//}
	//
	//cookie := &http.Cookie{
	//	Name:     s.cookieName,
	//	Value:    backend.String(),
	//	Path:     cp,
	//	Domain:   opt.Domain,
	//	Expires:  opt.Expires,
	//	MaxAge:   opt.MaxAge,
	//	Secure:   opt.Secure,
	//	HttpOnly: opt.HTTPOnly,
	//	SameSite: opt.SameSite,
	//}
	//http.SetCookie(*w, cookie)
}

func (s *StickySession) isBackendAlive(needle *url.URL, haystack []*url.URL) bool {
	if len(haystack) == 0 {
		return false
	}

	for _, serverURL := range haystack {
		if sameURL(needle, serverURL) {
			return true
		}
	}
	return false
}
