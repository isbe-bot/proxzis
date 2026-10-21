<script setup>
import { useRouter } from 'vue-router'
import RegisterPage from '../components/RegisterPage.vue'
import { refreshContenders, useCatalog } from '../catalogStore.js'

const router = useRouter()
const { allBots } = useCatalog()
const viewRegisteredBot = bot => router.push(`/agents/${encodeURIComponent(bot.id)}`)
</script>

<template>
  <div class="page container">
    <RegisterPage :all-bots="allBots" @registered="refreshContenders" @view-bot="viewRegisteredBot" />
  </div>
</template>
