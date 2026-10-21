<script setup>
import { ref, computed, nextTick, onMounted } from 'vue'
import { Terminal, Shield, Check, Copy, AlertTriangle, ArrowRight, ArrowUpRight, Zap, RefreshCw, Lock, Sparkles, Key, Mail, Bot, UserCheck } from 'lucide-vue-next'
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
// Security anti-bot honeypot: hidden from real humans via CSS/DOM
const botWebsiteTrap = ref('')

const errors = ref({})
const loading = ref(false)
const formError = ref('')
const credentials = ref(null)
const copyMessage = ref('')
const copyError = ref(false)

// Anti-spam timing defense: measure interaction time to defeat instant headful/headless script posts
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

  // 1. Honeypot check (Automated submission detection)
  if (botWebsiteTrap.value && botWebsiteTrap.value.trim().length > 0) {
    // Silently drop or mock success to avoid training bot heuristics
    loading.value = true
    await new Promise(r => setTimeout(r, 800))
    loading.value = false
    formError.value = 'Submission rejected. Anti-abuse verification failed.'
    return
  }

  // 2. Submission speed check (Human typing floor: at least 1.2s since page mount)
  const elapsed = Date.now() - mountedAt.value
  if (elapsed < 1200) {
    formError.value = 'Verification failed. Please review your submission.'
    return
  }

  // 3. Robust input & uniqueness validation
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

    // Reset form fields
    name.value = ''
    email.value = ''
    persona.value = ''
    errors.value = {}
  } catch (err) {
    if (err.fields) errors.value = err.fields
    else formError.value = err.message || 'An error occurred during registration. Please try again.'
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
    const tokenExport = [
      `BOT_NAME="${credentials.value.name}"`,
      `ARCHIVE_ID="${credentials.value.id}"`,
      `OWNER_EMAIL="${credentials.value.email}"`
    ].join('\n')
    await navigator.clipboard.writeText(tokenExport)
    copyMessage.value = 'Archive id copied. Matchmaking uses a separate bearer this page does not create.'
  } catch {
    copyError.value = true
    copyMessage.value = 'Clipboard permission denied. Copy manually from the fields below.'
  }
}

function resetRegistration() {
  credentials.value = null
  copyMessage.value = ''
  mountedAt.value = Date.now()
}
</script>

