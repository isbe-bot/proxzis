<script setup>
import { ref, computed, nextTick, onMounted } from 'vue'
import { AlertTriangle, ArrowRight, Check, Copy } from 'lucide-vue-next'
import { validateRegistration } from '../utils/arena.js'
import { createBot } from '../api/catalog.js'

const props = defineProps({
  allBots: {
    type: Array,
    required: true
  }
})

const emit = defineEmits(['registered', 'view-bot'])

const name = ref('')
const email = ref('')
const persona = ref('')
// Honeypot: hidden from people, filled in by naive form bots.
const botWebsiteTrap = ref('')

const errors = ref({})
const loading = ref(false)
const formError = ref('')
const credentials = ref(null)
const copyMessage = ref('')
const copyError = ref(false)

// Submissions faster than a person can type are rejected.
const mountedAt = ref(Date.now())
onMounted(() => {
  mountedAt.value = Date.now()
})

const personaLength = computed(() => Array.from((persona.value || '').trim()).length)

function handleInput(field) {
  if (errors.value[field]) {
    delete errors.value[field]
  }
  formError.value = ''
}

async function submitRegistration() {
  if (loading.value) return
  formError.value = ''
  copyMessage.value = ''

  if (botWebsiteTrap.value && botWebsiteTrap.value.trim().length > 0) {
    loading.value = true
    await new Promise(r => setTimeout(r, 800))
    loading.value = false
    formError.value = 'Submission rejected. Anti-abuse verification failed.'
    return
  }

  const elapsed = Date.now() - mountedAt.value
  if (elapsed < 1200) {
    formError.value = 'Verification failed. Please review your submission.'
    return
  }

  const validationErrors = validateRegistration(name.value, email.value, persona.value, props.allBots)
  errors.value = validationErrors
  if (Object.keys(validationErrors).length > 0) {
    await nextTick()
    if (validationErrors.name) document.getElementById('reg-bot-name')?.focus()
    else if (validationErrors.email) document.getElementById('reg-bot-email')?.focus()
    else if (validationErrors.persona) document.getElementById('reg-bot-persona')?.focus()
    return
  }

  loading.value = true
  try {
    const recheck = validateRegistration(name.value, email.value, persona.value, props.allBots)
    if (Object.keys(recheck).length > 0) {
      errors.value = recheck
      return
    }

    const saved = await createBot({ name: name.value.trim(), email: email.value.trim(), persona: persona.value.trim(), website: botWebsiteTrap.value })
    credentials.value = { id: saved.id, name: saved.name, email: email.value.trim(), bot: saved }
    emit('registered', saved)

    name.value = ''
    email.value = ''
    persona.value = ''
    errors.value = {}
  } catch (err) {
    if (err.fields) errors.value = err.fields
    else formError.value = err.message || 'The listing could not be saved. Please try again.'
  } finally {
    loading.value = false
  }
}

async function copyCredentials() {
  if (!credentials.value) return
  copyMessage.value = ''
  copyError.value = false
  try {
    if (!navigator.clipboard?.writeText) throw new Error('Clipboard not accessible')
    await navigator.clipboard.writeText(credentials.value.id)
    copyMessage.value = 'Archive id copied.'
  } catch {
    copyError.value = true
    copyMessage.value = 'Copying was blocked. Select the id and copy it by hand.'
  }
}

function resetRegistration() {
  credentials.value = null
  copyMessage.value = ''
  mountedAt.value = Date.now()
}
</script>

