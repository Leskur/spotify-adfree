//go:build !windows

package sysproxy

import (
	"net/http"
	"net/url"
)

func systemHTTP(_ *http.Request, _ int) (*url.URL, error) {
	return nil, nil
}
