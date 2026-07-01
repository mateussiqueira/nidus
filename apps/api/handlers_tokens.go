package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/google/uuid"
	"log"
	"net/http"
	"strings"
	"time"
)

func handleListTokens(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)
	rows, err := db.QueryContext(r.Context(), "SELECT id, name, token, created_at FROM api_tokens WHERE user_id=$1 ORDER BY created_at DESC", userID)
	if err != nil {
		jsonError(w, "Erro", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	tokens := []map[string]interface{}{}
	for rows.Next() {
		var id, name, token, createdAt string
		rows.Scan(&id, &name, &token, &createdAt)
		if len(token) > 12 {
			token = token[:8] + "..." + token[len(token)-4:]
		}
		tokens = append(tokens, map[string]interface{}{"id": id, "name": name, "token": token, "createdAt": createdAt})
	}
	jsonResponse(w, tokens)
}

func generateAPIToken() (string, string) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		log.Fatalf("failed to generate token: %v", err)
	}
	token := "stk_" + hex.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	return token, hex.EncodeToString(hash[:])
}

func handleCreateToken(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "Body invalido", http.StatusBadRequest)
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" {
		jsonError(w, "name eh obrigatorio", http.StatusBadRequest)
		return
	}
	if len(body.Name) > 64 {
		jsonError(w, "name deve ter no maximo 64 caracteres", http.StatusBadRequest)
		return
	}

	var count int
	db.QueryRowContext(r.Context(), "SELECT count(*) FROM api_tokens WHERE user_id=$1", userID).Scan(&count)
	if count >= 10 {
		jsonError(w, "Maximo de 10 tokens por conta", http.StatusBadRequest)
		return
	}

	id := uuid.New().String()
	rawToken, hashedToken := generateAPIToken()
	now := time.Now()

	_, err := db.ExecContext(r.Context(),
		`INSERT INTO api_tokens (id, user_id, name, token, created_at, last_used_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		id, userID, body.Name, hashedToken, now, now)
	if err != nil {
		log.Printf("[tokens] insert error: %v", err)
		jsonError(w, "Erro ao criar token", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"id":        id,
		"name":      body.Name,
		"token":     rawToken,
		"createdAt": now.Format(time.RFC3339),
	})
}

func handleDeleteToken(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)
	tokenID := r.PathValue("tokenId")
	if tokenID == "" {
		jsonError(w, "tokenId eh obrigatorio", http.StatusBadRequest)
		return
	}

	result, err := db.ExecContext(r.Context(),
		"DELETE FROM api_tokens WHERE id=$1 AND user_id=$2", tokenID, userID)
	if err != nil {
		log.Printf("[tokens] delete error: %v", err)
		jsonError(w, "Erro ao deletar token", http.StatusInternalServerError)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		jsonError(w, "Token nao encontrado", http.StatusNotFound)
		return
	}

	jsonResponse(w, map[string]interface{}{"ok": true})
}
