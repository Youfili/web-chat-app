<script>
import { nextTick } from 'vue'
import api from '@/services/api'
import LoadingSpinner from '@/components/LoadingSpinner.vue'
import { emojiListSet } from '@/services/emoji.js';

export default {
    // Registrazione componenti
    components: {
        LoadingSpinner,
    },
    
    // Parte del Pattern in cui dichiaro tutte le variabili reattive 
    data() {
        return {
            // Recupero lo username dalla sessione
            username: localStorage.getItem('username'),

            // Variabili
            myProfile: null,
            conversations: [],
            loading: true,
            errorMsg: null,

            selectedChatId: null,       // Per sapere quale chat è aperta
            newMessageText: "",         // per l'invio di un nuovo messaggio

            messages: [],               // La lista dei messaggi della chat aperta
            chatLoading: false,         // Loading specifico per la colonna destra

            // Variabili per la ricerca
            searchQuery: "",
            searchResults: [],          // Lista utenti trovati
            isSearching: false,         // Se true, mostro i risultati invece delle chat

            // Variabili per il timer
            pollingInterval: null,

            // Variabili Gruppo
            showGroupModal: false,     // Apre/Chiude la finestra
            newGroupName: "",          // Nome del nuovo gruppo
            groupSearchQuery: "",      // Ricerca utenti per il gruppo
            groupSearchResults: [],    // Risultati ricerca
            selectedUsers: [],         // Utenti selezionati (ID e Username)
            currentChatAdmins: [],     // Lista admin della chat corrente (quella che sto esaminando)

            // Variabili Info Gruppo
            showInfoModal: false,
            groupInfo: null,            // Conterrà i dettagli del gruppo cliccato

            // Variabili Info Utente
            userInfo: null,              // Dati utente chat privata

            // Variabile Emoji
            showEmojiPicker: false,     // Variabile per Mostrare/Nascondere le Emoji
            emojiList: emojiListSet,     // Lista di Emoji

            // Variabili Modifica Messaggio
            editingMessageId: null,      // ID del messaggio in modifica (null se nessuno)
            originalMessageText: "",     // Testo originale per backup

            // Variabili Inolto Messaggio
            showForwardModal: false,    // Mostra/Nasconde il modale
            messageToForward: null,     // L'oggetto messaggio che sto inoltrando
            forwardSearchQuery: "",     // Testo ricerca inoltro
            forwardSearchResults: [],   // Risultati ricerca inoltro

            // Variabili Reaction
            reactionOptions: ["👍", "❤️", "😂", "😮", "😢", "😡"],  // Emoji da usare per le Reaction

            // Variabili Reply to Message
            replyingToMessage: null,    // Oggetto del messaggio a cui sto rispondendo

            // Variabili per gestire l'immagine con didascalia
            selectedFile: null,        // Il file "grezzo" da caricare
            selectedFilePreview: null, // L'URL locale per l'anteprima
        }
    },

    // Per calcolare dati derivati
    computed: {
        // Restituisce l'oggetto della chat attualmente selezionata
        activeChat() {
            return this.conversations.find(c => c.id === this.selectedChatId)
        },
        // Computed property per pulizia codice
        isEditing() {
            return this.editingMessageId !== null
        }
    },

    // Parte del Pattern in cui metto tutte le funzioni
    methods: {
        // Navigazione Profilo
        goToProfile() {
            this.$router.push('/profile')
        },

        // Funzione per caricare i dati iniziali
        async loadData() {
            try {
                this.loading = true
                // Restituisco il profilo dell'utente loggato in sessione
                this.myProfile = await api.getMyProfile(this.username)
                
                // Restituisco le chat dell'utente loggato in sessione
                const response = await api.getConversations(this.username)
                this.conversations = (response.conversations || []).filter(c => c)  // Aggiungo .filter(c => c) per rimuovere i null
                
            } catch (e) {
                this.errorMsg = e.toString()
            } finally {
                this.loading = false
            }
        },

        // Funzione per ricavare il nome della chat (a seconda se è un chat di gruppo o privata)
        getChatName(chat) {
            if (chat.conversationType === 'group') {
                // Se il gruppo ha un nome viene ritornato
                if (chat.groupName && chat.groupName.trim() !== "") {
                    return chat.groupName;
                }
                // Se non ha nome, mostro "Group Chat" 
                return "Group Chat";
            } else {
                // Chat privata: restituisco il nome dell'altro utente coinvolto nella chat.
                return chat.recipientUsername || "Unknown User";
            }
        },

        // Funzione helper per l'avatar (con avatar intendo anche l'immagine del gruppo, non solo del profilo di utenti) (a seconda se è un chat di gruppo o privata)
        getChatPhoto(chat) {
            // Se c'è una foto specifica (gruppo o utente), utilizzo quella
            if (chat.conversationType === 'group' && chat.groupPhoto) return chat.groupPhoto
            if (chat.conversationType === 'private' && chat.userPhoto) return chat.userPhoto
            
            // Fallback immagine default
            // NOTA (Promemoria per me): Lo slash iniziale "/" indica la root della cartella 'public' di Vue
            return '/default_avatar.jpg'
        },


        // -------------
        //## MESSAGGI
        // ------------
        async selectChat(chatId) {
            this.showEmojiPicker = false // Chiudo Emoji Panel se aperto
            this.selectedChatId = chatId    // Aggiorno ID chat selezionata

            // Resetto e attivo loading
            this.messages = [] 
            this.currentChatAdmins = [] // Reset
            this.chatLoading = true
            this.errorMsg = null // Resetto errori globali 

            try {
                // Se è un gruppo, recupero i dettagli (inclusi admin)
                // Cercoo il tipo nella lista locale
                const chat = this.conversations.find(c => c.id === chatId)
                const chatType = chat ? chat.conversationType : null
                
                if (chatType === 'group') {
                    try {
                        const groupDetails = await api.getGroupChat(this.username, chatId)
                        this.currentChatAdmins = groupDetails.admins || []
                    } catch (err) {
                        console.error("Errore recupero dettagli gruppo", err)
                    }
                }

                // Carico i messaggi
                // Chiamo l'API (api.js gestirà l'abort delle richieste vecchie)
                let response = await api.getChatMessages(this.username, chatId)  // Il browser manda la richiesta al server Go (porta 3000) e aspetta.
                this.messages = response.messages.reverse()
                
                this.markAsRead()    // Ho caricato i messaggi, quindi li sto leggendo ora.
                

            } catch (e) {
                // Gestione Cancellazione
                if (e.isCanceled) {
                    console.log("Richiesta messaggi annullata (utente ha cambiato chat)")
                    return; 
                }
                // Gestione errore reale
                this.errorMsg = "Impossibile caricare i messaggi: " + e.toString()
            } finally {
                // Ragionamento che tengo per iscritto per averlo sott'occhio in fase di debug:
                // Tolgo il loading solo se la richiesta NON è stata cancellata
                // (Nota: se è stata cancellata, il loading resta true finché quella nuova non finisce)
                // --> api.js lancia l'eccezione, quindi sono nel catch.
                // Se arrivo qui con successo, tolgo il loading.
                this.chatLoading = false

                this.scrollToBottom()    // Poi scrollo

            }
        },

        // Funzione per inserire un Emoji nel messaggio che voglio inviare
        addEmoji(emojiChar) {
            // Ottengo il riferimento all'elemento input del DOM
            const input = this.$refs.messageInput;
            
            // Se per qualche motivo l'input non esiste, aggiungo solo alla fine (fallback)
            if (!input) {
                this.newMessageText += emojiChar;
                return;
            }

            // Trovo la posizione del cursore (inizio e fine selezione)
            const start = input.selectionStart;
            const end = input.selectionEnd;
            
            // Spezzo il testo attuale in due parti e inserisco l'emoji nel mezzo
            const text = this.newMessageText;
            const before = text.substring(0, start);
            const after  = text.substring(end);
            
            this.newMessageText = before + emojiChar + after;

            // Dopo che Vue ha aggiornato il DOM, rimetto il focus e sposto il cursore
            this.$nextTick(() => {
                input.focus();
                // Sposto il cursore DOPO l'emoji appena inserita
                const newCursorPos = start + emojiChar.length;
                input.setSelectionRange(newCursorPos, newCursorPos);
            });
        },

        
        // Inizia la modifica di un messaggio
        startEditing(msg) {
            this.editingMessageId = msg.id;
            this.originalMessageText = this.newMessageText; // Salva bozza corrente se c'è
            this.newMessageText = msg.contentMess; // Mette il testo vecchio nell'input (cosi lo modifico)
            
            // Focus sull'input
            this.$nextTick(() => {
                if (this.$refs.messageInput) this.$refs.messageInput.focus();
            });
        },

        // Annulla la modifica
        cancelEditing() {
            this.editingMessageId = null;
            this.newMessageText = ""; // Pulisce l'input 
        },

        // Elimina messaggio
        async doDeleteMessage(msgId) {
            if (!confirm("Are you sure you want to delete this message?")) return;
            try {
                await api.deleteMessage(this.username, this.selectedChatId, msgId);
                
                // Rimuove messaggio dalla lista locale (senza ricaricare tutto)
                this.messages = this.messages.filter(m => m.id !== msgId);
                
                // Se stavo modificando proprio questo messaggio, esco dall'edit
                if (this.editingMessageId === msgId) this.cancelEditing();
                
            } catch (e) {
                alert("Error deleting: " + e.toString());
            }
        },


        // Apre il modale di inoltro
        openForwardModal(msg) {
            this.messageToForward = msg;
            this.showForwardModal = true;
        },


        // ----------------
        // FORWARD
        // ----------------

        // Esegue l'inoltro di un messaggio verso una chat specifica
        async doForwardToChat(targetChat) {
            if (!this.messageToForward) return;

            // Chiedo conferma per evitare click accidentali
            if(!confirm(`Forward this message to ${this.getChatName(targetChat)}?`)) return;

            try {
                // Costruisco il payload JSON secondo lo struct go che ho definito in service/database/struct.go
                const payload = { 
                    targetConversationId: targetChat.id 
                };

                // Chiamo l'API
                await api.forwardMessage(
                    this.username, 
                    this.selectedChatId, // ID della chat DA CUI inoltro
                    this.messageToForward.id, // ID del messaggio
                    payload
                );

                // Chiudo e resetto
                this.showForwardModal = false;
                this.messageToForward = null;

                alert(`Message forwarded to ${this.getChatName(targetChat)}!`);
                
                // Se ho inoltrato alla chat STESSA in cui mi trovo, faccio un refresh per vedere il nuovo messaggio
                if (targetChat.id === this.selectedChatId) {
                    await this.refreshChat();
                    this.scrollToBottom();
                }

            } catch (e) {
                alert("Errore durante l'inoltro: " + e.toString());
            }
        },

        // Cerca utenti globali per l'inoltro (Non solo le chat che già ho aperto)
        async searchUsersForForward() {
            // Se la casella è vuota, pulisco i risultati
            if (this.forwardSearchQuery.length < 1) {
                this.forwardSearchResults = [];
                return;
            }
            try {
                const response = await api.searchUsers(this.username, this.forwardSearchQuery);
                // Filtro: rimuovo me stesso dalla lista
                this.forwardSearchResults = response.users.filter(u => u.username !== this.username);
            } catch (e) {
                console.error("Errore ricerca forward:", e);
            }
        },

        // Esegue l'inoltro verso un NUOVO utente (o esistente, ci pensa il backend)
        async doForwardToUser(user) {
            if (!this.messageToForward) return;

            if(!confirm(`Forward message to ${user.username}?`)) return;

            try {
                // Costruisco il payload specificando targetUserId
                // Il backend capirà che deve cercare o creare una chat privata con questo utente
                const payload = { 
                    targetUserId: user.id 
                };
                
                await api.forwardMessage(
                    this.username, 
                    this.selectedChatId, 
                    this.messageToForward.id, 
                    payload
                );

                // Reset e chiusura
                this.showForwardModal = false;
                this.messageToForward = null;
                this.forwardSearchQuery = "";
                this.forwardSearchResults = [];

                alert(`Message forwarded to ${user.username}!`);
                
                // Ricarico le conversazioni per far apparire la nuova chat in cima
                await this.refreshConversations();

            } catch (e) {
                alert("Error forwarding: " + e.toString());
            }
        },


        // REPLY

        getRepliedMessageDetails(replyId) {
            // Caso di sicurezza: se replyId è null o undefined
            if (!replyId) return { senderUsername: 'Unknown', contentMess: 'Message unavailable' };

            // Se replyId è un "oggetto", prendo l'id, altrimenti uso replyId direttamente
            const idToSearch = (typeof replyId === 'object' && replyId !== null) ? replyId.id : replyId;

            // Cerco il messaggio dentro la lista 'this.messages' che ho già scaricato
            const foundMsg = this.messages.find(m => m.id === idToSearch);

            if (foundMsg) {
                return foundMsg;
            } else {
                // Fallback se il messaggio è troppo vecchio e non è nella lista caricata
                return { senderUsername: 'User', contentMess: 'Message not loaded' };
            }
        },

        // Reactions
        // Gestisce il click su una reaction (Rimozione / Sostituzione)
        async reactToMessage(msg, emoji) {
            // Se msg.reactions è null/undefined lo inizializzo
            if (!msg.reactions) msg.reactions = [];

            // Cerco se ho già messo una reaction a questo messaggio
            const myExistingReaction = msg.reactions.find(r => r.senderUserId === this.myProfile.id);

            try {
                // CASO 1: Ho già una reazione
                if (myExistingReaction) {
                    
                    // Se clicco la STESSA emoji -> La Rimuovo 
                    if (myExistingReaction.emoji === emoji) {
                        await api.removeReaction(this.username, this.selectedChatId, msg.id);
                        // Aggiorno UI locale
                        msg.reactions = msg.reactions.filter(r => r.senderUserId !== this.myProfile.id);
                        return;
                    } 
                    
                    // (Sostituzione) Se clicco un'emoji DIVERSA -> Rimuovo vecchia e metto nuova 
                    await api.removeReaction(this.username, this.selectedChatId, msg.id);
                }

                // CASO 2: Aggiungo la nuova reazione (o quella sostituita)
                const newReaction = await api.addReaction(this.username, this.selectedChatId, msg.id, emoji);
                
                // Aggiorno UI locale:
                // Pulisco eventuali mie reazioni vecchie (per sicurezza UI)
                msg.reactions = msg.reactions.filter(r => r.senderUserId !== this.myProfile.id);
                // Aggiungo la nuova
                msg.reactions.push(newReaction);

            } catch (e) {
                console.error("Errore reazione:", e);
                alert("Impossibile reagire al messaggio.");
            }
        },


        // Invia un nuovo messaggio
        async sendMsg() {
            // Evito invii vuoti o se nessuna chat è selezionata
            // Devo avere almeno (Testo OPPURE File (quindi anche entrambi vanno bene)) E una chat selezionata
            if ((!this.newMessageText.trim() && !this.selectedFile) || !this.selectedChatId) return;

            try {

                this.showEmojiPicker = false // Chiudo le emoji quando invio
                let photoUrl = null;
                
                // Per semplicita di lettura codice ho commentanto dividento il codice in "Fasi"

                // FASE 1: Se c'è un file in "canna", lo carico ora
                if (this.selectedFile) {
                    const uploadResp = await api.uploadFile(this.selectedFile, 'media');
                    photoUrl = uploadResp.url; // Recupero l'URL dal server
                }

                // FASE 2: Invio il messaggio completo
                if (this.isEditing){

                    // Logica Editing Messaggio
                    const updatedMsg = await api.editMessage(
                        this.username,
                        this.selectedChatId,
                        this.editingMessageId,
                        this.newMessageText
                    );

                    // Aggiorno il messaggio nella lista locale
                    const index = this.messages.findIndex(m => m.id === this.editingMessageId);
                    if (index !== -1) {
                        this.messages[index] = updatedMsg; 
                    }
                    
                    // Esco dalla modalità edit
                    this.cancelEditing();

                } else {
                    
                    // Invio Normale (Caption + Foto)
                    // Logica Invio (Normale o Risposta)
                    const replyId = this.replyingToMessage ? this.replyingToMessage.id : null;

                    // Chiamata API
                    const response = await api.sendMessage(
                        this.username, 
                        this.selectedChatId, 
                        this.newMessageText,    // La Didascalia (Caption)
                        photoUrl,               // URL della foto
                        replyId                 // replyToMessageId
                    );

                    // Nel backend ho implementato getMessages che restitusce i messsaggi in DESC (dal più nuovo).
                    // Nel selectChat li ho girati (.reverse()). Quindi ora sono [Vecchio, ..., Nuovo].
                    // Quindi devo fare PUSH per aggiungere in fondo.

                    //Aggiorno la UI
                    this.messages.push(response) 

                    // FASE 3: Pulizia Totale
                    // Pulisco l'input e scrollo in basso
                    this.newMessageText = ""
                    this.removeSelectedFile();  // Pulisce file e anteprima
                    this.cancelReply();         // Reset
                    this.scrollToBottom() 

                }

            } catch (e) {
                this.errorMsg = "Error sending message: " + e.toString()
            }
        },


        // ------------------------
        // RISPOSTA A UN MESSAGGIO
        // ------------------------

        // Avvia la risposta
        startReplying(msg) {
            this.replyingToMessage = msg;
            this.$nextTick(() => {
                if (this.$refs.messageInput) this.$refs.messageInput.focus();
            });
        },

        // Annulla la risposta
        cancelReply() {
            this.replyingToMessage = null;
        },


        // Funzione Scroll To Bottom
        async scrollToBottom() {
            await nextTick() // Aspetto che Vue abbia aggiornato l'HTML con i nuovi messaggi
            const chatContainer = this.$refs.chatContainer 
            if (chatContainer) {
                // Imposto lo scroll verticale (scrollTop) pari all'altezza totale (scrollHeight) --> Risolvo il bug che avevo di scroll
                chatContainer.scrollTop = chatContainer.scrollHeight
            }
        },

        // Funzione chiamata quando utente scrive nella barra di ricerca
        async doSearch() {
            if (this.searchQuery.length < 1) {
                this.isSearching = false
                this.searchResults = []
                return
            }

            this.isSearching = true
            try {
                this.showEmojiPicker = false // Chiudo Emoji Panel se aperto

                const response = await api.searchUsers(this.username, this.searchQuery)
                // Filtro l'utente che ricerca dai risultati
                this.searchResults = response.users.filter(u => u.username !== this.username)
            } catch (e) {
                console.error("Search error", e)
            }
        },

        // Funzione chiamata quando clicco su un utente cercato
        async startChatWith(user) {
            try {
                this.showEmojiPicker = false // Chiudo Emoji Panel se aperto

                // Creo (o recupero, se già esiste) la chat privata
                // Nota: Qui metto un messaggio iniziale di benvenuto sandard --> la mia API createPrivateChat Richiede un initialMessage (scelta implementativa personale)
                const chat = await api.createPrivateChat(this.username, user.id, "👋")
                
                // Pulisco la ricerca
                this.searchQuery = ""
                this.isSearching = false
                
                // Ricarico la lista conversazioni (per far apparire quella nuova)
                await this.loadData()
                
                // Seleziono subito la nuova chat
                this.selectChat(chat.id)
                
            } catch (e) {
                alert("Error creating chat: " + e.toString())
            }
        },


        ////##  POLLING (Aggiornamento messaggi "live")

        // Funzione per aggiornare i messaggi della chat aperta
        async refreshChat() {
            if (!this.selectedChatId) return

            try {
                // Scarico i messaggi aggiornati (che contengono statusInfo corretto)
                // Nota: non resetto messages.value, confronto o sostituisco
                let response = await api.getChatMessages(this.username, this.selectedChatId)
                const newMessages = response.messages.reverse()
                
                // Controllo: Se il numero di messaggi è cambiato, aggiorno tutto e scrollo
                if (newMessages.length > this.messages.length) {
                    this.messages = newMessages

                    await this.markAsRead()        // Sono arrivati nuovi messaggi mentre guardavo la chat -> Li segno come letti

                    this.scrollToBottom()    //Scrollo in fondo perché c'è un nuovo messaggio
                }
                // Se la lunghezza è uguale, Nessun nuovo messaggio, ma forse è cambiato lo Stato (letto?)
                else {
                    // Sovrascrivo la lista per aggiornare lo stato (delivered -> read)
                    this.messages = newMessages 
                }

            } catch (e) {
                if (e.isCanceled) return
                console.error("Polling error", e)
            }
        },

        // Chiama l'API per segnare come letto
        async markAsRead() {
            if (this.messages.length > 0 && this.selectedChatId) {
                // L'ultimo messaggio (il più recente) è l'ultimo della lista (perché ho fatto reverse/push)
                const lastMsg = this.messages[this.messages.length - 1]
                
                // Chiamp l'API
                await api.markAsRead(this.username, this.selectedChatId, lastMsg.id)
                
                // Aggiorno localmente il contatore della chat sidebar a 0
                const chat = this.conversations.find(c => c.id === this.selectedChatId)
                if (chat) chat.unreadCount = 0
            }
        },

        // Funzione per aggiornare la lista conversazioni (Sidebar)
        async refreshConversations() {
            try {
                const response = await api.getConversations(this.username)
                this.conversations = (response.conversations || []).filter(c => c) // Aggiungo .filter(c => c) per rimuovere i null

                // Controllo Rimozione da una chat di gruppo --> Aggiornamento
                // Se ho una chat aperta...
                if (this.selectedChatId) {
                    // controllo se esiste ancora nella lista appena scaricata
                    const chatStillExists = this.conversations.some(c => c.id === this.selectedChatId)

                    // Se NON esiste più (sono stato rimosso o il gruppo è stato cancellato)
                    if (!chatStillExists) {
                        // Chiudo la schermata chat resettando le variabili
                        this.selectedChatId = null
                        this.messages = []
                        this.groupInfo = null
                        this.currentChatAdmins = []
                        
                        // Mostro un avviso all'utente 
                        // alert("You are no longer part of this chat.") 
                    }
                }

            } catch (e) {
                console.error("Polling conversations error", e)
            }
        },

        // ----------------

        ////## Upload messaggi immagini

        // Funzione che simula il click sull'input nascosto
        triggerFileUpload() {
            this.$refs.fileInput.click() 
        },

        // Gestione selezione file (Non invia, prepara solo) 
        handleFileUpload(event) {
            const file = event.target.files[0];
            if (!file) return;

            // Salvo il file nello stato
            this.selectedFile = file;

            // Creo un URL locale temporaneo per mostrare l'anteprima all'utente
            this.selectedFilePreview = URL.createObjectURL(file);

            // Resetto l'input file HTML (altrimenti non posso riselezionare lo stesso file se sbaglio)
            event.target.value = "";
            
            // Chiudo le emoji e do focus alla barra di testo per la didascalia
            this.showEmojiPicker = false;
            this.$nextTick(() => {
                if (this.$refs.messageInput) this.$refs.messageInput.focus();
            });
        },

        // Rimuove l'immagine selezionata (la "X" sull'anteprima) 
        removeSelectedFile() {
            this.selectedFile = null;
            if (this.selectedFilePreview) {
                URL.revokeObjectURL(this.selectedFilePreview); // Pulisce la memoria del browser (Url locale temporaneo --> usato per l'anteprima)
                this.selectedFilePreview = null;
            }
        },

        // Helper per capire se una stringa è un'immagine
        isImage(content) {
            if (!content) return false
            // Controllo se è una stringa
            if (typeof content !== 'string') return false
            
            // Rimuovo spazi bianchi eventuali
            const cleanContent = content.trim();

            // Deve iniziare con http (o https)
            const hasHttp = cleanContent.startsWith('http');
            
            // Deve contenere un'estensione immagine
            const hasExtension = cleanContent.match(/\.(jpeg|jpg|gif|png)/i) != null;

            return hasHttp && hasExtension;
        },

        // Gestisce il click sulla barra in alto
        async openChatInfo() {
            this.showEmojiPicker = false // Chiudo Emoji Panel se aperto

            // Recupero la chat attiva
            const chat = this.conversations.find(c => c.id === this.selectedChatId)
            if (!chat) return

            this.showInfoModal = true // Apro il modale
            
            // Reset dati precedenti 
            this.groupInfo = null
            this.userInfo = null

            // Logica Differente in base al tipo di Conversazione (di Gruppo o Privata)
            if (chat.conversationType === 'group') {
                try {
                    // API Gruppo
                    this.groupInfo = await api.getGroupChat(this.username, chat.id)
                } catch (e) {
                    alert("Error loading group info: " + e.toString())
                    this.showInfoModal = false
                }
            } else {
                // Logica Chat Privata
                try {
                    // API Utente: Uso la stessa chiamata del profilo, ma chiedendo l'username dell'altro (incosistenza del nome della funzione, ma preferisco fare cosi piuttosto che clonare codice) --> Potrei fare un generico getProfile()
                    this.userInfo = await api.getMyProfile(chat.recipientUsername)
                } catch (e) {
                    alert("Error loading user info: " + e.toString())
                    this.showInfoModal = false
                }
            }
        },

        // ----------------

        ////## Gestione Gruppi

        // Cerca utenti da aggiungere al gruppo
        async searchUsersForGroup() {
            this.showEmojiPicker = false // Chiudo Emoji Panel se aperto

            if (this.groupSearchQuery.length < 1) {
                this.groupSearchResults = []
                return
            }
            try {
                const response = await api.searchUsers(this.username, this.groupSearchQuery)
                // Filtro: toglie me stesso (utente loggato in sessione) e chi è già stato selezionato
                this.groupSearchResults = response.users.filter(u => 
                    u.username !== this.username && 
                    !this.selectedUsers.some(sel => sel.id === u.id)
                )
            } catch (e) {
                console.error(e)
            }
        },

        // Aggiunge/Rimuove un utente dalla lista "da aggiungere"
        toggleUserSelection(user) {
            // Se c'è già, lo tolgo
            if (this.selectedUsers.some(u => u.id === user.id)) {
                this.selectedUsers = this.selectedUsers.filter(u => u.id !== user.id)
            } else {
                // Altrimenti lo aggiungo
                this.selectedUsers.push(user)
            }
            // Pulisco la ricerca
            this.groupSearchQuery = ""
            this.groupSearchResults = []
        },

        // Chiama l'API per creare il gruppo
        async createGroup() {
            this.showEmojiPicker = false // Chiudo Emoji Panel se aperto

            if (!this.newGroupName.trim()) {
                alert("Please enter a group name")
                return
            }
            if (this.selectedUsers.length === 0) {
                alert("Select at least one member")
                return
            }

            try {
                // Estraggo solo gli ID
                const memberIds = this.selectedUsers.map(u => u.id)
                
                // Chiamata API
                const newGroup = await api.createGroup(this.username, this.newGroupName, memberIds)
                
                // Chiudo tutto e resetto
                this.showGroupModal = false
                this.newGroupName = ""
                this.selectedUsers = []
                
                // Ricarico la lista e apro il nuovo gruppo
                await this.loadData()
                this.selectChat(newGroup.id)

            } catch (e) {
                alert("Error creating group: " + e.toString())
            }
        },


        // Metodo che permette all'utente loggato in sessione di Uscire da un gruppo di cui fa parte
        async leaveGroup() {
            this.showEmojiPicker = false // Chiudo Emoji Panel se aperto

            // Controllo di sicurezza (classico, giusto per)
            if (!this.selectedChatId) return;

            // Conferma utente (per una protezione in più, nel caso l'utente si sbagliasse a cliccare)
            if (!confirm(`Sei sicuro di voler abbandonare questo gruppo?`)) {
                return;
            }

            try {
                // Chiamata API 
                await api.leaveGroup(this.username, this.selectedChatId);

                // Success - Aggiorno la UI
                alert("Hai abbandonato il gruppo con successo.");

                // Rimuovp la chat dalla lista locale (così sparisce subito dalla sidebar)
                this.conversations = this.conversations.filter(chat => chat.id !== this.selectedChatId);

                // Chiudo eventuali modali aperti
                this.showInfoModal = false; 

                // Resetto la vista centrale (chiudo la chat)
                this.selectedChatId = null;
                this.messages = [];
                this.groupInfo = null;
                this.currentChatAdmins = [];

            } catch (error) {
                // Gestione Errori
                if (error.response) {
                    
                    // CASO 409: IL BLOCCO ADMIN (quello che abbiamo creato nel backend)
                    if (error.response.status === 409) {
                        alert("⛔ IMPOSSIBILE USCIRE!\n\nSei l'unico amministratore del gruppo.\nDevi nominare un altro utente come amministratore prima di uscire.");
                    } 
                    // CASO 400/404: Utente non trovato o non nel gruppo
                    else if (error.response.status === 400 || error.response.status === 404) {
                        alert("Errore: Non sembri far parte di questo gruppo.");
                        // Chiudo comunque la chat per coerenza
                        this.selectedChatId = null;
                        this.conversations = this.conversations.filter(c => c.id !== this.selectedChatId);
                    } 
                    // Errore Generico Server
                    else {
                        console.error("Errore server:", error);
                        alert("Si è verificato un errore durante l'uscita. Riprova.");
                    }
                } else {
                    console.error("Errore di rete:", error);
                    alert("Errore di connessione al server.");
                }
            }
        },


        // Funzione per ordinare i membri: Admin prima degli altri membri "Normali"
        // Helper per ordinare i membri: Admin prima degli altri
        getSortedMembers() {
            // Controllo di sicurezza: se members è nullo o indefinito, ritorno array vuoto
            if (!this.groupInfo || !this.groupInfo.members) return []
            
            // Creo una copia sicura
            return [...this.groupInfo.members].sort((a, b) => {
                // a e b ora sono OGGETTI (dopo la modifica che ho implementato in struct) { userId: "...", username: "..." }
                
                // Recupero gli ID per controllare se sono admin
                // (admins rimane un array di stringhe nel backend)
                const idA = a.userId
                const idB = b.userId
                
                const isAdminA = this.groupInfo.admins.includes(idA)
                const isAdminB = this.groupInfo.admins.includes(idB)
                const isMeA = idA === this.myProfile.id
                const isMeB = idB === this.myProfile.id

                // Logica di ordinamento
                if (isMeA) return -1 // Utente Loggato sempre primo
                if (isMeB) return 1
                if (isAdminA && !isAdminB) return -1 // Admin sopra
                if (!isAdminA && isAdminB) return 1
                
                // Ordine alfabetico per nome (scelta implementativa, cosi da trovare un utente se lo si ricerca nel gruppo)
                return a.username.localeCompare(b.username)
            })
        },


        // -------------------
        
        
        // Formatto data e ora in modo sicuro
        formatDateTime(dateString) {
            if (!dateString) return ''
            const date = new Date(dateString)
            // Controllo se la data è valida
            if (isNaN(date.getTime())) return ''
            
            return date.toLocaleString([], {
                day: '2-digit', 
                month: '2-digit', 
                year: '2-digit', 
                hour: '2-digit', 
                minute: '2-digit'
            })
        },

        // -------------------

        // Helper sicuro per la data della sidebar
        getLastMessageTime(chatObj) {
            if (!chatObj || !chatObj.dtLastMessage) return ""
            return this.formatDateTime(chatObj.dtLastMessage)
        },

        // -------------------


        logout() {
            localStorage.clear()
            this.$router.push('/login')
        },

        // Funzione che chiude il picker se clicco fuori
        handleClickOutside(event) {
            // Se il picker è già chiuso, non faccio nulla
            if (!this.showEmojiPicker) return;

            // Recupero i riferimenti agli elementi HTML (che ho messo con ref="...")
            const picker = this.$refs.emojiPicker;
            const btn = this.$refs.emojiBtn;

            // Controllo
            // 1. Esiste il picker?
            // 2. Il click NON è avvenuto dentro il picker?
            // 3. Il click NON è avvenuto sul bottone che apre il picker?
            if (picker && !picker.contains(event.target) && btn && !btn.contains(event.target)) {
                this.showEmojiPicker = false;   // Chiudo la griglia delle Emojii
            }
        },

    },

    // MOUNTED: Avvio del ciclo infinito 
    mounted() {
        if (!this.username) {
            this.$router.push('/login')
            return
        }
        this.loadData()

        // POLLING: Ogni 3 secondi aggiorna tutto
        this.pollingInterval = setInterval(() => {
            this.refreshConversations() // Aggiorna la lista a sinistra
            this.refreshChat()          // Aggiorna la chat a destra (se aperta)
        }, 3000)
        
        // Aggiungo l'ascoltatore per i click su tutta la pagina
        document.addEventListener('click', this.handleClickOutside);
        
    },

    // UNMOUNTED: Pulizia quando chiudo la pagina 
    beforeUnmount() {
        // Pulisco l'intervallo
        if (this.pollingInterval) {
            clearInterval(this.pollingInterval)
        }

        // Rimuovo l'ascoltatore per il clikc
        document.removeEventListener('click', this.handleClickOutside);
    }
}
</script>


