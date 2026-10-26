package router

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"limen/internal/adapter/http/handler"
	"limen/internal/adapter/http/middleware"
	postgresacademic "limen/internal/adapter/persistence/postgres/academic"
	postgresattendance "limen/internal/adapter/persistence/postgres/attendance"
	postgresuser "limen/internal/adapter/persistence/postgres/user"
	academicuc "limen/internal/usecase/academic"
	"limen/internal/usecase/auth"
	deviceuc "limen/internal/usecase/device"
	settingsuc "limen/internal/usecase/settings"
	useruc "limen/internal/usecase/user"
)

func New(postgresDB *gorm.DB, redisClient *redis.Client, logger *logrus.Logger) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery(), middleware.RequestLogger(logger))

	userRepo := postgresuser.NewUserRepository(postgresDB)
	userRoleRepo := postgresuser.NewUserRoleRepository(postgresDB)
	sessionRepo := postgresuser.NewSessionRepository(postgresDB)
	studentRepo := postgresacademic.NewStudentRepository(postgresDB)

	loginUseCase := auth.NewLoginUseCase(userRepo, userRoleRepo, sessionRepo, logger)
	logoutUseCase := auth.NewLogoutUseCase(sessionRepo, logger)
	meUseCase := auth.NewMeUseCase(userRepo, userRoleRepo, logger)
	resetPasswordUseCase := auth.NewResetPasswordUseCase(userRepo, sessionRepo, logger)

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
	adminHandler := handler.NewAdminHandler(settingsuc.NewUseCase(settingsRepo, logger), deviceuc.NewUseCase(deviceRepo, logger), useruc.NewUseCase(userRepo, roleRepo, userRoleRepo, sessionRepo, logger))
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

	academicHandler := handler.NewAcademicHandler(academicuc.NewUseCase(
		postgresacademic.NewClassRepository(postgresDB),
		postgresacademic.NewSubjectRepository(postgresDB),
		postgresacademic.NewTeachingAssignmentRepository(postgresDB),
		userRepo,
		studentRepo,
		postgresacademic.NewQueryRepository(postgresDB),
		logger,
	))
	adminRead.GET("/classes", academicHandler.ListClasses)
	adminWrite.POST("/classes", academicHandler.CreateClass)
	adminWrite.PUT("/classes/:classId", academicHandler.UpdateClass)
	adminRead.GET("/subjects", academicHandler.ListSubjects)
	adminWrite.POST("/subjects", academicHandler.CreateSubject)
	adminRead.GET("/students", academicHandler.ListStudents)
	adminRead.GET("/students/:studentId", academicHandler.GetStudent)
	adminWrite.PATCH("/students/:studentId", academicHandler.UpdateStudent)
	adminRead.GET("/teaching-assignments", academicHandler.ListAssignments)
	adminWrite.POST("/teaching-assignments", academicHandler.CreateAssignment)

	return router
}
