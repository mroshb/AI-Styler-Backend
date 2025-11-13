package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Handlers handles all Telegram bot commands and messages
type Handlers struct {
	bot           *tgbotapi.BotAPI
	apiClient     *APIClient
	sessionMgr    *SessionManager
	rateLimiter   *RateLimiter
	config        *Config
}

// NewHandlers creates a new handlers instance
func NewHandlers(bot *tgbotapi.BotAPI, apiClient *APIClient, sessionMgr *SessionManager, rateLimiter *RateLimiter, config *Config) *Handlers {
	return &Handlers{
		bot:         bot,
		apiClient:   apiClient,
		sessionMgr:  sessionMgr,
		rateLimiter: rateLimiter,
		config:      config,
	}
}

// SetBot sets the bot API instance
func (h *Handlers) SetBot(bot *tgbotapi.BotAPI) {
	h.bot = bot
}

// HandleMessage handles incoming messages
func (h *Handlers) HandleMessage(msg *tgbotapi.Message) {
	ctx := context.Background()
	userID := msg.From.ID

	// Rate limiting
	allowed, err := h.rateLimiter.AllowUserMessage(ctx, userID, h.config.RateLimit.MessagesPerMinute, time.Minute)
	if err != nil {
		log.Printf("Rate limit check error: %v", err)
	}
	if !allowed {
		h.sendMessage(msg.Chat.ID, MsgErrorRateLimit)
		RecordRateLimitHit("message")
		return
	}

	// Handle commands
	if msg.IsCommand() {
		h.handleCommand(msg)
		return
	}

	// Handle text messages
	if msg.Text != "" {
		h.handleTextMessage(msg)
		return
	}

	// Handle photos
	if msg.Photo != nil && len(msg.Photo) > 0 {
		h.handlePhoto(msg)
		return
	}

	// Handle documents
	if msg.Document != nil {
		h.handleDocument(msg)
		return
	}
}

// handleCommand handles bot commands
func (h *Handlers) handleCommand(msg *tgbotapi.Message) {
	command := msg.Command()
	chatID := msg.Chat.ID

	switch command {
	case "start":
		h.handleStartCommand(msg)
	case "help":
		h.sendMessage(chatID, MsgHelp)
	default:
		h.sendMessage(chatID, "دستور نامعتبر است. از /help برای راهنما استفاده کنید.")
	}
}

// handleStartCommand handles /start command
func (h *Handlers) handleStartCommand(msg *tgbotapi.Message) {
	ctx := context.Background()
	userID := msg.From.ID
	chatID := msg.Chat.ID

	// Get or create session
	_, err := h.sessionMgr.GetSession(ctx, userID)
	if err != nil {
		log.Printf("Failed to get session: %v", err)
		h.sendMessage(chatID, MsgErrorGeneric)
		return
	}

	// Check if authenticated
	authenticated, err := h.sessionMgr.IsAuthenticated(ctx, userID)
	if err != nil {
		log.Printf("Failed to check authentication: %v", err)
	}

	if authenticated {
		h.sendMessageWithKeyboard(chatID, MsgWelcomeBack, MainMenuKeyboard())
	} else {
		// User not authenticated - show welcome and prompt for login
		h.sendMessageWithKeyboard(chatID, MsgWelcome+"\n\n"+MsgPleaseLogin, MainMenuKeyboard())
		// Set state to wait for phone number if user wants to start conversion
		// This will be handled when they click "start_conversion"
	}
}

// handleTextMessage handles text messages
func (h *Handlers) handleTextMessage(msg *tgbotapi.Message) {
	ctx := context.Background()
	userID := msg.From.ID
	chatID := msg.Chat.ID
	text := msg.Text

	// Get user state
	state, err := h.sessionMgr.GetState(ctx, userID)
	if err != nil {
		log.Printf("Failed to get state: %v", err)
		h.sendMessage(chatID, "لطفاً از منو استفاده کنید.")
		return
	}

	if state == nil {
		h.sendMessage(chatID, "لطفاً از منو استفاده کنید.")
		return
	}

	switch state.Action {
	case "waiting_phone":
		h.handlePhoneInput(msg, text)
	case "waiting_otp":
		h.handleOTPInput(msg, text)
	case "waiting_password":
		h.handlePasswordInput(msg, text)
	default:
		h.sendMessage(chatID, "لطفاً از منو استفاده کنید.")
	}
}

