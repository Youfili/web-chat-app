import {createRouter, createWebHashHistory} from 'vue-router'
import HomeView from '../views/HomeView.vue'
import LoginView from '../views/LoginView.vue'
import ProfileView from '../views/ProfileView.vue'
import EditGroupView from '../views/EditGroupView.vue'

const router = createRouter({
	history: createWebHashHistory(import.meta.env.BASE_URL),
	routes: [
		// Le Rotte
		{
			path: '/',
			name: 'home',
			component: HomeView
		},
		{
			path: '/login',
			name: 'login',
			component: LoginView
		},
		{ 
			path: '/profile', 
			name: 'profile', 
			component: ProfileView 
		},
		{ 
			path: '/groups/:id/edit', 
			name: 'edit-group', component: 
			EditGroupView 
		}
	]
})

// Navigation Guard (Protezione Rotte)
// Prima di ogni cambio pagina, controllo se sono loggato!
router.beforeEach((to, from, next) => {
    // Cerco il token nel "cassetto" del browser
    const publicPages = ['/login'];
    const authRequired = !publicPages.includes(to.path);
    const loggedIn = localStorage.getItem('token');

    if (authRequired && !loggedIn) {
        // Se serve auth e non ce l'ho -> Login
        next('/login');
    } else {
        // Altrimenti entro senza problemi nella nuova view
        next();
    }
});

export default router
