package telegram

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// InlineKeyboardMarkup builders for Telegram bot

// MainMenuKeyboard returns the main menu inline keyboard
func MainMenuKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnStartConversion, "start_conversion"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnMyConversions, "my_conversions"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnHelp, "help"),
			tgbotapi.NewInlineKeyboardButtonData(BtnSettings, "settings"),
		),
	)
}

// StyleSelectionKeyboard returns keyboard for style selection
func StyleSelectionKeyboard() tgbotapi.InlineKeyboardMarkup {
	// Predefined styles - can be fetched from backend in the future
	styles := []string{
		"vintage",
		"casual",
		"formal",
		"streetwear",
		"elegant",
	}

	rows := make([][]tgbotapi.InlineKeyboardButton, 0)
	row := make([]tgbotapi.InlineKeyboardButton, 0)

	for i, style := range styles {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(
			getStyleDisplayName(style),
			"style_"+style,
		))

		// Add 2 buttons per row
		if (i+1)%2 == 0 || i == len(styles)-1 {
			rows = append(rows, row)
			row = make([]tgbotapi.InlineKeyboardButton, 0)
		}
	}

	// Add cancel button
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(BtnCancel, "cancel"),
	})

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// ConversionConfirmationKeyboard returns keyboard for conversion confirmation
func ConversionConfirmationKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnConfirm, "confirm_conversion"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnChangeStyle, "change_style"),
			tgbotapi.NewInlineKeyboardButtonData(BtnCancel, "cancel"),
		),
	)
}

// ConversionResultKeyboard returns keyboard shown after conversion completion
func ConversionResultKeyboard(conversionID string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnFeedback, "feedback_"+conversionID),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnBackToMenu, "main_menu"),
		),
	)
}

// ConversionsListKeyboard returns keyboard for paginated conversions list
func ConversionsListKeyboard(page, totalPages int, conversionID string) tgbotapi.InlineKeyboardMarkup {
	rows := make([][]tgbotapi.InlineKeyboardButton, 0)

	// View result button
	if conversionID != "" {
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(BtnViewResult, "view_conversion_"+conversionID),
		})
	}

	// Pagination buttons
	if totalPages > 1 {
		paginationRow := make([]tgbotapi.InlineKeyboardButton, 0)

		if page > 1 {
			paginationRow = append(paginationRow,
				tgbotapi.NewInlineKeyboardButtonData(BtnPrevious, fmt.Sprintf("conversions_page_%d", page-1)),
			)
		}

		if page < totalPages {
			paginationRow = append(paginationRow,
				tgbotapi.NewInlineKeyboardButtonData(BtnNext, fmt.Sprintf("conversions_page_%d", page+1)),
			)
		}

		if len(paginationRow) > 0 {
			rows = append(rows, paginationRow)
		}
	}

	// Back to menu button
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(BtnBackToMenu, "main_menu"),
	})

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// CancelKeyboard returns a simple cancel button
func CancelKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnCancel, "cancel"),
		),
	)
}

// BackToMenuKeyboard returns a back to menu button
func BackToMenuKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnBackToMenu, "main_menu"),
		),
	)
}

// ShareContactKeyboard returns keyboard with share contact button
func ShareContactKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButtonContact("📱 Share Contact"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("❌ Cancel"),
		),
	)
}

// RemoveKeyboard removes the custom keyboard
func RemoveKeyboard() tgbotapi.ReplyKeyboardRemove {
	return tgbotapi.NewRemoveKeyboard(true)
}

// getStyleDisplayName returns Persian display name for style
func getStyleDisplayName(style string) string {
	styleNames := map[string]string{
		"vintage":    "کلاسیک",
		"casual":     "راحت",
		"formal":     "رسمی",
		"streetwear": "خیابانی",
		"elegant":    "زیبا",
	}

	if name, ok := styleNames[style]; ok {
		return name
	}
	return style
}

