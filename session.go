package session

import (
	"net/http"
	"time"
"fmt"

	"github.com/alexedwards/scs/v2"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type SessionConfig struct {
	Skipper        middleware.Skipper
	SessionManager *scs.SessionManager
}

var DefaultSessionConfig = SessionConfig{
	Skipper: middleware.DefaultSkipper,
}

func LoadAndSave(sessionManager *scs.SessionManager) echo.MiddlewareFunc {
	c := DefaultSessionConfig
	c.SessionManager = sessionManager

	return LoadAndSaveWithConfig(c)
}

func LoadAndSaveWithConfig(config SessionConfig) echo.MiddlewareFunc {

	if config.Skipper == nil {
		config.Skipper = DefaultSessionConfig.Skipper
	}

	if config.SessionManager == nil {
		panic("Session middleware requires a session manager")
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if config.Skipper(c) {
                               fmt.Printf("---SCS skipped\n")
				return next(c)
			}

			ctx := c.Request().Context()

			var token string
			cookie, err := c.Cookie(config.SessionManager.Cookie.Name)
			if err == nil {
				token = cookie.Value
                               fmt.Printf("SCS token = %v\n", token)
			} else {
                               fmt.Printf("SCS cookie error  = %v\n", err)
                        }


			ctx, err = config.SessionManager.Load(ctx, token)
			if err != nil {
                               fmt.Printf("---SCS load error\n")
				return err
			} else {
                               fmt.Printf("ctx = %v\n", ctx)
                        }



			c.SetRequest(c.Request().WithContext(ctx))

			c.Response().Before(func() {
				st := config.SessionManager.Status(ctx)
                                fmt.Printf("status = %v\n", st)

				if st != scs.Unmodified {

					responseCookie := &http.Cookie{
						Name:     config.SessionManager.Cookie.Name,
						Path:     config.SessionManager.Cookie.Path,
						Domain:   config.SessionManager.Cookie.Domain,
						Secure:   config.SessionManager.Cookie.Secure,
						HttpOnly: config.SessionManager.Cookie.HttpOnly,
						SameSite: config.SessionManager.Cookie.SameSite,
					}

					switch config.SessionManager.Status(ctx) {
					case scs.Modified:
                               fmt.Printf("---SCS Session modified\n")
						token, _, err := config.SessionManager.Commit(ctx)
						if err != nil {
							panic(err)
						}

						responseCookie.Value = token

					case scs.Destroyed:
                               fmt.Printf("---SCS Session Destroyed\n")
						responseCookie.Expires = time.Unix(1, 0)
						responseCookie.MaxAge = -1
					}

					c.SetCookie(responseCookie)
					addHeaderIfMissing(c.Response(), "Cache-Control", `no-cache="Set-Cookie"`)
					addHeaderIfMissing(c.Response(), "Vary", "Cookie")
				}
			})

			return next(c)
		}
	}
}

func addHeaderIfMissing(w http.ResponseWriter, key, value string) {
	for _, h := range w.Header()[key] {
		if h == value {
			return
		}
	}
	w.Header().Add(key, value)
}
