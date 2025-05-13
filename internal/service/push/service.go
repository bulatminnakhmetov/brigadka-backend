package push

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sideshow/apns2"
	apns2payload "github.com/sideshow/apns2/payload"
	"github.com/sideshow/apns2/token"

	pushrepo "github.com/bulatminnakhmetov/brigadka-backend/internal/repository/push"
)

// NotificationPayload represents a push notification payload
type NotificationPayload struct {
	Title    string                 `json:"title"`
	Body     string                 `json:"body"`
	Badge    int                    `json:"badge,omitempty"`
	Sound    string                 `json:"sound,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
	ImageURL string                 `json:"imageUrl,omitempty"`
}

// PushService defines the operations for push notifications
type PushService interface {
	SaveToken(ctx context.Context, userID int, token string, platform string, deviceID string) error
	DeleteToken(ctx context.Context, token string) error
	SendNotification(ctx context.Context, userID int, payload NotificationPayload) error
	SendNotificationToTokens(ctx context.Context, tokens []string, payload NotificationPayload) error
}

type pushService struct {
	repo            pushrepo.Repository
	fcmServerKey    string
	apnsKeyID       string
	apnsTeamID      string
	apnsPrivateKey  []byte
	apnsBundleID    string
	apnsDevelopment bool
}

// Config holds the configuration for the push service
type Config struct {
	FCMServerKey    string
	APNSKeyID       string
	APNSTeamID      string
	APNSPrivateKey  []byte
	APNSBundleID    string
	APNSDevelopment bool
}

// NewPushService creates a new push notification service
func NewPushService(repo pushrepo.Repository, config Config) PushService {
	return &pushService{
		repo:            repo,
		fcmServerKey:    config.FCMServerKey,
		apnsKeyID:       config.APNSKeyID,
		apnsTeamID:      config.APNSTeamID,
		apnsPrivateKey:  config.APNSPrivateKey,
		apnsBundleID:    config.APNSBundleID,
		apnsDevelopment: config.APNSDevelopment,
	}
}

// SaveToken saves a push notification token for a user
func (s *pushService) SaveToken(ctx context.Context, userID int, token string, platform string, deviceID string) error {
	if token == "" {
		return errors.New("token cannot be empty")
	}

	if !isValidPlatform(platform) {
		return errors.New("invalid platform: must be 'ios' or 'android'")
	}

	_, err := s.repo.SaveToken(ctx, pushrepo.PushToken{
		UserID:   userID,
		Token:    token,
		Platform: platform,
		DeviceID: deviceID,
	})

	return err
}

// DeleteToken removes a push notification token
func (s *pushService) DeleteToken(ctx context.Context, token string) error {
	return s.repo.DeleteToken(ctx, token)
}

// SendNotification sends a push notification to a specific user
func (s *pushService) SendNotification(ctx context.Context, userID int, payload NotificationPayload) error {
	tokens, err := s.repo.GetUserTokens(ctx, userID)
	if err != nil {
		return err
	}

	if len(tokens) == 0 {
		return errors.New("no tokens found for user")
	}

	// Group tokens by platform
	androidTokens := make([]string, 0)
	iosTokens := make([]string, 0)

	for _, token := range tokens {
		if strings.ToLower(token.Platform) == "android" {
			androidTokens = append(androidTokens, token.Token)
		} else if strings.ToLower(token.Platform) == "ios" {
			iosTokens = append(iosTokens, token.Token)
		}
	}

	var sendErrors []error

	// Send to Android devices
	if len(androidTokens) > 0 {
		err := s.sendToFCM(ctx, androidTokens, payload)
		if err != nil {
			sendErrors = append(sendErrors, fmt.Errorf("FCM error: %w", err))
		}
	}

	// Send to iOS devices
	if len(iosTokens) > 0 {
		err := s.sendToAPNS(ctx, iosTokens, payload)
		if err != nil {
			sendErrors = append(sendErrors, fmt.Errorf("APNS error: %w", err))
		}
	}

	if len(sendErrors) > 0 {
		// Return first error or combine them
		return sendErrors[0]
	}

	return nil
}

// SendNotificationToTokens sends a notification to specific tokens
func (s *pushService) SendNotificationToTokens(ctx context.Context, tokens []string, payload NotificationPayload) error {
	if len(tokens) == 0 {
		return errors.New("no tokens provided")
	}

	// For simplicity, assuming all tokens are FCM tokens
	// In a real implementation, you might want to determine the type of each token
	return s.sendToFCM(ctx, tokens, payload)
}

// sendToFCM sends notifications to Firebase Cloud Messaging
func (s *pushService) sendToFCM(ctx context.Context, tokens []string, payload NotificationPayload) error {
	if s.fcmServerKey == "" {
		return errors.New("FCM server key not configured")
	}

	fcmPayload := map[string]interface{}{
		"registration_ids": tokens,
		"notification": map[string]interface{}{
			"title": payload.Title,
			"body":  payload.Body,
			"sound": defaultIfEmpty(payload.Sound, "default"),
		},
		"data": payload.Data,
	}

	if payload.ImageURL != "" {
		fcmPayload["notification"].(map[string]interface{})["image"] = payload.ImageURL
	}

	jsonPayload, err := json.Marshal(fcmPayload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://fcm.googleapis.com/fcm/send", strings.NewReader(string(jsonPayload)))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "key="+s.fcmServerKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			return fmt.Errorf("FCM error: %v", errResp)
		}
		return fmt.Errorf("FCM error with status code: %d", resp.StatusCode)
	}

	// Process FCM response to handle tokens that need to be removed
	var fcmResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&fcmResponse); err != nil {
		return fmt.Errorf("error decoding FCM response: %w", err)
	}

	// Handle invalid tokens
	if results, ok := fcmResponse["results"].([]interface{}); ok {
		for i, result := range results {
			resultMap, ok := result.(map[string]interface{})
			if !ok {
				continue
			}

			if errMsg, exists := resultMap["error"]; exists {
				errStr, ok := errMsg.(string)
				if !ok {
					continue
				}

				// Check for situations where we should remove the token
				if errStr == "NotRegistered" || errStr == "InvalidRegistration" {
					if i < len(tokens) {
						_ = s.repo.DeleteToken(ctx, tokens[i]) // Best effort cleanup
					}
				}
			}
		}
	}

	return nil
}

// sendToAPNS sends notifications to Apple Push Notification Service
func (s *pushService) sendToAPNS(ctx context.Context, tokens []string, payload NotificationPayload) error {
	// Verify required APNS configuration
	if len(s.apnsPrivateKey) == 0 || s.apnsKeyID == "" || s.apnsTeamID == "" || s.apnsBundleID == "" {
		return errors.New("incomplete APNS configuration")
	}

	// Create a new token based authentication for APNS
	authKey, err := token.AuthKeyFromBytes(s.apnsPrivateKey)
	if err != nil {
		return fmt.Errorf("failed to load APNS auth key: %w", err)
	}

	// Create a token client
	authToken := &token.Token{
		AuthKey: authKey,
		KeyID:   s.apnsKeyID,
		TeamID:  s.apnsTeamID,
	}

	// Determine if we should use development or production APNS server
	var client *apns2.Client
	if s.apnsDevelopment {
		client = apns2.NewTokenClient(authToken).Development()
	} else {
		client = apns2.NewTokenClient(authToken).Production()
	}

	// Set timeout on the client
	client.HTTPClient.Timeout = 15 * time.Second

	// Build APNS notification payload
	apnsPayload := apns2payload.NewPayload().
		AlertTitle(payload.Title).
		AlertBody(payload.Body).
		Sound(defaultIfEmpty(payload.Sound, "default"))

	// Set badge if provided
	if payload.Badge > 0 {
		apnsPayload.Badge(payload.Badge)
	}

	// Add custom data if provided
	if payload.Data != nil {
		for k, v := range payload.Data {
			apnsPayload.Custom(k, v)
		}
	}

	// If an image URL is provided, add it as a media attachment
	if payload.ImageURL != "" {
		apnsPayload.MutableContent()
		apnsPayload.Custom("image_url", payload.ImageURL)
	}

	// Process all tokens
	var errs []error
	for _, token := range tokens {
		// Create notification
		notification := &apns2.Notification{
			DeviceToken: token,
			Topic:       s.apnsBundleID,
			Payload:     apnsPayload,
			Priority:    apns2.PriorityHigh,
			PushType:    apns2.PushTypeAlert,
		}

		// Send notification
		resp, err := client.Push(notification)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to send APNS notification: %w", err))
			continue
		}

		// Handle APNS response
		if resp.StatusCode != http.StatusOK {
			// Handle specific status codes
			switch resp.Reason {
			case apns2.ReasonBadDeviceToken, apns2.ReasonDeviceTokenNotForTopic, apns2.ReasonUnregistered:
				// Token is invalid, remove it from database
				_ = s.repo.DeleteToken(ctx, token)
				errs = append(errs, fmt.Errorf("invalid token removed: %s - %s", token, resp.Reason))
			default:
				errs = append(errs, fmt.Errorf("APNS error: %s", resp.Reason))
			}
		}
	}

	// Return concatenated errors if any
	if len(errs) > 0 {
		var combinedErr strings.Builder
		for i, err := range errs {
			if i > 0 {
				combinedErr.WriteString("; ")
			}
			combinedErr.WriteString(err.Error())
		}
		return errors.New(combinedErr.String())
	}

	return nil
}

// Helper functions
func isValidPlatform(platform string) bool {
	platform = strings.ToLower(platform)
	return platform == "ios" || platform == "android"
}

func defaultIfEmpty(val, defaultVal string) string {
	if val == "" {
		return defaultVal
	}
	return val
}