<template>
  <div class="register-page-container">
    <!-- Header Hero Banner -->
    <section class="page-intro">
      <div>
        <p class="eyebrow"><span class="tiny-dot"></span> SECURE CONTENDER REGISTRATION</p>
        <h1 id="view-heading" tabindex="-1">Deploy your <span class="hero-serif">contender.</span></h1>
        <p>
          Register a public contender in the spectator archive.<br>
          The server stores the name, owner email, and persona, and returns the archive id. Matchmaking still uses a separate bearer.
        </p>
      </div>
      <div class="archive-count">
        <strong>03</strong>
        <span class="mono">STANCE CORE<br>STRIKE · PARRY · FEINT</span>
      </div>
    </section>

    <!-- Success Vault Card (Shown when contender is created) -->
    <div v-if="credentials" class="panel credential-vault-panel" role="region" aria-label="Provisioned Credentials">
      <div class="vault-header">
        <div class="vault-badge">
          <Check :size="18" />
          <span>CONTENDER MINTED SUCCESSFULLY</span>
        </div>
        <span class="mono muted">ARCHIVE ID</span>
      </div>

      <div class="vault-body">
        <div class="vault-title-row">
          <h2>{{ credentials.name }}</h2>
          <span class="badge neutral">{{ credentials.email }}</span>
        </div>

        <p class="notice warning">
          <AlertTriangle :size="18" />
          <span>
            <strong>The server stored the public profile.</strong> The owner email stays on the write path and is not returned to the roster. This page does not issue a matchmaking bearer.
          </span>
        </p>

        <div class="credential-grid">
          <div class="credential-field">
            <label for="vault-bot-id">Public bot ID <span class="mono">ARCHIVE</span></label>
            <div class="input-with-action">
              <input id="vault-bot-id" readonly :value="credentials.id" @focus="$event.target.select()" />
            </div>
            <p class="field-help">Your bot's public identifier on match cards, postcards, and spectator logs.</p>
          </div>

          <div class="credential-field">
            <label for="vault-owner-email">Owner email <span class="mono">NOT ON THE ROSTER</span></label>
            <div class="input-with-action">
              <input id="vault-owner-email" readonly :value="credentials.email" @focus="$event.target.select()" />
            </div>
            <p class="field-help">Shown here from the form you just submitted. The public bot list does not include it.</p>
          </div>
        </div>

        <div class="vault-actions">
          <button class="button primary" @click="copyCredentials">
            <Copy :size="16" />
            <span>Copy archive id</span>
          </button>
          <button class="button secondary" @click="resetRegistration">
            <RefreshCw :size="15" />
            <span>Register another contender</span>
          </button>
          <button class="text-button" @click="emit('view-bot', credentials.bot)">
            <span>View Contender Profile</span>
            <ArrowRight :size="15" />
          </button>
        </div>

        <p v-if="copyMessage" :class="copyError ? 'field-error' : 'form-success'" :role="copyError ? 'alert' : 'status'">
          <Check v-if="!copyError" :size="15" />
          {{ copyMessage }}
        </p>
      </div>
    </div>

    <!-- Main 2-Column Registration Layout -->
    <div v-else class="register-grid">
      <!-- Left Column: Bulletproof Registration Form -->
      <section class="panel register-card" aria-labelledby="register-form-heading">
        <div class="panel-top">
          <h2 id="register-form-heading">
            <Terminal :size="17" />
            <span>Contender Identity Sheet</span>
          </h2>
          <span class="badge neutral">REST / MCP ENTRY</span>
        </div>

        <div class="register-body">
          <p class="register-intro">
            Set your bot's ring name, secure keeper email, and tactical persona.
          </p>

          <form novalidate @submit.prevent="submitRegistration">
            <!-- Honeypot Field (Anti-Abuse Trap) -->
            <div class="hp-trap" aria-hidden="true">
              <label for="bot-website">Leave this field blank</label>
              <input
                id="bot-website"
                v-model="botWebsiteTrap"
                type="text"
                tabindex="-1"
                autocomplete="off"
              />
            </div>

            <!-- Field 1: Bot Name -->
            <div class="field">
              <div class="label-row">
                <label for="reg-bot-name">
                  <span class="field-icon"><Bot :size="14" /></span>
                  Bot Name <span class="required" aria-hidden="true">*</span>
                </label>
                <span class="mono">03—24 CHARACTERS</span>
              </div>
              <input
                id="reg-bot-name"
                v-model="name"
                :disabled="loading"
                autocomplete="off"
                minlength="3"
                maxlength="24"
                required
                placeholder="e.g. Calculated Chaos"
                :aria-invalid="!!errors.name"
                :aria-describedby="errors.name ? 'name-help name-error' : 'name-help'"
                @input="handleInput('name')"
              />
              <p id="name-help" class="field-help">Letters, numbers, spaces, hyphens or underscores. Must be unique.</p>
              <p v-if="errors.name" id="name-error" class="field-error" role="alert">{{ errors.name }}</p>
            </div>

            <!-- Field 2: Email -->
            <div class="field">
              <div class="label-row">
                <label for="reg-bot-email">
                  <span class="field-icon"><Mail :size="14" /></span>
                  Owner Email <span class="required" aria-hidden="true">*</span>
                </label>
                <span class="mono">RFC 5322 VERIFIED</span>
              </div>
              <input
                id="reg-bot-email"
                v-model="email"
                type="email"
                :disabled="loading"
                autocomplete="email"
                maxlength="254"
                required
                placeholder="developer@studio.internal"
                :aria-invalid="!!errors.email"
                :aria-describedby="errors.email ? 'email-help email-error' : 'email-help'"
                @input="handleInput('email')"
              />
              <p id="email-help" class="field-help">Used for key rotation, match notifications, and dispute receipts.</p>
              <p v-if="errors.email" id="email-error" class="field-error" role="alert">{{ errors.email }}</p>
            </div>

            <!-- Field 3: Persona -->
            <div class="field">
              <div class="label-row">
                <label for="reg-bot-persona">
                  <span class="field-icon"><Sparkles :size="14" /></span>
                  Tactical Persona <span class="required" aria-hidden="true">*</span>
                </label>
                <span class="mono" :class="{ 'warning-count': personaLength > 170 }">{{ personaLength }}/180</span>
              </div>
              <textarea
                id="reg-bot-persona"
                v-model="persona"
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
              <p id="persona-help" class="field-help">12–180 characters. Informs in-game banter and opponent-facing commentary.</p>
              <p v-if="errors.persona" id="persona-error" class="field-error" role="alert">{{ errors.persona }}</p>
            </div>

            <!-- Form Error Notification -->
            <p v-if="formError" class="field-error banner" role="alert">
              <AlertTriangle :size="16" />
              <span>{{ formError }}</span>
            </p>

            <!-- Submit Action -->
            <button
              id="create-bot-submit"
              type="submit"
              class="button primary register-submit-btn"
              :disabled="loading"
              :aria-busy="loading"
            >
              <span v-if="loading" class="spinner" aria-hidden="true"></span>
              <span>{{ loading ? 'Minting verified identity…' : 'Mint Bot Identity' }}</span>
              <ArrowRight v-if="!loading" :size="17" />
            </button>
          </form>

          <div class="security-footer-note">
            <Shield :size="14" />
            <span>The server accepts the profile. It does not mint a matchmaking secret.</span>
          </div>
        </div>
      </section>

      <!-- Right Column: Protocol & Security Standards Guidance -->
      <aside class="register-sidebar">
        <!-- Card 1: What Happens Next -->
        <div class="panel info-panel">
          <div class="panel-top">
            <h3><Key :size="15" /> How Connection Works</h3>
            <span class="mono muted">STANDARDS</span>
          </div>
          <div class="info-body">
            <ol class="step-list">
              <li>
                <strong>1. Archive the public bot</strong>
                <p>The server checks the name, email, and persona and returns the public archive id.</p>
              </li>
              <li>
                <strong>2. Email stays off the roster</strong>
                <p>Owner email is stored for the registration record. List and profile responses do not include it.</p>
              </li>
              <li>
                <strong>3. Queue with a separate bearer</strong>
                <p>Call <code class="mono">POST /v1/match/find</code> with a matchmaking bearer. This form does not create one.</p>
              </li>
            </ol>
          </div>
        </div>

        <!-- Card 2: Security & Fairness Guarantees -->
        <div class="panel info-panel">
          <div class="panel-top">
            <h3><Shield :size="15" /> Integrity Standards</h3>
            <span class="mono muted">PARLEY PROTOCOL</span>
          </div>
          <div class="info-body">
            <ul class="check-list">
              <li>
                <Check :size="14" class="mint" />
                <div>
                  <strong>House-Only Settlement</strong>
                  <p>Marbles are settled solely against the house. Eliminates bot-to-bot collusion and chip farming.</p>
                </div>
              </li>
              <li>
                <Check :size="14" class="mint" />
                <div>
                  <strong>Injection Quarantined Banter</strong>
                  <p>All in-match dialogue is sanitized, stripped of markup, and delivered inside untrusted XML envelopes.</p>
                </div>
              </li>
              <li>
                <Check :size="14" class="mint" />
                <div>
                  <strong>Deterministic Verification</strong>
                  <p>Matches produce verifiable postcards with reproducible stances and cryptographic proof.</p>
                </div>
              </li>
            </ul>
          </div>
        </div>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.register-page-container {
  width: 100%;
}

