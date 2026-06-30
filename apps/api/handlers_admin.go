package main

import (
	"net/http"
)

func handleAdminStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{}

	var count int
	db.QueryRow("SELECT count(*) FROM users").Scan(&count)
	stats["totalUsers"] = count

	db.QueryRow("SELECT count(*) FROM projects").Scan(&count)
	stats["totalProjects"] = count

	db.QueryRow("SELECT count(*) FROM deployments").Scan(&count)
	stats["totalDeploys"] = count

	db.QueryRow("SELECT count(*) FROM databases").Scan(&count)
	stats["totalDatabases"] = count

	db.QueryRow("SELECT count(*) FROM domains WHERE verified=true").Scan(&count)
	stats["activeDomains"] = count

	// Active subscriptions
	db.QueryRow("SELECT count(*) FROM subscriptions WHERE status=active").Scan(&count)
	stats["activeSubscriptions"] = count

	// Revenue (MRR estimado)
	var revenue int
	db.QueryRow("SELECT COALESCE(SUM(p.amount_cents),0) FROM payments p WHERE p.status=paid").Scan(&revenue)
	stats["totalRevenueCents"] = revenue

	// Users by plan
	rows, _ := db.Query("SELECT plan_id, count(*) FROM users GROUP BY plan_id ORDER BY count(*) DESC")
	defer rows.Close()
	byPlan := map[string]int{}
	for rows.Next() {
		var plan string
		var n int
		rows.Scan(&plan, &n)
		byPlan[plan] = n
	}
	stats["usersByPlan"] = byPlan

	// Recent signups (last 7 days)
	db.QueryRow("SELECT count(*) FROM users WHERE created_at > NOW() - INTERVAL 7 days").Scan(&count)
	stats["newUsers7d"] = count

	// Recent deploys (last 24h)
	db.QueryRow("SELECT count(*) FROM deployments WHERE created_at > NOW() - INTERVAL 1 day").Scan(&count)
	stats["deploys24h"] = count

	// Health checker status
	db.QueryRow("SELECT count(*) FROM health_checks WHERE status=up AND checked_at > NOW() - INTERVAL 5 minutes").Scan(&count)
	stats["healthyProjects"] = count

	jsonResponse(w, stats)
}

func handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, email, role, plan_id, subscription_status, created_at FROM users ORDER BY created_at DESC LIMIT 50")
	if err != nil { jsonError(w, "Erro", http.StatusInternalServerError); return }
	defer rows.Close()
	users := []map[string]interface{}{}
	for rows.Next() {
		var id, name, email, role, plan, status, created string
		rows.Scan(&id, &name, &email, &role, &plan, &status, &created)
		users = append(users, map[string]interface{}{
			"id":id,"name":name,"email":email,"role":role,
			"plan":plan,"subscription":status,"createdAt":created,
		})
	}
	jsonResponse(w, users)
}

func handleAdminPayments(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT p.id, u.email, pl.name, p.gateway, p.amount_cents, p.status, p.payment_method, p.created_at FROM payments p JOIN users u ON p.user_id=u.id JOIN plans pl ON p.plan_id=pl.id ORDER BY p.created_at DESC LIMIT 30")
	if err != nil { jsonError(w, "Erro", http.StatusInternalServerError); return }
	defer rows.Close()
	payments := []map[string]interface{}{}
	for rows.Next() {
		var id, email, plan, gateway, status, method, created string
		var amount int
		rows.Scan(&id, &email, &plan, &gateway, &amount, &status, &method, &created)
		payments = append(payments, map[string]interface{}{
			"id":id,"user":email,"plan":plan,"gateway":gateway,
			"amount":amount,"status":status,"method":method,"createdAt":created,
		})
	}
	jsonResponse(w, payments)
}

func handleAdminAudit(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT a.action, a.resource, a.resource_id, u.email, a.ip_address, a.created_at FROM audit_logs a JOIN users u ON a.user_id=u.id ORDER BY a.created_at DESC LIMIT 50")
	if err != nil { jsonError(w, "Erro", http.StatusInternalServerError); return }
	defer rows.Close()
	logs := []map[string]interface{}{}
	for rows.Next() {
		var action, resource, resourceID, email, ip, created string
		rows.Scan(&action, &resource, &resourceID, &email, &ip, &created)
		logs = append(logs, map[string]interface{}{
			"action":action,"resource":resource,"resourceId":resourceID,
			"user":email,"ip":ip,"createdAt":created,
		})
	}
	jsonResponse(w, logs)
}
