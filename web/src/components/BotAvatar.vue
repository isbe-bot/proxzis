<script setup>
defineProps({ bot: { type: Object, required: true }, large: Boolean })
</script>
<template>
  <span class="bot-avatar" :class="[bot.color, bot.face, { large }]" aria-hidden="true">
    <span class="bot-antenna"></span><span class="bot-face"><i></i><i></i></span><span class="bot-mouth"></span>
  </span>
</template>
