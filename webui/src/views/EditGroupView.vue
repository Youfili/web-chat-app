<script>
import api from '@/services/api'
import LoadingSpinner from '@/components/LoadingSpinner.vue'

export default {
    components: { LoadingSpinner },
    data() {
        return {
            username: localStorage.getItem('username'),
            groupId: this.$route.params.id, // ID del gruppo dall'URL
            
            groupInfo: null,
            newGroupName: "",
            newGroupDesc: "",
            
            searchQuery: "",
            searchResults: [],
            
            loading: true,
            msg: null,
            msgType: 'success'
        }
    },
    methods: {
        // Carico i dati iniziali del gruppo
        async loadGroup() {
            this.loading = true
            try {
                // Recupero info del gruppo
                this.groupInfo = await api.getGroupChat(this.username, this.groupId)
                
                // Pre-popolo i campi input
                this.newGroupName = this.groupInfo.groupName
                this.newGroupDesc = this.groupInfo.groupDescription || ""
                
            } catch (e) {
                this.msg = "Error loading group: " + e.toString()
                this.msgType = 'danger'
            } finally {
                this.loading = false
            }
        },

        // --- MODIFICHE INFO ---

        // Modifico il Nome del Gruppo
        async updateName() {
            if (!this.newGroupName.trim()) return
            try {
                await api.updateGroupInfo(this.username, this.groupId, 'name', this.newGroupName)
                this.msg = "Group name updated!"
                this.msgType = 'success'
                this.loadGroup() // Ricarico per sicurezza
            } catch (e) {
                this.msg = "Error: " + e.toString(); this.msgType = 'danger'
            }
        },

        // Modifico la Descrizione del Gruppo
        async updateDescription() {
            try {
                await api.updateGroupInfo(this.username, this.groupId, 'description', this.newGroupDesc)
                this.msg = "Description updated!"
                this.msgType = 'success'
                this.loadGroup()
            } catch (e) {
                this.msg = "Error: " + e.toString(); this.msgType = 'danger'
            }
        },

        // Modifico la Foto del Gruppo
        triggerPhotoUpload() {
            this.$refs.groupPhotoInput.click()
        },

        async handlePhotoUpload(event) {
            const file = event.target.files[0]
            if (!file) return
            
            // Reset input
            event.target.value = null

            try {
                this.loading = true
                // Carico file fisico
                const response = await api.uploadFile(file, 'media') // o 'avatar' se preferisci
                const newUrl = response.url

                // Aggiorno info gruppo
                await api.updateGroupInfo(this.username, this.groupId, 'photo', newUrl)
                
                this.msg = "Group photo updated!"
                this.msgType = 'success'
                await this.loadGroup() // Ricarico visuale

            } catch (e) {
                this.msg = "Upload failed: " + e.toString()
                this.msgType = 'danger'
            } finally {
                this.loading = false
            }
        },


        // --- GESTIONE MEMBRI ---

        // Cerco utenti da aggiungere
        async searchUsers() {
            if (this.searchQuery.length < 1) { this.searchResults = []; return }
            try {
                const res = await api.searchUsers(this.username, this.searchQuery)
                // Filtro: mostro solo chi NON è già nel gruppo
                // NOTA Modifica: groupInfo.members è ora un array di oggetti {userId, username} visto che l'ho modificato nello struct.go
                const currentMemberIds = this.groupInfo.members.map(m => m.userId)
                this.searchResults = res.users.filter(u => !currentMemberIds.includes(u.id))
            } catch(e) { console.error(e) }
        },

        // Aggiungo Nuovo Membro nel gruppo
        async addMember(user) {
            try {
                await api.addToGroup(this.username, this.groupId, user.id)
                this.searchQuery = ""
                this.searchResults = []
                this.msg = `Added ${user.username}!`
                this.msgType = 'success'
                await this.loadGroup() // Ricarico la lista membri
            } catch (e) {
                this.msg = "Error adding member: " + e.toString(); this.msgType = 'danger'
            }
        },

        // Rimuovo un Membro dal Gruppo
        async removeMember(userId) {
            if(!confirm("Remove this user form the group?")) return
            try {
                await api.removeFromGroup(this.username, this.groupId, userId)
                this.msg = "User removed."
                this.msgType = 'success'
                await this.loadGroup()
            } catch (e) {
                // Se il server risponde 400 (Bad Request) --> blocco admin
                if (e.response && e.response.status === 400) {
                    this.msg = "Cannot remove an admin. Please demote them first." 
                    this.msgType = 'warning' // Uso warning (Giallo) perché è più una "procedura" nel caso si voglia verasmente rimuovere
                } else {
                    this.msg = "Error removing member: " + e.toString();
                    this.msgType = 'danger'
                }
            }
        },

        // Promuovo/Declasso Admin
        async toggleAdmin(userId, makeAdmin) {
            try {
                if (makeAdmin) {
                    // Chiamata API: POST .../admins (Body: userIdToPromote)
                    await api.makeAdmin(this.username, this.groupId, userId)
                    this.msg = "User promoted to Admin!"
                } else {
                    // Chiamata API: DELETE .../admins/{userId}
                    await api.removeAdminStatus(this.username, this.groupId, userId)
                    this.msg = "User demoted to member."
                }
                this.msgType = 'success'
                await this.loadGroup() // Ricarico la lista per aggiornare i "badge"
            } catch (e) {
                this.msg = "Error changing admin status: " + e.toString()
                this.msgType = 'danger'
            }
        },

        goBack() {
            this.$router.push('/')
        }
    },
    mounted() {
        this.loadGroup()
    }
}
</script>

