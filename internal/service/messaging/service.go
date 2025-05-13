package messaging

import (
	"context"
	"errors"
	"log"
	"time"

	apierrors "github.com/bulatminnakhmetov/brigadka-backend/internal/errors"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/repository/messaging"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/repository/profile"
)

type Chat = messaging.Chat

// Service interface defines the messaging service operations
type Service interface {
	GetUserChats(userID int) ([]messaging.Chat, error)
	GetChat(chatID string, userID int) (*messaging.Chat, error)
	CreateChat(ctx context.Context, chatID string, creatorID int, chatName string, participants []int) error
	AddMessage(messageID string, chatID string, senderID int, content string) (time.Time, error)
	GetChatParticipants(chatID string) ([]int, error)
	IsUserInChat(userID int, chatID string) (bool, error)
	AddParticipant(chatID string, userID int) error
	RemoveParticipant(chatID string, userID int) error
	AddReaction(reactionID string, messageID string, userID int, reactionCode string) error
	RemoveReaction(messageID string, userID int, reactionCode string) error
	GetChatIDForMessage(messageID string) (string, error)
	GetChatMessages(chatID string, userID int, limit, offset int) ([]messaging.ChatMessage, error)
	StoreTypingIndicator(userID int, chatID string) error
	StoreReadReceipt(userID int, chatID string, messageID string) error
	GetUserChatRooms(userID int) (map[string]struct{}, error)
	GetChatParticipantsForBroadcast(chatID string) ([]int, error)
	GetOrCreateDirectChat(ctx context.Context, userID1 int, userID2 int) (string, error)
}

type ProfileRepository interface {
	GetProfile(userID int) (*profile.ProfileModel, error)
}

// ServiceImpl implements the messaging service
type ServiceImpl struct {
	messagingRepo messaging.MessagingRepository
	profileRepo   ProfileRepository
}

// NewService creates a new messaging service
func NewService(messagingRepo messaging.MessagingRepository, profileRepo ProfileRepository) *ServiceImpl {
	return &ServiceImpl{
		messagingRepo: messagingRepo,
		profileRepo:   profileRepo,
	}
}

// GetUserChats retrieves all chats for a user
func (s *ServiceImpl) GetUserChats(userID int) ([]messaging.Chat, error) {
	rawChats, err := s.messagingRepo.GetUserChats(userID)
	if err != nil {
		return nil, err
	}

	var chats []messaging.Chat

	// TODO: use batch query for profile retrieval
	for _, rawChat := range rawChats {
		chat, err := s.setChatName(&rawChat, userID)
		if err != nil {
			log.Printf("Error setting chat name: %v", err)
			continue
		}
		chats = append(chats, *chat)
	}

	return chats, nil
}

func (s *ServiceImpl) setChatName(chat *messaging.Chat, userID int) (*messaging.Chat, error) {
	if chat == nil || chat.IsGroup {
		return chat, nil
	}

	for _, participant := range chat.Participants {
		if participant != userID {
			profile, err := s.profileRepo.GetProfile(participant)
			if err != nil {
				return nil, err
			}
			chat.ChatName = &profile.FullName
		}
	}

	return chat, nil
}

// GetChat retrieves details for a specific chat
func (s *ServiceImpl) GetChat(chatID string, userID int) (*messaging.Chat, error) {
	chat, err := s.messagingRepo.GetChat(chatID, userID)
	if err != nil {
		return nil, err
	}

	return s.setChatName(chat, userID)
}

// CreateChat creates a new chat with the specified participants
func (s *ServiceImpl) CreateChat(ctx context.Context, chatID string, creatorID int, chatName string, participants []int) error {
	return s.messagingRepo.CreateChat(ctx, chatID, creatorID, chatName, participants)
}

// AddMessage adds a new message to a chat
func (s *ServiceImpl) AddMessage(messageID string, chatID string, senderID int, content string) (time.Time, error) {
	// Check if user can send messages to this chat
	inChat, err := s.IsUserInChat(senderID, chatID)
	if err != nil {
		return time.Time{}, err
	}

	if !inChat {
		return time.Time{}, errors.New(apierrors.ErrorUserNotInChat)
	}

	return s.messagingRepo.AddMessage(messageID, chatID, senderID, content)
}