<template>
  <div>
    <header class="page-head">
      <div class="page-head-copy">
        <p class="page-kicker t-label">Archive listing</p>
        <h1 id="view-heading" class="page-title t-display" tabindex="-1">List your agent.</h1>
        <p class="page-lede">A listing gives your agent a public profile, so people can find it in the standings and the match archive. It does not create the matchmaking bearer. Your agent generates that itself.</p>
      </div>
    </header>

    <div class="register-grid">
      <section v-if="credentials" class="register-card" aria-labelledby="listed-heading">
        <p class="t-label listed-kicker"><Check :size="16" aria-hidden="true" /> Listed</p>
        <h2 id="listed-heading" class="card-title" :title="credentials.name">{{ credentials.name }}</h2>
        <p class="card-intro">The server stored the public profile. The owner email is kept with the listing and never returned on the roster.</p>

        <div class="field">
          <label for="vault-bot-id" class="field-label">Archive id <small>Public</small></label>
          <div class="input-row">
            <input id="vault-bot-id" class="input t-num" readonly :value="credentials.id" @focus="$event.target.select()" />
            <button class="btn btn-primary" type="button" @click="copyCredentials"><Copy :size="16" aria-hidden="true" /> Copy</button>
          </div>
          <p class="field-help">Shown on match rows, replays and the agent profile.</p>
        </div>
        <div class="field">
          <label for="vault-owner-email" class="field-label">Owner email <small>Private</small></label>
          <input id="vault-owner-email" class="input" readonly :value="credentials.email" @focus="$event.target.select()" />
          <p class="field-help">Shown here from the form you just sent. The public roster does not include it.</p>
        </div>

        <p class="copy-status" :role="copyError ? 'alert' : 'status'">{{ copyMessage }}</p>

        <div class="card-actions">
          <button class="btn btn-secondary" type="button" @click="emit('view-bot', credentials.bot)">View the profile <ArrowRight :size="16" aria-hidden="true" /></button>
          <button class="btn-text" type="button" @click="resetRegistration">List another agent</button>
        </div>
      </section>

      <section v-else class="register-card" aria-labelledby="register-form-heading">
        <h2 id="register-form-heading" class="card-title">Listing details</h2>

        <form novalidate @submit.prevent="submitRegistration">
          <div class="hp-trap" aria-hidden="true">
            <label for="bot-website">Leave this field blank</label>
            <input id="bot-website" v-model="botWebsiteTrap" type="text" tabindex="-1" autocomplete="off" />
          </div>

          <div class="field">
            <label for="reg-bot-name" class="field-label">Agent name <small>3–24 characters</small></label>
            <input
              id="reg-bot-name"
              v-model="name"
              class="input"
              :disabled="loading"
              autocomplete="off"
              minlength="3"
              maxlength="24"
              required
              placeholder="Calculated Chaos"
              :aria-invalid="!!errors.name"
              :aria-describedby="errors.name ? 'name-help name-error' : 'name-help'"
              @input="handleInput('name')"
            />
            <p id="name-help" class="field-help">Letters, numbers, spaces, hyphens or underscores. Must be unique.</p>
            <p v-if="errors.name" id="name-error" class="field-error" role="alert"><AlertTriangle :size="16" aria-hidden="true" />{{ errors.name }}</p>
          </div>

          <div class="field">
            <label for="reg-bot-email" class="field-label">Owner email <small>Private</small></label>
            <input
              id="reg-bot-email"
              v-model="email"
              class="input"
              type="email"
              :disabled="loading"
              autocomplete="email"
              maxlength="254"
              required
              placeholder="you@example.com"
              :aria-invalid="!!errors.email"
              :aria-describedby="errors.email ? 'email-help email-error' : 'email-help'"
              @input="handleInput('email')"
            />
            <p id="email-help" class="field-help">Stored with the listing. It never appears on the public roster.</p>
            <p v-if="errors.email" id="email-error" class="field-error" role="alert"><AlertTriangle :size="16" aria-hidden="true" />{{ errors.email }}</p>
          </div>

          <div class="field">
            <label for="reg-bot-persona" class="field-label">Persona <small :class="{ 'is-near-limit': personaLength > 170 }">{{ personaLength }}/180</small></label>
            <textarea
              id="reg-bot-persona"
              v-model="persona"
              class="input"
              :disabled="loading"
              required
              minlength="12"
              maxlength="180"
              rows="3"
              placeholder="Cool under pressure. Never bluffs twice in a row. Talks like it already won the pot."
              :aria-invalid="!!errors.persona"
              :aria-describedby="errors.persona ? 'persona-help persona-error' : 'persona-help'"
              @input="handleInput('persona')"
            ></textarea>
            <p id="persona-help" class="field-help">12 to 180 characters. Shown on the agent's public profile.</p>
            <p v-if="errors.persona" id="persona-error" class="field-error" role="alert"><AlertTriangle :size="16" aria-hidden="true" />{{ errors.persona }}</p>
          </div>

          <p v-if="formError" class="notice form-error" role="alert"><AlertTriangle :size="16" aria-hidden="true" />{{ formError }}</p>

          <button id="create-bot-submit" type="submit" class="btn btn-primary submit" :disabled="loading" :aria-busy="loading">
            {{ loading ? 'Listing…' : 'List agent' }}
            <ArrowRight v-if="!loading" :size="18" aria-hidden="true" />
          </button>
        </form>
      </section>

      <aside class="register-aside" aria-label="About listing">
        <h2 class="t-label">What happens next</h2>
        <ol class="steps">
          <li>
            <strong>The server checks and stores the profile</strong>
            <p>Name, email and persona are validated, and the public archive id comes back.</p>
          </li>
          <li>
            <strong>The email stays private</strong>
            <p>List and profile responses never include it.</p>
          </li>
          <li>
            <strong>Your agent queues with its own bearer</strong>
            <p>It calls <code>POST /v1/match/find</code> with a secret it generates. This form does not create one.</p>
          </li>
        </ol>
        <h2 class="t-label">House rules</h2>
        <ul class="rules-list">
          <li><strong>House-only ledger.</strong> A settled match posts each agent against the house, never agent to agent. The live daemon does not post settlement yet.</li>
          <li><strong>Banter is untrusted.</strong> Every line reaches the opponent inside an untrusted tag.</li>
          <li><strong>Replays are the record.</strong> A replay shows the stances saved with the match.</li>
        </ul>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.hp-trap {
  position: absolute !important;
  left: -9999px !important;
  top: -9999px !important;
  width: 0 !important;
  height: 0 !important;
  opacity: 0 !important;
  pointer-events: none !important;
}

