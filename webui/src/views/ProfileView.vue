<script>
import api from '@/services/api'
import LoadingSpinner from '@/components/LoadingSpinner.vue'

export default {
    components: {
        LoadingSpinner
    },
    data() {
        return {
            username: localStorage.getItem('username'),
            
            // Stato locale del profilo
            userProfile: {
                username: '',
                status: '',
                profilePhoto: ''
            },

            loading: false,
            msg: null,
            msgType: 'success', // 'success' o 'danger'
            
            // La uso per forzare il refresh dell'immagine
            imageKey: 0
        }
    },
    methods: {
        // Carico i dati attuali
        async loadProfile() {
            this.loading = true
            try {
                const response = await api.getMyProfile(this.username)

                // Assegnazione sicura: se il backend non manda campi, uso stringhe vuote
                this.userProfile = {
                    username: response.username || '',
                    status: response.status || 'Hey there! I am using WASAtext.',   // Se response.status è vuoto, uso la frase default
                    profilePhoto: response.profilePhoto || ''
                }

            } catch(e) {
                this.msg = "Error loading profile: " + e.toString()
                this.msgType = 'danger' 
            } finally {
                this.loading = false
            }
        },

        // Aggiorno username
        async updateUsername() {

            //// ## Faccio 2 controlli per evitare errori di user experience
            // Controllo validità input
            if (!this.userProfile.username || this.userProfile.username.trim().length < 3) {    // trim() metodo delle stringhe in JavaScript che rimuove gli spazi iniziali e finali dalla stringa
                this.msg = "Username must be at least 3 characters long."
                this.msgType = 'danger'
                return
            }

            // Controllo se è uguale a quello attuale
            if (this.userProfile.username === this.username) {
                this.msg = "You are already using this username."
                this.msgType = 'warning' 
                return
            }


            try {
                const res = await api.setMyUsername(this.username, this.userProfile.username)
                // Aggiorno il localStorage e ricarico
                localStorage.setItem('username', res.username)
                this.msg = "Username updated! (You might need to login again)"
                this.msgType = 'success'

                // Timeout per far leggere il messaggio e poi logout
                setTimeout(() => {
                    this.$router.push('/login')
                }, 1500)

            } catch (e) {
                // Gestione Errori in modo specifico --> Cosi utente capisce il motivo per il quale non riesce a cambiare username
                if (e.response && e.response.status === 400) {
                    // Se il server risponde 400, significa che il nome è preso o non-valido
                    this.msg = "This username is already taken. Please choose another one."
                } else if (e.response && e.response.data) {
                    this.msg = "Error: " + e.response.data
                } else {
                    this.msg = "Error updating username: " + e.toString()
                }
                this.msgType = 'danger'
            }
        },

        // Aggiorna Status
        async updateStatus() {
            try {
                // Invio lo stato al backend
                await api.setMyStatus(this.username, this.userProfile.status)
                
                this.msg = "Status updated successfully!"
                this.msgType = 'success'
            } catch (e) {
                this.msg = "Error updating status: " + e.toString()
                this.msgType = 'danger'
            }
        },

        // Gestione Upload Foto
        triggerUpload() {
            this.$refs.fileInput.click()
        },

        async handleFileUpload(event) {
            // Controlli di sicurezza preliminari (cosi da non far crashare)
            if (!event || !event.target || !event.target.files) {
                console.error("Evento upload non valido o nessun file lista trovata.")
                return
            }

            const file = event.target.files[0]
            
            // Controllo se il file Non viene scelto (es: Se l'utente ha aperto la finestra ma ha cliccato "Annulla")
            if (!file) {
                console.log("Nessun file selezionato (Annullato).")
                return
            }

            // Procedo con l'Upload
            try {
                this.loading = true
                console.log("Caricamento file:", file.name)

                // Upload fisico
                const response = await api.uploadFile(file, 'avatar')
                const newUrl = response.url

                // Aggiornamento DB
                await api.setMyPhoto(this.username, newUrl)
                
                // Aggiornamento Vue
                this.userProfile.profilePhoto = newUrl
                this.imageKey += 1                          // Refresh immagine
                
                this.msg = "Profile photo updated!"
                this.msgType = 'success'
                
            } catch (e) {
                console.error(e)
                this.msg = "Upload failed: " + e.toString()
                this.msgType = 'danger'
            } finally {
                this.loading = false
                // Reset Sicuro: Lo faccio solo alla fine di tutto
                // Questo permette di ricaricare lo stesso file se si vuole riprovare
                if (event.target) {
                    event.target.value = null
                }
            }
        },

        goBack() {
            this.$router.push('/')
        }
    },
    mounted() {
        this.loadProfile()
    }
}
</script>

<template>
    <div class="container mt-5">
        <div class="row justify-content-center">
            <div class="col-md-8 col-lg-6">
                <div class="card shadow-sm border-0">
                    
                    <div class="card-header bg-white border-bottom-0 pt-4 pb-0 d-flex justify-content-between align-items-center">
                        <h4 class="mb-0 fw-bold text-primary">Edit Profile</h4>
                        <button @click="goBack" class="btn btn-outline-secondary btn-sm">
                            <i class="feather icon-arrow-left"></i> Back
                        </button>
                    </div>
                    
                    <div class="card-body p-4">
                        
                        <div v-if="msg" :class="`alert alert-${msgType} mb-4`" role="alert">
                            {{ msg }}
                        </div>

                        <div v-if="loading" class="text-center my-5">
                            <LoadingSpinner />
                        </div>

                        <div v-else>
                            <div class="d-flex flex-column align-items-center mb-5">
                                <div class="position-relative">
                                    <img 
                                        :key="imageKey"
                                        :src="userProfile.profilePhoto || '/default_avatar.jpg'" 
                                        class="rounded-circle border shadow-sm" 
                                        width="150" height="150" 
                                        style="object-fit: cover;"
                                        alt="Profile Photo"
                                    >
                                    <button 
                                        @click="triggerUpload"
                                        class="btn btn-primary position-absolute bottom-0 end-0 rounded-circle shadow p-0 d-flex align-items-center justify-content-center"
                                        style="width: 40px; height: 40px; border: 3px solid white;"
                                        title="Change Photo"
                                    >
                                        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="feather feather-camera"><path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"></path><circle cx="12" cy="13" r="4"></circle></svg>
                                    </button>
                                </div>
                                <input type="file" ref="fileInput" class="d-none" @change="handleFileUpload" accept="image/png, image/jpeg">
                            </div>

                            <form @submit.prevent>
                                
                                <div class="mb-4">
                                    <label class="form-label fw-bold text-muted small text-uppercase">Username</label>
                                    <div class="input-group">
                                        <span class="input-group-text bg-light border-end-0">@</span>
                                        <input type="text" class="form-control border-start-0" v-model="userProfile.username">
                                        <button class="btn btn-dark" @click="updateUsername">Save</button>
                                    </div>
                                    <div class="form-text small">Changing username requires re-login.</div>
                                </div>

                                <div class="mb-4">
                                    <label class="form-label fw-bold text-muted small text-uppercase">Status</label>
                                    
                                    <p class="mb-2 fst-italic text-secondary" v-if="userProfile.status">
                                        Currently: "{{ userProfile.status }}"
                                    </p>
                                    <p class="mb-2 text-muted" v-else>
                                        Currently: "Hey there! I am using WASAtext."
                                    </p>

                                    <div class="input-group">
                                        <input type="text" class="form-control" v-model="userProfile.status" placeholder="e.g. Available, At work...">
                                        <button class="btn btn-outline-primary" @click="updateStatus">Update</button>
                                    </div>
                                </div>

                            </form>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>