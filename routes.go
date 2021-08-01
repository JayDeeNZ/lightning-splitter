package main

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type (
	ResponseStatus string

	Response struct {
		Status  ResponseStatus `json:"status"`
		Message string         `json:"message,omitempty"`
		Data    interface{}    `json:"data,omitempty"`
	}

	RegisterPayeeForm struct {
		Invoice string `json:"invoice" validate:"required"`
	}
)

const (
	ResponseStatusSuccess ResponseStatus = "success"
	ResponseStatusFail    ResponseStatus = "fail"
	ResponseStatusError   ResponseStatus = "error"
)

func GetNodeInfo(w http.ResponseWriter, r *http.Request) {
	info, err := lndClient.GetNodeInfo(r.Context())
	if err != nil {
		log.WithError(err).Error("Unable to get info from lnd node")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{
			Status:  ResponseStatusError,
			Message: "An internal error occurred",
		})
		return
	}

	json.NewEncoder(w).Encode(Response{
		Status: ResponseStatusSuccess,
		Data:   info,
	})
}

func RegisterPayee(w http.ResponseWriter, r *http.Request) {
	var form RegisterPayeeForm

	err := json.NewDecoder(r.Body).Decode(&form)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Status: ResponseStatusFail,
			Data:   err.Error(),
		})
		return
	}

	err = validate.Struct(form)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Status: ResponseStatusFail,
			Data:   err.Error(),
		})
		return
	}

	err = lndClient.RegisterPayee(r.Context(), form.Invoice)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{
			Status:  ResponseStatusError,
			Message: "An internal error occurred",
		})
		return
	}

	json.NewEncoder(w).Encode(Response{
		Status: ResponseStatusSuccess,
		Data:   "New payee registered successfully",
	})
}
