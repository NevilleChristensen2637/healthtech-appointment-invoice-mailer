package main

import (
	"fmt"
	"html"
	"net/mail"
	"strings"
)

type Order struct {
	ID                string `json:"id"`
	AppointmentStatus string `json:"appointment_status"`
	PaymentStatus     string `json:"payment_status"`
	PatientEmail      string `json:"patient_email"`
	AmountCents       int64  `json:"amount_cents"`
}

func (o Order) validate() error {
	if strings.TrimSpace(o.ID) == "" || strings.ContainsAny(o.ID, "\r\n") {
		return fmt.Errorf("invalid order id")
	}
	if o.AppointmentStatus != "completed" || o.PaymentStatus != "paid" {
		return fmt.Errorf("invoice requires a completed appointment and paid order")
	}
	address, err := mail.ParseAddress(o.PatientEmail)
	if err != nil || address.Address != o.PatientEmail {
		return fmt.Errorf("invalid patient email")
	}
	if o.AmountCents <= 0 {
		return fmt.Errorf("amount_cents must be positive")
	}
	return nil
}

func invoiceHTML(o Order) string {
	return fmt.Sprintf(`<!doctype html><html><head><meta charset="utf-8"><title>Invoice</title></head><body><h1>Invoice</h1><p>Order %s</p><p>Appointment service</p><p>Paid: $%d.%02d</p></body></html>`, html.EscapeString(o.ID), o.AmountCents/100, o.AmountCents%100)
}

type InvoiceResult struct {
	OrderID   string `json:"order_id"`
	PDFURL    string `json:"pdf_url"`
	MessageID string `json:"message_id"`
}
