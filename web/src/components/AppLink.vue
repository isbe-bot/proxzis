<script setup>
import { inject } from 'vue'
import { RouterLink, routerKey } from 'vue-router'

defineProps({ to: { type: [String, Object], required: true } })
const hasRouter = Boolean(inject(routerKey, null))
</script>

<template>
  <RouterLink v-if="hasRouter" :to="to"><slot /></RouterLink>
  <a v-else :href="typeof to === 'string' ? to : to.path"><slot /></a>
</template>
