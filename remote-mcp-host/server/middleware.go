package server

import (
	"encoding/json"
	"log"
	"net/http"
)

type noBody = bool

func toJson[Req any, Res any, Dat any](handler func(Req, Dat, *http.Request) (Res, error), data Dat, checkContentType bool) func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); checkContentType && ct != "application/json" {
			http.Error(w, "Unsupported media type: Expected Content-Type: application/json", http.StatusUnsupportedMediaType)
			return
		}
		if at := r.Header.Get("Accept"); at != "application/json" && at != "*/*" {
			http.Error(w, "Unsupported media type: Expected Accept: application/json", http.StatusUnsupportedMediaType)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		var requestObject Req
		if _, noBody := any(requestObject).(noBody); !noBody {
			if err := json.NewDecoder(r.Body).Decode(&requestObject); err != nil {
				log.Printf("Invalid data to be marshaled: %s", err.Error())
				http.Error(w, "Could not parse request body.", 400)
				return
			}
		}

		responseObject, err := handler(requestObject, data, r)
		if err != nil {
			errorMessage, err := json.Marshal(map[string]string{"error": err.Error()})
			if err != nil {
				http.Error(w, "Internal Error: could not marshal error message", http.StatusInternalServerError)
			}
			w.Write(errorMessage)
			return
		}

		responseJson, err := json.Marshal(responseObject)
		if err != nil {
			http.Error(w, "Internal Error: could not marshal output data", http.StatusInternalServerError)
			return
		}
		w.Write(responseJson)
	}
}
