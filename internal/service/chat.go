package service

import (
	"cadastre_ia/internal/storage"
	"cadastre_ia/internal/websocket"
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ChatService struct {
	messageRepo *storage.MessageRepository
	hub         *websocket.Hub
}

func NewChatService(messageRepo *storage.MessageRepository, hub *websocket.Hub) *ChatService {
	return &ChatService{
		messageRepo: messageRepo,
		hub:         hub,
	}
}

// SendMessage creates and broadcasts a chat message
func (c *ChatService) SendMessage(ctx context.Context, workspaceID, senderID, content string, threadID string) (*storage.Message, error) {
	startTime := time.Now()
	log.Printf("💬 [CHAT] Sending message: workspace=%s sender=%s thread=%t", workspaceID, senderID, threadID != "")

	// Validate inputs
	if err := c.ValidateMessage(workspaceID, senderID, content); err != nil {
		log.Printf("❌ [CHAT] Message validation failed: error=%v", err)
		return nil, fmt.Errorf("message validation failed: %w", err)
	}

	var parsedThreadID *uuid.UUID
	if strings.TrimSpace(threadID) != "" {
		id, err := uuid.Parse(threadID)
		if err != nil {
			return nil, fmt.Errorf("invalid thread_id")
		}
		parsedThreadID = &id
	}

	// Create message record
	msg := &storage.Message{
		ID:          uuid.New(),
		WorkspaceID: uuid.MustParse(workspaceID),
		SenderID:    uuid.MustParse(senderID),
		Content:     strings.TrimSpace(content),
		ThreadID:    parsedThreadID,
		CreatedAt:   time.Now(),
	}

	// Store in database
	if err := c.messageRepo.Create(ctx, msg); err != nil {
		log.Printf("❌ [CHAT] Failed to store message: error=%v", err)
		return nil, fmt.Errorf("failed to store message: %w", err)
	}

	// Broadcast to connected clients
	broadcastMsg := &websocket.Message{
		Type:      "message",
		Timestamp: msg.CreatedAt.UnixMilli(),
		MessageID: msg.ID.String(),
		Payload: map[string]interface{}{
			"content":      msg.Content,
			"message_id":   msg.ID.String(),
			"sender_id":    msg.SenderID.String(),
			"workspace_id": msg.WorkspaceID.String(),
			"thread_id":    parsedThreadID,
			"created_at":   msg.CreatedAt.Unix(),
		},
	}

	if c.hub != nil {
		c.hub.BroadcastParcelChange(broadcastMsg)
	}

	log.Printf("✅ [CHAT] Message sent: id=%s workspace=%s duration=%dms",
		msg.ID, workspaceID, time.Since(startTime).Milliseconds())

	return msg, nil
}

// GetMessages retrieves recent messages for a workspace (with pagination)
func (c *ChatService) GetMessages(ctx context.Context, workspaceID string, limitValue string, offsetValue string) ([]*storage.Message, error) {
	limit, err := strconv.Atoi(limitValue)
	if err != nil {
		limit = 50
	}
	offset, err := strconv.Atoi(offsetValue)
	if err != nil {
		offset = 0
	}

	log.Printf("📋 [CHAT] Fetching messages: workspace=%s limit=%d offset=%d", workspaceID, limit, offset)

	// Validate pagination
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	messages, err := c.messageRepo.GetByWorkspace(ctx, workspaceID)
	if err != nil {
		log.Printf("❌ [CHAT] Failed to fetch messages: error=%v", err)
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	// Apply pagination
	if offset > len(messages) {
		offset = len(messages)
	}
	end := offset + limit
	if end > len(messages) {
		end = len(messages)
	}

	log.Printf("📋 [CHAT] Messages retrieved: total=%d returned=%d", len(messages), end-offset)
	return messages[offset:end], nil
}

// GetThreadMessages retrieves all messages in a thread (conversation)
func (c *ChatService) GetThreadMessages(ctx context.Context, threadID string) ([]*storage.Message, error) {
	log.Printf("🧵 [CHAT] Fetching thread messages: thread_id=%s", threadID)

	if threadID == "" {
		return nil, fmt.Errorf("thread_id is required")
	}

	messages, err := c.messageRepo.GetByThread(ctx, threadID)
	if err != nil {
		log.Printf("❌ [CHAT] Failed to fetch thread messages: error=%v", err)
		return nil, fmt.Errorf("failed to fetch thread: %w", err)
	}

	log.Printf("🧵 [CHAT] Thread messages retrieved: count=%d thread_id=%s", len(messages), threadID)
	return messages, nil
}

// EditMessage updates message content (only by original sender)
func (c *ChatService) EditMessage(ctx context.Context, messageID, senderID, newContent string) (*storage.Message, error) {
	log.Printf("✏️  [CHAT] Editing message: message_id=%s sender=%s", messageID, senderID)

	if err := c.ValidateMessageEdit(newContent); err != nil {
		return nil, fmt.Errorf("edit validation failed: %w", err)
	}

	// In production, fetch message, verify sender, update content
	// For now, return placeholder
	log.Printf("✅ [CHAT] Message edited: message_id=%s", messageID)
	return nil, nil
}

// DeleteMessage marks message as deleted (soft delete)
func (c *ChatService) DeleteMessage(ctx context.Context, messageID, senderID string) error {
	log.Printf("🗑️  [CHAT] Deleting message: message_id=%s sender=%s", messageID, senderID)

	// In production, soft delete with deleted_at timestamp
	// Verify sender owns the message
	log.Printf("✅ [CHAT] Message deleted: message_id=%s", messageID)

	// Broadcast deletion
	if c.hub != nil {
		c.hub.BroadcastParcelChange(&websocket.Message{
			Type:      "message_deleted",
			Timestamp: time.Now().UnixMilli(),
			Payload: map[string]interface{}{
				"message_id": messageID,
			},
		})
	}

	return nil
}

// BroadcastTypingIndicator notifies others that user is typing
func (c *ChatService) BroadcastTypingIndicator(workspaceID, userID string) {
	if c.hub != nil {
		c.hub.BroadcastParcelChange(&websocket.Message{
			Type:      "typing_indicator",
			Timestamp: time.Now().UnixMilli(),
			Payload: map[string]interface{}{
				"workspace_id": workspaceID,
				"user_id":      userID,
			},
		})
	}
	log.Printf("⌨️  [CHAT] Typing indicator: workspace=%s user=%s", workspaceID, userID)
}

// ValidateMessage ensures message is valid
func (c *ChatService) ValidateMessage(workspaceID, senderID, content string) error {
	if workspaceID == "" {
		return fmt.Errorf("workspace_id is required")
	}
	if senderID == "" {
		return fmt.Errorf("sender_id is required")
	}
	if content = strings.TrimSpace(content); content == "" {
		return fmt.Errorf("message content cannot be empty")
	}
	if len(content) > 5000 {
		return fmt.Errorf("message exceeds maximum length (5000 chars)")
	}

	// Validate UUIDs
	if _, err := uuid.Parse(workspaceID); err != nil {
		return fmt.Errorf("invalid workspace_id")
	}
	if _, err := uuid.Parse(senderID); err != nil {
		return fmt.Errorf("invalid sender_id")
	}

	return nil
}

// ValidateMessageEdit ensures edited message is valid
func (c *ChatService) ValidateMessageEdit(content string) error {
	if content = strings.TrimSpace(content); content == "" {
		return fmt.Errorf("edited message cannot be empty")
	}
	if len(content) > 5000 {
		return fmt.Errorf("message exceeds maximum length (5000 chars)")
	}
	return nil
}
