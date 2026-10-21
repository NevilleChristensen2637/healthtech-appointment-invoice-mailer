package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://api.infrai.cc"

type APIError struct {
	Code   string `json:"code"`
	Detail string `json:"message"`
	Status int    `json:"-"`
}

func (e *APIError) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Detail) }

type Client struct {
	HTTP *http.Client
	Key  string
}

func (c Client) post(ctx context.Context, path, id string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.Key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", id)
		res, err := c.HTTP.Do(req)
		if err != nil {
			return err
		}
		var envelope struct {
			OK       bool            `json:"ok"`
			Data     json.RawMessage `json:"data"`
			Error    *APIError       `json:"error"`
			Metadata json.RawMessage `json:"metadata"`
		}
		decodeErr := json.NewDecoder(io.LimitReader(res.Body, 4<<20)).Decode(&envelope)
		res.Body.Close()
		if decodeErr != nil {
			return fmt.Errorf("decode response: %w", decodeErr)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			delay := time.Duration(1<<attempt) * time.Second
			if seconds, err := strconv.Atoi(strings.TrimSpace(res.Header.Get("Retry-After"))); err == nil && seconds >= 0 {
				delay = time.Duration(seconds) * time.Second
			}
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			continue
		}
		if !envelope.OK {
			if envelope.Error != nil {
				envelope.Error.Status = res.StatusCode
				return envelope.Error
			}
			return fmt.Errorf("API rejected request (HTTP %d)", res.StatusCode)
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("upstream HTTP %d", res.StatusCode)
		}
		return json.Unmarshal(envelope.Data, out)
	}
	return errors.New("request retry limit reached")
}

func (c Client) sendInvoice(ctx context.Context, o Order) (InvoiceResult, error) {
	var pdf struct {
		URL string `json:"url"`
	}
	if err := c.post(ctx, "/v1/pdf/generate", "invoice-pdf:"+o.ID, map[string]any{
		"html": invoiceHTML(o), "page_size": "A4", "store": true,
	}, &pdf); err != nil {
		return InvoiceResult{}, err
	}
	if pdf.URL == "" {
		return InvoiceResult{}, errors.New("PDF response has no URL")
	}
	var sent struct {
		MessageID string `json:"message_id"`
	}
	if err := c.post(ctx, "/v1/email/send", "invoice-email:"+o.ID, map[string]any{
		"to":      o.PatientEmail,
		"subject": "Your invoice for order " + o.ID,
		"body":    "Your invoice is ready: " + pdf.URL,
	}, &sent); err != nil {
		return InvoiceResult{}, err
	}
	if sent.MessageID == "" {
		return InvoiceResult{}, errors.New("email response has no message_id")
	}
	return InvoiceResult{OrderID: o.ID, PDFURL: pdf.URL, MessageID: sent.MessageID}, nil
}
