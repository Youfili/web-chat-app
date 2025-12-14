import axios from "axios";

const instance = axios.create({
	baseURL: __API_URL__,
	timeout: 1000 * 5
});

/*
Questo codice intercetta ogni richiesta in uscita, controlla se c'è un utente salvato (token/ID)
e inserisce l'header di Autorizzazione
*/

// REQUEST INTERCEPTOR
// Prima di inviare la richiesta, aggiunge l'header Authorization
instance.interceptors.request.use(
	(config) => {
		// Recupero il token (ID utente) dal localStorage (l'ho settato nella view del Login)
		// Nota: "token" è la chiave che userò quando farò il login
		const token = localStorage.getItem("token");

		if (token) {
			config.headers["Authorization"] = `Bearer ${token}`;
		}
		return config;
	},
	(error) => {
		return Promise.reject(error);
	}
);

export default instance;