// handlePhoneInput handles phone number input
func (h *Handlers) handlePhoneInput(msg *tgbotapi.Message, phone string) {
	ctx := context.Background()
	userID := msg.From.ID
	chatID := msg.Chat.ID

	// Normalize phone number
	phone = normalizePhone(phone)
	if phone == "" {
		h.sendMessage(chatID, "شماره تلفن نامعتبر است. لطفاً دوباره وارد کنید:")
		return
	}

	// Send OTP
	resp, err := h.apiClient.SendOTP(ctx, phone)
	if err != nil {
		log.Printf("Failed to send OTP: %v", err)
		h.sendMessage(chatID, MsgErrorGeneric)
		return
	}

	if !resp.Sent {
		h.sendMessage(chatID, "ارسال کد تأیید با خطا مواجه شد. لطفاً دوباره تلاش کنید.")
		return
	}

	// Update state
	h.sessionMgr.SetState(ctx, userID, "waiting_otp", phone)
	h.sendMessage(chatID, MsgOTPSent)
}

// handleOTPInput handles OTP code input
func (h *Handlers) handleOTPInput(msg *tgbotapi.Message, code string) {
	ctx := context.Background()
	userID := msg.From.ID
	chatID := msg.Chat.ID

	// Get phone from state
	state, err := h.sessionMgr.GetState(ctx, userID)
	if err != nil || state == nil {
		h.sendMessage(chatID, "خطا در دریافت اطلاعات. لطفاً دوباره شروع کنید.")
		return
	}

	phone := state.Data
	if len(code) != 6 {
		h.sendMessage(chatID, "کد باید ۶ رقمی باشد. لطفاً دوباره وارد کنید:")
		return
	}

	// Verify OTP
	resp, err := h.apiClient.VerifyOTP(ctx, phone, code)
	if err != nil {
		log.Printf("Failed to verify OTP: %v", err)
		h.sendMessage(chatID, MsgOTPInvalid)
		return
	}

	if !resp.Verified {
		h.sendMessage(chatID, MsgOTPInvalid)
		return
	}

	// Check if user exists - try to auto-register first
	// In production, you might want to ask for password or use phone-only auth
	defaultPassword := generateDefaultPassword(phone)
	userName := msg.From.FirstName
	if msg.From.LastName != "" {
		userName += " " + msg.From.LastName
	}

	registerReq := RegisterRequest{
		Phone:    phone,
		Password: defaultPassword,
		Name:     userName,
		Role:     "user",
	}

	registerResp, err := h.apiClient.Register(ctx, registerReq)
	if err != nil {
		// User might already exist, try login with default password
		loginResp, loginErr := h.apiClient.Login(ctx, phone, defaultPassword)
		if loginErr != nil {
			// If login fails, user might have changed password
			// For now, ask user to register via web/app
			log.Printf("Failed to register/login: %v, %v", err, loginErr)
			h.sendMessage(chatID, "کاربری با این شماره وجود دارد اما رمز عبور متفاوت است. لطفاً از طریق وب‌سایت یا اپلیکیشن وارد شوید.")
			h.sessionMgr.ClearState(ctx, userID)
			return
		}

		// Update session with login response
		session, _ := h.sessionMgr.GetSession(ctx, userID)
		if session != nil {
			userIDStr := loginResp.User.ID
			accessToken := loginResp.AccessToken
			refreshToken := loginResp.RefreshToken
			session.BackendUserID = &userIDStr
			session.Phone = &phone
			session.AccessToken = &accessToken
			session.RefreshToken = &refreshToken
			h.sessionMgr.UpdateSession(ctx, session)

			// Store tokens in Redis
			ttl := time.Duration(loginResp.AccessExpiresIn) * time.Second
			_ = h.sessionMgr.GetStorage().StoreToken(ctx, userID, accessToken, refreshToken, ttl)
		}

		h.sessionMgr.ClearState(ctx, userID)
		h.sendMessageWithKeyboard(chatID, MsgLoginSuccess, MainMenuKeyboard())
		return
	}

	// Update session
	session, _ := h.sessionMgr.GetSession(ctx, userID)
	if session != nil {
		userIDStr := registerResp.UserID
		accessToken := registerResp.AccessToken
		refreshToken := registerResp.RefreshToken
		session.BackendUserID = &userIDStr
		session.Phone = &phone
		session.AccessToken = &accessToken
		session.RefreshToken = &refreshToken
		h.sessionMgr.UpdateSession(ctx, session)

		// Store tokens in Redis
		if registerResp.AccessExpiresIn > 0 {
			ttl := time.Duration(registerResp.AccessExpiresIn) * time.Second
			_ = h.sessionMgr.GetStorage().StoreToken(ctx, userID, accessToken, refreshToken, ttl)
		}
	}

	h.sessionMgr.ClearState(ctx, userID)
	h.sendMessageWithKeyboard(chatID, MsgRegistrationSuccess, MainMenuKeyboard())
}

