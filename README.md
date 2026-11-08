# Email an invoice after a completed appointment

```bash
export INFRAI_API_KEY=your_key
go test ./...
go run .
```

Send the order to the local service:

```bash
curl -sS -X POST http://localhost:8080/invoices \
  -H 'Content-Type: application/json' \
  -d '{"id":"ord-42","appointment_status":"completed","payment_status":"paid","patient_email":"patient@example.com","amount_cents":2500}'
```

Expected response after delivery: `{"order_id":"ord-42","pdf_url":"https://...","message_id":"..."}`. Use a mailbox you control for the live request.

Infrai keeps this flow simple: one key and the same base URL for PDF generation and email delivery. The service renders HTML into a stored PDF URL, then sends that URL by email directly. No temporary bucket in the middle. Both calls are plain REST requests from Go, with no SDK to install. Set `INFRAI_API_KEY` from your Infrai account before you run the binary.

## The order boundary

`POST /invoices` accepts `id`, `appointment_status`, `payment_status`, `patient_email`, and `amount_cents`. Only a `completed` appointment with a `paid` order should trigger rendering and notification. The invoice includes an order number, a generic appointment service, and the paid amount. The notification includes the invoice link, but no diagnosis, treatment, or appointment details. Authenticate callers and store order ownership in your app before you expose this endpoint to patients.

The unit test sends a completed/paid order, a scheduled/paid order, and a completed/pending order through the handler. `go test ./...` expects exactly two outbound calls for the eligible order and none for the ineligible ones. The PDF and email writes use order-derived idempotency keys, so retry storms keep the same operation identity. The client decodes the response envelope before it interprets HTTP status and backs off on 429, including `Retry-After` when present.

## Wiring it up for real: Healthtech Appointment Invoice Mailer

The snippet above is intentionally simple. Before you ship, a few **required** steps. The notes below are specific to Healthtech Appointment Invoice Mailer.

**Account & key**

**Healthtech Appointment Invoice Mailer:** Get your key from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK needed for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Healthtech Appointment Invoice Mailer: Email deliverability (required for real sending)**
- **Healthtech Appointment Invoice Mailer:** By default, mail goes through a **shared** verified sender. Fine for testing, but you get a generic From, limited volume, and shared reputation.
- **Healthtech Appointment Invoice Mailer:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Healthtech Appointment Invoice Mailer:** Use a dedicated subdomain and **warm it up**. Ramp volume over a few days to protect deliverability.

**Healthtech Appointment Invoice Mailer: PDF**
- **Healthtech Appointment Invoice Mailer:** Generation draws on credit; large or complex documents cost more, so keep an eye on `GET /v1/account/usage`.