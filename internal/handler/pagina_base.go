package handler

import (
	"database/sql"
	"fincontrol/internal/auth"
	"fincontrol/internal/models"
	"fincontrol/internal/store"
	"fmt"
	"net/http"
	"time"
)

func bp(db *sql.DB, r *http.Request, active, title string) models.BasePage {
	u := auth.CurrentUser(r)
	page := models.BasePage{CurrentUser: u, Active: active, Title: title}
	if u != nil && !u.Admin && !u.FamilyAdmin {
		page.BlockedScreens, _ = store.GetBlockedScreens(db, u.ID)
	}
	return page
}

func mesFromRequest(r *http.Request) (time.Time, string) {
	mesStr := r.URL.Query().Get("mes")
	if mesStr == "" {
		mesStr = time.Now().Format("2006-01")
	}
	t, err := time.Parse("2006-01", mesStr)
	if err != nil {
		t = time.Now()
		mesStr = t.Format("2006-01")
	}
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC), mesStr
}

func mesDisplay(s string) string {
	t, err := time.Parse("2006-01", s)
	if err != nil {
		return s
	}
	meses := []string{"", "Janeiro", "Fevereiro", "Março", "Abril", "Maio", "Junho",
		"Julho", "Agosto", "Setembro", "Outubro", "Novembro", "Dezembro"}
	return fmt.Sprintf("%s/%d", meses[t.Month()], t.Year())
}