// handlePasswordInput handles password input (for future use)
func (h *Handlers) handlePasswordInput(msg *tgbotapi.Message, password string) {
	// Implementation for password-based registration
}

// handlePhoto handles photo uploads
func (h *Handlers) handlePhoto(msg *tgbotapi.Message) {
	ctx := context.Background()
	userID := msg.From.ID
	chatID := msg.Chat.ID

	// Check authentication
	authenticated, err := h.sessionMgr.IsAuthenticated(ctx, userID)
	if err != nil || !authenticated {
		h.sendMessage(chatID, MsgErrorUnauthorized)
		return
	}

	// Get the largest photo
	photo := msg.Photo[len(msg.Photo)-1]

	// Download photo
	file, err := h.bot.GetFile(tgbotapi.FileConfig{FileID: photo.FileID})
	if err != nil {
		log.Printf("Failed to get file: %v", err)
		h.sendMessage(chatID, MsgErrorGeneric)
		return
	}

	// Download file data
	fileURL := file.Link(h.bot.Token)
	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Get(fileURL)
	if err != nil {
		log.Printf("Failed to download file: %v", err)
		h.sendMessage(chatID, MsgErrorGeneric)
		return
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to download file: HTTP %d", resp.StatusCode)
		h.sendMessage(chatID, MsgErrorGeneric)
		return
	}

	fileData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read file: %v", err)
		h.sendMessage(chatID, MsgErrorGeneric)
		return
	}

	// Detect MIME type from file content or extension
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		// Try to detect from file extension
		ext := strings.ToLower(file.FilePath)
		if strings.HasSuffix(ext, ".jpg") || strings.HasSuffix(ext, ".jpeg") {
			mimeType = "image/jpeg"
		} else if strings.HasSuffix(ext, ".png") {
			mimeType = "image/png"
		} else if strings.HasSuffix(ext, ".webp") {
			mimeType = "image/webp"
		} else {
			mimeType = "image/jpeg" // Default fallback
		}
	}

	// Validate file size
	if int64(len(fileData)) > h.config.Security.MaxUploadSize {
		h.sendMessage(chatID, fmt.Sprintf(MsgImageTooLarge, formatSize(h.config.Security.MaxUploadSize)))
		return
	}

	// Get access token
	accessToken, err := h.sessionMgr.GetAccessToken(ctx, userID)
	if err != nil || accessToken == "" {
		h.sendMessage(chatID, MsgErrorUnauthorized)
		return
	}

	// Get state to determine image type
	// First image is user photo, second image is cloth/garment
	state, _ := h.sessionMgr.GetState(ctx, userID)
	imageType := "user" // Default: user photo
	if state != nil && state.Action == "waiting_cloth_image" {
		// Second image is the cloth/garment - also use "user" type as it belongs to the user
		// The backend API expects userImageId and clothImageId, both can be "user" type images
		imageType = "user"
	}

	// Upload to backend
	uploadResp, err := h.apiClient.UploadImage(ctx, accessToken, fileData, file.FilePath, mimeType, imageType)
	if err != nil {
		log.Printf("Failed to upload image: %v", err)
		h.sendMessage(chatID, MsgErrorGeneric)
		return
	}

	h.sendMessage(chatID, fmt.Sprintf(MsgImageUploaded, uploadResp.ID))

	// Update state based on image type
	if state == nil || state.Action == "waiting_user_image" {
		// Store user image ID and request cloth image
		h.sessionMgr.SetState(ctx, userID, "waiting_cloth_image", uploadResp.ID)
		h.sendMessage(chatID, MsgImageReceived)
	} else if state.Action == "waiting_cloth_image" {
		// Store cloth image ID and show style selection
		userImageID := state.Data
		h.sessionMgr.SetState(ctx, userID, "waiting_style", userImageID+":"+uploadResp.ID)
		h.sendMessageWithKeyboard(chatID, MsgSelectStyle, StyleSelectionKeyboard())
	}
}