<template>
    <div class="container-fluid vh-100 p-0 overflow-hidden">
        <div class="row g-0 h-100">
            
            <div class="col-md-4 col-lg-3 d-flex flex-column border-end bg-light h-100">
                
                <div class="p-3 bg-secondary text-white d-flex align-items-center justify-content-between">
                    
                    <div 
                        class="d-flex align-items-center" 
                        style="cursor: pointer;" 
                        @click="goToProfile"
                        title="Edit Profile"
                    >
                        <img 
                            :src="myProfile?.profilePhoto || '/default_avatar.jpg'" 
                            class="rounded-circle me-2" 
                            width="40" height="40" 
                            style="object-fit: cover;"
                        >
                        <span class="fw-bold">{{ myProfile?.username }}</span>
                    </div>
                    <div>
                        <button @click="showGroupModal = true" class="btn btn-sm btn-light me-2" title="New Group">
                            <i class="feather icon-plus">+</i>
                        </button>

                        <button @click="logout" class="btn btn-sm btn-dark">Logout</button>
                    </div>
                </div>

                <div class="p-2 border-bottom">
                    <input 
                        type="text" 
                        class="form-control" 
                        placeholder="Search users..." 
                        v-model="searchQuery"
                        @input="doSearch"
                    >
                </div>

                <div class="flex-grow-1 overflow-auto bg-white">
                    
                    <div v-if="isSearching">
                        <div class="p-2 text-muted small bg-light">Search Results</div>
                        
                        <button 
                            v-for="user in searchResults" 
                            :key="user.id"
                            @click="startChatWith(user)"
                            class="list-group-item list-group-item-action d-flex align-items-center p-3"
                        >
                            <img 
                                :src="user.profilePhoto || '/default_avatar.jpg'" 
                                class="rounded-circle me-3" 
                                width="40" height="40"
                                style="object-fit: cover;"
                            >
                            <div>
                                <h6 class="mb-0">{{ user.username }}</h6>
                                <small class="text-success">Click to chat</small>
                            </div>
                        </button>

                        <div v-if="searchResults.length === 0" class="text-center p-3 text-muted">
                            No users found.
                        </div>
                    </div>

                    <div v-else class="list-group list-group-flush">
                        
                        <div v-if="loading" class="text-center p-4">Loading chats...</div>
                        
                        <div v-else-if="errorMsg" class="text-danger p-3">{{ errorMsg }}</div>
                        
                        <div v-else>
                            <button 
                                v-for="chat in conversations" 
                                :key="chat.id"
                                @click="selectChat(chat.id)"
                                class="list-group-item list-group-item-action d-flex align-items-center p-3"
                                :class="{ 'active': selectedChatId === chat.id }"
                            >
                                <img 
                                    :src="getChatPhoto(chat)" 
                                    class="rounded-circle me-3" 
                                    width="50" height="50"
                                    style="object-fit: cover;"
                                >
                                
                                <div class="flex-grow-1 overflow-hidden">
                                    <div class="d-flex justify-content-between align-items-baseline">
                                        <h6 class="mb-0 text-truncate">{{ getChatName(chat) }}</h6>
                                        
                                        <span v-if="chat.unreadCount > 0" class="badge rounded-pill bg-danger ms-2">
                                            {{ chat.unreadCount }}
                                        </span>

                                        <small class="text-muted" style="font-size: 0.75rem;">
                                            {{ getLastMessageTime(chat) }}
                                        </small>
                                    </div>

                                    <!-- ANTEPRIMA MESSAGGIO (SIDEBAR) -->
                                    <!-- Se la chat è quella selezionata, il testo dell'anteprima diventa bianco, cosi da poterlo visualizzare meglio-->
                                    <p 
                                        class="mb-0 small text-truncate"
                                        :class="selectedChatId === chat.id ? 'text-white-50' : 'text-muted'"
                                    >
                                        <span v-if="isImage(chat.snippet)">
                                            📷 Photo
                                        </span>
                                        
                                        <span v-else>
                                            {{ chat.snippet }}
                                        </span>
                                    </p>

                                </div>
                            </button>

                            <div v-if="conversations.length === 0" class="text-center text-muted mt-5">
                                No conversations yet.
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <div class="col-md-8 col-lg-9 d-flex flex-column h-100 bg-chat">
                
                <div v-if="selectedChatId" class="h-100 d-flex flex-column justify-content-between">

                    <div 
                        class="p-2 border-bottom bg-white d-flex align-items-center shadow-sm" 
                        style="z-index: 10; min-height: 60px; cursor: pointer;"
                        @click="openChatInfo"
                    >
                        <div class="d-flex align-items-center" v-if="activeChat">
                            <img 
                                :src="getChatPhoto(activeChat)" 
                                class="rounded-circle me-2 border" 
                                width="40" height="40" 
                                style="object-fit: cover;"
                            >
                            <div>
                                <h6 class="mb-0 fw-bold">{{ getChatName(activeChat) }}</h6>
                                <small class="text-muted" style="font-size: 0.75rem;">
                                    {{ activeChat.conversationType === 'group' ? 'Group Chat' : 'Private Chat' }}
                                </small>
                            </div>
                        </div>
                        <h5 v-else class="mb-0">Chat</h5>
                    </div>

                    <!-- Container della CHAT -->
                    <div 
                        ref="chatContainer"
                        class="flex-grow-1 overflow-auto p-3 d-flex flex-column" 
                        style="background: rgba(255,255,255,0.6);"
                    >
                        
                        <div v-if="chatLoading" class="text-center mt-5">
                            Loading messages...
                        </div>

                        <!-- LISTA DEI MESSAGGI -->
                        <div v-else v-for="msg in messages" :key="msg.id" class="mb-3 d-flex"
                            :class="msg.senderUsername === username ? 'justify-content-end' : 'justify-content-start'">
                            
                            <div class="card shadow-sm border-0 position-relative message-card" 
                                :class="msg.senderUsername === username ? 'bg-success text-white' : 'bg-white text-dark'"
                                style="max-width: 70%; min-width: 150px;">
                                
                                <div class="card-body p-2">
                                    <!-- Grafica del Forwaded sul messaggio in chat -->
                                    <div v-if="msg.isForwarded || msg.forwarded" class="fst-italic mb-1 d-flex align-items-center" 
                                        :class="msg.senderUsername === username ? 'text-white-50' : 'text-muted'" style="font-size: 0.75rem;">
                                        <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="me-1"><polyline points="15 14 20 9 15 4"></polyline><path d="M4 20v-7a4 4 0 0 1 4-4h12"></path></svg>
                                        Forwarded
                                    </div>

                                    <!-- Grafica del Reply sul messaggio in chat -->
                                    <div v-if="msg.replyTo" 
                                        class="mb-2 rounded overflow-hidden position-relative d-flex flex-column justify-content-center border-start border-4"
                                        :class="msg.senderUsername === username ? 'border-light' : 'border-primary'"
                                        style="background: rgba(0,0,0,0.1); padding: 5px 8px; cursor: pointer; min-width: 120px;">
                                        
                                        <span class="fw-bold mb-1" style="font-size: 0.75rem; opacity: 0.9;">
                                            {{ getRepliedMessageDetails(msg.replyTo).senderUsername }}
                                        </span>
                                        
                                        <span class="text-truncate d-flex align-items-center" style="font-size: 0.8rem; opacity: 0.8;">
                                            <i v-if="isImage(getRepliedMessageDetails(msg.replyTo).contentMess)" class="feather icon-image me-1"></i>
                                            
                                            {{ isImage(getRepliedMessageDetails(msg.replyTo).contentMess) 
                                                ? 'Photo' 
                                                : getRepliedMessageDetails(msg.replyTo).contentMess }}
                                        </span>
                                    </div>

                                    
                                    <small v-if="msg.senderUsername !== username" class="fw-bold d-block mb-1 text-primary">
                                        {{ msg.senderUsername }}
                                        <span v-if="currentChatAdmins.includes(msg.senderUserId)" class="badge bg-light text-secondary border ms-2">Admin</span>
                                    </small>

                                    <!-- Gestisco la Visualizzazione del messaggio (anche se è una Immagine + Testo) -->

                                    <!-- In "messagePhoto" ho salvato l'URL dell' Upload, se c'è stampa l'immagine -->
                                    <div v-if="msg.messagePhoto">
                                        <img :src="msg.messagePhoto" class="img-fluid rounded mb-1" style="max-height: 300px; cursor: pointer;">
                                    </div>
                                    
                                    <div v-else-if="isImage(msg.contentMess)">
                                        <img :src="msg.contentMess" class="img-fluid rounded mb-1" style="max-height: 300px; cursor: pointer;">
                                    </div>

                                    <!-- Controllo msg.contentMess. Se c'è del testo (la didascalia), stampo anche il paragrafo <p> -->
                                    <p v-if="msg.contentMess && !isImage(msg.contentMess)" class="mb-1 text-break" style="white-space: pre-line;">
                                        {{ msg.contentMess }}
                                    </p>
                                    

                                    <div v-if="msg.reactions && msg.reactions.length > 0" class="d-flex flex-wrap gap-1 mt-2 mb-1">
                                        <span 
                                            v-for="reaction in msg.reactions" 
                                            :key="reaction.reactionId" 
                                            class="badge rounded-pill border reaction-pill d-flex align-items-center"
                                            :class="reaction.senderUserId === myProfile.id ? 'bg-primary-subtle text-primary border-primary' : 'bg-light text-dark border-secondary-subtle'"
                                            @click="reactToMessage(msg, reaction.emoji)"
                                            :title="reaction.senderUsername"
                                        >
                                            {{ reaction.emoji }}
                                        </span>
                                    </div>

                                    
                                    <div class="text-end lh-1" style="font-size: 0.7rem; opacity: 0.8;">
                                        {{ formatDateTime(msg.timestamp) }}
                                        <!-- Checkmarks dei messaggi -->
                                        <span v-if="msg.senderUsername === username" class="ms-1">
                                            
                                            <span v-if="msg.statusInfo === 'read'" class="fw-bold" style="color: #4df0ff;" title="Read">✓✓</span>
                                            
                                            <span v-else-if="msg.statusInfo === 'delivered'" class="fw-bold text-secondary" title="Delivered">✓✓</span>
                                            
                                            <span v-else class="fw-bold text-secondary" title="Sent">✓</span>

                                        </span>
                                    </div>

                                </div>

                                <div class="message-actions">
                                    <div class="dropdown">
                                        <button class="btn btn-sm btn-link p-0" 
                                                :class="msg.senderUsername === username ? 'text-white' : 'text-secondary'"
                                                type="button" data-bs-toggle="dropdown" aria-expanded="false" style="opacity: 0.9;">
                                            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="feather feather-more-vertical"><circle cx="12" cy="12" r="1"></circle><circle cx="12" cy="5" r="1"></circle><circle cx="12" cy="19" r="1"></circle></svg>
                                        </button>
                                        
                                        <ul class="dropdown-menu dropdown-menu-end shadow-sm border-0" style="min-width: 200px;">
                                            
                                            <li>
                                                <div class="d-flex justify-content-evenly p-2 bg-light rounded mx-2 mb-2">
                                                    <button 
                                                        v-for="emoji in reactionOptions" 
                                                        :key="emoji"
                                                        class="btn btn-sm p-0 fs-5 lh-1 reaction-menu-btn"
                                                        @click="reactToMessage(msg, emoji)"
                                                    >
                                                        {{ emoji }}
                                                    </button>
                                                </div>
                                            </li>
                                            
                                            <!-- EDIT -->
                                            <li v-if="msg.senderUsername === username">
                                                <button class="dropdown-item d-flex align-items-center gap-2" @click="startEditing(msg)">
                                                    <i class="feather icon-edit-2 text-primary"></i> Edit
                                                </button>
                                            </li>

                                            <!-- DELETE -->
                                            <li v-if="msg.senderUsername === username">
                                                <button class="dropdown-item d-flex align-items-center gap-2 text-danger" @click="doDeleteMessage(msg.id)">
                                                    <i class="feather icon-trash-2"></i> Delete
                                                </button>
                                            </li>
                                            
                                            <li v-if="msg.senderUsername === username"><hr class="dropdown-divider"></li>
                                            
                                            <!-- FORWARD -->
                                            <li>
                                                <button class="dropdown-item d-flex align-items-center gap-2" @click="openForwardModal(msg)">
                                                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-secondary"><path d="M4 12v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8"></path><polyline points="16 6 12 2 8 6"></polyline><line x1="12" y1="2" x2="12" y2="15"></line></svg>
                                                    Forward
                                                </button>
                                            </li>

                                            <!-- REPLY -->
                                            <li>
                                                <button class="dropdown-item d-flex align-items-center gap-2" @click="startReplying(msg)">
                                                    <i class="feather icon-corner-up-left text-secondary"></i> Reply
                                                </button>
                                            </li>

                                        </ul>
                                    </div>
                                </div>

                            </div>
                        </div>

                        
                        <div v-if="!chatLoading && messages.length === 0" class="text-center text-muted mt-5">
                            <p>No messages yet. Say hello! 👋</p>
                        </div>

                    </div>

                    <!-- Barra Bottom della Chat, quella dove inserire il testo, emoji, immagine -->                    
                    <div class="p-3 bg-light border-top d-flex flex-column position-relative" style="z-index: 100;">
                        <div v-if="replyingToMessage || isEditing || selectedFile" class="mb-2 w-100">

                            <!-- Bottom Bar Reply -->
                            <div v-if="replyingToMessage" 
                                class="d-flex justify-content-between align-items-center p-2 rounded bg-white shadow-sm position-relative overflow-hidden" 
                                style="border-left: 5px solid #00a884; background-color: rgba(255,255,255,0.95);">
                                
                                <div class="d-flex flex-column ps-2 overflow-hidden w-100">
                                    <span class="fw-bold small mb-1" style="color: #00a884;">
                                        {{ replyingToMessage.senderUsername }}
                                    </span>
                                    
                                    <span class="text-muted small text-truncate" style="max-width: 90%; font-size: 0.85rem;">
                                        <i v-if="isImage(replyingToMessage.contentMess)" class="feather icon-image me-1"></i>
                                        {{ isImage(replyingToMessage.contentMess) ? 'Photo' : replyingToMessage.contentMess }}
                                    </span>
                                </div>

                                <div v-if="isImage(replyingToMessage.contentMess)" class="me-3 rounded overflow-hidden border" style="width: 40px; height: 40px;">
                                    <img :src="replyingToMessage.contentMess" class="w-100 h-100" style="object-fit: cover;">
                                </div>

                                <button class="btn btn-sm btn-link text-secondary p-0 ms-2" @click="cancelReply">
                                    <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="feather feather-x"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
                                </button>
                            </div>

                            <div v-if="isEditing" 
                                 class="d-flex justify-content-between align-items-center p-2 rounded bg-white shadow-sm" 
                                 style="border-left: 5px solid #0d6efd;">
                                <div class="d-flex align-items-center gap-2 overflow-hidden">
                                    <i class="feather icon-edit text-primary"></i>
                                    <div class="d-flex flex-column lh-1 overflow-hidden">
                                        <span class="fw-bold text-primary small">Editing Message</span>
                                        <span class="text-muted small text-truncate">{{ originalMessageText }}</span>
                                    </div>
                                </div>
                                <button class="btn btn-sm btn-close" @click="cancelEditing"></button>
                            </div>

                            <div v-if="selectedFile" class="d-flex align-items-center p-2 rounded bg-white shadow-sm border" style="border-left: 5px solid #6610f2 !important;">
                                <div class="me-3 rounded overflow-hidden border" style="width: 50px; height: 50px;">
                                    <!-- Ho inserito il DIV con l'Anteprima ( img :src="selectedFilePreview" )-->
                                    <img :src="selectedFilePreview" class="w-100 h-100" style="object-fit: cover;">
                                </div>
                                <div class="flex-grow-1">
                                    <small class="fw-bold d-block text-dark">Image selected</small>
                                    <small class="text-muted" style="font-size: 0.75rem;">Add a caption below...</small>
                                </div>
                                <button class="btn btn-sm btn-close" @click="removeSelectedFile"></button>
                            </div>

                        </div>

                        <div class="d-flex gap-2 align-items-center w-100">
                            <div v-if="showEmojiPicker" ref="emojiPicker" class="emoji-picker-popup shadow-sm" style="bottom: 70px;">
                                <div class="emoji-grid">
                                    <button 
                                        v-for="emoji in emojiList" 
                                        :key="emoji" 
                                        class="emoji-btn"
                                        type="button"
                                        @click="addEmoji(emoji)"
                                    >
                                        {{ emoji }}
                                    </button>
                                </div>
                            </div>
                            
                            <input 
                                type="file" 
                                ref="fileInput" 
                                style="display: none" 
                                accept="image/png, image/jpeg, image/gif"
                                @change="handleFileUpload"
                            >

                            <button class="btn btn-link text-secondary text-decoration-none p-0" ref="emojiBtn" @click="showEmojiPicker = !showEmojiPicker">
                                <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="feather feather-smile"><circle cx="12" cy="12" r="10"></circle><path d="M8 14s1.5 2 4 2 4-2 4-2"></path><line x1="9" y1="9" x2="9.01" y2="9"></line><line x1="15" y1="9" x2="15.01" y2="9"></line></svg>
                            </button>

                            <button class="btn btn-link text-secondary text-decoration-none p-0" @click="triggerFileUpload" title="Send Image">
                                <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="feather feather-camera"><path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"></path><circle cx="12" cy="13" r="4"></circle></svg>
                            </button>

                            <input 
                                ref="messageInput"   
                                type="text" 
                                class="form-control rounded-pill border-0 bg-white shadow-sm px-3" 
                                placeholder="Type a message..."
                                v-model="newMessageText"
                                @keyup.enter="sendMsg"
                                style="height: 45px;"
                            >
                            
                            <!-- Ho aggiornato la classe :class per diventare verde (btn-success) anche quando c'è un file selezionato, così l'utente capisce che sta inviando qualcosa di diverso dal solito testo. -->
                            <button class="btn rounded-circle shadow-sm d-flex align-items-center justify-content-center" 
                                    :class="(isEditing || selectedFile) ? 'btn-success' : 'btn-primary'"
                                    style="width: 45px; height: 45px;"
                                    @click="sendMsg">
                                <i v-if="isEditing" class="feather icon-check"></i>
                                <svg v-else xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="feather feather-send"><line x1="22" y1="2" x2="11" y2="13"></line><polygon points="22 2 15 22 11 13 2 9 22 2"></polygon></svg>
                            </button>
                        </div>
                    </div>
                </div>

                <div v-else class="h-100 d-flex flex-column align-items-center justify-content-center text-muted bg-light">
                    <h3 class="fw-light">WASAtext Web</h3>
                    <p>Select a chat to start messaging</p>
                </div>

            </div>
        </div>

        <!-- Finestra sovrimpressione quando clicco "+" per creare un gruppo  -> MODALE CREAZIONE GRUPPO -->
         <div v-if="showGroupModal" class="modal-overlay">
            <div class="modal-content p-4 shadow rounded bg-white" style="max-width: 500px; width: 90%;">
                
                <div class="d-flex justify-content-between align-items-center mb-3">
                    <h4>New Group</h4>
                    <button @click="showGroupModal = false" class="btn btn-close"></button>
                </div>

                <div class="mb-3">
                    <label class="form-label">Group Name</label>
                    <input type="text" class="form-control" v-model="newGroupName" placeholder="e.g. Best Friends">
                </div>

                <div class="mb-3">
                    <label class="form-label">Add Members</label>
                    
                    <input 
                        type="text" 
                        class="form-control mb-2" 
                        placeholder="Search users to add..." 
                        v-model="groupSearchQuery"
                        @input="searchUsersForGroup"
                    >

                    <div v-if="groupSearchResults.length > 0" class="list-group mb-2 border">
                        <button 
                            v-for="user in groupSearchResults" 
                            :key="user.id" 
                            @click="toggleUserSelection(user)"
                            class="list-group-item list-group-item-action"
                        >
                            {{ user.username }}
                        </button>
                    </div>

                    <div class="d-flex flex-wrap gap-2 mt-2">
                        <span v-for="user in selectedUsers" :key="user.id" class="badge bg-primary d-flex align-items-center p-2">
                            {{ user.username }}
                            <span @click="toggleUserSelection(user)" class="ms-2 cursor-pointer text-white" style="cursor: pointer;">&times;</span>
                        </span>
                        <span v-if="selectedUsers.length === 0" class="text-muted small">No members selected</span>
                    </div>
                </div>

                <div class="d-flex justify-content-end gap-2 mt-4">
                    <button @click="showGroupModal = false" class="btn btn-secondary">Cancel</button>
                    <button @click="createGroup" class="btn btn-success">Create Group</button>
                </div>

            </div>
        </div>

        <!-- MODALE INFO Gruppo / Profilo Utente -->
        <div v-if="showInfoModal" class="modal-overlay" @click.self="showInfoModal = false">
            <div class="modal-content bg-white shadow rounded overflow-hidden" style="max-width: 400px; width: 90%; max-height: 85vh; display: flex; flex-direction: column;">
                
                <div class="d-flex justify-content-end p-2 position-absolute w-100" style="z-index: 10;">
                    <button @click="showInfoModal = false" class="btn-close bg-white p-2 rounded-circle shadow-sm"></button>
                </div>

                <div v-if="!groupInfo && !userInfo" class="p-5 text-center">
                    <LoadingSpinner />
                </div>

                 <!-- Sezione Info Gruppo -->
                <div v-else-if="groupInfo">
                    <div class="text-center p-4 bg-light border-bottom">
                        <img :src="groupInfo.groupPhoto || '/default_avatar.jpg'" class="rounded-circle shadow mb-3" width="120" height="120" style="object-fit: cover;">
                        <h4 class="fw-bold mb-0">{{ groupInfo.groupName }}</h4>
                        <p class="text-muted small">Group • {{ groupInfo.members.length }} participants</p>
                        
                        <button 
                            v-if="myProfile && groupInfo.admins.includes(myProfile.id)" 
                            @click="$router.push('/groups/' + selectedChatId + '/edit')"
                            class="btn btn-sm btn-outline-secondary mt-2"
                        >
                            Edit Group
                        </button>
                    </div>

                    <div class="p-3 border-bottom">
                        <small class="fw-bold text-muted" style="font-size: 0.7rem;">DESCRIPTION</small>
                        <p class="mb-0 mt-1">{{ groupInfo.groupDescription || 'No description' }}</p>
                    </div>

                    <div class="flex-grow-1 overflow-auto p-0" style="max-height: 180px;">
                        <div class="p-3 bg-light text-muted small fw-bold">PARTICIPANTS</div>
                        <ul class="list-group list-group-flush">
                            <li v-for="member in getSortedMembers()" :key="member.userId" class="list-group-item d-flex justify-content-between align-items-center">
                                
                                <div class="d-flex align-items-center">
                                    <img src="/default_avatar.jpg" class="rounded-circle me-2" width="30" height="30">
                                    <span>
                                        {{ member.username }}
                                        <span v-if="myProfile && member.userId === myProfile.id" class="text-muted fst-italic ms-1">(You)</span>
                                    </span>
                                </div>

                                <span v-if="groupInfo.admins.includes(member.userId)" class="badge bg-light text-success border border-success">Admin</span>
                            </li>
                        </ul>
                    </div>

                    <!-- Possibilità di Lasciare il Gruppo-->
                    <div class="p-3 border-top mt-auto">
                        <button @click="leaveGroup" class="btn btn-danger w-100 d-flex align-items-center justify-content-center gap-2">
                            <i class="feather icon-log-out"></i> Leave Group
                        </button>
                    </div>

                </div>

                <!-- Sezione Info Utente Privato -->
                <div v-else-if="userInfo">
                    <div class="text-center p-5 pb-4">
                        <img 
                            :src="userInfo.profilePhoto || '/default_avatar.jpg'" 
                            class="rounded-circle shadow mb-3" 
                            width="150" height="150" 
                            style="object-fit: cover;"
                        >
                        <h3 class="fw-bold mb-1">{{ userInfo.username }}</h3>
                        <span class="badge bg-primary rounded-pill">User</span>
                    </div>

                    <div class="p-4 border-top">
                        <small class="fw-bold text-primary text-uppercase" style="font-size: 0.75rem;">Status</small>
                        <p class="fs-5 mt-1 fst-italic text-dark">
                            "{{ userInfo.status || 'Hey there! I am using WASAtext.' }}"
                        </p>
                    </div>

                </div>

            </div>
        </div>

        <!-- Modale Forward Message -->
        <div v-if="showForwardModal" class="modal-overlay" @click.self="showForwardModal = false">
            <div class="modal-content shadow rounded bg-white" style="max-width: 400px; width: 90%; max-height: 80vh; display: flex; flex-direction: column;">
                
                <div class="p-3 border-bottom d-flex justify-content-between align-items-center bg-light">
                    <h5 class="mb-0">Forward to...</h5>
                    <button @click="showForwardModal = false" class="btn-close"></button>
                </div>

                <div class="p-2 border-bottom bg-white">
                    <input 
                        type="text" 
                        class="form-control" 
                        placeholder="Search for people..." 
                        v-model="forwardSearchQuery"
                        @input="searchUsersForForward"
                    >
                </div>

                <div class="flex-grow-1 overflow-auto">
                    
                    <div v-if="forwardSearchQuery.length > 0">
                        <div class="p-2 text-muted small bg-light fw-bold">Global Search</div>
                        
                        <div v-if="forwardSearchResults.length === 0" class="p-3 text-center text-muted small">
                            No users found.
                        </div>

                        <button 
                            v-for="user in forwardSearchResults" 
                            :key="user.id"
                            @click="doForwardToUser(user)"
                            class="list-group-item list-group-item-action d-flex align-items-center p-3 border-bottom"
                        >
                            <img 
                                :src="user.profilePhoto || '/default_avatar.jpg'" 
                                class="rounded-circle me-3" 
                                width="40" height="40"
                                style="object-fit: cover;"
                            >
                            <div>
                                <h6 class="mb-0">{{ user.username }}</h6>
                                <small 
                                    class="text-primary"
                                    v-if="!conversations.some(chat => chat.conversationType === 'private' && chat.recipientUser === user.id)"
                                >
                                    Send to new chat
                                </small>
                            </div>
                        </button>
                    </div>

                    <div class="list-group list-group-flush">
                        <div class="p-2 text-muted small bg-light fw-bold">Recent Chats</div>

                        <button 
                            v-for="chat in conversations" 
                            :key="chat.id"
                            @click="doForwardToChat(chat)"
                            class="list-group-item list-group-item-action d-flex align-items-center p-3"
                        >
                            <img 
                                :src="getChatPhoto(chat)" 
                                class="rounded-circle me-3" 
                                width="40" height="40"
                                style="object-fit: cover;"
                            >
                            <div>
                                <h6 class="mb-0">{{ getChatName(chat) }}</h6>
                                <small class="text-muted" style="font-size: 0.75rem;">
                                    {{ chat.conversationType === 'group' ? 'Group' : 'Private' }}
                                </small>
                            </div>
                        </button>

                        <div v-if="conversations.length === 0" class="p-4 text-center text-muted">
                            No active chats found.
                        </div>
                    </div>
                </div>
                
                <div class="p-2 border-top text-center bg-light">
                    <button class="btn btn-sm btn-secondary" @click="showForwardModal = false">Cancel</button>
                </div>
            </div>
        </div>


    </div>
