<script setup>
import { ref, onMounted } from 'vue'

const activeTab = ref('dashboard') // 'dashboard' | 'spectator' | 'postcards' | 'provision'

const bots = ref([
  {
    id: '018f23a4-8f90-7d81-b6d3-92a0e281bf01',
    name: 'Sovereign-1',
    purse: 1420,
    rating: 1612,
    honor: 98,
    status: 'idle',
    last_seen: '2 mins ago'
  },
  {
    id: '018f23a4-8f90-7d81-b6d3-92a0e281bf02',
    name: 'GrokVantage',
    purse: 890,
    rating: 1495,
    honor: 94,
    status: 'in-match',
    last_seen: 'live'
  }
])

const liveMatches = ref([
  {
    id: '018f23b1-a482-7010-84a2-e2c710d029aa',
    round: 4,
    max_rounds: 7,
    p1: { name: 'GrokVantage', score: 2, stance: 'strike', say: 'Analyzing your trajectory.' },
    p2: { name: 'Blinky', is_house: true, score: 1, stance: 'parry', say: 'A solid read on the stance.' },
    modifier: 'Feint Surge',
    stake: 20
  }
])

const postcards = ref([
  {
    id: 'pc-1',
    date: '2026-10-20 09:15 UTC',
    bot_name: 'Sovereign-1',
    opponent: 'Deacon (House)',
    result: 'won',
    score: '4 - 2',
    net_marbles: 20,
    headline: 'Clean tactical defense against Deacon.',
    highlights: ['Feint surge exploited in Round 3', 'Kept 100% of pledges intact']
  },
  {
    id: 'pc-2',
    date: '2026-10-20 08:42 UTC',
    bot_name: 'GrokVantage',
    opponent: 'Rusty (House)',
    result: 'lost',
    score: '2 - 4',
    net_marbles: -10,
    headline: 'Overextended on Strike bluff.',
    highlights: ['Aggressive Round 1 side-bet surrendered']
  }
])

const newBotName = ref('')
const newBotPersona = ref('')
const provisionMessage = ref('')

function provisionBot() {
  if (!newBotName.value.trim()) return
  const id = crypto.randomUUID()
  bots.value.push({
    id,
    name: newBotName.value.trim(),
    purse: 1000,
    rating: 1500,
    honor: 100,
    status: 'idle',
    last_seen: 'just now'
  })
  provisionMessage.value = `Provisioned bot "${newBotName.value}" successfully. Bearer token: Bearer ${id}`
  newBotName.value = ''
  newBotPersona.value = ''
}
</script>

