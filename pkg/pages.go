package barertc

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"git.kirsle.net/apps/barertc/pkg/config"
	"git.kirsle.net/apps/barertc/pkg/jwt"
	"git.kirsle.net/apps/barertc/pkg/log"
	"git.kirsle.net/apps/barertc/pkg/util"
)

// IndexPage returns the HTML template for the chat room.
func IndexPage() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Load the template, TODO: once on server startup.
		tmpl := template.New("index")

		// Handle a JWT authentication token.
		var (
			tokenStr  = r.FormValue("jwt")
			claims    = &jwt.Claims{}
			authOK    bool
			blocklist = []string{} // cached blocklist from your website, for JWT auth only
		)
		if tokenStr != "" {
			parsed, ok, err := jwt.ParseAndValidate(tokenStr)
			if err != nil {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(
					fmt.Sprintf("Error parsing your JWT token: %s", err),
				))
				return
			}

			authOK = ok
			claims = parsed
			blocklist = GetCachedBlocklist(claims.Subject)
		}

		// Are we enforcing strict JWT authentication?
		if config.Current.JWT.Enabled && config.Current.JWT.Strict && !authOK {
			// Do we have a landing page to redirect to?
			if config.Current.JWT.LandingPageURL != "" {
				w.Header().Add("Location", config.Current.JWT.LandingPageURL)
				w.WriteHeader(http.StatusFound)
				return
			}

			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(
				"Authentication denied. Please go back and try again.",
			))
			return
		}

		// Variables to give to the front-end page.
		var values = map[string]interface{}{
			// A cache-busting hash for JS and CSS includes.
			"CacheHash": util.RandomString(8),

			// The current website settings.
			"Config": config.Current,

			// Authentication settings.
			"JWTTokenString": tokenStr,
			"JWTAuthOK":      authOK,
			"JWTClaims":      claims,

			// Cached user blocklist sent by your website.
			"CachedBlocklist": blocklist,
		}

		tmpl.Funcs(template.FuncMap{
			"AsHTML": func(v string) template.HTML {
				return template.HTML(v)
			},
			"AsJS": func(v interface{}) template.JS {
				return template.JS(fmt.Sprintf("%v", v))
			},
		})
		tmpl, err := tmpl.ParseFiles("dist/index.html")
		if err != nil {
			panic(err.Error())
		}
		// END load the template

		log.Info("GET / [%s] %s", r.RemoteAddr, strings.Join([]string{
			r.Header.Get("X-Forwarded-For"),
			r.Header.Get("X-Real-IP"),
			r.Header.Get("User-Agent"),
			util.IPAddress(r),
		}, " "))
		tmpl.ExecuteTemplate(w, "index", values)
	})
}

// AboutPage returns the HTML template for the about page.
func AboutPage() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Load the template, TODO: once on server startup.
		tmpl := template.New("index")

		// Variables to give to the front-end page.
		var values = map[string]interface{}{
			// A cache-busting hash for JS and CSS includes.
			"CacheHash": util.RandomString(8),

			// The current website settings.
			"Config":   config.Current,
			"Hostname": r.Host,
		}

		tmpl.Funcs(template.FuncMap{
			"AsHTML": func(v string) template.HTML {
				return template.HTML(v)
			},
		})
		tmpl, err := tmpl.ParseFiles("web/templates/about.html")
		if err != nil {
			panic(err.Error())
		}

		tmpl.ExecuteTemplate(w, "index", values)
	})
}

// LogoutPage returns the HTML template for the logout page.
func LogoutPage() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Load the template, TODO: once on server startup.
		tmpl := template.New("index")
		tmpl, err := tmpl.ParseFiles("web/templates/logout.html")
		if err != nil {
			panic(err.Error())
		}
		tmpl.ExecuteTemplate(w, "index", nil)
	})
}