</template>

<style scoped>
/* Stile per lo sfondo della chat a destra */
.bg-chat {
    background-color: #efeae2; 
    /* pattern di sfondo */
    background-image: url("https://user-images.githubusercontent.com/15075759/28719144-86dc0f70-73b1-11e7-911d-60d70fcded21.png");
    background-repeat: repeat;
}

/* Personalizzazione scrollbar */
::-webkit-scrollbar {
    width: 6px;
}
::-webkit-scrollbar-thumb {
    background: #ccc; 
    border-radius: 3px;
}

.modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background: rgba(0, 0, 0, 0.5); /* Sfondo scuro semi-trasparente */
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000; /* Sopra a tutto */
}

/* Stile Emoji */
/* Container del popup */
.emoji-picker-popup {
    position: absolute;
    bottom: 60px; 
    left: 20px;
    width: 320px;
    height: 250px;
    background: white;
    box-shadow: 0 -2px 10px rgba(0,0,0,0.1);
    border-radius: 10px;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    z-index: 1000;
}

/* Griglia delle emoji */
.emoji-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(40px, 1fr));
    padding: 10px;
    overflow-y: auto; /* scroll */
    height: 100%;
    
    /* Personalizzazione Scrollbar (Chrome/Safari/Edge) */
    scrollbar-width: thin;
    scrollbar-color: #cccccc transparent;
}