// handleDocument handles document uploads (for images sent as files)
func (h *Handlers) handleDocument(msg *tgbotapi.Message) {
	// Similar to handlePhoto but for documents
	// For now, just handle photos
	h.sendMessage(msg.Chat.ID, "لطفاً عکس را به صورت عکس ارسال کنید، نه فایل.")
}

// HandleCallbackQuery handles inline keyboard callbacks
func (h *Handlers) HandleCallbackQuery(query *tgbotapi.CallbackQuery) {
	ctx := context.Background()
	userID := query.From.ID
	chatID := query.Message.Chat.ID
	data := query.Data

	// Rate limiting
	allowed, _ := h.rateLimiter.AllowUserMessage(ctx, userID, h.config.RateLimit.MessagesPerMinute, time.Minute)
	if !allowed {
		h.answerCallback(query.ID, "تعداد درخواست‌های شما بیش از حد مجاز است.")
		RecordRateLimitHit("callback")
		return
	}

	// Handle different callback actions
	switch {
	case data == "start_conversion":
		h.handleStartConversion(query)
	case data == "my_conversions":
		h.handleMyConversions(query)
	case data == "help":
		h.answerCallback(query.ID, "")
		h.sendMessage(chatID, MsgHelp)
	case data == "settings":
		h.answerCallback(query.ID, "")
		h.sendMessage(chatID, MsgSettings)
	case data == "main_menu":
		h.answerCallback(query.ID, "")
		h.sendMessageWithKeyboard(chatID, "منوی اصلی:", MainMenuKeyboard())
	case strings.HasPrefix(data, "style_"):
		h.handleStyleSelection(query, strings.TrimPrefix(data, "style_"))
	case data == "confirm_conversion":
		h.handleConfirmConversion(query)
	case data == "cancel":
		h.handleCancel(query)
	case strings.HasPrefix(data, "view_conversion_"):
		h.handleViewConversion(query, strings.TrimPrefix(data, "view_conversion_"))
	case strings.HasPrefix(data, "conversions_page_"):
		page, _ := strconv.Atoi(strings.TrimPrefix(data, "conversions_page_"))
		h.handleConversionsPage(query, page)
	default:
		h.answerCallback(query.ID, "عملیات نامعتبر")
	}
}

// handleStartConversion handles conversion start
func (h *Handlers) handleStartConversion(query *tgbotapi.CallbackQuery) {
	ctx := context.Background()
	userID := query.From.ID
	chatID := query.Message.Chat.ID

	// Check authentication
	authenticated, err := h.sessionMgr.IsAuthenticated(ctx, userID)
	if err != nil || !authenticated {
		h.answerCallback(query.ID, "")
		h.sendMessage(chatID, MsgErrorUnauthorized+"\n\n"+MsgPleaseLogin)
		// Prompt for phone number
		h.sessionMgr.SetState(ctx, userID, "waiting_phone", "")
		h.sendMessage(chatID, MsgEnterPhone)
		return
	}

	// Check rate limit for conversions
	allowed, err := h.rateLimiter.AllowUserConversion(ctx, userID, h.config.RateLimit.ConversionsPerHour, time.Hour)
	if err != nil {
		log.Printf("Rate limit check error: %v", err)
	}
	if !allowed {
		h.answerCallback(query.ID, "")
		h.sendMessage(chatID, MsgErrorRateLimit)
		RecordRateLimitHit("conversion")
		return
	}

	h.answerCallback(query.ID, "")
	// Clear any previous state and set new state
	h.sessionMgr.ClearState(ctx, userID)
	h.sessionMgr.SetState(ctx, userID, "waiting_user_image", "")
	h.sendMessage(chatID, "لطفاً عکس خودتون رو ارسال کنید:")
}

