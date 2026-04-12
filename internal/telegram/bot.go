package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type FinanceBot struct {
	bot    *tgbotapi.BotAPI
	apiURL string
	client *http.Client
}

type UserState struct {
	Action    string
	Data      map[string]interface{}
	LastMsgID int
}

var userStates = make(map[int64]*UserState)

func NewFinanceBot(token, apiURL string) (*FinanceBot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	bot.Debug = true

	return &FinanceBot{
		bot:    bot,
		apiURL: apiURL,
		client: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (fb *FinanceBot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := fb.bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			fb.handleMessage(update.Message)
		} else if update.CallbackQuery != nil {
			fb.handleCallback(update.CallbackQuery)
		}
	}
}

func (fb *FinanceBot) handleMessage(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	// Проверяем команды
	switch text {
	case "/start":
		fb.handleStart(chatID)
		return
	case "/help":
		fb.handleHelp(chatID)
		return
	case "/accounts":
		fb.handleAccounts(chatID)
		return
	case "/income":
		fb.handleIncomeCommand(chatID)
		return
	case "/expense":
		fb.handleExpenseCommand(chatID)
		return
	case "/transfer":
		fb.handleTransferCommand(chatID)
		return
	case "/history":
		fb.handleHistoryCommand(chatID)
		return
	case "/cancel":
		fb.handleCancel(chatID)
		return
	}

	// Обрабатываем состояния диалога
	state, exists := userStates[chatID]
	if !exists {
		return
	}

	switch state.Action {
	case "income_select_account":
		fb.processIncomeAccount(chatID, text, state)
	case "income_input_amount":
		fb.processIncomeAmount(chatID, text, state)
	case "income_input_description":
		fb.processIncomeDescription(chatID, text, state)

	case "expense_select_account":
		fb.processExpenseAccount(chatID, text, state)
	case "expense_input_amount":
		fb.processExpenseAmount(chatID, text, state)
	case "expense_input_description":
		fb.processExpenseDescription(chatID, text, state)

	case "transfer_select_from":
		fb.processTransferFrom(chatID, text, state)
	case "transfer_select_to":
		fb.processTransferTo(chatID, text, state)
	case "transfer_input_amount":
		fb.processTransferAmount(chatID, text, state)
	case "transfer_input_description":
		fb.processTransferDescription(chatID, text, state)

	case "history_select_account":
		fb.processHistory(chatID, text, state)
	}
}

func (fb *FinanceBot) handleCallback(callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID

	// Отвечаем на callback
	callbackResp := tgbotapi.NewCallback(callback.ID, "")
	fb.bot.Send(callbackResp)

	// Обрабатываем как обычное сообщение
	fb.handleMessage(&tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: chatID},
		Text: callback.Data,
	})
}

// ========== ОСНОВНЫЕ КОМАНДЫ ==========

func (fb *FinanceBot) handleStart(chatID int64) {
	text := "💰 *Финансовый бот*\n\n"
	text += "Я помогаю отслеживать финансы. Доступные команды:\n\n"
	text += "/accounts - показать все счета\n"
	text += "/income - добавить доход (пополнение)\n"
	text += "/expense - добавить расход (трату)\n"
	text += "/transfer - перевод между счетами\n"
	text += "/history - история операций\n"
	text += "/cancel - отменить текущее действие\n"
	text += "/help - показать это сообщение"

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	fb.bot.Send(msg)
}

func (fb *FinanceBot) handleHelp(chatID int64) {
	fb.handleStart(chatID)
}

func (fb *FinanceBot) handleCancel(chatID int64) {
	delete(userStates, chatID)
	msg := tgbotapi.NewMessage(chatID, "❌ Действие отменено")
	fb.bot.Send(msg)
}

func (fb *FinanceBot) handleAccounts(chatID int64) {
	resp, err := fb.client.Get(fb.apiURL + "/accounts")
	if err != nil {
		fb.sendError(chatID, "Не удалось получить список счетов")
		return
	}
	defer resp.Body.Close()

	var accounts []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&accounts); err != nil {
		fb.sendError(chatID, "Ошибка обработки данных")
		return
	}

	if len(accounts) == 0 {
		msg := tgbotapi.NewMessage(chatID, "📭 У вас пока нет счетов. Создайте их через API")
		fb.bot.Send(msg)
		return
	}

	text := "🏦 *Ваши счета:*\n\n"
	for _, acc := range accounts {
		name := acc["name"].(string)
		balance := acc["balance"].(float64)
		text += fmt.Sprintf("• %s: %.2f ₽\n", name, balance)
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	fb.bot.Send(msg)
}

