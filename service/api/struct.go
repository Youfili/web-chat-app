package api

import (
	"time"
)

// AuthResponse represents the response for Login and Registration
type AuthResponse struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	ProfilePhoto string `json:"profilePhoto,omitempty"` // omitempty perché opzionale
}

// User represents a user in the system
type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	ProfilePhoto string `json:"profilePhoto"`
	Status       string `json:"status,omitempty"`
}

// ---------------------------------------------------------
// CONVERSATIONS (Private & Group)
// ---------------------------------------------------------

// Conversation is a unified struct that can represent BOTH a PrivateChat and a GroupChat.
// I use pointers (*) and 'omitempty' so that fields not relevant to the specific type
// are omitted from the JSON output, satisfying the OpenAPI "oneOf" requirement.
type Conversation struct {
	// Common Fields
	ConversationType string    `json:"conversationType"` // "private" or "group"
	ID               string    `json:"id"`
	Snippet          string    `json:"snippet"`
	DtLastMessage    time.Time `json:"dtLastMessage"`

	// Private Chat Fields (Only if type="private")
	RecipientUser     *string `json:"recipientUser,omitempty"`
	RecipientUsername *string `json:"recipientUsername,omitempty"`
	UserPhoto         *string `json:"userPhoto,omitempty"`

	// Group Chat Fields (Only if type="group")
	GroupName        *string   `json:"groupName,omitempty"`
	GroupDescription *string   `json:"groupDescription,omitempty"`
	GroupPhoto       *string   `json:"groupPhoto,omitempty"`
	Members          *[]string `json:"members,omitempty"` // Array of UUIDs
	Admins           *[]string `json:"admins,omitempty"`  // Array of UUIDs
}

// ConversationList is the response for GET /conversations
type ConversationList struct {
	Conversations []Conversation `json:"conversations"`
}

// ---------------------------------------------------------
// MESSAGES
// ---------------------------------------------------------

// Message represents a single message
type Message struct {
	ID             string     `json:"id"`
	ContentMess    string     `json:"contentMess"`
	Timestamp      time.Time  `json:"timestamp"`
	StatusInfo     string     `json:"statusInfo"` // "delivered", "read"
	ConversationID string     `json:"conversationId"`
	SenderUserID   string     `json:"senderUserId"`
	SenderUsername string     `json:"senderUsername"`
	Forwarded      bool       `json:"forwarded"`
	Reactions      []Reaction `json:"reactions"`
}

// MessageList is the response for GET /messages
type MessageList struct {
	Messages []Message `json:"messages"`
}

// NewMessageRequest is the payload for sending a message
type NewMessageRequest struct {
	ContentMess string `json:"contentMess"`
}

// MessageUpdate is the payload for editing a message
type MessageUpdate struct {
	ContentMess string `json:"contentMess"`
}

// ForwardMessageRequest payload
type ForwardMessageRequest struct {
	TargetConversationID *string `json:"targetConversationId,omitempty"`
	TargetUserID         *string `json:"targetUserId,omitempty"`
}

// ReadStatusRequest is the payload for Blue Ticks
type ReadStatusRequest struct {
	LastReadMessageID string `json:"lastReadMessageId"`
}

// ---------------------------------------------------------
// REACTIONS
// ---------------------------------------------------------

// Reaction represents a user reaction to a message
type Reaction struct {
	ReactionID     string    `json:"reactionId"`
	Emoji          string    `json:"emoji"`
	Timestamp      time.Time `json:"timestamp"`
	MessToReactID  string    `json:"messToReactId"`
	SenderUserID   string    `json:"senderUserId"`
	SenderUsername string    `json:"senderUsername"`
}

// ReactionList response
type ReactionList struct {
	Reactions []Reaction `json:"reactions"`
}

// NewReactionRequest payload
type NewReactionRequest struct {
	Emoji string `json:"emoji"`
}

// ---------------------------------------------------------
// GROUP MANAGEMENT REQUESTS
// ---------------------------------------------------------

// GroupChatPrototype is the payload to CREATE a new group
type GroupChatPrototype struct {
	GroupName        string   `json:"groupName"`
	GroupDescription string   `json:"groupDescription"`
	Members          []string `json:"members"` // List of User IDs
}

// AddMembersRequest payload
type AddMembersRequest struct {
	UsersIDsToAdd []string `json:"usersIdsToAdd"`
}

// AdminRequest payload
type AdminRequest struct {
	UserIDToPromote string `json:"userIdToPromote"`
}

// GroupNameUpdate payload
type GroupNameUpdate struct {
	GroupName string `json:"groupName"`
}

// GroupDescriptionUpdate payload
type GroupDescriptionUpdate struct {
	GroupDescription string `json:"groupDescription"`
}

// GroupPhotoUpdate payload
type GroupPhotoUpdate struct {
	GroupPhoto string `json:"groupPhoto"`
}

// ---------------------------------------------------------
// PRIVATE CHAT REQUESTS
// ---------------------------------------------------------

// PrivateChatPrototype is the payload to START a private chat
type PrivateChatPrototype struct {
	RecipientUser  string `json:"recipientUser"`
	InitialMessage string `json:"initialMessage"`
}

// ---------------------------------------------------------
// USER PROFILE REQUESTS
// ---------------------------------------------------------

// LoginRequest payload
type LoginRequest struct {
	Username string `json:"username"`
}

// RegisterRequest payload
type RegisterRequest struct {
	Username     string `json:"username"`
	Status       string `json:"status,omitempty"`
	ProfilePhoto string `json:"profilePhoto,omitempty"`
}

// UpdateUsernameRequest payload
type UpdateUsernameRequest struct {
	NewUsername string `json:"newUsername"`
}

// UpdateStatusRequest payload
type UpdateStatusRequest struct {
	Status string `json:"status"`
}

// UpdatePhotoRequest payload
type UpdatePhotoRequest struct {
	ProfilePhoto string `json:"profilePhoto"`
}

// UserListResponse for Search
type UserListResponse struct {
	Users []User `json:"users"`
}