// handleStyleSelection handles style selection
func (h *Handlers) handleStyleSelection(query *tgbotapi.CallbackQuery, style string) {
	ctx := context.Background()
	userID := query.From.ID
	chatID := query.Message.Chat.ID

	// Get state
	state, err := h.sessionMgr.GetState(ctx, userID)
	if err != nil || state == nil || state.Action != "waiting_style" {
		h.answerCallback(query.ID, "خطا در دریافت اطلاعات")
		return
	}

	// Parse image IDs
	parts := strings.Split(state.Data, ":")
	if len(parts) != 2 {
		h.answerCallback(query.ID, "خطا در دریافت اطلاعات")
		return
	}

	userImageID := parts[0]
	clothImageID := parts[1]

	// Store conversion data
	conversionData := map[string]string{
		"userImageID":  userImageID,
		"clothImageID": clothImageID,
		"style":        style,
	}
	dataJSON, _ := json.Marshal(conversionData)
	h.sessionMgr.SetState(ctx, userID, "confirming_conversion", string(dataJSON))

	h.answerCallback(query.ID, fmt.Sprintf("استایل انتخاب شد: %s", getStyleDisplayName(style)))
	h.sendMessageWithKeyboard(chatID, "آیا می‌خواهید تبدیل را شروع کنید؟", ConversionConfirmationKeyboard())
}

// handleConfirmConversion handles conversion confirmation
func (h *Handlers) handleConfirmConversion(query *tgbotapi.CallbackQuery) {
	ctx := context.Background()
	userID := query.From.ID
	chatID := query.Message.Chat.ID

	// Get access token
	accessToken, err := h.sessionMgr.GetAccessToken(ctx, userID)
	if err != nil || accessToken == "" {
		h.answerCallback(query.ID, "")
		h.sendMessage(chatID, MsgErrorUnauthorized)
		return
	}

	// Get state
	state, err := h.sessionMgr.GetState(ctx, userID)
	if err != nil || state == nil || state.Action != "confirming_conversion" {
		h.answerCallback(query.ID, "خطا در دریافت اطلاعات")
		return
	}

	// Parse conversion data
	var conversionData map[string]string
	if err := json.Unmarshal([]byte(state.Data), &conversionData); err != nil {
		h.answerCallback(query.ID, "خطا در دریافت اطلاعات")
		return
	}

	// Create conversion
	convReq := ConversionRequest{
		UserImageID:  conversionData["userImageID"],
		ClothImageID: conversionData["clothImageID"],
		StyleName:    conversionData["style"],
	}

	convResp, err := h.apiClient.CreateConversion(ctx, accessToken, convReq)
	if err != nil {
		log.Printf("Failed to create conversion: %v", err)
		h.answerCallback(query.ID, "")
		h.sendMessage(chatID, MsgErrorGeneric)
		return
	}

	h.answerCallback(query.ID, "")
	h.sessionMgr.ClearState(ctx, userID)
	h.sendMessage(chatID, fmt.Sprintf(MsgConversionStarted, convResp.ID))

	// Start polling for conversion status
	go h.pollConversionStatus(ctx, userID, chatID, convResp.ID, accessToken)
}

// pollConversionStatus polls conversion status and updates user
func (h *Handlers) pollConversionStatus(ctx context.Context, userID int64, chatID int64, conversionID, accessToken string) {
	// Create a context with timeout for polling
	pollCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	var lastMessageID int

	for {
		select {
		case <-pollCtx.Done():
			if pollCtx.Err() == context.DeadlineExceeded {
				h.sendMessage(chatID, "زمان پردازش به پایان رسید. لطفاً دوباره تلاش کنید.")
			}
			return
		case <-ticker.C:
			conv, err := h.apiClient.GetConversion(pollCtx, accessToken, conversionID)
			if err != nil {
				log.Printf("Failed to get conversion: %v", err)
				continue
			}

			switch conv.Status {
			case "completed":
				if conv.ResultImageID != nil {
					// Get image URL and send result
					imageURL, err := h.apiClient.GetImageURL(pollCtx, accessToken, *conv.ResultImageID)
					if err == nil {
						photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(imageURL))
						photo.Caption = MsgConversionCompleted
						photo.ReplyMarkup = ConversionResultKeyboard(conversionID)
						h.bot.Send(photo)
					} else {
						h.sendMessageWithKeyboard(chatID, MsgConversionCompleted, ConversionResultKeyboard(conversionID))
					}
				} else {
					h.sendMessageWithKeyboard(chatID, MsgConversionCompleted, ConversionResultKeyboard(conversionID))
				}
				RecordConversion("completed")
				return
			case "failed":
				errorMsg := "خطای نامشخص"
				if conv.ErrorMessage != nil {
					errorMsg = *conv.ErrorMessage
				}
				h.sendMessage(chatID, fmt.Sprintf(MsgConversionFailed, errorMsg))
				RecordConversion("failed")
				return
			case "processing":
				// Estimate progress (simplified - in production, get actual progress from API)
				progress := 50 // Default progress
				if lastMessageID == 0 {
					msg := tgbotapi.NewMessage(chatID, GetProgressMessage(progress))
					sent, _ := h.bot.Send(msg)
					if sent.MessageID != 0 {
						lastMessageID = sent.MessageID
					}
				} else {
					edit := tgbotapi.NewEditMessageText(chatID, lastMessageID, GetProgressMessage(progress))
					h.bot.Send(edit)
				}
			case "pending":
				// Still waiting, continue polling
			}
		}
	}
}