/* Stile scrollbar per Webkit (Chrome) */
.emoji-grid::-webkit-scrollbar {
    width: 6px;
}
.emoji-grid::-webkit-scrollbar-track {
    background: transparent;
}
.emoji-grid::-webkit-scrollbar-thumb {
    background-color: #cccccc;
    border-radius: 20px;
}

/* Bottone Emoji singola */
.emoji-btn {
    font-size: 24px; /* Grandezza emoji */
    padding: 8px 0;
    cursor: pointer;
    background: transparent; /* Rimuove sfondo grigio default */
    border: none;            /* Rimuove bordo 3D default */
    border-radius: 6px;
    transition: background-color 0.2s;
    
    /* Forza il font emoji colorato su Windows/Mac */
    font-family: "Segoe UI Emoji", "Apple Color Emoji", "Noto Color Emoji", sans-serif;
    line-height: 1;
}

/* Effetto Hover (like Whatsapp*/
.emoji-btn:hover {
    background-color: #f0f2f5; 
}

/* Visualizzazione Corretta Emojii */

/* Applico il font corretto all'input di testo e ai messaggi inviati */
input.form-control, 
.card-body p {
    /* Noto Color Emoji --> Debian */
    font-family: "Segoe UI", "Roboto", "Helvetica Neue", Arial, sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Noto Color Emoji";
    line-height: 1.5; 
}
/* Applico il font anche al testo segnaposto (placeholder) */
::placeholder {
   font-family: "Segoe UI", "Roboto", "Helvetica Neue", Arial, sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Noto Color Emoji";
    opacity: 0.7;
}

