// Modulo che contiene le chiamate HTTP


import axios from "./axios";

// Variabile globale per gestire l'annullamento delle richieste dei messaggi
// (Evita race conditions quando si cambia chat velocemente)
let messagesAbortController = null;

export default {
	// =================================================================
	// AUTHENTICATION (/session)
	// =================================================================
	
	async login(username) {
		const response = await axios.post("/session/login", { username });
		return response.data;
	},

	async register(username) {
		const response = await axios.post("/session/register", { username });
		return response.data;
	},

	// =================================================================
	// USERS & PROFILE (/wasatext/:username)
	// =================================================================

	// Ottiene il profilo dell'utente loggato
	async getMyProfile(username) {
		const response = await axios.get(`/wasatext/${username}/profile`);
		return response.data;
	},

	// Cerca utenti (Fuzzy search)
	async searchUsers(username, query) {
		const response = await axios.get(`/wasatext/${username}/user_search`, {
			params: { usernameSearched: query }
		});
		return response.data;
	},

	// Ottiene il profilo di un altro utente (dato lo username target)
	async getUserProfile(myUsername, targetUsername) {
		const response = await axios.get(`/wasatext/${myUsername}/users/${targetUsername}`);
		return response.data;
	},

	// Aggiorna lo username
	async setMyUsername(currentUsername, newUsername) {
		const response = await axios.put(`/wasatext/${currentUsername}/profile/username`, {
			newUsername: newUsername
		});
		return response.data;
	},

	// Aggiorna lo stato (es. "Available")
	async setMyStatus(username, status) {
		const response = await axios.patch(`/wasatext/${username}/profile/status`, {
			status: status
		});
		return response.data;
	},

	// Aggiorna la foto profilo
	async setMyPhoto(username, photoUrl) {
		const response = await axios.put(`/wasatext/${username}/profile/photo`, {
			profilePhoto: photoUrl
		});
		return response.data;
	},

	// =================================================================
	// CONVERSATIONS & CHATS
	// =================================================================

	// Ottiene la lista di tutte le conversazioni 
	async getConversations(username) {
		const response = await axios.get(`/wasatext/${username}/conversations`);
		return response.data;
	},

	// Crea una nuova chat privata (e invia il primo messaggio)
	// Nota: recipientId deve essere l'uuid dell'altro utente (quello che riceverà il messaggio)
	async createPrivateChat(username, recipientId, initialMessage) {
		const response = await axios.post(`/wasatext/${username}/private_chats`, {
			recipientUser: recipientId,
			initialMessage: initialMessage
		});
		return response.data;
	},

	// =================================================================
	// GROUPS MANAGEMENT
	// =================================================================

	// Crea un nuovo gruppo
	async createGroup(username, groupName, membersIds) {
		const response = await axios.post(`/wasatext/${username}/groups`, {
			groupName: groupName,
			groupDescription: "", // Opzionale iniziale
			members: membersIds
		});
		return response.data;
	},

	// Aggiorna info gruppo (Nome, Descrizione o Foto)
	// Uso la logica dell'helper backend: passo solo il campo che mi interessa
	async updateGroupInfo(username, conversationId, type, value) {
		// type può essere: "name", "description", "photo"
		let payload = {};
		if (type === 'name') payload.groupName = value;
		if (type === 'description') payload.groupDescription = value;
		if (type === 'photo') payload.groupPhoto = value;

		const response = await axios.put(
			`/wasatext/${username}/groups/${conversationId}/${type}`, 
			payload
		);
		return response.data;
	},

	async addToGroup(username, conversationId, userIdToAdd) {
		const response = await axios.post(`/wasatext/${username}/groups/${conversationId}/members`, {
			usersIdsToAdd: [userIdToAdd]
		});
		return response.data;
	},

	async removeFromGroup(username, conversationId, userIdToRemove) {
		await axios.delete(`/wasatext/${username}/groups/${conversationId}/members/${userIdToRemove}`);
	},

	async leaveGroup(username, conversationId) {
		await axios.post(`/wasatext/${username}/groups/${conversationId}/leave`);
	},

	// Promuove un utente ad Admin
    async makeAdmin(username, conversationId, userId) {
        const response = await axios.post(`/wasatext/${username}/groups/${conversationId}/admins`, {
            userIdToPromote: userId
        });
        return response.data;
    },

    // Rimuove lo stato di Admin da un utente (lo declassa a membro semplice)
    async removeAdminStatus(username, conversationId, userId) {
        const response = await axios.delete(`/wasatext/${username}/groups/${conversationId}/admins/${userId}`);
        return response.data;
    },

	// Ottiene i dettagli completi di un gruppo (inclusi membri e admin)
	async getGroupChat(username, conversationId) {
		const response = await axios.get(`/wasatext/${username}/groups/${conversationId}`);
		return response.data;
	},

	// Uscita dell'utente dal gruppo
    async leaveGroup(username, conversationId) {
        // Nota: non serve che passo i dati nel body (secondo parametro) perché il backend legge tutto dall'URL
        const response = await axios.post(`/wasatext/${username}/groups/${conversationId}/leave`);
        return response.data;
    },

	// =================================================================
	// MESSAGES (Con Logica di Abort)
	// =================================================================

	async getChatMessages(username, conversationId, limit = 50, beforeTimestamp = null) {
		// 1. Se st0 caricando una NUOVA chat (non paginazione), annulla richieste precedenti
		if (!beforeTimestamp && messagesAbortController) {
			messagesAbortController.abort();
		}

		// 2. Crea nuovo controller se non esiste o se è stato abortito
		if (!beforeTimestamp || !messagesAbortController) {
			messagesAbortController = new AbortController();
		}
		
		try {
			const params = { limit };
			if (beforeTimestamp) params.before = beforeTimestamp;

			const response = await axios.get(
				`/wasatext/${username}/conversations/${conversationId}/messages`, 
				{
					signal: messagesAbortController.signal,
					params: params
				}
			);
			return response.data;
		} catch (error) {
			if (axios.isCancel(error)) {
				// Errore intenzionale (cambio chat), lo lancio in modo che la UI possa ignorarlo
				throw { isCanceled: true };
			}
			throw error;
		}
	},

	async sendMessage(username, conversationId, content) {
		const response = await axios.post(
			`/wasatext/${username}/conversations/${conversationId}/messages`,
			{ contentMess: content }
		);
		return response.data;
	},

	async deleteMessage(username, conversationId, messageId) {
		await axios.delete(`/wasatext/${username}/conversations/${conversationId}/messages/${messageId}`);
	},

	// Smart Forwarding
	// targetType: 'conversation' (uuid chat) oppure 'user' (uuid utente)
	async forwardMessage(username, currentConvId, messageId, targetType, targetId) {
		const payload = {};
		if (targetType === 'conversation') {
			payload.targetConversationId = targetId;
		} else {
			payload.targetUserId = targetId;
		}

		const response = await axios.post(
			`/wasatext/${username}/conversations/${currentConvId}/messages/${messageId}/forward`,
			payload
		);
		return response.data;
	},

	// =================================================================
	// REACTIONS & READ STATUS
	// =================================================================

	async addReaction(username, conversationId, messageId, emoji) {
		const response = await axios.post(
			`/wasatext/${username}/conversations/${conversationId}/messages/${messageId}/reactions`,
			{ emoji: emoji }
		);
		return response.data;
	},

	async removeReaction(username, conversationId, messageId) {
		await axios.delete(
			`/wasatext/${username}/conversations/${conversationId}/messages/${messageId}/reactions`
		);
	},

	async markAsRead(username, conversationId, lastMessageId) {
		await axios.put(
			`/wasatext/${username}/conversations/${conversationId}/read_status`,
			{ lastReadMessageId: lastMessageId }
		);
	},

	// =================================================================
	// MEDIA UPLOAD
	// =================================================================

	async uploadFile(fileObj, type = 'media') {
		// Per inviare file utilizzo FormData
		const formData = new FormData();
		formData.append('file', fileObj);

		// type può essere 'avatar' o 'media'
		const response = await axios.post(`/media/upload`, formData, {
			headers: {
				'Content-Type': 'multipart/form-data'
			},
			params: { type: type }
		});
		
		// Ritorna l'oggetto { url: "..." }
		return response.data;
	}
};