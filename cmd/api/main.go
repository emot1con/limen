package main

import (
	"log"
	"net/http"

	"limen/internal/adapter/http/handler"
	"limen/internal/adapter/persistence/postgres"
	schoolusecase "limen/internal/usecase/school"
	userusecase "limen/internal/usecase/user"
)

func main() {
	userRepository := postgres.NewUserRepository()
	schoolRepository := postgres.NewSchoolRepository()

	userHandler := handler.NewUserHandler(userusecase.NewCreateUseCase(userRepository))
	schoolHandler := handler.NewSchoolHandler(schoolusecase.NewCreateUseCase(schoolRepository))

	mux := http.NewServeMux()
	mux.HandleFunc("POST /users", userHandler.Create)
	mux.HandleFunc("POST /schools", schoolHandler.Create)

	log.Fatal(http.ListenAndServe(":8080", mux))
}
