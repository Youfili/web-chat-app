package api

import (
	"net/http"
)

// Handler returns an instance of httprouter.Router that handle APIs registered here
func (rt *_router) Handler() http.Handler {

	// Register routes
	rt.router.GET("/liveness", rt.liveness)
	rt.router.GET("/context", rt.wrap(rt.getContextReply)

	// Login & Registration
	rt.router.POST("/wasatext/login", rt.doLogin)
	rt.router.POST("/wasatext/register", rt.registerUser)

	// User Profile & Status
	rt.router.GET("/wasatext/:username/profile", rt.myProfileDetails)
	rt.router.PATCH("/wasatext/:username/profile/status", rt.modifyProfileStatus)
	rt.router.PUT("/wasatext/:username/profile/username", rt.setMyUserName)
	rt.router.PUT("/wasatext/:username/profile/photo", rt.setMyPhoto)

	// User Search & Blocking
	rt.router.GET("/wasatext/:username/users/search", rt.searchUser)
	rt.router.GET("/wasatext/:username/users/:userId", rt.getUserProfile)
	rt.router.GET("/wasatext/:username/users/blocked", rt.allUsersBlocked)
	rt.router.POST("/wasatext/:username/users/:userId/block", rt.blockUser)
	rt.router.DELETE("/wasatext/:username/users/:userId/block", rt.unblockUser)

	// Conversations (General List & Search)
	rt.router.GET("/wasatext/:username/conversations", rt.getMyConversations)
	rt.router.GET("/wasatext/:username/conversations/search", rt.searchConversation)
	rt.router.GET("/wasatext/:username/conversations/:conversationId/stats", rt.getConversationStats)
	rt.router.POST("/wasatext/:username/conversations/:conversationId/typing", rt.isTyping)

	// Private Chats (Creation & Deletion specifically defined)
	rt.router.POST("/wasatext/:username/conversations/private_chats", rt.newPrivateChat)
	rt.router.GET("/wasatext/:username/conversations/private_chats/:conversationId", rt.getPrivateChatById)
	rt.router.DELETE("/wasatext/:username/conversations/private_chats/:conversationId", rt.deletePrivateChat)

	// Group Chats (Creation & Management)
	rt.router.POST("/wasatext/:username/conversations/groups", rt.createNewGroup)
	rt.router.GET("/wasatext/:username/conversations/groups/:conversationId", rt.getGroupChat)
	rt.router.PUT("/wasatext/:username/conversations/groups/:conversationId/name", rt.setGroupName)
	rt.router.PUT("/wasatext/:username/conversations/groups/:conversationId/description", rt.setGroupDescription)
	rt.router.PUT("/wasatext/:username/conversations/groups/:conversationId/photo", rt.setGroupPhoto)
	rt.router.POST("/wasatext/:username/conversations/groups/:conversationId/leave", rt.leaveGroup)

	// Group Members Management
	rt.router.POST("/wasatext/:username/conversations/groups/:conversationId/members", rt.addToGroup)
	rt.router.DELETE("/wasatext/:username/conversations/groups/:conversationId/members/:userId", rt.removeMember)

	// Group Admins Management
	rt.router.POST("/wasatext/:username/conversations/groups/:conversationId/admins", rt.makeAdmin)
	rt.router.DELETE("/wasatext/:username/conversations/groups/:conversationId/admins/:usernameAdmin", rt.removeAdminStatus)

	// Messages (List, Send, Edit, Delete)
	rt.router.GET("/wasatext/:username/conversations/:conversationId/messages", rt.allMessagesOfConversation)
	rt.router.POST("/wasatext/:username/conversations/:conversationId/messages", rt.sendMessage)
	rt.router.PUT("/wasatext/:username/conversations/:conversationId/messages/:messageId", rt.editMessage)
	rt.router.DELETE("/wasatext/:username/conversations/:conversationId/messages/:messageId", rt.deleteMessage)

	// Message Interactions (Reply & Forward)
	rt.router.POST("/wasatext/:username/conversations/:conversationId/messages/reply", rt.replyToMessage)
	rt.router.POST("/wasatext/:username/conversations/:conversationId/messages/forward", rt.forwardMessage)

	// Reactions
	rt.router.POST("/wasatext/:username/conversations/:conversationId/messages/:messageId/reactions", rt.commentMessage)
	rt.router.DELETE("/wasatext/:username/conversations/:conversationId/messages/:messageId/reactions", rt.uncommentMessage)
	rt.router.GET("/wasatext/:username/conversations/:conversationId/messages/:messageId/reactions", rt.getAllMessageReactions)

	// Global (users) Search (Messages)
	rt.router.GET("/wasatext/:username/search/messages", rt.searchMessageGlob)

	// Favourites
	rt.router.GET("/wasatext/:username/favourites", rt.getAllUserFavourites)
	rt.router.GET("/wasatext/:username/conversations/:conversationId/favourites", rt.getConversationFavourites)
	rt.router.POST("/wasatext/:username/conversations/:conversationId/messages/:messageId/favourite", rt.addMessageToFavourites)
	rt.router.DELETE("/wasatext/:username/conversations/:conversationId/messages/:messageId/favourite", rt.removeMessageFromFavourites)

	return rt.router
}
