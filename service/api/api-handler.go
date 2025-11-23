package api

import (
	"net/http"
)

// Handler returns an instance of httprouter.Router that handle APIs registered here
func (rt *_router) Handler() http.Handler {

	// Register routes
	rt.router.GET("/liveness", rt.liveness)
	rt.router.GET("/context", rt.wrap(rt.getContextReply))

	// ------------------------------------------------
	// LOGIN & REGISTRATION
	// ------------------------------------------------
	rt.router.POST("/wasatext/login", rt.doLogin)
	rt.router.POST("/wasatext/register", rt.registerUser)

	// ------------------------------------------------
	// USERS (Profile, Status, Search)
	// ------------------------------------------------
	// Get current user profile
	rt.router.GET("/wasatext/:username/profile", rt.myProfileDetails)

	// Update current user status
	rt.router.PATCH("/wasatext/:username/profile/status", rt.modifyProfileStatus)

	// Update username
	rt.router.PUT("/wasatext/:username/profile/username", rt.setMyUserName)

	// Update profile photo
	rt.router.PUT("/wasatext/:username/profile/photo", rt.setMyPhoto)

	// Search user
	// Returns a list of users
	rt.router.GET("/wasatext/:username/user_search", rt.searchUser)

	// Get another user's profile by username
	rt.router.GET("/wasatext/:username/users/:usernameSearched", rt.getUserProfile)

	// ------------------------------------------------
	// CONVERSATIONS
	// ------------------------------------------------
	// Get all conversations
	rt.router.GET("/wasatext/:username/conversations", rt.getMyConversations)

	// ------------------------------------------------
	// PRIVATE CHATS (Root level)
	// ------------------------------------------------
	// Create new private chat (or append message)
	rt.router.POST("/wasatext/:username/private_chats", rt.newPrivateChat)

	// Get specific private chat
	rt.router.GET("/wasatext/:username/private_chats/:conversationId", rt.getPrivateChatById)

	// Delete private chat
	rt.router.DELETE("/wasatext/:username/private_chats/:conversationId", rt.deletePrivateChat)

	// ------------------------------------------------
	// GROUP CHATS (Root level)
	// ------------------------------------------------
	// Create new group
	rt.router.POST("/wasatext/:username/groups", rt.createNewGroup)

	// Get specific group chat
	rt.router.GET("/wasatext/:username/groups/:conversationId", rt.getGroupChat)

	// Add members to group
	rt.router.POST("/wasatext/:username/groups/:conversationId/members", rt.addToGroup)

	// Remove member from group
	rt.router.DELETE("/wasatext/:username/groups/:conversationId/members/:userId", rt.removeMember)

	// Leave group
	rt.router.POST("/wasatext/:username/groups/:conversationId/leave", rt.leaveGroup)

	// Make user admin
	rt.router.POST("/wasatext/:username/groups/:conversationId/admins", rt.makeAdmin)

	// Remove admin status
	rt.router.DELETE("/wasatext/:username/groups/:conversationId/admins/:userId", rt.removeAdminStatus)

	// Update group name
	rt.router.PUT("/wasatext/:username/groups/:conversationId/name", rt.setGroupName)

	// Update group description
	rt.router.PUT("/wasatext/:username/groups/:conversationId/description", rt.setGroupDescription)

	// Update group photo
	rt.router.PUT("/wasatext/:username/groups/:conversationId/photo", rt.setGroupPhoto)

	// ------------------------------------------------
	// MESSAGES
	// ------------------------------------------------
	// Get messages in conversation
	rt.router.GET("/wasatext/:username/conversations/:conversationId/messages", rt.getConversation)

	// Send new message
	rt.router.POST("/wasatext/:username/conversations/:conversationId/messages", rt.sendMessage)

	// Mark messages as read (Blue Ticks)
	rt.router.PUT("/wasatext/:username/conversations/:conversationId/read_status", rt.markMessagesAsRead)

	// Edit message
	rt.router.PATCH("/wasatext/:username/conversations/:conversationId/messages/:messageId", rt.editMessage)

	// Delete message
	rt.router.DELETE("/wasatext/:username/conversations/:conversationId/messages/:messageId", rt.deleteMessage)

	// Forward message
	rt.router.POST("/wasatext/:username/conversations/:conversationId/messages/:messageId/forward", rt.forwardMessage)

	// ------------------------------------------------
	// REACTIONS
	// ------------------------------------------------
	// Add reaction
	rt.router.POST("/wasatext/:username/conversations/:conversationId/messages/:messageId/reactions", rt.commentMessage)

	// Remove reaction
	rt.router.DELETE("/wasatext/:username/conversations/:conversationId/messages/:messageId/reactions", rt.uncommentMessage)

	// Get all reactions for a message
	rt.router.GET("/wasatext/:username/conversations/:conversationId/messages/:messageId/reactions", rt.getAllMessageReactions)

	return rt.router
}
