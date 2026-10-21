<script setup>
import { ref, watch, nextTick, onBeforeUnmount } from 'vue'
import { X } from 'lucide-vue-next'
const props = defineProps({ open: Boolean, title: String, id: { type: String, required: true }, wide: Boolean, restoreFocus: String })
const emit = defineEmits(['close'])
const dialog = ref(null)
let previousFocus, previousOverflow = '', locked = false
function release() {
  if (!locked) return
  document.body.style.overflow = previousOverflow
  locked = false
  if (previousFocus?.isConnected) previousFocus.focus()
}
watch(() => props.open, async open => {
  if (open) {
    previousFocus = (props.restoreFocus && document.querySelector(props.restoreFocus)) || document.activeElement
    previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    locked = true
    await nextTick()
    dialog.value?.showModal()
    dialog.value?.querySelector('[data-initial-focus]')?.focus()
  } else { dialog.value?.close(); release() }
}, { immediate: true })
function backdrop(event) {
  if (event.target !== dialog.value) return
  const rect = dialog.value.getBoundingClientRect()
  if (event.clientX < rect.left || event.clientX > rect.right || event.clientY < rect.top || event.clientY > rect.bottom) emit('close')
}
function trap(event) {
  if (event.key !== 'Tab') return
  const nodes = [...dialog.value.querySelectorAll('button, a[href], input, textarea, select, [tabindex="0"]')].filter(el => !el.disabled && el.getClientRects().length)
  const first = nodes[0], last = nodes.at(-1)
  if (!first) { event.preventDefault(); dialog.value.focus(); return }
  if (event.shiftKey && (document.activeElement === first || document.activeElement === dialog.value)) { event.preventDefault(); last.focus() }
  else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus() }
}
onBeforeUnmount(() => { dialog.value?.close(); release() })
</script>
<template>
  <Teleport to="body">
    <dialog ref="dialog" class="arena-modal" :class="{ wide }" :aria-labelledby="`${id}-title`" @cancel.prevent="emit('close')" @click="backdrop" @keydown="trap">
      <header class="modal-header"><div><p class="eyebrow">BOTTAGON / DEMO ARENA</p><h2 :id="`${id}-title`">{{ title }}</h2></div><button data-initial-focus class="icon-button" aria-label="Close dialog" @click="emit('close')"><X :size="20" /></button></header>
      <div class="modal-content"><slot /></div>
    </dialog>
  </Teleport>
</template>
