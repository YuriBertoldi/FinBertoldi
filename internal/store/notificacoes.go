package store

import (
	"database/sql"
	"fincontrol/internal/models"
	"strings"
	"time"
)

func GetNotificacoes(db *sql.DB, userID int) models.Notificacoes {
	n := models.Notificacoes{DiaEnvio: 1}
	db.QueryRow(`SELECT email_mensal, telegram_mensal, telegram_chat_id, dia_envio, mes_seguinte, ultimo_envio
		FROM user_notificacoes WHERE user_id=$1`, userID).
		Scan(&n.EmailMensal, &n.TelegramMensal, &n.TelegramChatID, &n.DiaEnvio, &n.MesSeguinte, &n.UltimoEnvio)
	return n
}

func SaveNotificacoes(db *sql.DB, userID int, n models.Notificacoes) error {
	_, err := db.Exec(`
		INSERT INTO user_notificacoes (user_id, email_mensal, telegram_mensal, telegram_chat_id, dia_envio, mes_seguinte)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (user_id) DO UPDATE SET email_mensal=$2, telegram_mensal=$3, telegram_chat_id=$4, dia_envio=$5, mes_seguinte=$6`,
		userID, n.EmailMensal, n.TelegramMensal, strings.TrimSpace(n.TelegramChatID), n.DiaEnvio, n.MesSeguinte)
	return err
}

// GetNotificacoesPendentes lista usuários ativos com envio mensal ligado cujo dia é hoje
// (ou já passou, quando o mês não tem o dia escolhido) e que ainda não receberam no mês.
func GetNotificacoesPendentes(db *sql.DB, hoje int, mes time.Time) ([]models.NotificacaoPendente, error) {
	ultimoDia := time.Date(mes.Year(), mes.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	rows, err := db.Query(`
		SELECT u.id, u.nome, u.email, u.family_id,
		       n.email_mensal, n.telegram_mensal, n.telegram_chat_id, n.dia_envio, n.mes_seguinte, n.ultimo_envio
		FROM user_notificacoes n
		JOIN users u ON u.id = n.user_id
		WHERE u.ativo = true
		  AND (n.email_mensal OR n.telegram_mensal)
		  AND (n.dia_envio = $1 OR ($1 = $2 AND n.dia_envio > $2))
		  AND (n.ultimo_envio IS NULL OR n.ultimo_envio < $3::date)`, hoje, ultimoDia, mes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.NotificacaoPendente
	for rows.Next() {
		var p models.NotificacaoPendente
		if err := rows.Scan(&p.User.ID, &p.User.Nome, &p.User.Email, &p.User.FamilyID,
			&p.Notificacoes.EmailMensal, &p.Notificacoes.TelegramMensal, &p.Notificacoes.TelegramChatID,
			&p.Notificacoes.DiaEnvio, &p.Notificacoes.MesSeguinte, &p.Notificacoes.UltimoEnvio); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func MarcarEnvioMensal(db *sql.DB, userID int, mes time.Time) error {
	_, err := db.Exec(`UPDATE user_notificacoes SET ultimo_envio=$2 WHERE user_id=$1`, userID, mes)
	return err
}
