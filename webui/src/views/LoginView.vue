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
				let response;
				if (this.isRegistering) {
                    // Chiamata POST /session/register
					response = await api.register(this.username);
				} else {
                    // Chiamata POST /session/login
					response = await api.login(this.username);
				}

                // Punto Fondamentale, utilizzo questo token (che inserisco qui) nell' Interceptor in axios.js
				// Salvo l'ID (token) e lo username nel browser
				localStorage.setItem('token', response.id); 
				localStorage.setItem('username', response.username);
                if (response.profilePhoto) {
                    localStorage.setItem('profilePhoto', response.profilePhoto);
                }
				
				// Reindirizzo alla Home
				this.$router.push('/');

			} catch (e) {
				// Gestione errori
				if (e.response && e.response.data) {
                    // Uso il messaggio inviato dal Backend (es. "Invalid username format" o "Username taken")
                    this.errormsg = e.response.data.toString();
				} else {
					this.errormsg = e.toString();
				}
			}
			this.loading = false;
		},
		toggleMode() {
			this.isRegistering = !this.isRegistering;
			this.errormsg = null;
		}
	}
}
</script>

<template>
	<div class="d-flex justify-content-center align-items-center vh-100 bg-light">
		<div class="card p-4 shadow" style="max-width: 400px; width: 100%;">
			<h3 class="text-center mb-3">WASAtext</h3>
			
			<h5 class="text-center mb-4">
				{{ isRegistering ? 'Create Account' : 'Login' }}
			</h5>

			<form @submit.prevent="doLoginOrRegister">
				<div class="mb-3">
					<label for="username" class="form-label">Username</label>
					<input 
						type="text" 
						class="form-control" 
						id="username" 
						v-model="username" 
						required 
						minlength="3" 
						maxlength="30"
						pattern="[a-zA-Z0-9._~-]+"
						placeholder="Insert yuor username..."
						@input="username = username.replace(/\s/g, '')"
					>
					<div class="form-text">Min 3 chars, letters/numbers only.</div>
				</div>

				<div class="d-grid gap-2">
					<button type="submit" class="btn btn-primary" :disabled="loading">
						<LoadingSpinner v-if="loading" />
						<span v-else>{{ isRegistering ? 'Register' : 'Login' }}</span>
					</button>
				</div>
			</form>

			<div class="text-center mt-3">
				<a href="#" @click.prevent="toggleMode">
					{{ isRegistering ? 'Already have an account? Login' : 'New here? Register' }}
				</a>
			</div>

			<ErrorMsg v-if="errormsg" :msg="errormsg" class="mt-3" />
		</div>
	</div>
</template>

<style scoped>
.vh-100 {
	height: 100vh;
}
</style>