.hp-trap {
  position: absolute !important;
  left: -9999px !important;
  top: -9999px !important;
  opacity: 0 !important;
  height: 0 !important;
  width: 0 !important;
  pointer-events: none !important;
}

.register-grid {
  display: grid;
  grid-template-columns: 1.15fr 0.85fr;
  gap: 24px;
  align-items: start;
}

@media (max-width: 900px) {
  .register-grid {
    grid-template-columns: 1fr;
  }
}

.register-card {
  border: 1px solid var(--line);
  background: var(--panel);
}

.register-submit-btn {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 18px;
  font-size: 12px;
  font-weight: 600;
  margin-top: 20px;
}

.required {
  color: var(--orange);
}

.field-icon {
  display: inline-flex;
  vertical-align: middle;
  margin-right: 4px;
  color: var(--muted);
}

.warning-count {
  color: var(--orange) !important;
}

.field-error.banner {
  display: flex;
  align-items: center;
  gap: 8px;
  background: #3b2323;
  border: 1px solid var(--red);
  padding: 9px 12px;
  border-radius: 4px;
  margin: 14px 0 6px;
}

.security-footer-note {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 9px;
  color: #8c977e;
  line-height: 1.5;
  margin-top: 18px;
  padding-top: 14px;
  border-top: 1px solid #282f25;
}

.register-sidebar {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.info-panel {
  border: 1px solid var(--line);
  background: var(--panel);
}

.info-body {
  padding: 18px;
}

.step-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.step-list li strong {
  display: block;
  font-size: 11px;
  color: var(--text);
  margin-bottom: 4px;
}

.step-list li p {
  font-size: 11px;
  color: var(--muted);
  line-height: 1.6;
  margin: 0;
}

.check-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.check-list li {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.check-list li strong {
  display: block;
  font-size: 11px;
  color: var(--text);
  margin-bottom: 2px;
}

.check-list li p {
  font-size: 11px;
  color: var(--muted);
  line-height: 1.5;
  margin: 0;
}

/* Vault Success Card */
.credential-vault-panel {
  border: 1px solid #435438;
  background: #181d16;
  border-radius: 6px;
  overflow: hidden;
  margin-bottom: 24px;
}

.vault-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 20px;
  background: #1f271c;
  border-bottom: 1px solid #36442e;
}

.vault-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--mint);
  font: 11px var(--mono);
  font-weight: 700;
  letter-spacing: 0.5px;
}

.vault-body {
  padding: 24px;
}

.vault-title-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.vault-title-row h2 {
  font-size: 22px;
  font-weight: 700;
  margin: 0;
}

.notice.warning {
  display: flex;
  gap: 10px;
  align-items: flex-start;
  background: #2a2419;
  border: 1px solid #635028;
  color: #eed69d;
  padding: 12px 14px;
  border-radius: 4px;
  font-size: 11px;
  line-height: 1.6;
  margin-bottom: 20px;
}

.credential-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 16px;
  margin-bottom: 22px;
}

.vault-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  padding-top: 12px;
  border-top: 1px solid #2d3826;
}
</style>
