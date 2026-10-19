package handler

import (
	"encoding/json"
	"net/http"

	userusecase "limen/internal/usecase/user"
)

type UserHandler struct {
	createUser userusecase.CreateUseCase
}

func NewUserHandler(createUser userusecase.CreateUseCase) UserHandler {
	return UserHandler{createUser: createUser}
}

func (h UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input userusecase.CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.createUser.Execute(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}
