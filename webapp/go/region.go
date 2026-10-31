package isureal

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

// serverRegionHeader は、リクエストを受けたサーバーのリージョン。s2・s3 の nginx は /api を s1 のアプリに転送し、
// europe・america を入れる（DB の隣の s1 で処理して、DB との往復を 100ms から 1ms にする）。s1 の nginx は asia で
// 上書きするので、クライアントが送った値はアプリに届かない。付いていなければ ISUREAL_REGION を使う
const serverRegionHeader = "X-Isureal-Region"

func serverRegion(c *echo.Context, h *HandlerContext) string {
	if region := c.Request().Header.Get(serverRegionHeader); validRegion(region) {
		return region
	}
	return h.Config.Region
}

func RegionMiddleware(h *HandlerContext) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			session := GetSession(c)
			if session == nil {
				return NewAPIError(http.StatusUnauthorized, "invalid_session")
			}
			if session.Region != serverRegion(c, h) {
				return NewAPIError(http.StatusForbidden, "region_mismatch")
			}
			return next(c)
		}
	}
}