// GetChatParticipants retrieves all participants in a chat
func (s *ServiceImpl) GetChatParticipants(chatID string) ([]int, error) {
	return s.messagingRepo.GetChatParticipants(chatID)
}

// IsUserInChat checks if a user is a participant in a chat
func (s *ServiceImpl) IsUserInChat(userID int, chatID string) (bool, error) {
	return s.messagingRepo.IsUserInChat(userID, chatID)
}

// AddParticipant adds a user to a chat
func (s *ServiceImpl) AddParticipant(chatID string, userID int) error {
	return s.messagingRepo.AddParticipant(chatID, userID)
}

// RemoveParticipant removes a user from a chat
func (s *ServiceImpl) RemoveParticipant(chatID string, userID int) error {
	return s.messagingRepo.RemoveParticipant(chatID, userID)
}

// AddReaction adds a reaction to a message
func (s *ServiceImpl) AddReaction(reactionID string, messageID string, userID int, reactionCode string) error {
	// Business logic moved from repository to service
	chatID, err := s.GetChatIDForMessage(messageID)
	if err != nil {
		return err
	}

	// Check if user is in the chat
	inChat, err := s.IsUserInChat(userID, chatID)
	if err != nil {
		return err
	}

	if !inChat {
		return errors.New(apierrors.ErrorNotAuthorizedToReact)
	}

	return s.messagingRepo.AddReaction(reactionID, messageID, userID, reactionCode)
}

// RemoveReaction removes a reaction from a message
func (s *ServiceImpl) RemoveReaction(messageID string, userID int, reactionCode string) error {
	return s.messagingRepo.RemoveReaction(messageID, userID, reactionCode)
}

// GetChatIDForMessage retrieves the chat ID for a message
func (s *ServiceImpl) GetChatIDForMessage(messageID string) (string, error) {
	return s.messagingRepo.GetChatIDForMessage(messageID)
}

// GetChatMessages retrieves messages for a chat with pagination
func (s *ServiceImpl) GetChatMessages(chatID string, userID int, limit, offset int) ([]messaging.ChatMessage, error) {
	// Check if user is in chat
	inChat, err := s.IsUserInChat(userID, chatID)
	if err != nil {
		return nil, err
	}

	if !inChat {
		return nil, errors.New(apierrors.ErrorUserNotInChat)
	}

	return s.messagingRepo.GetChatMessages(chatID, userID, limit, offset)
}

// StoreTypingIndicator records that a user is typing in a chat
func (s *ServiceImpl) StoreTypingIndicator(userID int, chatID string) error {
	return s.messagingRepo.StoreTypingIndicator(userID, chatID)
}

// StoreReadReceipt records that a user has read messages up to a certain point
func (s *ServiceImpl) StoreReadReceipt(userID int, chatID string, messageID string) error {
	return s.messagingRepo.StoreReadReceipt(userID, chatID, messageID)
}

// GetUserChatRooms retrieves all chat IDs a user is part of
func (s *ServiceImpl) GetUserChatRooms(userID int) (map[string]struct{}, error) {
	return s.messagingRepo.GetUserChatRooms(userID)
}

// GetChatParticipantsForBroadcast retrieves all participants of a chat for broadcasting
func (s *ServiceImpl) GetChatParticipantsForBroadcast(chatID string) ([]int, error) {
	return s.messagingRepo.GetChatParticipantsForBroadcast(chatID)
}

// GetOrCreateDirectChat finds or creates a direct chat between two users
func (s *ServiceImpl) GetOrCreateDirectChat(ctx context.Context, userID1 int, userID2 int) (string, error) {
	// Business logic moved from handler to service
	if userID1 == userID2 {
		return "", errors.New(apierrors.ErrorCannotCreateChatWithSelf)
	}

	return s.messagingRepo.GetOrCreateDirectChat(ctx, userID1, userID2)
}
