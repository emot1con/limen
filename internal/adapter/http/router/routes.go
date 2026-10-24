package router

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"limen/internal/adapter/http/handler"
	"limen/internal/adapter/http/middleware"
	postgresattendance "limen/internal/adapter/persistence/postgres/attendance"
	postgresuser "limen/internal/adapter/persistence/postgres/user"
	"limen/internal/usecase/auth"
	deviceuc "limen/internal/usecase/device"
	settingsuc "limen/internal/usecase/settings"
	useruc "limen/internal/usecase/user"
)

func New(postgresDB *gorm.DB, redisClient *redis.Client) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	userRepo := postgresuser.NewUserRepository(postgresDB)
	userRoleRepo := postgresuser.NewUserRoleRepository(postgresDB)
	sessionRepo := postgresuser.NewSessionRepository(postgresDB)

	loginUseCase := auth.NewLoginUseCase(userRepo, userRoleRepo, sessionRepo)
	logoutUseCase := auth.NewLogoutUseCase(sessionRepo)
	meUseCase := auth.NewMeUseCase(userRepo, userRoleRepo)
	resetPasswordUseCase := auth.NewResetPasswordUseCase(userRepo, sessionRepo)

	authHandler := handler.NewAuthHandler(loginUseCase, logoutUseCase, meUseCase, resetPasswordUseCase)
	authMiddleware := middleware.NewAuthMiddleware(sessionRepo, userRoleRepo)
	loginRateLimiter := middleware.EmailRateLimiterMiddleware(redisClient, middleware.LoginEmailRateLimiter())

	v1 := router.Group("/api/v1")
	v1.POST("/auth/login", loginRateLimiter, authHandler.Login)
	v1.POST("/auth/logout", authMiddleware.Authenticate(), authMiddleware.RequireCSRF(), authHandler.Logout)
	v1.GET("/auth/me", authMiddleware.Authenticate(), authHandler.Me)
	v1.POST("/users/:userId/password-reset", authMiddleware.Authenticate(), authMiddleware.RequireCSRF(), authMiddleware.RequireRole("admin"), authHandler.ResetPassword)

	settingsRepo := postgresattendance.NewSchoolSettingsRepository(postgresDB)
	deviceRepo := postgresattendance.NewDeviceRepository(postgresDB)
	roleRepo := postgresuser.NewRoleRepository(postgresDB)
	adminHandler := handler.NewAdminHandler(settingsuc.NewUseCase(settingsRepo), deviceuc.NewUseCase(deviceRepo), useruc.NewUseCase(userRepo, roleRepo, userRoleRepo, sessionRepo))
	adminRead := v1.Group("", authMiddleware.Authenticate(), authMiddleware.RequireRole("admin"))
	adminWrite := v1.Group("", authMiddleware.Authenticate(), authMiddleware.RequireCSRF(), authMiddleware.RequireRole("admin"))
	adminRead.GET("/settings/school", adminHandler.GetSettings)
	adminWrite.PUT("/settings/school", adminHandler.UpdateSettings)
	adminRead.GET("/devices", adminHandler.ListDevices)
	adminWrite.POST("/devices", adminHandler.CreateDevice)
	adminWrite.PUT("/devices/:deviceId", adminHandler.UpdateDevice)
	adminRead.GET("/users", adminHandler.ListUsers)
	adminWrite.POST("/users", adminHandler.CreateUser)
	adminRead.GET("/users/:userId", adminHandler.GetUser)
	adminWrite.PUT("/users/:userId", adminHandler.UpdateUser)

	return router
}