func (fb *FinanceBot) handleIncomeCommand(chatID int64) {
	accounts, err := fb.getAccounts()
	if err != nil {
		fb.sendError(chatID, "Не удалось получить список счетов")
		return
	}

	if len(accounts) == 0 {
		msg := tgbotapi.NewMessage(chatID, "❌ Нет доступных счетов")
		fb.bot.Send(msg)
		return
	}

	userStates[chatID] = &UserState{
		Action: "income_select_account",
		Data: map[string]interface{}{
			"accounts": accounts,
		},
	}

	keyboard := fb.buildAccountKeyboard(accounts)
	msg := tgbotapi.NewMessage(chatID, "💰 Выберите счет для пополнения:")
	msg.ReplyMarkup = keyboard
	sentMsg, _ := fb.bot.Send(msg)
	userStates[chatID].LastMsgID = sentMsg.MessageID
}

func (fb *FinanceBot) handleExpenseCommand(chatID int64) {
	accounts, err := fb.getAccounts()
	if err != nil || len(accounts) == 0 {
		fb.sendError(chatID, "Нет доступных счетов")
		return
	}

	userStates[chatID] = &UserState{
		Action: "expense_select_account",
		Data: map[string]interface{}{
			"accounts": accounts,
		},
	}

	keyboard := fb.buildAccountKeyboard(accounts)
	msg := tgbotapi.NewMessage(chatID, "💸 Выберите счет для списания:")
	msg.ReplyMarkup = keyboard
	sentMsg, _ := fb.bot.Send(msg)
	userStates[chatID].LastMsgID = sentMsg.MessageID
}

func (fb *FinanceBot) handleTransferCommand(chatID int64) {
	accounts, err := fb.getAccounts()
	if err != nil || len(accounts) < 2 {
		fb.sendError(chatID, "Для перевода нужно минимум 2 счета")
		return
	}

	userStates[chatID] = &UserState{
		Action: "transfer_select_from",
		Data: map[string]interface{}{
			"accounts": accounts,
		},
	}

	keyboard := fb.buildAccountKeyboard(accounts)
	msg := tgbotapi.NewMessage(chatID, "🔄 Выберите счет ОТКУДА:")
	msg.ReplyMarkup = keyboard
	sentMsg, _ := fb.bot.Send(msg)
	userStates[chatID].LastMsgID = sentMsg.MessageID
}

func (fb *FinanceBot) handleHistoryCommand(chatID int64) {
	accounts, err := fb.getAccounts()
	if err != nil || len(accounts) == 0 {
		fb.sendError(chatID, "Нет доступных счетов")
		return
	}

	userStates[chatID] = &UserState{
		Action: "history_select_account",
		Data: map[string]interface{}{
			"accounts": accounts,
		},
	}

	keyboard := fb.buildAccountKeyboard(accounts)
	msg := tgbotapi.NewMessage(chatID, "📜 Выберите счет для истории:")
	msg.ReplyMarkup = keyboard
	fb.bot.Send(msg)
}

// ========== ПРОЦЕСС ДОХОДА ==========

func (fb *FinanceBot) processIncomeAccount(chatID int64, text string, state *UserState) {
	accountID := fb.extractAccountID(text, state.Data["accounts"].([]map[string]interface{}))
	if accountID == 0 {
		if text == "cancel" {
			delete(userStates, chatID)
			fb.sendMessage(chatID, "Отменено")
			return
		}
		fb.sendMessage(chatID, "Пожалуйста, выберите счет из списка")
		return
	}

	state.Action = "income_input_amount"
	state.Data["account_id"] = accountID

	if state.LastMsgID != 0 {
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, state.LastMsgID)
		fb.bot.Send(deleteMsg)
	}

	msg := tgbotapi.NewMessage(chatID, "💰 Введите сумму пополнения:")
	sentMsg, _ := fb.bot.Send(msg)
	state.LastMsgID = sentMsg.MessageID
}

func (fb *FinanceBot) processIncomeAmount(chatID int64, text string, state *UserState) {
	amount, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil || amount <= 0 {
		fb.sendMessage(chatID, "❌ Введите корректную сумму")
		return
	}

	state.Action = "income_input_description"
	state.Data["amount"] = amount

	msg := tgbotapi.NewMessage(chatID, "📝 Введите описание (например: 'Зарплата'):")
	sentMsg, _ := fb.bot.Send(msg)
	state.LastMsgID = sentMsg.MessageID
}