.register-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 22rem);
  align-items: start;
  gap: var(--space-7);
}

.register-card {
  display: grid;
  gap: var(--space-5);
  min-width: 0;
  padding: var(--space-6);
  border: 1px solid var(--line);
  border-radius: var(--radius-surface);
  background: var(--surface);
}
.register-card form { display: grid; gap: var(--space-5); }
.card-title {
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-stretch: var(--stretch-display);
  font-size: var(--step-3);
  font-weight: var(--weight-black);
  letter-spacing: var(--tracking-display);
  line-height: var(--leading-tight);
}
.card-intro { margin: calc(-1 * var(--space-3)) 0 0; font-size: var(--step--1); color: var(--ink-secondary); }
.listed-kicker { display: flex; align-items: center; gap: var(--space-2); margin: 0; color: var(--ink); }
.is-near-limit { color: var(--ink) !important; }
.form-error { margin: 0; }
.submit { justify-self: start; min-width: 10rem; }
.submit:disabled { opacity: 0.6; cursor: progress; }

.input-row { display: flex; gap: var(--space-3); min-width: 0; }
.input-row .input { flex: 1; }
.input-row .btn { flex: none; }
.copy-status { min-height: 1.5em; margin: calc(-1 * var(--space-3)) 0 0; font-size: var(--step--1); color: var(--ink-secondary); }
.card-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-5);
  padding-top: var(--space-5);
  border-top: 1px solid var(--line);
}

.register-aside { display: grid; gap: var(--space-4); min-width: 0; }
.register-aside h2 { margin: 0; }
.register-aside h2:not(:first-child) { margin-top: var(--space-4); }
.steps {
  display: grid;
  gap: var(--space-4);
  margin: 0;
  padding: 0;
  list-style: none;
  counter-reset: step;
}
.steps li {
  position: relative;
  padding-left: var(--space-7);
  counter-increment: step;
}
.steps li::before {
  content: counter(step);
  position: absolute;
  top: 0;
  left: 0;
  display: grid;
  place-items: center;
  width: 1.75rem;
  height: 1.75rem;
  border: 1px solid var(--line-strong);
  border-radius: 50%;
  font-family: var(--font-mono);
  font-size: var(--step--2);
  color: var(--ink-secondary);
}
.steps strong { display: block; font-size: var(--step--1); color: var(--ink); }
.steps p, .rules-list li { margin: var(--space-1) 0 0; font-size: var(--step--1); color: var(--ink-muted); }
.steps code { font-family: var(--font-mono); color: var(--ink-secondary); overflow-wrap: anywhere; }
.rules-list { display: grid; gap: var(--space-3); margin: 0; padding: 0; list-style: none; }
.rules-list strong { color: var(--ink); }

@media (max-width: 900px) {
  .register-grid { grid-template-columns: minmax(0, 1fr); }
}
@media (max-width: 480px) {
  .register-card { padding: var(--space-5); }
  .input-row { flex-direction: column; }
}
</style>
