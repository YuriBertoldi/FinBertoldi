package integrations

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"fincontrol/internal/store"
)

// SendTelegramMessage envia uma mensagem via Telegram Bot
func SendTelegramMessage(botToken, chatID, msg string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	resp, err := http.PostForm(apiURL, url.Values{
		"chat_id":    {chatID},
		"text":       {msg},
		"parse_mode": {"HTML"},
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if !result.OK {
		return fmt.Errorf("%s", explicarErroTelegram(resp.StatusCode, result.Description))
	}
	return nil
}

var reTokenTelegram = regexp.MustCompile(`^[0-9]+:[A-Za-z0-9_-]{30,}$`)

// ValidarTokenTelegram confere o formato "123456789:AA..." devolvido pelo BotFather.
func ValidarTokenTelegram(token string) error {
	if !reTokenTelegram.MatchString(token) {
		return fmt.Errorf("token do bot inválido: copie a linha inteira do BotFather, no formato 123456789:AAE... (número, dois-pontos e o código)")
	}
	return nil
}

// explicarErroTelegram traduz as respostas mais comuns da API do Telegram.
func explicarErroTelegram(status int, desc string) string {
	switch {
	case status == http.StatusNotFound || status == http.StatusUnauthorized:
		return "token do bot inválido ou revogado — gere/copie novamente no BotFather"
	case strings.Contains(desc, "chat not found"):
		return "chat não encontrado — confira o Chat ID e envie /start para o bot antes"
	case strings.Contains(desc, "bot was blocked"):
		return "você bloqueou o bot — desbloqueie e envie /start"
	case strings.Contains(desc, "bot can't initiate"):
		return "envie /start para o bot antes de receber mensagens"
	case desc != "":
		return "Telegram recusou: " + desc
	}
	return fmt.Sprintf("Telegram recusou a mensagem (HTTP %d)", status)
}

// SendResumoSemanal envia resumo financeiro semanal via Telegram
func SendResumoSemanal(db *sql.DB, fid int) error {
	botToken := store.GetIntegracaoKV(db, fid, "telegram", "bot_token")
	chatID := store.GetIntegracaoKV(db, fid, "telegram", "chat_id")
	if botToken == "" || chatID == "" {
		return nil
	}

	mes := time.Now().Format("2006-01")
	txns, _ := store.GetTransacoesBanco(db, fid, mes, "", "", "")

	var entradas, saidas float64
	for _, t := range txns {
		if t.Tipo == "credito" {
			entradas += t.Valor
		} else {
			saidas += t.Valor
		}
	}

	msg := fmt.Sprintf(
		"<b>📊 Resumo Semanal - FinBertoldi</b>\n\n"+
			"📅 Mês: %s\n"+
			"💰 Entradas: R$ %.2f\n"+
			"💸 Saídas: R$ %.2f\n"+
			"📈 Saldo: R$ %.2f\n"+
			"📋 Transações: %d",
		mes, entradas, saidas, entradas-saidas, len(txns))

	return SendTelegramMessage(botToken, chatID, msg)
}

// SendAlertaVencimento envia alerta de despesas pendentes
func SendAlertaVencimento(db *sql.DB, fid int) error {
	botToken := store.GetIntegracaoKV(db, fid, "telegram", "bot_token")
	chatID := store.GetIntegracaoKV(db, fid, "telegram", "chat_id")
	if botToken == "" || chatID == "" {
		return nil
	}

	now := time.Now()
	mesStr := now.Format("2006-01")
	mesTime := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	basicas, cartao, vr, _ := store.GetDespesasMes(db, fid, mesTime, mesStr)

	var pendentes int
	var totalPendente float64
	all := append(append(basicas, cartao...), vr...)
	for _, d := range all {
		if !d.Pago {
			pendentes++
			totalPendente += d.Valor
		}
	}

	if pendentes == 0 {
		return nil
	}

	msg := fmt.Sprintf(
		"<b>⚠️ Alerta de Vencimento - FinBertoldi</b>\n\n"+
			"Você tem <b>%d</b> despesa(s) pendente(s) este mês.\n"+
			"Total pendente: <b>R$ %.2f</b>\n\n"+
			"Acesse o sistema para conferir.",
		pendentes, totalPendente)

	return SendTelegramMessage(botToken, chatID, msg)
}

// TestTelegram envia mensagem de teste
func TestTelegram(botToken, chatID string) error {
	return SendTelegramMessage(botToken, chatID, "✅ <b>FinBertoldi</b> conectado com sucesso!")
}

var botUsernameCache sync.Map // token -> username

// TelegramBotUsername devolve o @username do bot (sem "@") consultando getMe uma vez por token.
func TelegramBotUsername(token string) string {
	if token == "" {
		return ""
	}
	if v, ok := botUsernameCache.Load(token); ok {
		return v.(string)
	}
	cli := http.Client{Timeout: 5 * time.Second}
	resp, err := cli.Get(fmt.Sprintf("https://api.telegram.org/bot%s/getMe", token))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var r struct {
		OK     bool `json:"ok"`
		Result struct {
			Username string `json:"username"`
		} `json:"result"`
	}
	if json.NewDecoder(resp.Body).Decode(&r) != nil || !r.OK {
		return ""
	}
	botUsernameCache.Store(token, r.Result.Username)
	return r.Result.Username
}
