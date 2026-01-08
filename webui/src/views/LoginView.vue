<script>
// Importo il servizio API, quello che ho realizzato in services/api.js
import api from '@/services/api'

export default {
	data() {
		return {
			username: '',
			loading: false,
			errormsg: null,
			isRegistering: false // Stato per sapere se sto facendo Login o Register
		}
	},
	methods: {
		async doLoginOrRegister() {
			this.loading = true;
			this.errormsg = null;

			try {
                // Provo a fare il LOGIN
                let response = await api.login(this.username);
                
                // Se arrivo qui, l'utente esisteva --> Faccio salvataggio sessione
                this.handleSuccess(response);

			} catch (e) {
				// Se ricevo errore 404 (User Not Found), allora provo a REGISTRARE
                if (e.response && e.response.status === 404) {
                    try {
                        console.log("Utente non trovato, procedo con la Registrazione...");
                        // Chiamo la registrazione
                        let regResponse = await api.register(this.username);
                        // Se va a buon fine, procedo al salvataggio sessione
                        this.handleSuccess(regResponse);
                        
                    } catch (regError) {
                        // Se fallisce anche la registrazione (es. nome non valido), mostro l'errore
                        this.handleError(regError);
                    }
                } else {
                    // Se era un altro tipo di errore (es. 500 server error), lo mostro subito
                    this.handleError(e);
                }
			}
		},

		// Funzione helper per salvare i dati e reindirizzare
        handleSuccess(response) {
            localStorage.setItem('token', response.id);
            localStorage.setItem('username', response.username);
            if (response.profilePhoto) {
                localStorage.setItem('profilePhoto', response.profilePhoto);
            }
            this.loading = false;
            this.$router.push('/');
        },

        // Funzione helper per gestire gli errori
        handleError(e) {
            this.loading = false;
            if (e.response && e.response.data) {
                this.errormsg = e.response.data.toString();
            } else {
                this.errormsg = e.toString();
            }
        }

	}
}
</script>

<template>
	<div class="d-flex justify-content-center align-items-center vh-100 bg-light">
		<div class="card p-4 shadow" style="max-width: 400px; width: 100%;">
			<h3 class="text-center mb-3">WASAtext</h3>

			<h5 class="text-center mb-4">Welcome</h5>
		
			<form @submit.prevent="doLoginOrRegister">
                <div class="mb-3">
                    <label for="username" class="form-label">Username</label>
                    <div class="input-group">
                        <span class="input-group-text bg-white text-muted">@</span>
                        <input 
                            type="text" 
                            class="form-control" 
                            id="username" 
                            v-model="username" 
                            required 
                            minlength="3" 
                            maxlength="30"
                            pattern="[a-zA-Z0-9._~\-]+"
                            placeholder="Choose your username"
                            @input="username = username.replace(/\s/g, '')"
                        >
                    </div>
                    <div class="form-text">Enter your username, to login or create an account automatically.</div>
                </div>

                <div class="d-grid gap-2 mt-4">
                    <button type="submit" class="btn btn-primary btn-lg" :disabled="loading">
                        <LoadingSpinner v-if="loading" />
                        <span v-else>Enter Chat</span>
                    </button>
                </div>
            </form>

            <div v-if="errormsg" class="alert alert-danger mt-3 d-flex align-items-center" role="alert">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-exclamation-triangle-fill flex-shrink-0 me-2" viewBox="0 0 16 16">
                    <path d="M8.982 1.566a1.13 1.13 0 0 0-1.96 0L.165 13.233c-.457.778.091 1.767.98 1.767h13.713c.889 0 1.438-.99.98-1.767L8.982 1.566zM8 5c.535 0 .954.462.9.995l-.35 3.507a.552.552 0 0 1-1.1 0L7.1 5.995A.905.905 0 0 1 8 5zm.002 6a1 1 0 1 1 0 2 1 1 0 0 1 0-2z"/>
                </svg>
                <div>{{ errormsg }}</div>
            </div>
        </div>
    </div>
</template>

<style scoped>
.vh-100 {
    height: 100vh;
}
/* Un tocco di stile in più per l'input */
.input-group-text {
    border-right: none;
}
.form-control {
    border-left: none;
}
.form-control:focus {
    box-shadow: none;
    border-color: #ced4da;
}
.input-group:focus-within {
    box-shadow: 0 0 0 0.25rem rgba(13, 110, 253, 0.25);
    border-radius: 0.375rem;
}
.input-group:focus-within .form-control, 
.input-group:focus-within .input-group-text {
    border-color: #86b7fe;
}
</style>