func (fb *FinanceBot) processIncomeDescription(chatID int64, text string, state *UserState) {
	accountID := state.Data["account_id"].(int)
	amount := state.Data["amount"].(float64)
	description := text

	reqBody := map[string]interface{}{
		"account_id":  accountID,
		"amount":      amount,
		"description": description,
		"date":        time.Now().Format("2006-01-02"),
	}

	jsonData, _ := json.Marshal(reqBody)
	resp, err := fb.client.Post(fb.apiURL+"/transactions/income", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fb.sendError(chatID, "Ошибка при сохранении")
		delete(userStates, chatID)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		fb.sendError(chatID, "Ошибка при сохранении")
		delete(userStates, chatID)
		return
	}

	fb.sendSuccess(chatID, fmt.Sprintf("💰 Доход %.2f ₽ добавлен!\nОписание: %s", amount, description))
	delete(userStates, chatID)
}

// ========== ПРОЦЕСС РАСХОДА ==========

func (fb *FinanceBot) processExpenseAccount(chatID int64, text string, state *UserState) {
	accountID := fb.extractAccountID(text, state.Data["accounts"].([]map[string]interface{}))
	if accountID == 0 {
		if text == "cancel" {
			delete(userStates, chatID)
			fb.sendMessage(chatID, "Отменено")
			return
		}
		fb.sendMessage(chatID, "Выберите счет из списка")
		return
	}

	state.Action = "expense_input_amount"
	state.Data["account_id"] = accountID

	if state.LastMsgID != 0 {
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, state.LastMsgID)
		fb.bot.Send(deleteMsg)
	}

	msg := tgbotapi.NewMessage(chatID, "💸 Введите сумму расхода:")
	sentMsg, _ := fb.bot.Send(msg)
	state.LastMsgID = sentMsg.MessageID
}

func (fb *FinanceBot) processExpenseAmount(chatID int64, text string, state *UserState) {
	amount, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil || amount <= 0 {
		fb.sendMessage(chatID, "❌ Введите корректную сумму")
		return
	}

	state.Action = "expense_input_description"
	state.Data["amount"] = amount

	msg := tgbotapi.NewMessage(chatID, "📝 Что купили?")
	sentMsg, _ := fb.bot.Send(msg)
	state.LastMsgID = sentMsg.MessageID
}

func (fb *FinanceBot) processExpenseDescription(chatID int64, text string, state *UserState) {
	accountID := state.Data["account_id"].(int)
	amount := state.Data["amount"].(float64)
	description := text

	reqBody := map[string]interface{}{
		"account_id":  accountID,
		"amount":      amount,
		"description": description,
		"date":        time.Now().Format("2006-01-02"),
	}

	jsonData, _ := json.Marshal(reqBody)
	resp, err := fb.client.Post(fb.apiURL+"/transactions/expense", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fb.sendError(chatID, "Ошибка при сохранении")
		delete(userStates, chatID)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		fb.sendError(chatID, "Недостаточно средств или ошибка")
		delete(userStates, chatID)
		return
	}

	fb.sendSuccess(chatID, fmt.Sprintf("💸 Расход %.2f ₽ записан!\nОписание: %s", amount, description))
	delete(userStates, chatID)
}

// ========== ПРОЦЕСС ПЕРЕВОДА ==========

func (fb *FinanceBot) processTransferFrom(chatID int64, text string, state *UserState) {
	fromID := fb.extractAccountID(text, state.Data["accounts"].([]map[string]interface{}))
	if fromID == 0 {
		if text == "cancel" {
			delete(userStates, chatID)
			fb.sendMessage(chatID, "Отменено")
			return
		}
		fb.sendMessage(chatID, "Выберите счет из списка")
		return
	}

	state.Action = "transfer_select_to"
	state.Data["from_account_id"] = fromID

	if state.LastMsgID != 0 {
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, state.LastMsgID)
		fb.bot.Send(deleteMsg)
	}

	accounts := state.Data["accounts"].([]map[string]interface{})
	var filteredAccounts []map[string]interface{}
	for _, acc := range accounts {
		if int(acc["id"].(float64)) != fromID {
			filteredAccounts = append(filteredAccounts, acc)
		}
	}

	keyboard := fb.buildAccountKeyboard(filteredAccounts)
	msg := tgbotapi.NewMessage(chatID, "🔄 Выберите счет КУДА:")
	msg.ReplyMarkup = keyboard
	sentMsg, _ := fb.bot.Send(msg)
	state.LastMsgID = sentMsg.MessageID
	state.Data["filtered_accounts"] = filteredAccounts
}

func (fb *FinanceBot) processTransferTo(chatID int64, text string, state *UserState) {
	toID := fb.extractAccountID(text, state.Data["filtered_accounts"].([]map[string]interface{}))
	if toID == 0 {
		fb.sendMessage(chatID, "Выберите счет из списка")
		return
	}

	state.Action = "transfer_input_amount"
	state.Data["to_account_id"] = toID

	if state.LastMsgID != 0 {
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, state.LastMsgID)
		fb.bot.Send(deleteMsg)
	}

	msg := tgbotapi.NewMessage(chatID, "💰 Введите сумму перевода:")
	sentMsg, _ := fb.bot.Send(msg)
	state.LastMsgID = sentMsg.MessageID
}