// handleMyConversions handles my conversions list
func (h *Handlers) handleMyConversions(query *tgbotapi.CallbackQuery) {
	ctx := context.Background()
	userID := query.From.ID
	chatID := query.Message.Chat.ID

	// Check authentication
	authenticated, err := h.sessionMgr.IsAuthenticated(ctx, userID)
	if err != nil || !authenticated {
		h.answerCallback(query.ID, "")
		h.sendMessage(chatID, MsgErrorUnauthorized+"\n\n"+MsgPleaseLogin)
		// Prompt for phone number
		h.sessionMgr.SetState(ctx, userID, "waiting_phone", "")
		h.sendMessage(chatID, MsgEnterPhone)
		return
	}

	// Get access token
	accessToken, err := h.sessionMgr.GetAccessToken(ctx, userID)
	if err != nil || accessToken == "" {
		h.answerCallback(query.ID, "")
		h.sendMessage(chatID, MsgErrorUnauthorized)
		return
	}

	// Get conversions
	conversions, err := h.apiClient.ListConversions(ctx, accessToken, 1, 10, "")
	if err != nil {
		log.Printf("Failed to list conversions: %v", err)
		h.answerCallback(query.ID, "")
		h.sendMessage(chatID, MsgErrorGeneric)
		return
	}

	h.answerCallback(query.ID, "")

	if len(conversions.Conversions) == 0 {
		h.sendMessage(chatID, MsgNoConversions)
		return
	}

	// Format conversions list
	text := MsgMyConversions + "\n\n"
	for i, conv := range conversions.Conversions {
		// Safely truncate ID for display
		displayID := conv.ID
		if len(displayID) > 8 {
			displayID = displayID[:8]
		}
		text += fmt.Sprintf("%d. تبدیل #%s\n   وضعیت: %s\n   تاریخ: %s\n\n",
			i+1, displayID, getStatusText(conv.Status), conv.CreatedAt.Format("2006-01-02 15:04"))
	}

	// Send with pagination if needed
	if conversions.TotalPages > 1 {
		h.sendMessageWithKeyboard(chatID, text, ConversionsListKeyboard(1, conversions.TotalPages, ""))
	} else {
		h.sendMessageWithKeyboard(chatID, text, BackToMenuKeyboard())
	}
}

// handleViewConversion handles viewing a specific conversion
func (h *Handlers) handleViewConversion(query *tgbotapi.CallbackQuery, conversionID string) {
	ctx := context.Background()
	userID := query.From.ID
	chatID := query.Message.Chat.ID

	// Get access token
	accessToken, err := h.sessionMgr.GetAccessToken(ctx, userID)
	if err != nil || accessToken == "" {
		h.answerCallback(query.ID, "")
		h.sendMessage(chatID, MsgErrorUnauthorized)
		return
	}

	// Get conversion
	conv, err := h.apiClient.GetConversion(ctx, accessToken, conversionID)
	if err != nil {
		log.Printf("Failed to get conversion: %v", err)
		h.answerCallback(query.ID, "")
		h.sendMessage(chatID, MsgConversionNotFound)
		return
	}

	h.answerCallback(query.ID, "")

	// Safely truncate ID for display
	displayID := conversionID
	if len(displayID) > 8 {
		displayID = displayID[:8]
	}
	text := fmt.Sprintf("تبدیل #%s\nوضعیت: %s\n", displayID, getStatusText(conv.Status))
	if conv.ResultImageID != nil {
		imageURL, err := h.apiClient.GetImageURL(ctx, accessToken, *conv.ResultImageID)
		if err == nil {
			photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(imageURL))
			photo.Caption = text
			photo.ReplyMarkup = BackToMenuKeyboard()
			h.bot.Send(photo)
			return
		}
	}

	h.sendMessageWithKeyboard(chatID, text, BackToMenuKeyboard())
}

