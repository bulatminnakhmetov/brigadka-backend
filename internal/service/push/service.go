package push

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/sideshow/apns2"
	apns2payload "github.com/sideshow/apns2/payload"
	"github.com/sideshow/apns2/token"

	"firebase.google.com/go/v4/messaging"
	pushrepo "github.com/bulatminnakhmetov/brigadka-backend/internal/repository/push"
)

var (
	ErrTokenNotFound = errors.New("token not found")
)

// NotificationPayload represents a push notification payload
type NotificationPayload struct {
	Title    string `json:"title"`
	Body     string `json:"body"`
	Badge    int    `json:"badge,omitempty"`
	Sound    string `json:"sound,omitempty"`
	ImageURL string `json:"imageUrl,omitempty"`
}

// PushService defines the operations for push notifications
type PushService interface {
	SaveToken(ctx context.Context, userID int, token string, platform string, deviceID string) error
	DeleteToken(ctx context.Context, userID int, token string) error
	SendNotification(ctx context.Context, userID int, payload NotificationPayload) error
	SendNotificationToTokens(ctx context.Context, userID int, tokens []string, payload NotificationPayload) error
}

type pushService struct {
	repository      pushrepo.Repository
	firebaseClient  *messaging.Client
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
func NewPushService(repo pushrepo.Repository, config Config, firebaseClient *messaging.Client) PushService {
	return &pushService{
		repository:      repo,
		firebaseClient:  firebaseClient,
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

	_, err := s.repository.SaveToken(ctx, pushrepo.PushToken{
		UserID:   userID,
		Token:    token,
		Platform: platform,
		DeviceID: deviceID,
	})

	return err
}

// DeleteToken removes a push notification token
func (s *pushService) DeleteToken(ctx context.Context, userID int, token string) error {
	isExists, err := s.repository.IsTokenExists(ctx, token, userID)
	if err != nil {
		return errors.New("failed to check token existence")
	}
	if !isExists {
		return ErrTokenNotFound
	}
	return s.repository.DeleteToken(ctx, userID, token)
}

// SendNotification sends a push notification to a specific user
func (s *pushService) SendNotification(ctx context.Context, userID int, payload NotificationPayload) error {
	tokens, err := s.repository.GetUserTokens(ctx, userID)
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
		err := s.sendToFCM(ctx, userID, androidTokens, payload)
		if err != nil {
			sendErrors = append(sendErrors, fmt.Errorf("FCM error: %w", err))
		}
	}

	// Send to iOS devices
	if len(iosTokens) > 0 {
		err := s.sendToAPNS(ctx, userID, iosTokens, payload)
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
func (s *pushService) SendNotificationToTokens(ctx context.Context, userID int, tokens []string, payload NotificationPayload) error {
	if len(tokens) == 0 {
		return errors.New("no tokens provided")
	}

	// For simplicity, assuming all tokens are FCM tokens
	// In a real implementation, you might want to determine the type of each token
	return s.sendToFCM(ctx, userID, tokens, payload)
}

// sendToFCM sends notifications to Firebase Cloud Messaging
func (s *pushService) sendToFCM(ctx context.Context, userID int, tokens []string, payload NotificationPayload) error {
	if s.firebaseClient == nil {
		return errors.New("firebase messaging client not initialized")
	}

	if len(tokens) == 0 {
		return errors.New("no tokens provided")
	}

	// Track success and failures
	var failedTokens []string
	var invalidTokens []string
	successCount := 0

	// Send messages individually to each token
	for _, token := range tokens {
		// Create notification
		notification := &messaging.Notification{
			Title: payload.Title,
			Body:  payload.Body,
		}

		if payload.ImageURL != "" {
			notification.ImageURL = payload.ImageURL
		}

		// Create message for a single token
		message := &messaging.Message{
			Token:        token,
			Notification: notification,
			Android: &messaging.AndroidConfig{
				Notification: &messaging.AndroidNotification{
					Sound: defaultIfEmpty(payload.Sound, "default"),
				},
			},
		}

		// Send individual message
		_, err := s.firebaseClient.Send(ctx, message)
		if err != nil {
			failedTokens = append(failedTokens, token)

			log.Printf("Failed to send FCM message to user %d: %v", userID, err)

			// Check if error is due to an invalid token
			if messaging.IsUnregistered(err) || messaging.IsInvalidArgument(err) {
				invalidTokens = append(invalidTokens, token)
			}
		} else {
			successCount++
		}
	}

	// Clean up invalid tokens
	for _, token := range invalidTokens {
		_ = s.repository.DeleteToken(ctx, userID, token) // Best effort cleanup
	}

	// Return error if all messages failed to send
	if successCount == 0 && len(failedTokens) > 0 {
		return fmt.Errorf("all FCM messages failed to send")
	}

	return nil
}

// sendToAPNS sends notifications to Apple Push Notification Service
func (s *pushService) sendToAPNS(ctx context.Context, userID int, tokens []string, payload NotificationPayload) error {
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
				_ = s.repository.DeleteToken(ctx, userID, token)
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
