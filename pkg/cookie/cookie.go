package cookie

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	// "github.com/rdeusser/oauth2-proxy/pkg/structs"
	"github.com/rdeusser/oauth2-proxy/pkg/cfg"
	"github.com/rdeusser/oauth2-proxy/pkg/domains"
)

const maxCookieSize = 4000

var log = cfg.Cfg.Logger

// SetCookie http
func SetCookie(w http.ResponseWriter, r *http.Request, val string) {
	setCookie(w, r, val, cfg.Cfg.Cookie.MaxAge*60) // convert minutes to seconds
}

func setCookie(w http.ResponseWriter, r *http.Request, val string, maxAge int) {
	cookieName := cfg.Cfg.Cookie.Name
	// foreach domain
	domain := domains.Matches(r.Host)
	// Allow overriding the cookie domain in the config file
	if cfg.Cfg.Cookie.Domain != "" {
		domain = cfg.Cfg.Cookie.Domain
		log.Debugf("setting the cookie domain to %v", domain)
	}
	cookie := http.Cookie{
		Name:     cfg.Cfg.Cookie.Name,
		Value:    val,
		Path:     "/",
		Domain:   domain,
		MaxAge:   maxAge,
		Secure:   cfg.Cfg.Cookie.Secure,
		HttpOnly: cfg.Cfg.Cookie.HTTPOnly,
	}
	cookieSize := len(cookie.String())
	cookie.Value = ""
	emptyCookieSize := len(cookie.String())
	// Cookies have a max size of 4096 bytes, but to support most browsers, we should stay below 4000 bytes
	// https://tools.ietf.org/html/rfc6265#section-6.1
	// http://browsercookielimits.squawky.net/
	if cookieSize > maxCookieSize {
		// https://www.lifewire.com/cookie-limit-per-domain-3466809
		log.Warnf("cookie size: %d.  cookie sizes over ~4093 bytes(depending on the browser and platform) have shown to cause issues or simply aren't supported.", cookieSize)
		cookieParts := SplitCookie(val, maxCookieSize-emptyCookieSize)
		for i, cookiePart := range cookieParts {
			// Cookies are named 1of3, 2of3, 3of3
			cookieName = fmt.Sprintf("%s_%dof%d", cfg.Cfg.Cookie.Name, i+1, len(cookieParts))
			http.SetCookie(w, &http.Cookie{
				Name:     cookieName,
				Value:    cookiePart,
				Path:     "/",
				Domain:   domain,
				MaxAge:   maxAge,
				Secure:   cfg.Cfg.Cookie.Secure,
				HttpOnly: cfg.Cfg.Cookie.HTTPOnly,
			})
		}
	} else {
		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    val,
			Path:     "/",
			Domain:   domain,
			MaxAge:   maxAge,
			Secure:   cfg.Cfg.Cookie.Secure,
			HttpOnly: cfg.Cfg.Cookie.HTTPOnly,
		})
	}
}

// Cookie get the oauth2 jwt cookie
func Cookie(r *http.Request) (string, error) {

	var cookieParts []string
	var numParts = -1

	var err error
	cookies := r.Cookies()
	// Get the remaining parts
	// search for cookie parts in order
	// this is the hotpath so we're trying to only walk once
	for _, cookie := range cookies {
		if cookie.Name == cfg.Cfg.Cookie.Name {
			return cookie.Value, nil
		}
		if strings.HasPrefix(cookie.Name, fmt.Sprintf("%s_", cfg.Cfg.Cookie.Name)) {
			log.Debugw("cookie",
				"cookieName", cookie.Name,
				"cookieValue", cookie.Value,
			)
			xOFy := strings.Split(cookie.Name, "_")[1]
			xyArray := strings.Split(xOFy, "of")
			if numParts == -1 { // then its uninitialized
				if numParts, err = strconv.Atoi(xyArray[1]); err != nil {
					return "", fmt.Errorf("multipart cookie fail: %s", err)
				}
				log.Debugf("make cookieParts of size %d", numParts)
				cookieParts = make([]string, numParts)
			}
			var i int
			if i, err = strconv.Atoi(xyArray[0]); err != nil {
				return "", fmt.Errorf("multipart cookie fail: %s", err)
			}
			cookieParts[i-1] = cookie.Value
		}

	}
	// combinedCookieStr := combinedCookie.String()
	combinedCookieStr := strings.Join(cookieParts, "")
	if combinedCookieStr == "" {
		return "", errors.New("Cookie token empty")
	}

	log.Debugw("combined cookie",
		"cookieValue", combinedCookieStr,
	)
	return combinedCookieStr, err
}

// ClearCookie get rid of the existing cookie
func ClearCookie(w http.ResponseWriter, r *http.Request) {
	cookies := r.Cookies()
	domain := domains.Matches(r.Host)
	// Allow overriding the cookie domain in the config file
	if cfg.Cfg.Cookie.Domain != "" {
		domain = cfg.Cfg.Cookie.Domain
		log.Debugf("setting the cookie domain to %v", domain)
	}
	// search for cookie parts
	for _, cookie := range cookies {
		if strings.HasPrefix(cookie.Name, cfg.Cfg.Cookie.Name) {
			log.Debugf("deleting cookie: %s", cookie.Name)
			http.SetCookie(w, &http.Cookie{
				Name:     cookie.Name,
				Value:    "delete",
				Path:     "/",
				Domain:   domain,
				MaxAge:   -1,
				Secure:   cfg.Cfg.Cookie.Secure,
				HttpOnly: cfg.Cfg.Cookie.HTTPOnly,
			})
		}
	}
}

// SplitCookie separate string into several strings of specified length
func SplitCookie(longString string, maxLen int) []string {
	splits := []string{}

	var l, r int
	for l, r = 0, maxLen; r < len(longString); l, r = r, r+maxLen {
		for !utf8.RuneStart(longString[r]) {
			r--
		}
		splits = append(splits, longString[l:r])
	}
	splits = append(splits, longString[l:])
	return splits
}
