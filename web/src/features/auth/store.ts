import { ref } from 'vue'
import { defineStore } from 'pinia'
import { currentUser, logout as apiLogout, type User } from '@/api/auth'
import { APIError } from '@/api/client'
export const useAuthStore=defineStore('auth',()=>{const user=ref<User|null>(null);const initialized=ref(false);async function initialize(){if(initialized.value)return;try{user.value=await currentUser()}catch(e){if(!(e instanceof APIError&&e.status===401))console.warn(e)}finally{initialized.value=true}}async function logout(){await apiLogout();user.value=null}return{user,initialized,initialize,logout,isOwner:()=>user.value?.role==='owner'}})
