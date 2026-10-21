package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"
)

func invoiceHandler(client Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()
		var order Order
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&order); err != nil {
			http.Error(w, "invalid order JSON", http.StatusBadRequest)
			return
		}
		if err := order.validate(); err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		result, err := client.sendInvoice(r.Context(), order)
		if err != nil {
			var apiErr *APIError
			if errors.As(err, &apiErr) && apiErr.Status >= 400 && apiErr.Status < 500 {
				http.Error(w, apiErr.Error(), http.StatusUnprocessableEntity)
				return
			}
			log.Printf("invoice delivery: %v", err)
			http.Error(w, "invoice delivery failed", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("set INFRAI_API_KEY")
	}
	client := Client{HTTP: &http.Client{Timeout: 20 * time.Second}, Key: key}
	http.HandleFunc("/invoices", invoiceHandler(client))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
