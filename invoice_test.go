package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvoiceEligibility(t *testing.T) {
	cases := []struct {
		name, appointment, payment string
		want                       int
	}{
		{"completed and paid", "completed", "paid", http.StatusOK},
		{"scheduled", "scheduled", "paid", http.StatusUnprocessableEntity},
		{"unpaid", "completed", "pending", http.StatusUnprocessableEntity},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := `{"id":"ord-42","appointment_status":"` + tc.appointment + `","payment_status":"` + tc.payment + `","patient_email":"patient@example.com","amount_cents":2500}`
			calls := 0
			w := httptest.NewRecorder()
			client := Client{HTTP: &http.Client{Transport: stubTransport{respond: func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Header.Get("Idempotency-Key") == "" {
					t.Fatal("missing idempotency key")
				}
				var body string
				switch req.URL.Path {
				case "/v1/pdf/generate":
					body = `{"ok":true,"data":{"url":"https://example.com/invoice.pdf"}}`
				case "/v1/email/send":
					body = `{"ok":true,"data":{"message_id":"msg-42"}}`
				default:
					t.Fatalf("unexpected path: %s", req.URL.Path)
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			}}}}
			invoiceHandler(client)(w, httptest.NewRequest(http.MethodPost, "/invoices", strings.NewReader(input)))
			if w.Code != tc.want {
				t.Fatalf("status = %d; want %d", w.Code, tc.want)
			}
			if tc.want == http.StatusOK && (calls != 2 || !strings.Contains(w.Body.String(), "msg-42")) {
				t.Fatalf("delivery = %s, calls = %d", w.Body.String(), calls)
			}
			if tc.want != http.StatusOK && calls != 0 {
				t.Fatalf("rejected order made %d API calls", calls)
			}
		})
	}
}

type stubTransport struct {
	respond func(*http.Request) (*http.Response, error)
}

func (s stubTransport) RoundTrip(r *http.Request) (*http.Response, error) { return s.respond(r) }