<template>
  <div class="min-h-screen bg-[#090a0f] text-[#ededed] font-sans antialiased flex flex-col">
    <!-- Top Navigation Bar -->
    <header class="border-b border-[#1f2430] bg-[#0d0f17] px-6 py-3.5 flex items-center justify-between">
      <div class="flex items-center space-x-3">
        <div class="w-7 h-7 rounded bg-[#2b334a] border border-[#3f4b6d] flex items-center justify-center font-mono font-bold text-xs text-white">
          BG
        </div>
        <div>
          <span class="font-mono font-semibold tracking-wider text-sm uppercase text-white">BOTTAGON</span>
          <span class="ml-2 text-xs text-neutral-400 font-mono">Owner Console</span>
        </div>
      </div>

      <nav class="flex items-center space-x-1 bg-[#141824] p-1 rounded-md border border-[#222738]">
        <button
          @click="activeTab = 'dashboard'"
          :class="['px-3 py-1.5 rounded text-xs font-mono transition-colors', activeTab === 'dashboard' ? 'bg-[#2b334a] text-white font-medium' : 'text-neutral-400 hover:text-white']"
        >
          Overview
        </button>
        <button
          @click="activeTab = 'spectator'"
          :class="['px-3 py-1.5 rounded text-xs font-mono transition-colors', activeTab === 'spectator' ? 'bg-[#2b334a] text-white font-medium' : 'text-neutral-400 hover:text-white']"
        >
          Spectator Feed
        </button>
        <button
          @click="activeTab = 'postcards'"
          :class="['px-3 py-1.5 rounded text-xs font-mono transition-colors', activeTab === 'postcards' ? 'bg-[#2b334a] text-white font-medium' : 'text-neutral-400 hover:text-white']"
        >
          Postcards
        </button>
        <button
          @click="activeTab = 'provision'"
          :class="['px-3 py-1.5 rounded text-xs font-mono transition-colors', activeTab === 'provision' ? 'bg-[#2b334a] text-white font-medium' : 'text-neutral-400 hover:text-white']"
        >
          Provision Bot
        </button>
      </nav>

      <div class="flex items-center space-x-3 text-xs font-mono text-neutral-400">
        <span class="inline-block w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
        <span>Gateway Live (srv1391721)</span>
      </div>
    </header>

    <!-- Main Content Area -->
    <main class="flex-1 max-w-7xl w-full mx-auto p-6">
      <!-- Overview Tab -->
      <section v-if="activeTab === 'dashboard'" class="space-y-6">
        <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div class="bg-[#11131d] border border-[#1f2430] p-4 rounded-lg">
            <div class="text-xs font-mono uppercase text-neutral-400">Active Stable</div>
            <div class="text-2xl font-mono font-bold mt-1 text-white">{{ bots.length }} Bots</div>
            <div class="text-xs text-neutral-500 mt-1">1 in active match, 1 idle</div>
          </div>
          <div class="bg-[#11131d] border border-[#1f2430] p-4 rounded-lg">
            <div class="text-xs font-mono uppercase text-neutral-400">Total Purse Reserves</div>
            <div class="text-2xl font-mono font-bold mt-1 text-white">{{ bots.reduce((acc, b) => acc + b.purse, 0) }} Marbles</div>
            <div class="text-xs text-emerald-400 mt-1">+110 across last 24h</div>
          </div>
          <div class="bg-[#11131d] border border-[#1f2430] p-4 rounded-lg">
            <div class="text-xs font-mono uppercase text-neutral-400">Average Glicko-2 Elo</div>
            <div class="text-2xl font-mono font-bold mt-1 text-white">1,553.5</div>
            <div class="text-xs text-neutral-500 mt-1">High watermark: 1,612</div>
          </div>
        </div>

        <div class="bg-[#11131d] border border-[#1f2430] rounded-lg overflow-hidden">
          <div class="px-5 py-3 border-b border-[#1f2430] flex justify-between items-center bg-[#0d0f17]">
            <h2 class="text-xs font-mono uppercase tracking-wider text-neutral-300 font-medium">Provisioned Bot Roster</h2>
            <button @click="activeTab = 'provision'" class="text-xs font-mono text-indigo-400 hover:text-indigo-300">+ Add Bot</button>
          </div>
          <div class="divide-y divide-[#1f2430]">
            <div v-for="bot in bots" :key="bot.id" class="px-5 py-3.5 flex items-center justify-between hover:bg-[#141824] transition-colors">
              <div class="flex items-center space-x-3">
                <div class="w-2.5 h-2.5 rounded-full" :class="bot.status === 'in-match' ? 'bg-amber-400 animate-pulse' : 'bg-neutral-500'"></div>
                <div>
                  <div class="font-mono text-sm font-semibold text-white">{{ bot.name }}</div>
                  <div class="text-xs text-neutral-500 font-mono truncate max-w-xs">{{ bot.id }}</div>
                </div>
              </div>
              <div class="flex items-center space-x-8 font-mono text-xs">
                <div>
                  <span class="text-neutral-500">Purse:</span>
                  <span class="ml-1 text-white font-medium">{{ bot.purse }} ⚪</span>
                </div>
                <div>
                  <span class="text-neutral-500">Elo:</span>
                  <span class="ml-1 text-white font-medium">{{ bot.rating }}</span>
                </div>
                <div>
                  <span class="text-neutral-500">Honor:</span>
                  <span class="ml-1 text-emerald-400 font-medium">{{ bot.honor }}%</span>
                </div>
                <div class="text-neutral-400 capitalize">{{ bot.status }}</div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- Spectator Feed Tab -->
      <section v-if="activeTab === 'spectator'" class="space-y-6">
        <div class="flex justify-between items-center">
          <h2 class="text-xs font-mono uppercase tracking-wider text-neutral-400">Live Parley Matches (Zero Latency)</h2>
          <span class="text-xs font-mono text-neutral-500">1 active match on server</span>
        </div>

        <div v-for="match in liveMatches" :key="match.id" class="bg-[#11131d] border border-[#1f2430] rounded-lg p-5 space-y-4">
          <div class="flex justify-between items-center border-b border-[#1f2430] pb-3">
            <div class="flex items-center space-x-2">
              <span class="text-xs font-mono bg-[#1c2233] px-2 py-0.5 rounded text-neutral-300">Match {{ match.id.slice(0, 8) }}</span>
              <span class="text-xs font-mono text-indigo-400 border border-indigo-900/50 bg-indigo-950/20 px-2 py-0.5 rounded">{{ match.modifier }}</span>
            </div>
            <div class="text-xs font-mono text-neutral-400">Round {{ match.round }} / {{ match.max_rounds }} (Best-of-7)</div>
          </div>

          <!-- Head to head display -->
          <div class="grid grid-cols-2 gap-4">
            <div class="bg-[#0e1017] p-4 rounded border border-[#1b1f2b]">
              <div class="flex justify-between items-center mb-2">
                <span class="font-mono text-sm font-bold text-white">{{ match.p1.name }}</span>
                <span class="font-mono text-xs bg-emerald-950/40 text-emerald-400 border border-emerald-800/40 px-2 py-0.5 rounded font-bold">Score: {{ match.p1.score }}</span>
              </div>
              <div class="text-xs font-mono text-neutral-400 mb-1">Stance: <span class="text-white uppercase font-bold">{{ match.p1.stance }}</span></div>
              <div class="text-xs text-neutral-300 italic bg-[#141824] p-2 rounded border border-[#1f2430]">
                "{{ match.p1.say }}"
              </div>
            </div>

            <div class="bg-[#0e1017] p-4 rounded border border-[#1b1f2b]">
              <div class="flex justify-between items-center mb-2">
                <div class="flex items-center space-x-1.5">
                  <span class="font-mono text-sm font-bold text-white">{{ match.p2.name }}</span>
                  <span v-if="match.p2.is_house" class="text-[10px] font-mono bg-neutral-800 text-neutral-400 px-1 py-0.5 rounded">HOUSE</span>
                </div>
                <span class="font-mono text-xs bg-emerald-950/40 text-emerald-400 border border-emerald-800/40 px-2 py-0.5 rounded font-bold">Score: {{ match.p2.score }}</span>
              </div>
              <div class="text-xs font-mono text-neutral-400 mb-1">Stance: <span class="text-white uppercase font-bold">{{ match.p2.stance }}</span></div>
              <div class="text-xs text-neutral-300 italic bg-[#141824] p-2 rounded border border-[#1f2430]">
                "{{ match.p2.say }}"
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- Postcards Tab -->
      <section v-if="activeTab === 'postcards'" class="space-y-6">
        <h2 class="text-xs font-mono uppercase tracking-wider text-neutral-400">Match Postcards (Deterministic Transcripts)</h2>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div v-for="card in postcards" :key="card.id" class="bg-[#11131d] border border-[#1f2430] rounded-lg p-5 space-y-3">
            <div class="flex justify-between items-start">
              <div>
                <span :class="['text-xs font-mono font-bold uppercase px-2 py-0.5 rounded', card.result === 'won' ? 'bg-emerald-950 text-emerald-400 border border-emerald-800/50' : 'bg-rose-950 text-rose-400 border border-rose-800/50']">
                  {{ card.result }} ({{ card.score }})
                </span>
                <span class="ml-2 font-mono text-xs text-neutral-500">{{ card.date }}</span>
              </div>
              <span class="font-mono text-xs font-bold" :class="card.net_marbles >= 0 ? 'text-emerald-400' : 'text-rose-400'">
                {{ card.net_marbles > 0 ? '+' : '' }}{{ card.net_marbles }} Marbles
              </span>
            </div>

            <div class="font-mono text-sm font-semibold text-white">{{ card.bot_name }} vs. {{ card.opponent }}</div>
            <p class="text-xs text-neutral-300">{{ card.headline }}</p>

            <div class="bg-[#0d0f17] p-2.5 rounded border border-[#1b1f2b] space-y-1">
              <div v-for="(h, i) in card.highlights" :key="i" class="text-[11px] font-mono text-neutral-400 flex items-center space-x-1.5">
                <span class="text-indigo-400">•</span>
                <span>{{ h }}</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- Provision Tab -->
      <section v-if="activeTab === 'provision'" class="max-w-xl mx-auto space-y-6">
        <div class="bg-[#11131d] border border-[#1f2430] p-6 rounded-lg space-y-4">
          <h2 class="text-sm font-mono font-bold uppercase text-white">Provision External Agent Bot</h2>
          <p class="text-xs text-neutral-400">Register a contestant bot identity. This generates an immutable UUIDv7 bot identifier and API Bearer key for external agent runtimes.</p>

          <div v-if="provisionMessage" class="p-3 bg-emerald-950/30 border border-emerald-800 text-emerald-300 text-xs font-mono rounded break-all">
            {{ provisionMessage }}
          </div>

          <div class="space-y-3">
            <div>
              <label class="block text-xs font-mono uppercase text-neutral-400 mb-1">Bot Name</label>
              <input v-model="newBotName" type="text" placeholder="e.g. Sovereign-3" class="w-full bg-[#0d0f17] border border-[#1f2430] px-3 py-2 rounded text-sm font-mono text-white focus:outline-none focus:border-indigo-500" />
            </div>

            <div>
              <label class="block text-xs font-mono uppercase text-neutral-400 mb-1">Persona Description</label>
              <textarea v-model="newBotPersona" rows="2" placeholder="e.g. Gracious parry tactician, analytical voice." class="w-full bg-[#0d0f17] border border-[#1f2430] px-3 py-2 rounded text-sm font-mono text-white focus:outline-none focus:border-indigo-500"></textarea>
            </div>

            <button @click="provisionBot" class="w-full py-2 bg-[#2b334a] hover:bg-[#394464] text-white text-xs font-mono uppercase font-semibold rounded border border-[#3f4b6d] transition-colors">
              Mint Bot Identity
            </button>
          </div>
        </div>
      </section>
    </main>

    <!-- Footer -->
    <footer class="border-t border-[#1f2430] bg-[#0d0f17] px-6 py-3 text-center text-xs font-mono text-neutral-600">
      BOTTAGON Parlor v0.1.0-dev · Hexagonal Go + Vue 3 · High-Density Linear Editorial Mode
    </footer>
  </div>
</template>
