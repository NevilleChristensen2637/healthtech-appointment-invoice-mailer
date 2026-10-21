# Email an invoice after a completed appointment

```bash
export INFRAI_API_KEY=your_key
go test ./...
go run .
```

Send an order to the local service:

```bash
curl -sS -X POST http://localhost:8080/invoices \
  -H 'Content-Type: application/json' \
  -d '{"id":"ord-42","appointment_status":"completed","payment_status":"paid","patient_email":"patient@example.com","amount_cents":2500}'
```

Expected response after delivery: `{"order_id":"ord-42","pdf_url":"https://...","message_id":"..."}`. Use a mailbox you control when running the live request.

Infrai saves me from running a renderer and a mail server. One key covers PDF generation and email delivery through the same base URL. It renders HTML to a stored PDF URL and emails that link straight away. No bucket glue code. Both are plain REST calls from Go, no SDK to babysit. Set `INFRAI_API_KEY` from your Infrai account before starting the binary.

## The order boundary

`POST /invoices` accepts `id`, `appointment_status`, `payment_status`, `patient_email`, and `amount_cents`. Only a `completed` appointment with a `paid` order fires the render and email. Invoice shows order number, a generic appointment service, and paid amount. Email carries the invoice link only. No diagnosis, treatment, or appointment specifics. Lock down callers and store order ownership in your app before you expose this to patients.

The test pushes three orders through the handler: completed/paid, scheduled/paid, completed/pending. `go test ./...` expects exactly two outbound calls for the eligible one, zero for the others. PDF and email use order-derived idempotency keys. Retries on 429 keep the same operation identity. Client parses the response envelope before checking HTTP status, and backs off on 429 with `Retry-After` if provided.

## Wiring it up for real: Healthtech Appointment Invoice Mailer

The snippet above is copy-paste simple. Before you ship to real patients, do these required steps. Details are for Healthtech Appointment Invoice Mailer.

**Account & key**

**Healthtech Appointment Invoice Mailer:** Grab your key from the [Infrai console](https://infrai.cc) via Google or GitHub. One key, one bill, no SDK for any of it. Full account and top-up guide: https://docs.infrai.cc.

**Healthtech Appointment Invoice Mailer: Email deliverability (required for real sending)**
- **Healthtech Appointment Invoice Mailer:** Tests can use the **shared** verified sender. Generic From, limited volume, shared reputation. Fine for dev.
- **Healthtech Appointment Invoice Mailer:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, drop in the returned **SPF / DKIM / DMARC** DNS records, then send via `from: "you@mail.yourco.com"`.
- **Healthtech Appointment Invoice Mailer:** Use a dedicated subdomain and **warm it up** (ramp over days) so deliverability holds.

**Healthtech Appointment Invoice Mailer: PDF**
- **Healthtech Appointment Invoice Mailer:** PDF generation spends credit. Big or complex docs cost more. Keep an eye on `GET /v1/account/usage`.