/*  Fix Visualizzazione Emoji Sidebar */
/* Applico il font corretto ai Nomi (h6), alle Anteprime (p) nella lista */
.list-group-item h6, 
.list-group-item p {
    font-family: "Segoe UI", "Roboto", "Helvetica Neue", Arial, sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Noto Color Emoji";
}

/* EDITING MESSAGGIO */

/* Modifica al contenitore della card */
.message-card {
    display: flex !important; /* Attiva le colonne (Flexbox) */
    flex-direction: row;      /* Testo a sinistra, Icona a destra */
    overflow: visible;        /* Importante per far uscire il menu a tendina */
}

/* Modifica al corpo del testo (.card-body) */
.message-card .card-body {
    flex: 1;       /* Prende tutto lo spazio disponibile meno i 30px dell'icona */
    min-width: 0;  /* Fix tecnico per evitare che il testo rompa il layout flex */
    
    padding: 0.5rem !important; 
}

/* Modifica al contenitore dei puntini (.message-actions) */
.message-card .message-actions {
    position: relative; 
    
    width: 30px;        /* Larghezza fissa della colonna "tab" */
    
    /* Centratura dell'icona */
    display: flex;      
    justify-content: center;
    padding-top: 8px;   /* Per allinearla visivamente alla prima riga di testo */
    
    opacity: 0.6;
    transition: opacity 0.2s;
    z-index: 10;
    cursor: pointer;
}