// handleConversionsPage handles pagination for conversions list
func (h *Handlers) handleConversionsPage(query *tgbotapi.CallbackQuery, page int) {
	ctx := context.Background()
	userID := query.From.ID
	chatID := query.Message.Chat.ID

	// Check authentication
	authenticated, err := h.sessionMgr.IsAuthenticated(ctx, userID)
	if err != nil || !authenticated {
		h.answerCallback(query.ID, "")
		h.sendMessage(chatID, MsgErrorUnauthorized)
		return
	}

	// Get access token
	accessToken, err := h.sessionMgr.GetAccessToken(ctx, userID)
	if err != nil || accessToken == "" {
		h.answerCallback(query.ID, "")
		h.sendMessage(chatID, MsgErrorUnauthorized)
		return
	}

	// Get conversions for specific page
	conversions, err := h.apiClient.ListConversions(ctx, accessToken, page, 10, "")
	if err != nil {
		log.Printf("Failed to list conversions: %v", err)
		h.answerCallback(query.ID, "")
		h.sendMessage(chatID, MsgErrorGeneric)
		return
	}

	h.answerCallback(query.ID, "")

	if len(conversions.Conversions) == 0 {
		h.sendMessage(chatID, MsgNoConversions)
		return
	}

	// Format conversions list
	text := MsgMyConversions + "\n\n"
	for i, conv := range conversions.Conversions {
		// Safely truncate ID for display
		displayID := conv.ID
		if len(displayID) > 8 {
			displayID = displayID[:8]
		}
		text += fmt.Sprintf("%d. تبدیل #%s\n   وضعیت: %s\n   تاریخ: %s\n\n",
			(page-1)*10+i+1, displayID, getStatusText(conv.Status), conv.CreatedAt.Format("2006-01-02 15:04"))
	}

	// Send with pagination if needed
	if conversions.TotalPages > 1 {
		h.sendMessageWithKeyboard(chatID, text, ConversionsListKeyboard(page, conversions.TotalPages, ""))
	} else {
		h.sendMessageWithKeyboard(chatID, text, BackToMenuKeyboard())
	}
}

// handleCancel handles cancel action
func (h *Handlers) handleCancel(query *tgbotapi.CallbackQuery) {
	ctx := context.Background()
	userID := query.From.ID
	chatID := query.Message.Chat.ID

	h.sessionMgr.ClearState(ctx, userID)
	h.answerCallback(query.ID, "لغو شد")
	h.sendMessageWithKeyboard(chatID, "عملیات لغو شد.", MainMenuKeyboard())
}

// Helper functions

func (h *Handlers) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := h.bot.Send(msg)
	if err != nil {
		log.Printf("Failed to send message: %v", err)
		RecordError("send_message", "handler")
	}
}

func (h *Handlers) sendMessageWithKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	_, err := h.bot.Send(msg)
	if err != nil {
		log.Printf("Failed to send message: %v", err)
		RecordError("send_message", "handler")
	}
}

func (h *Handlers) answerCallback(callbackID string, text string) {
	callback := tgbotapi.NewCallback(callbackID, text)
	_, err := h.bot.Request(callback)
	if err != nil {
		log.Printf("Failed to answer callback: %v", err)
	}
}

func normalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	
	if !strings.HasPrefix(phone, "+") {
		if strings.HasPrefix(phone, "0") {
			phone = "+98" + phone[1:]
		} else {
			phone = "+98" + phone
		}
	}
	
	return phone
}

func generateDefaultPassword(phone string) string {
	// Generate a default password based on phone
	// In production, you might want to ask user for password
	return "TelegramBot" + phone[len(phone)-6:]
}

func formatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.2f MB", float64(bytes)/(1024*1024))
}

func getStatusText(status string) string {
	statusMap := map[string]string{
		"pending":    "در انتظار",
		"processing": "در حال پردازش",
		"completed":  "تکمیل شده",
		"failed":     "ناموفق",
	}
	if text, ok := statusMap[status]; ok {
		return text
	}
	return status
}

