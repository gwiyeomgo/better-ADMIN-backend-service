package middlewares

import (
	"better-admin-backend-service/helpers"
	"better-admin-backend-service/security"
	"github.com/labstack/echo"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strings"
)

var (
	jwtAuthentication security.JwtAuthentication
)

func JwtToken() echo.MiddlewareFunc {
	jwtAuthentication = security.JwtAuthentication{}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			accessToken := c.Request().Header.Get("Authorization")
			if len(accessToken) == 0 {
				return next(c)
			}

			index := strings.Index(accessToken, "Bearer")
			if index < 0 {
				index = strings.Index(accessToken, "Bearer")
			}
			if index >= 0 {
				accessToken = accessToken[index+len("Bearer"):]
				accessToken = strings.Trim(accessToken, " ")
			}

			userClaim, err := jwtAuthentication.ConvertTokenUserClaim(accessToken)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
			}

			req := c.Request()
			req = req.WithContext(helpers.ContextHelper().SetUserClaim(req.Context(), userClaim))
			c.SetRequest(req)
			return next(c)
		}
	}
}

func CheckAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			_, err := helpers.ContextHelper().GetUserClaim(c.Request().Context())
			if err != nil {
				log.Warnf("No valid credentials: %s", c.Request().RequestURI)
				return echo.NewHTTPError(http.StatusUnauthorized, "Please provide valid credentials")
			}
			return next(c)
		}
	}
}
