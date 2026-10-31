package isureal

import (
	"context"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

type HandlerContext struct {
	DB            *sqlx.DB
	Config        *Config
	IPBans        *IPBanList
	Notifications *notificationHub
}

// s2/s3 を 30 のまま上げないのは、プール待ちの根拠がなく、s4 の CPU がベンチ中ほぼ埋まっていて同時実行を増やしたくないため
func dbPoolSize(region string) int {
	if region == "asia" {
		return 60
	}
	return 30
}

func NewServer(config *Config) *echo.Echo {
	if !validRegion(config.Region) {
		panic("invalid server region: " + config.Region)
	}
	e := echo.New()
	e.IPExtractor = extractClientIP

	e.Use(ActionLogMiddleware(config.LogLevel, os.Stdout))
	e.Use(middleware.Recover())
	e.HTTPErrorHandler = APIErrorHandler

	pgxConfig, err := pgx.ParseConfig(config.DatabaseDSN())
	if err != nil {
		panic("failed to parse database config: " + err.Error())
	}
	db := sqlx.NewDb(stdlib.OpenDB(*pgxConfig, stdlib.OptionShouldPing(shouldPingIdleConn)), "pgx")
	poolSize := dbPoolSize(config.Region)
	db.SetMaxOpenConns(poolSize)
	db.SetMaxIdleConns(poolSize)
	publishDBStats(db)
	// 全台再起動で DB (s4) より先に起動しても落ちないよう、最大60秒待つ
	for i := 0; i < 60; i++ {
		if err = db.Ping(); err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		panic("failed to ping database: " + err.Error())
	}

	handlerContext := &HandlerContext{
		DB:     db,
		Config: config,
		IPBans: &IPBanList{},
	}
	handlerContext.Notifications = newNotificationHub(db, e.Logger)
	go handlerContext.Notifications.run()
	go handlerContext.IPBans.run(db, e.Logger)
	e.Use(IPBanMiddleware(handlerContext))

	e.POST("/api/initialize", initializeHandler(handlerContext))
	e.POST("/api/auth/register", registerHandler(handlerContext))
	e.POST("/api/auth/login", loginHandler(handlerContext))
	e.POST("/api/auth/logout", logoutHandler(handlerContext), SessionMiddleware(handlerContext, true))
	e.GET("/api/users", searchUsersHandler(handlerContext), SessionMiddleware(handlerContext, true), RegionMiddleware(handlerContext))
	e.GET("/api/users/:userId", getUserHandler(handlerContext), SessionMiddleware(handlerContext, true), RegionMiddleware(handlerContext))
	e.GET("/api/me/export", exportPostsHandler(handlerContext), SessionMiddleware(handlerContext, true), RegionMiddleware(handlerContext))
	e.GET("/api/me", getMeHandler(handlerContext), SessionMiddleware(handlerContext, true), RegionMiddleware(handlerContext))
	e.PATCH("/api/me", updateMeHandler(handlerContext), SessionMiddleware(handlerContext, true))
	e.PUT("/api/me/profile-image", putProfileImageHandler(handlerContext), SessionMiddleware(handlerContext, true))
	e.DELETE("/api/me/profile-image", deleteProfileImageHandler(handlerContext), SessionMiddleware(handlerContext, true))
	e.POST("/api/me/account-deletion", deleteAccountHandler(handlerContext), SessionMiddleware(handlerContext, true), RegionMiddleware(handlerContext))
	e.POST("/api/posts", createPostHandler(handlerContext), FoldedSessionMiddleware())
	e.GET("/api/posting-window/current", getCurrentPostingWindowHandler(handlerContext), FoldedSessionMiddleware())
	e.GET("/api/posts/me/current", getMyCurrentPostHandler(handlerContext), FoldedSessionMiddleware())
	e.GET("/api/posts/:postId", getPostHandler(handlerContext), SessionMiddleware(handlerContext, true), RegionMiddleware(handlerContext))
	e.PATCH("/api/posts/:postId", updatePostHandler(handlerContext), SessionMiddleware(handlerContext, true), RegionMiddleware(handlerContext))
	e.DELETE("/api/posts/:postId", deletePostHandler(handlerContext), SessionMiddleware(handlerContext, true), RegionMiddleware(handlerContext))
	e.GET("/api/feeds/:feed", getFeedHandler(handlerContext), FoldedSessionMiddleware())
	e.GET("/api/memories", getMemoriesHandler(handlerContext), FoldedSessionMiddleware())
	e.GET("/api/memories/:date", getMemoriesByDateHandler(handlerContext), FoldedSessionMiddleware())
	e.POST("/api/posts/:postId/comments", createCommentHandler(handlerContext), SessionMiddleware(handlerContext, true), RegionMiddleware(handlerContext))
	e.GET("/api/posts/:postId/comments", getCommentsHandler(handlerContext), SessionMiddleware(handlerContext, true), RegionMiddleware(handlerContext))
	e.DELETE("/api/comments/:commentId", deleteCommentHandler(handlerContext), SessionMiddleware(handlerContext, true), RegionMiddleware(handlerContext))
	e.POST("/api/posts/:postId/realmojis", replaceRealMojiHandler(handlerContext), SessionMiddleware(handlerContext, true), RegionMiddleware(handlerContext))
	e.GET("/api/posts/:postId/realmojis", getRealMojisHandler(handlerContext), SessionMiddleware(handlerContext, true), RegionMiddleware(handlerContext))
	e.DELETE("/api/realmojis/:realMojiId", deleteRealMojiHandler(handlerContext), SessionMiddleware(handlerContext, true), RegionMiddleware(handlerContext))
	e.GET("/api/images/:imageId", getImageHandler(handlerContext))
	e.POST("/api/internal/image-signatures", signImagesHandler)
	e.GET("/api/notifications", getNotificationsHandler(handlerContext), SessionMiddleware(handlerContext, true), RegionMiddleware(handlerContext))
	e.POST("/api/friend-requests", createFriendRequestHandler(handlerContext), SessionMiddleware(handlerContext, true), RegionMiddleware(handlerContext))
	e.POST("/api/friend-requests/:requestId/:action", transitionFriendRequestHandler(handlerContext), SessionMiddleware(handlerContext, true), RegionMiddleware(handlerContext))
	e.GET("/api/relationships/:kind", getRelationshipsHandler(handlerContext), SessionMiddleware(handlerContext, true), RegionMiddleware(handlerContext))
	e.DELETE("/api/friendships/:userId", deleteFriendshipHandler(handlerContext), SessionMiddleware(handlerContext, true), RegionMiddleware(handlerContext))
	e.PUT("/api/follows/:userId", followHandler(handlerContext), SessionMiddleware(handlerContext, true), RegionMiddleware(handlerContext))
	e.DELETE("/api/follows/:userId", unfollowHandler(handlerContext), SessionMiddleware(handlerContext, true), RegionMiddleware(handlerContext))
	e.PUT("/api/blocks/:userId", blockHandler(handlerContext), SessionMiddleware(handlerContext, true), RegionMiddleware(handlerContext))
	e.DELETE("/api/blocks/:userId", unblockHandler(handlerContext), SessionMiddleware(handlerContext, true), RegionMiddleware(handlerContext))
	e.POST("/api/admin/auth/login", adminLoginHandler(handlerContext))
	e.POST("/api/admin/auth/logout", adminLogoutHandler(handlerContext), AdminSessionMiddleware(handlerContext, true))
	e.PUT("/api/admin/users/:userId/certification", certifyUserHandler(handlerContext), AdminSessionMiddleware(handlerContext, true))
	e.DELETE("/api/admin/users/:userId/certification", deleteUserCertificationHandler(handlerContext), AdminSessionMiddleware(handlerContext, true))
	e.POST("/api/admin/posting-window", openPostingWindowHandler(handlerContext), AdminSessionMiddleware(handlerContext, true))

	return e
}

// The default ping on reuse of a conn idle >1s costs a full round trip (~100ms from s2/s3).
// A dead conn (e.g. after a PostgreSQL restart) has already received FATAL/EOF, which a
// local non-blocking read detects; pinging it then fails and database/sql retries on a new conn.
func shouldPingIdleConn(_ context.Context, p stdlib.ShouldPingParams) bool {
	if p.IdleDuration <= time.Second {
		return false
	}
	return p.Conn.PgConn().CheckConn() != nil
}
