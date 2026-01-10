// Modulo che contiene le chiamate HTTP
import axios from "./axios";

// Variabile globale per gestire l'annullamento delle richieste dei messaggi --> (Evita race conditions quando si cambia chat velocemente)
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

	// Ottiene una specifica chat privata tramite ID
    async getPrivateChatById(username, conversationId) {
        const response = await axios.get(`/wasatext/${username}/private_chats/${conversationId}`);
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
	async setGroupName(username, conversationId, newName) {
        const response = await axios.put(
            `/wasatext/${username}/groups/${conversationId}/name`, 
            { groupName: newName }
        );
        return response.data;
    },

    async setGroupDescription(username, conversationId, newDesc) {
        const response = await axios.put(
            `/wasatext/${username}/groups/${conversationId}/description`, 
            { groupDescription: newDesc }
        );
        return response.data;
    },

    async setGroupPhoto(username, conversationId, newPhotoUrl) {
        const response = await axios.put(
            `/wasatext/${username}/groups/${conversationId}/photo`, 
            { groupPhoto: newPhotoUrl }
        );
        return response.data;
    },


	// Richiama internamente le funzioni specifiche qui sopra
	// HO usato questa implementazione a causa dell'errore presentato dal professore in cui
	// GLi endpoints nella OpenAPI Documentation vanno specificati in modo chiaro
    async updateGroupInfo(username, conversationId, type, value) {
        switch (type) {
            case 'name':
                return this.setGroupName(username, conversationId, value);
            case 'description':
                return this.setGroupDescription(username, conversationId, value);
            case 'photo':
                return this.setGroupPhoto(username, conversationId, value);
            default:
                console.error(`Update type '${type}' not recognized.`);
                return null;
        }
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
		//  Se st0 caricando una NUOVA chat, annulla richieste precedenti	--> (Annulla richieste vecchie se cambio chat veloce)
		if (!beforeTimestamp && messagesAbortController) {
			messagesAbortController.abort();
		}

		// Crea nuovo controller se non esiste o se è stato abortito
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
			if (error.code === "ERR_CANCELED" || error.name === "CanceledError") {
                throw { isCanceled: true };
            }
            throw error;
		}
	},

	async sendMessage(username, conversationId, content, photoUrl = null, replyToMessageId = null) {
        const payload = { contentMess: content };
		
		// Se c'è una foto, la aggiungo al payload
        if (photoUrl) {
            payload.messagePhoto = photoUrl;
        } 

		// Se c'è una risposta, la aggiungo
        if (replyToMessageId) payload.replyTo = replyToMessageId;

        const response = await axios.post(
            `/wasatext/${username}/conversations/${conversationId}/messages`,
            payload
        );
        return response.data;
    },

	async deleteMessage(username, conversationId, messageId) {
		await axios.delete(`/wasatext/${username}/conversations/${conversationId}/messages/${messageId}`);
	},

	// Funzione per modificare un messaggio
    async editMessage(username, conversationId, messageId, newContent) {
        // La chiave "contentMess" corrisponde alla struct Go: type MessageUpdate struct { ContentMess string `json:"contentMess"` }
        const response = await axios.patch(
            `/wasatext/${username}/conversations/${conversationId}/messages/${messageId}`,
            { contentMess: newContent } 
        );
        return response.data;
    },

	// Funzione per inoltrare un messaggio
    async forwardMessage(username, currentConversationId, messageId, targetDict) {
        // targetDict sarà un oggetto tipo: { targetConversationId: "..." } oppure { targetUserId: "..." }
        return axios.post(
            `/wasatext/${username}/conversations/${currentConversationId}/messages/${messageId}/forward`,
            targetDict
        );
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

    async getMessageReactions(username, conversationId, messageId) {
        const response = await axios.get(
            `/wasatext/${username}/conversations/${conversationId}/messages/${messageId}/reactions`
        );
        return response.data; // Torna { reactions: [...] }
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