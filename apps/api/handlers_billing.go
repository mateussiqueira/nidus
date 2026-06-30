package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/google/uuid"
)

func handleBillingCheckout(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)
	var body struct {
		PlanID        string `json:"planId"`
		PaymentMethod string `json:"paymentMethod"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.PlanID == "" { body.PlanID = "cloud" }
	if body.PaymentMethod == "" { body.PaymentMethod = "pix" }

	var priceCents int
	var planName string
	db.QueryRow("SELECT price_cents, name FROM plans WHERE id=$1", body.PlanID).Scan(&priceCents, &planName)
	if body.PaymentMethod == "pix" { priceCents = priceCents * 75 / 100 }

	paymentID := uuid.New().String()
	gateway := "stripe"
	country := r.Header.Get("X-Country")
	if country == "" { country = "BR" }
	if country == "BR" { gateway = "abacatepay" }

	db.Exec(
		"INSERT INTO payments (id, user_id, plan_id, gateway, amount_cents, status, payment_method) VALUES ($1,$2,$3,$4,$5,$6,$7)",
		paymentID, userID, body.PlanID, gateway, priceCents, "pending", body.PaymentMethod,
	)

	checkoutURL := "https://stackrun.vercel.app/dashboard/billing?payment=" + paymentID + "&status=pending"
	jsonResponse(w, map[string]interface{}{
		"paymentId":   paymentID,
		"gateway":     gateway,
		"amountCents": priceCents,
		"plan":        planName,
		"checkoutUrl": checkoutURL,
	}, http.StatusCreated)
}

func handleBillingWebhook(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	webhookSecret := os.Getenv("WEBHOOK_SECRET")
	sigHeader := r.Header.Get("X-Webhook-Signature")

	if webhookSecret != "" && sigHeader != "" && sigHeader != webhookSecret {
		jsonError(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	var webhook struct {
		PaymentID string `json:"paymentId"`
		Status    string `json:"status"`
		GatewayID string `json:"gatewayId"`
	}
	json.Unmarshal(bodyBytes, &webhook)

	if webhook.Status == "paid" || webhook.Status == "confirmed" {
		db.Exec("UPDATE payments SET status=paid, gateway_id=$1, paid_at=NOW() WHERE id=$2", webhook.GatewayID, webhook.PaymentID)
		var userID, planID string
		db.QueryRow("SELECT user_id, plan_id FROM payments WHERE id=$1", webhook.PaymentID).Scan(&userID, &planID)
		db.Exec("UPDATE users SET plan_id=$1, subscription_status=active WHERE id=$2", planID, userID)
		db.Exec("INSERT INTO subscriptions (user_id, plan_id, status, current_period_start, current_period_end) VALUES ($1,$2,active,NOW(),NOW()+INTERVAL 365 days) ON CONFLICT DO NOTHING", userID, planID)
	}

	jsonResponse(w, map[string]interface{}{"ok": true})
}
