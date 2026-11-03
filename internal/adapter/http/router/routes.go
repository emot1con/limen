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
	postgresassignment "limen/internal/adapter/persistence/postgres/assignment"
	postgresattendance "limen/internal/adapter/persistence/postgres/attendance"
	postgrescounseling "limen/internal/adapter/persistence/postgres/counseling"
	postgresaudit "limen/internal/adapter/persistence/postgres/audit"
	postgresfingerprint "limen/internal/adapter/persistence/postgres/fingerprint"
	postgresquiz "limen/internal/adapter/persistence/postgres/quiz"
	postgresreports "limen/internal/adapter/persistence/postgres/reports"
	postgresuser "limen/internal/adapter/persistence/postgres/user"
	fingerprintdomain "limen/internal/domain/fingerprint"
	"limen/internal/platform/cache"
	"limen/internal/platform/config"
	academicuc "limen/internal/usecase/academic"
	aptitudeuc "limen/internal/usecase/aptitude"
	assessmentuc "limen/internal/usecase/assessment"
	assignmentuc "limen/internal/usecase/assignment"
	"limen/internal/usecase/auth"
	counselinguc "limen/internal/usecase/counseling"
	deviceuc "limen/internal/usecase/device"
	fingerprintuc "limen/internal/usecase/fingerprint"
	notificationuc "limen/internal/usecase/notification"
	questionbankuc "limen/internal/usecase/question_bank"
	quizuc "limen/internal/usecase/quiz"
	reportsuc "limen/internal/usecase/reports"
	scheduleuc "limen/internal/usecase/schedule"
	settingsuc "limen/internal/usecase/settings"
	useruc "limen/internal/usecase/user"
)

