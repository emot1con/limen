package handler

// import (
// 	"encoding/json"
// 	"net/http"

// 	schoolusecase "limen/internal/usecase/school"

// 	"github.com/gin-gonic/gin"
// )

// type SchoolHandler struct {
// 	createSchool schoolusecase.CreateUseCase
// }

// func NewSchoolHandler(createSchool schoolusecase.CreateUseCase) SchoolHandler {
// 	return SchoolHandler{createSchool: createSchool}
// }

// func (u *SchoolHandler) RegisterRoutes(r *gin.Engine) {
// 	// Protected routes (require auth)
// 	paymentRoutes := r.Group("/payment")
// 	paymentRoutes.Use(middleware.ProtectedEndpoint())
// 	paymentRoutes.Use(middleware.RateLimiterMiddleware(middleware.DefaultRateLimiterConfig()))

// 	paymentRoutes.GET("/order/:order_id", u.GetPaymentByOrderId)
// 	paymentRoutes.POST("/initiate", u.InitiatePayment)

// 	// Webhook route (no auth - called by Midtrans)
// 	r.POST("/payment/webhook/midtrans", u.HandleMidtransWebhook)
// }


// func (h SchoolHandler) Create(w http.ResponseWriter, r *http.Request) {
// 	var input schoolusecase.CreateInput
// 	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
// 		http.Error(w, "invalid request body", http.StatusBadRequest)
// 		return
// 	}

// 	school, err := h.createSchool.Execute(r.Context(), input)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
// 		return
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusCreated)
// 	json.NewEncoder(w).Encode(school)
// }
