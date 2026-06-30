package main

import "net/http"

func handleListTokens(w http.ResponseWriter, r *http.Request) {
        userID := r.Context().Value("userID").(string)
        rows, err := db.QueryContext(r.Context(), "SELECT id, name, token, created_at FROM api_tokens WHERE user_id=$1 ORDER BY created_at DESC", userID)
        if err != nil { jsonError(w, "Erro", http.StatusInternalServerError); return }
        defer rows.Close()
        tokens := []map[string]interface{}{}
        for rows.Next() {
                var id, name, token, createdAt string
                rows.Scan(&id, &name, &token, &createdAt)
                if len(token) > 12 { token = token[:8] + "..." + token[len(token)-4:] }
                tokens = append(tokens, map[string]interface{}{"id":id,"name":name,"token":token,"createdAt":createdAt})
        }
        jsonResponse(w, tokens)
}

func handleCreateToken(w http.ResponseWriter, r *http.Request) { jsonResponse(w, map[string]interface{}{"ok":true}) }
func handleDeleteToken(w http.ResponseWriter, r *http.Request) { jsonResponse(w, map[string]interface{}{"ok":true}) }