func New(postgresDB *gorm.DB, redisClient *redis.Client, cacheClient cache.Cache, cfg *config.Config, logger *logrus.Logger, fingerprintGateway fingerprintdomain.Gateway) *gin.Engine {
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

	v1 := router.Group("/api/v1")
	v1.Use(middleware.RateLimiterMiddleware(redisClient, middleware.CustomRateLimiter(300, 100)))

	loginIPLimiter := middleware.ScopedIPRateLimiterMiddleware(redisClient, middleware.CustomRateLimiter(60, 1), "login")
	loginAccountLimiter := middleware.IdentifierRateLimiterMiddleware(redisClient, middleware.LoginEmailRateLimiter().WithFailClosed())
	v1.POST("/auth/login", loginIPLimiter, loginAccountLimiter, authHandler.Login)
	v1.POST("/auth/logout", authMiddleware.Authenticate(), authMiddleware.RequireCSRF(), authHandler.Logout)
	v1.GET("/auth/me", authMiddleware.Authenticate(), authHandler.Me)

	passwordResetLimiter := middleware.UserOperationRateLimiterMiddleware(redisClient, middleware.StrictRateLimiter().WithFailClosed(), "password_reset")
	v1.POST("/users/:userId/password-reset", authMiddleware.Authenticate(), authMiddleware.RequireCSRF(), authMiddleware.RequireRole("admin"), passwordResetLimiter, authHandler.ResetPassword)

	settingsRepo := postgresattendance.NewSchoolSettingsRepository(postgresDB)
	deviceRepo := postgresattendance.NewDeviceRepository(postgresDB)
	adminHandler := handler.NewAdminHandler(settingsuc.NewUseCase(settingsRepo, cacheClient, cfg.CacheTTLSchoolSettings, logger), deviceuc.NewUseCase(deviceRepo, cacheClient, cfg.CacheTTLAcademic, logger), useruc.NewUseCase(userRepo, userRoleRepo, sessionRepo, logger))

	writeLimiter := middleware.UserOperationRateLimiterMiddleware(redisClient, middleware.ConservativeRateLimiter(), "write")
	sensitiveReadLimiter := middleware.UserOperationRateLimiterMiddleware(redisClient, middleware.ModerateRateLimiter(), "read_sensitive")
	masterReadLimiter := middleware.UserOperationRateLimiterMiddleware(redisClient, middleware.GenerousRateLimiter(), "read_master")

	adminRead := v1.Group("", authMiddleware.Authenticate(), authMiddleware.RequireRole("admin"))
	adminWrite := v1.Group("", authMiddleware.Authenticate(), authMiddleware.RequireCSRF(), authMiddleware.RequireRole("admin"))

	adminRead.GET("/settings/school", masterReadLimiter, adminHandler.GetSettings)
	adminWrite.PUT("/settings/school", writeLimiter, adminHandler.UpdateSettings)

	adminRead.GET("/devices", masterReadLimiter, adminHandler.ListDevices)
	adminWrite.POST("/devices", writeLimiter, adminHandler.CreateDevice)
	adminWrite.PUT("/devices/:deviceId", writeLimiter, adminHandler.UpdateDevice)

	adminRead.GET("/users", sensitiveReadLimiter, adminHandler.ListUsers)
	adminWrite.POST("/users", writeLimiter, adminHandler.CreateUser)
	adminRead.GET("/users/:userId", sensitiveReadLimiter, adminHandler.GetUser)
	adminWrite.PUT("/users/:userId", writeLimiter, adminHandler.UpdateUser)

	academicHandler := handler.NewAcademicHandler(academicuc.NewUseCase(
		postgresacademic.NewClassRepository(postgresDB),
		postgresacademic.NewSubjectRepository(postgresDB),
		postgresacademic.NewTeachingAssignmentRepository(postgresDB),
		userRepo,
		studentRepo,
		postgresacademic.NewQueryRepository(postgresDB),
		cacheClient,
		cfg.CacheTTLAcademic,
		logger,
	))

	adminRead.GET("/classes", masterReadLimiter, academicHandler.ListClasses)
	adminWrite.POST("/classes", writeLimiter, academicHandler.CreateClass)
	adminWrite.PUT("/classes/:classId", writeLimiter, academicHandler.UpdateClass)

	adminRead.GET("/subjects", masterReadLimiter, academicHandler.ListSubjects)
	adminWrite.POST("/subjects", writeLimiter, academicHandler.CreateSubject)

	adminRead.GET("/students", sensitiveReadLimiter, academicHandler.ListStudents)
	adminRead.GET("/students/:studentId", sensitiveReadLimiter, academicHandler.GetStudent)
	adminWrite.PATCH("/students/:studentId", writeLimiter, academicHandler.UpdateStudent)

	scheduleHandler := handler.NewScheduleHandler(scheduleuc.NewUseCase(
		postgresacademic.NewScheduleRepository(postgresDB),
		postgresacademic.NewClassRepository(postgresDB),
		postgresacademic.NewSubjectRepository(postgresDB),
		userRepo,
		cacheClient,
		cfg.CacheTTLSchedules,
		logger,
	))

	adminRead.GET("/schedules", masterReadLimiter, scheduleHandler.ListSchedules)
	adminWrite.POST("/schedules", writeLimiter, scheduleHandler.CreateSchedule)
	adminWrite.PUT("/schedules/:scheduleId", writeLimiter, scheduleHandler.UpdateSchedule)
	adminWrite.DELETE("/schedules/:scheduleId", writeLimiter, scheduleHandler.DeleteSchedule)

	adminRead.GET("/teaching-assignments", masterReadLimiter, academicHandler.ListAssignments)
	adminWrite.POST("/teaching-assignments", writeLimiter, academicHandler.CreateAssignment)

	assessmentHandler := handler.NewAssessmentHandler(assessmentuc.NewUseCase(
		postgresassessment.NewAssessmentRepository(postgresDB),
		postgresassessment.NewAssessmentScoreRepository(postgresDB),
		postgresacademic.NewClassRepository(postgresDB),
		postgresacademic.NewSubjectRepository(postgresDB),
		studentRepo,
		userRepo,
		logger,
	))

	adminRead.GET("/assessments", masterReadLimiter, assessmentHandler.List)
	adminWrite.POST("/assessments", writeLimiter, assessmentHandler.Create)
	adminRead.GET("/assessments/:assessmentId/scores", sensitiveReadLimiter, assessmentHandler.ListScores)
	adminWrite.PUT("/assessments/:assessmentId/scores/:studentUserId", writeLimiter, assessmentHandler.UpsertScore)

	authRead := v1.Group("", authMiddleware.Authenticate())
	authWrite := v1.Group("", authMiddleware.Authenticate(), authMiddleware.RequireCSRF())

	aptitudeHandler := handler.NewAptitudeHandler(aptitudeuc.NewUseCase(
		postgrescounseling.NewAptitudeResultRepository(postgresDB),
		postgrescounseling.NewTargetMajorRepository(postgresDB),
		postgrescounseling.NewPassingGradeRepository(postgresDB),
		studentRepo,
		cacheClient,
		cfg.CacheTTLPassingGrades,
		logger,
	))

	authRead.GET("/students/:studentId/aptitude-results", sensitiveReadLimiter, aptitudeHandler.ListAptitudeResults)
	authWrite.POST("/students/:studentId/target-majors", writeLimiter, aptitudeHandler.CreateTargetMajor)
	authRead.GET("/passing-grades", masterReadLimiter, aptitudeHandler.ListPassingGrades)
	authRead.GET("/passing-grades/:passingGradeId", masterReadLimiter, aptitudeHandler.GetPassingGrade)

	counselingHandler := handler.NewCounselingHandler(counselinguc.NewUseCase(
		postgrescounseling.NewCounselingNoteRepository(postgresDB),
		postgrescounseling.NewCounselingFollowupRepository(postgresDB),
		studentRepo,
		logger,
	))

	authRead.GET("/counseling-notes", sensitiveReadLimiter, counselingHandler.ListNotes)
	authWrite.POST("/counseling-notes", writeLimiter, counselingHandler.CreateNote)
	authWrite.PUT("/counseling-notes/:noteId", writeLimiter, counselingHandler.UpdateNote)
	authWrite.POST("/counseling-notes/:noteId/followups", writeLimiter, counselingHandler.AddFollowup)

	reportHandler := handler.NewReportsHandler(reportsuc.NewUseCase(
		postgresreports.NewQueryRepository(postgresDB),
		studentRepo,
		logger,
	))

	authRead.GET("/reports/attendance", sensitiveReadLimiter, reportHandler.Attendance)
	authRead.GET("/reports/grades", sensitiveReadLimiter, reportHandler.Grades)
	authRead.GET("/students/:studentId/report-card", sensitiveReadLimiter, reportHandler.ReportCard)

	assignmentHandler := handler.NewAssignmentHandler(assignmentuc.NewUseCase(
		postgresassignment.NewAssignmentRepository(postgresDB),
		postgresassignment.NewAssignmentSubmissionRepository(postgresDB),
		postgresacademic.NewClassRepository(postgresDB),
		postgresacademic.NewSubjectRepository(postgresDB),
		studentRepo,
		userRepo,
		logger,
	))

	authRead.GET("/assignments", sensitiveReadLimiter, assignmentHandler.List)
	authWrite.POST("/assignments", writeLimiter, assignmentHandler.Create)
	authRead.GET("/assignments/:assignmentId/submissions", sensitiveReadLimiter, assignmentHandler.ListSubmissions)
	authWrite.PUT("/assignments/:assignmentId/submissions/:studentUserId", writeLimiter, assignmentHandler.GradeSubmission)

	questionBankHandler := handler.NewQuestionBankHandler(questionbankuc.NewUseCase(
		postgresquiz.NewQuestionBankRepository(postgresDB),
		postgresquiz.NewQuestionRepository(postgresDB),
		postgresacademic.NewSubjectRepository(postgresDB),
		logger,
	))

	authRead.GET("/question-banks", sensitiveReadLimiter, questionBankHandler.ListBanks)
	authWrite.POST("/question-banks", writeLimiter, questionBankHandler.CreateBank)
	authRead.GET("/question-banks/:bankId/questions", sensitiveReadLimiter, questionBankHandler.ListQuestions)
	authWrite.POST("/question-banks/:bankId/questions", writeLimiter, questionBankHandler.CreateQuestion)
	authWrite.PUT("/questions/:questionId", writeLimiter, questionBankHandler.UpdateQuestion)
	authWrite.DELETE("/questions/:questionId", writeLimiter, questionBankHandler.DeleteQuestion)

	quizHandler := handler.NewQuizHandler(quizuc.NewUseCase(
		postgresquiz.NewQuizRepository(postgresDB),
		postgresquiz.NewQuestionRepository(postgresDB),
		postgresquiz.NewQuizQuestionRepository(postgresDB),
		postgresquiz.NewQuizAttemptRepository(postgresDB),
		postgresquiz.NewQuizAnswerRepository(postgresDB),
		postgresquiz.NewQueryRepository(postgresDB),
		postgresacademic.NewSubjectRepository(postgresDB),
		cacheClient,
		cfg.CacheTTLQuiz,
		logger,
	))

	authRead.GET("/quizzes", sensitiveReadLimiter, quizHandler.ListQuizzes)
	authWrite.POST("/quizzes", writeLimiter, quizHandler.CreateQuiz)
	authRead.GET("/quizzes/:quizId", sensitiveReadLimiter, quizHandler.GetQuiz)
	authWrite.POST("/quizzes/:quizId/attempts", writeLimiter, quizHandler.StartAttempt)
	authRead.GET("/quiz-attempts", sensitiveReadLimiter, quizHandler.ListAttempts)
	authRead.GET("/quiz-attempts/:attemptId", sensitiveReadLimiter, quizHandler.GetAttempt)
	authWrite.PATCH("/quiz-attempts/:attemptId/answers/:questionId", writeLimiter, quizHandler.SaveAnswer)
	authWrite.POST("/quiz-attempts/:attemptId/submit", writeLimiter, quizHandler.SubmitAttempt)
	authWrite.PUT("/quiz-attempts/:attemptId/answers/:answerId/grade", writeLimiter, quizHandler.GradeAnswer)

	notificationHandler := handler.NewNotificationHandler(notificationuc.NewUseCase(
		postgresaudit.NewNotificationRepository(postgresDB),
		logger,
	))

	authRead.GET("/notifications", masterReadLimiter, notificationHandler.List)
	authWrite.POST("/notifications", writeLimiter, notificationHandler.Create)
	authWrite.POST("/notifications/:notificationId/read", writeLimiter, notificationHandler.MarkRead)

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