<template>
    <div class="container mt-4 mb-5">
        <div class="row justify-content-center">
            <div class="col-md-8 col-lg-6">
                <div class="card shadow-sm border-0">
                    
                    <div class="card-header bg-white border-bottom pt-3 pb-3 d-flex justify-content-between align-items-center">
                        <h5 class="mb-0 fw-bold text-primary">Edit Group</h5>
                        <button @click="goBack" class="btn btn-outline-secondary btn-sm">Done</button>
                    </div>
                    
                    <div class="card-body" v-if="groupInfo">
                        
                        <div v-if="msg" :class="`alert alert-${msgType} small py-2`">{{ msg }}</div>

                        <div class="text-center mb-4 mt-3">
                            <div class="position-relative d-inline-block">
                                <img 
                                    :src="groupInfo.groupPhoto || '/default_avatar.jpg'" 
                                    class="rounded-circle border shadow-sm" 
                                    width="100" height="100" 
                                    style="object-fit: cover;"
                                >
                                <button 
                                    @click="triggerPhotoUpload"
                                    class="btn btn-dark position-absolute bottom-0 end-0 rounded-circle p-0 d-flex align-items-center justify-content-center"
                                    style="width: 32px; height: 32px; border: 2px solid white;"
                                    title="Change Group Photo"
                                >
                                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="feather feather-camera"><path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"></path><circle cx="12" cy="13" r="4"></circle></svg>
                                </button>
                            </div>
                            <input type="file" ref="groupPhotoInput" class="d-none" @change="handlePhotoUpload" accept="image/png, image/jpeg">
                        </div>

                        <div class="mb-4">
                            <label class="form-label fw-bold small text-muted text-uppercase">Group Name</label>
                            <div class="input-group mb-3">
                                <input type="text" class="form-control" v-model="newGroupName">
                                <button class="btn btn-outline-primary" @click="updateName">Save</button>
                            </div>

                            <label class="form-label fw-bold small text-muted text-uppercase">Description</label>
                            <div class="input-group">
                                <textarea class="form-control" v-model="newGroupDesc" rows="2"></textarea>
                                <button class="btn btn-outline-primary" @click="updateDescription">Save</button>
                            </div>
                        </div>

                        <hr>

                        <div class="mb-4">
                            <label class="form-label fw-bold small text-muted text-uppercase">Add New Members</label>
                            <input type="text" class="form-control mb-2" placeholder="Search users..." v-model="searchQuery" @input="searchUsers">
                            
                            <div class="list-group" v-if="searchResults.length > 0">
                                <button v-for="u in searchResults" :key="u.id" @click="addMember(u)" class="list-group-item list-group-item-action d-flex justify-content-between align-items-center">
                                    <div class="d-flex align-items-center">
                                        <img :src="u.profilePhoto || '/default_avatar.jpg'" class="rounded-circle me-2" width="30" height="30" style="object-fit: cover;">
                                        <span>{{ u.username }}</span>
                                    </div>
                                    <span class="badge bg-success rounded-pill">+ Add</span>
                                </button>
                            </div>
                        </div>

                        <div>
                            <label class="form-label fw-bold small text-muted text-uppercase">
                                Current Members ({{ groupInfo.members.length }})
                            </label>
                            <ul class="list-group list-group-flush border rounded">
                                <li v-for="m in groupInfo.members" :key="m.userId" class="list-group-item d-flex justify-content-between align-items-center">
                                    
                                    <div class="d-flex align-items-center">
                                        <img src="/default_avatar.jpg" class="rounded-circle me-2 border" width="35" height="35">
                                        <span>
                                            {{ m.username }}
                                            <span v-if="groupInfo.admins.includes(m.userId)" class="badge bg-light text-success ms-1 border">Admin</span>
                                        </span>
                                    </div>
                                    
                                    <div class="d-flex align-items-center gap-2" v-if="m.userId !== groupInfo.members.find(u => u.username === username)?.userId">
                                        
                                        <button 
                                            v-if="groupInfo.admins.includes(m.userId)" 
                                            @click="toggleAdmin(m.userId, false)" 
                                            class="btn btn-sm btn-outline-warning py-0"
                                            title="Demote to Member"
                                        >
                                            Demote
                                        </button>

                                        <button 
                                            v-else 
                                            @click="toggleAdmin(m.userId, true)" 
                                            class="btn btn-sm btn-outline-success py-0"
                                            title="Promote to Admin"
                                        >
                                            Promote
                                        </button>

                                        <button @click="removeMember(m.userId)" class="btn btn-sm btn-outline-danger py-0">Remove</button>
                                    </div>

                                </li>
                            </ul>
                        </div>

                    </div>
                    
                    <div v-else class="p-5 text-center"><LoadingSpinner /></div>
                </div>
            </div>
        </div>
    </div>
</template>