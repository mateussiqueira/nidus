package main

import "net/http"

func handleListCronJobs(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectId")
	rows, err := db.Query( "SELECT id, name, schedule, command, status FROM cron_jobs WHERE project_id=$1 ORDER BY name", projectID)
	if err != nil { jsonError(w, "Erro", http.StatusInternalServerError); return }
	defer rows.Close()
	jobs := []map[string]interface{}{}
	for rows.Next() {
		var id, name, schedule, command, status string
		rows.Scan(&id, &name, &schedule, &command, &status)
		jobs = append(jobs, map[string]interface{}{"id":id,"name":name,"schedule":schedule,"command":command,"status":status})
	}
	jsonResponse(w, jobs)
}

func handleCreateCronJob(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]interface{}{"ok":true})
}

func handleDeleteCronJob(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]interface{}{"ok":true})
}