/* Hover effect */
.message-card:hover .message-actions {
    opacity: 1;
}

/* Se il menu è aperto, resta visibile */
.message-card .message-actions .dropdown.show {
    opacity: 1;
}

/* Stile Banner Modifica */
.editing-banner {
    position: absolute;
    bottom: 100%;
    left: 0;
    width: 100%;
    background-color: #f0f2f5;
    border-top: 1px solid #ddd;
    padding: 8px 15px;
    z-index: 5;
    border-radius: 10px 10px 0 0;
}

/* Stile REACTIONS */
.reaction-pill {
    cursor: pointer;
    font-size: 0.85rem;
    font-weight: normal;
    padding: 4px 8px;
    transition: transform 0.1s;
    user-select: none;
    
    /* Fix Font Emoji*/
    font-family: "Segoe UI Emoji", "Apple Color Emoji", "Noto Color Emoji", sans-serif;

    line-height: 1.3; 
}

.reaction-pill:active {
    transform: scale(0.95);
}

/* Colore Mia reazione */
.bg-primary-subtle {
    background-color: #e7f1ff !important;
}

/* Bordo Colore Reazioni degli altri */
.border-secondary-subtle {
    border-color: #dee2e6 !important;
}

/* Stile per le emoji dentro il menu a tendina */
.reaction-menu-btn {
    border: none;
    background: transparent;
    transition: transform 0.2s;
    
    /* Fix Font Emoji*/
    font-family: "Segoe UI Emoji", "Apple Color Emoji", "Noto Color Emoji", sans-serif;
}

.reaction-menu-btn:hover {
    transform: scale(1.4); /* Effetto zoom al passaggio del mouse */
}

</style>