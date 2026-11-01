package router

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"limen/internal/adapter/http/handler"
	"limen/internal/adapter/http/middleware"
	postgresacademic "limen/internal/adapter/persistence/postgres/academic"
	postgresassessment "limen/internal/adapter/persistence/postgres/assessment"
	postgresattendance "limen/internal/adapter/persistence/postgres/attendance"
	postgrescounseling "limen/internal/adapter/persistence/postgres/counseling"
	postgresfingerprint "limen/internal/adapter/persistence/postgres/fingerprint"
	postgresuser "limen/internal/adapter/persistence/postgres/user"
	fingerprintdomain "limen/internal/domain/fingerprint"
	academicuc "limen/internal/usecase/academic"
	aptitudeuc "limen/internal/usecase/aptitude"
	assessmentuc "limen/internal/usecase/assessment"
	"limen/internal/usecase/auth"
	counselinguc "limen/internal/usecase/counseling"
	deviceuc "limen/internal/usecase/device"
	fingerprintuc "limen/internal/usecase/fingerprint"
	scheduleuc "limen/internal/usecase/schedule"
	settingsuc "limen/internal/usecase/settings"
	useruc "limen/internal/usecase/user"
)

func New(postgresDB *gorm.DB, redisClient *redis.Client, logger *logrus.Logger, fingerprintGateway fingerprintdomain.Gateway) *gin.Engine {
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

	scheduleHandler := handler.NewScheduleHandler(scheduleuc.NewUseCase(
		postgresacademic.NewScheduleRepository(postgresDB),
		postgresacademic.NewClassRepository(postgresDB),
		postgresacademic.NewSubjectRepository(postgresDB),
		userRepo,
		logger,
	))
	adminRead.GET("/schedules", scheduleHandler.ListSchedules)
	adminWrite.POST("/schedules", scheduleHandler.CreateSchedule)
	adminWrite.PUT("/schedules/:scheduleId", scheduleHandler.UpdateSchedule)
	adminWrite.DELETE("/schedules/:scheduleId", scheduleHandler.DeleteSchedule)

	adminRead.GET("/teaching-assignments", academicHandler.ListAssignments)
	adminWrite.POST("/teaching-assignments", academicHandler.CreateAssignment)

	assessmentHandler := handler.NewAssessmentHandler(assessmentuc.NewUseCase(
		postgresassessment.NewAssessmentRepository(postgresDB),
		postgresassessment.NewAssessmentScoreRepository(postgresDB),
		postgresacademic.NewClassRepository(postgresDB),
		postgresacademic.NewSubjectRepository(postgresDB),
		studentRepo,
		userRepo,
		logger,
	))
	adminRead.GET("/assessments", assessmentHandler.List)
	adminWrite.POST("/assessments", assessmentHandler.Create)
	adminRead.GET("/assessments/:assessmentId/scores", assessmentHandler.ListScores)
	adminWrite.PUT("/assessments/:assessmentId/scores/:studentUserId", assessmentHandler.UpsertScore)

	aptitudeHandler := handler.NewAptitudeHandler(aptitudeuc.NewUseCase(
		postgrescounseling.NewAptitudeResultRepository(postgresDB),
		postgrescounseling.NewTargetMajorRepository(postgresDB),
		postgrescounseling.NewPassingGradeRepository(postgresDB),
		studentRepo,
		logger,
	))
	aptitudeRead := v1.Group("", authMiddleware.Authenticate())
	aptitudeWrite := v1.Group("", authMiddleware.Authenticate(), authMiddleware.RequireCSRF())
	aptitudeRead.GET("/students/:studentId/aptitude-results", aptitudeHandler.ListAptitudeResults)
	aptitudeWrite.POST("/students/:studentId/target-majors", aptitudeHandler.CreateTargetMajor)
	aptitudeRead.GET("/passing-grades", aptitudeHandler.ListPassingGrades)
	aptitudeRead.GET("/passing-grades/:passingGradeId", aptitudeHandler.GetPassingGrade)

	counselingHandler := handler.NewCounselingHandler(counselinguc.NewUseCase(
		postgrescounseling.NewCounselingNoteRepository(postgresDB),
		postgrescounseling.NewCounselingFollowupRepository(postgresDB),
		studentRepo,
		logger,
	))
	counselingRead := v1.Group("", authMiddleware.Authenticate())
	counselingWrite := v1.Group("", authMiddleware.Authenticate(), authMiddleware.RequireCSRF())
	counselingRead.GET("/counseling-notes", counselingHandler.ListNotes)
	counselingWrite.POST("/counseling-notes", counselingHandler.CreateNote)
	counselingWrite.PUT("/counseling-notes/:noteId", counselingHandler.UpdateNote)
	counselingWrite.POST("/counseling-notes/:noteId/followups", counselingHandler.AddFollowup)

	// Legacy Express gateway endpoints, merged into the main API without auth.
	fingerprintHandler := handler.NewFingerprintHandler(fingerprintuc.NewUseCase(
		fingerprintGateway,
		postgresfingerprint.NewIdentificationRepository(postgresDB),
		userRepo,
		logger,
	))
	v1.POST("/fingerprint/identify", fingerprintHandler.Identify)
	v1.POST("/fingerprint/enroll", fingerprintHandler.Enroll)
	v1.POST("/fingerprint/enroll/raw", fingerprintHandler.EnrollRaw)
	v1.GET("/fingerprint/latest", fingerprintHandler.Latest)
	v1.GET("/fingerprint/stream", fingerprintHandler.Stream)

	return router
}
