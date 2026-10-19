package handler

import (
	"encoding/json"
	"net/http"

	schoolusecase "limen/internal/usecase/school"
)

type SchoolHandler struct {
	createSchool schoolusecase.CreateUseCase
}

func NewSchoolHandler(createSchool schoolusecase.CreateUseCase) SchoolHandler {
	return SchoolHandler{createSchool: createSchool}
}

func (h SchoolHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input schoolusecase.CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	school, err := h.createSchool.Execute(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(school)
}
