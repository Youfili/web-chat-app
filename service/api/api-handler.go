package api

import (
	"net/http"
)

// Handler returns an instance of httprouter.Router that handle APIs registered here
func (rt *_router) Handler() http.Handler {

	// Register routes
	rt.router.GET("/liveness", rt.liveness)
	rt.router.GET("/context", rt.wrap(rt.getContextReply)

	// ------------------------------------------------
	// LOGIN & REGISTRATION
	// ------------------------------------------------
	rt.router.POST("/wasatext/login", rt.doLogin)
	rt.router.POST("/wasatext/register", rt.registerUser)

	// ------------------------------------------------
	// USERS (Profile, Status, Search, Block)
	// ------------------------------------------------
	// Get current user profile
	rt.router.GET("/wasatext/:username/profile", rt.myProfileDetails)
	
	// Update current user status (PATCH /status)
	rt.router.PATCH("/wasatext/:username/profile/status", rt.modifyProfileStatus)
	
	// Update username (PUT /username)
	rt.router.PUT("/wasatext/:username/profile/username", rt.setMyUserName)
	
	// Update profile photo (PUT /photo)
	rt.router.PUT("/wasatext/:username/profile/photo", rt.setMyPhoto)

	// Get another user's profile
	rt.router.GET("/wasatext/:username/users/:userId", rt.getUserProfile)


	// ------------------------------------------------
	// CONVERSATIONS (General)
	// ------------------------------------------------
	// Get all conversations
	rt.router.GET("/wasatext/:username/conversations", rt.getMyConversations)


	// ------------------------------------------------
	// PRIVATE CHATS
	// ------------------------------------------------
	// Create new private chat (or append message)
	rt.router.POST("/wasatext/:username/conversations/private_chats", rt.newPrivateChat)

	// Get specific private chat
	rt.router.GET("/wasatext/:username/conversations/private_chats/:conversationId", rt.getPrivateChatById)

	// Delete private chat
	rt.router.DELETE("/wasatext/:username/conversations/private_chats/:conversationId", rt.deletePrivateChat)

	// ------------------------------------------------
	// GROUP CHATS
	// ------------------------------------------------
	// Create new group
	rt.router.POST("/wasatext/:username/conversations/groups", rt.createNewGroup)

	// Get specific group chat
	rt.router.GET("/wasatext/:username/conversations/groups/:conversationId", rt.getGroupChat)

	// Add members to group
	rt.router.POST("/wasatext/:username/conversations/groups/:conversationId/members", rt.addToGroup)

	// Remove member from group
	rt.router.DELETE("/wasatext/:username/conversations/groups/:conversationId/members/:userId", rt.removeMember)

	// Leave group
	rt.router.POST("/wasatext/:username/conversations/groups/:conversationId/leave", rt.leaveGroup)

	// Make user admin
	rt.router.POST("/wasatext/:username/conversations/groups/:conversationId/admins", rt.makeAdmin)

	// Remove admin status
	rt.router.DELETE("/wasatext/:username/conversations/groups/:conversationId/admins/:usernameAdmin", rt.removeAdminStatus)

	// Update group name
	rt.router.PUT("/wasatext/:username/conversations/groups/:conversationId/name", rt.setGroupName)

	// Update group description
	rt.router.PUT("/wasatext/:username/conversations/groups/:conversationId/description", rt.setGroupDescription)

	// Update group photo
	rt.router.PUT("/wasatext/:username/conversations/groups/:conversationId/photo", rt.setGroupPhoto)

	// ------------------------------------------------
	// MESSAGES
	// ------------------------------------------------
	// Get messages in conversation
	rt.router.GET("/wasatext/:username/conversations/:conversationId/messages", rt.getConversation)

	// Send new message
	rt.router.POST("/wasatext/:username/conversations/:conversationId/messages", rt.sendMessage)

	// Edit message
	rt.router.PATCH("/wasatext/:username/conversations/:conversationId/messages/:messageId", rt.editMessage)

	// Delete message
	rt.router.DELETE("/wasatext/:username/conversations/:conversationId/messages/:messageId", rt.deleteMessage)

	// Forward message
	rt.router.POST("/wasatext/:username/conversations/:conversationId/messages/forward", rt.forwardMessage)

	// ------------------------------------------------
	// REACTIONS
	// ------------------------------------------------
	// Add reaction
	rt.router.POST("/wasatext/:username/conversations/:conversationId/messages/:messageId/reactions", rt.commentMessage)

	// Remove reaction
	rt.router.DELETE("/wasatext/:username/conversations/:conversationId/messages/:messageId/reactions", rt.uncommentMessage)

	// Get all reactions for a message
	rt.router.GET("/wasatext/:username/conversations/:conversationId/messages/:messageId/reactions", rt.getAllMessageReactions)

	// ------------------------------------------------
	// FAVOURITES
	// ------------------------------------------------
	// Get all favourites (Global)
	rt.router.GET("/wasatext/:username/favourites", rt.getAllUserFavourites)

	// Get favourites in conversation
	rt.router.GET("/wasatext/:username/conversations/:conversationId/favourites", rt.getConversationFavourites)

	// Add message to favouritesgetConversationFavourites
	rt.router.POST("/wasatext/:username/conversations/:conversationId/messages/:messageId/favourite", rt.addMessageToFavourites)

	// Remove message from favourites
	rt.router.DELETE("/wasatext/:username/conversations/:conversationId/messages/:messageId/favourite", rt.removeMessageFromFavourites)


	return rt.router
}
