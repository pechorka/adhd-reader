package respond

import (
	"encoding/json"
	"log"
	"net/http"
)

type Error struct {
	Code int    `json:"code"`
	Text string `json:"text,omitempty"`
}

func ErrorWithCode(w http.ResponseWriter, httpCode, appCode int) {
	w.WriteHeader(httpCode)
	JSON(w, Error{Code: appCode})
}

func RespondErrorWithText(w http.ResponseWriter, httpCode, appCode int, errText string) {
	w.WriteHeader(httpCode)
	JSON(w, Error{Code: appCode, Text: errText})
}

func JSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}