func (fb *FinanceBot) processTransferAmount(chatID int64, text string, state *UserState) {
	amount, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil || amount <= 0 {
		fb.sendMessage(chatID, "Введите корректную сумму")
		return
	}

	state.Action = "transfer_input_description"
	state.Data["amount"] = amount

	msg := tgbotapi.NewMessage(chatID, "📝 Назначение перевода:")
	sentMsg, _ := fb.bot.Send(msg)
	state.LastMsgID = sentMsg.MessageID
}

func (fb *FinanceBot) processTransferDescription(chatID int64, text string, state *UserState) {
	reqBody := map[string]interface{}{
		"from_account_id": state.Data["from_account_id"].(int),
		"to_account_id":   state.Data["to_account_id"].(int),
		"amount":          state.Data["amount"].(float64),
		"description":     text,
	}

	jsonData, _ := json.Marshal(reqBody)
	resp, err := fb.client.Post(fb.apiURL+"/transactions/transfer", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fb.sendError(chatID, "Ошибка при переводе")
		delete(userStates, chatID)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		fb.sendError(chatID, "Ошибка: недостаточно средств")
		delete(userStates, chatID)
		return
	}

	fb.sendSuccess(chatID, fmt.Sprintf("✅ Перевод %.2f ₽ выполнен!", state.Data["amount"].(float64)))
	delete(userStates, chatID)
}

// ========== ИСТОРИЯ ==========

func (fb *FinanceBot) processHistory(chatID int64, text string, state *UserState) {
	accountID := fb.extractAccountID(text, state.Data["accounts"].([]map[string]interface{}))
	if accountID == 0 {
		if text == "cancel" {
			delete(userStates, chatID)
			fb.sendMessage(chatID, "Отменено")
			return
		}
		fb.sendMessage(chatID, "Выберите счет")
		return
	}

	resp, err := fb.client.Get(fmt.Sprintf("%s/transactions?account_id=%d&limit=10", fb.apiURL, accountID))
	if err != nil {
		fb.sendError(chatID, "Ошибка получения истории")
		delete(userStates, chatID)
		return
	}
	defer resp.Body.Close()

	var transactions []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&transactions)

	if len(transactions) == 0 {
		fb.sendMessage(chatID, "📭 Нет операций по этому счету")
		delete(userStates, chatID)
		return
	}

	textResult := "📜 *Последние операции:*\n\n"
	for _, t := range transactions {
		date := t["date"].(string)
		amount := t["amount"].(float64)
		desc := t["description"].(string)

		var sign string
		fromID := t["from_account_id"]
		toID := t["to_account_id"]

		if fromID != nil && toID == nil {
			sign = fmt.Sprintf("-%.2f ₽", amount)
		} else if fromID == nil && toID != nil {
			sign = fmt.Sprintf("+%.2f ₽", amount)
		} else {
			sign = fmt.Sprintf("↺ %.2f ₽", amount)
		}

		textResult += fmt.Sprintf("• %s: %s - %s\n", date[:10], sign, desc)
	}

	msg := tgbotapi.NewMessage(chatID, textResult)
	msg.ParseMode = "Markdown"
	fb.bot.Send(msg)
	delete(userStates, chatID)
}

// ========== ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ==========

func (fb *FinanceBot) getAccounts() ([]map[string]interface{}, error) {
	resp, err := fb.client.Get(fb.apiURL + "/accounts")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var accounts []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&accounts); err != nil {
		return nil, err
	}
	return accounts, nil
}

func (fb *FinanceBot) buildAccountKeyboard(accounts []map[string]interface{}) *tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	for _, acc := range accounts {
		id := int(acc["id"].(float64))
		name := acc["name"].(string)

		button := tgbotapi.NewInlineKeyboardButtonData(name, fmt.Sprintf("acc_%d", id))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(button))
	}

	cancelBtn := tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cancel")
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(cancelBtn))

	return &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func (fb *FinanceBot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	fb.bot.Send(msg)
}

func (fb *FinanceBot) sendError(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, "❌ "+text)
	fb.bot.Send(msg)
}

func (fb *FinanceBot) sendSuccess(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, "✅ "+text)
	fb.bot.Send(msg)
}

func (fb *FinanceBot) extractAccountID(text string, accounts []map[string]interface{}) int {
	if strings.HasPrefix(text, "acc_") {
		idStr := strings.TrimPrefix(text, "acc_")
		id, _ := strconv.Atoi(idStr)
		return id
	}

	for _, acc := range accounts {
		if strings.EqualFold(text, acc["name"].(string)) {
			return int(acc["id"].(float64))
		}
	}
	return 0
}
