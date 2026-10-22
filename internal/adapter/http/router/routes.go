package router

// import (
// 	"broker/handler"
// 	"broker/middleware"
// 	"broker/repository"

// 	"github.com/gin-contrib/cors"
// 	"github.com/gin-gonic/gin"
// 	"github.com/sirupsen/logrus"
// )

// func Routes() *gin.Engine {
// 	r := gin.Default()

// 	// CORS middleware - allow all origins for development
// 	r.Use(cors.New(cors.Config{
// 		AllowAllOrigins:  true,
// 		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
// 		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
// 		ExposeHeaders:    []string{"Content-Length"},
// 		AllowCredentials: false,
// 	}))

// 	if err := middleware.InitRedis(); err != nil {
// 		logrus.Warnf("Failed to start Redis: %v", err)
// 	}

// 	userRepo := repository.NewUserRepository()
// 	userHandler := handler.NewUserHandler(userRepo)
// 	userHandler.RegisterRoutes(r)

// 	productRepo := repository.NewProductRepository()
// 	productHandler := handler.NewProductHandler(productRepo)
// 	productHandler.RegisterRoutes(r)

// 	orderRepo := repository.NewOrderRepository()
// 	orderHandler := handler.NewOrderHandler(userRepo, productRepo, orderRepo)
// 	orderHandler.RegisterRoutes(r)

// 	paymentRepo := repository.NewPaymentRepository()
// 	paymentHandler := handler.NewPaymentHandler(paymentRepo)
// 	paymentHandler.RegisterRoutes(r)

// 	return r
// }
