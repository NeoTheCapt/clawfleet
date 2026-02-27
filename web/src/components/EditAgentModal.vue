<template>
  <Modal :show="show" @close="$emit('close')" :width="720">
    <h3 class="text-lg font-semibold mb-4">⚙️ Edit Agent</h3>
    <div class="text-accent-blue text-sm mb-3">{{ agent.name }} ({{ agent.agent_type }})</div>

    <!-- Tabs -->
    <div class="flex border-b border-border mb-4">
      <button v-for="(tab, idx) in tabs" :key="idx"
        @click="activeTab = idx"
        :class="['px-4 py-2 text-sm transition', activeTab === idx ? 'border-b-2 border-accent-blue text-accent-blue' : 'text-text-muted hover:text-text']">
        {{ tab }}
      </button>
    </div>

    <!-- Tab 0: Basic -->
    <div v-show="activeTab === 0">
      <div class="grid grid-cols-2 gap-3 mb-3">
        <div>
          <label class="block text-sm text-text-muted mb-1">Role</label>
          <input v-model="form.role" type="text" placeholder="e.g. developer, researcher" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
        </div>
        <div>
          <label class="block text-sm text-text-muted mb-1">Docker Image</label>
          <input v-model="form.image" type="text" placeholder="auto (based on agent type)" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
        </div>
      </div>
      <label class="block text-sm text-text-muted mb-1">Description</label>
      <input v-model="form.description" type="text" placeholder="What does this agent do?" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
    </div>

    <!-- Tab 1: Model & Keys -->
    <div v-show="activeTab === 1">
      <div class="bg-bg-card border border-border rounded-lg p-4 mb-3">
        <h4 class="text-sm font-semibold mb-3">🧠 Model Configuration</h4>
        <div class="grid grid-cols-2 gap-3 mb-3">
          <div>
            <label class="block text-sm text-text-muted mb-1">Primary Provider</label>
            <select v-model="form.provider" @change="onProviderChange" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
              <option value="">Select provider...</option>
              <option v-for="p in providers" :key="p.value" :value="p.value">{{ p.label }}</option>
            </select>
          </div>
          <div>
            <label class="block text-sm text-text-muted mb-1">Model</label>
            <input v-model="form.model" type="text" :placeholder="modelPlaceholder" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm font-mono focus:outline-none focus:border-accent-blue">
            <div class="text-xs text-text-dim mt-1">{{ modelHint }}</div>
          </div>
        </div>
        <div v-if="form.provider === 'custom' || form.provider === 'azure'" class="mb-3">
          <label class="block text-sm text-text-muted mb-1">{{ form.provider === 'azure' ? 'Azure Endpoint' : 'Custom API Base URL' }}</label>
          <input v-model="form.custom_api_base" type="text" :placeholder="form.provider === 'azure' ? 'https://your-resource.openai.azure.com' : 'https://api.example.com/v1'" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
          <div class="text-xs text-text-dim mt-1">{{ form.provider === 'azure' ? 'Azure OpenAI endpoint (without /v1)' : 'OpenAI-compatible API base URL' }}</div>
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="block text-sm text-text-muted mb-1">Fallback Provider</label>
            <select v-model="form.fallback_provider" @change="onFallbackProviderChange" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
              <option value="">None</option>
              <option v-for="p in providers.filter(p => p.value !== 'custom')" :key="p.value" :value="p.value">{{ p.label }}</option>
            </select>
          </div>
          <div v-if="form.fallback_provider">
            <label class="block text-sm text-text-muted mb-1">Fallback Model</label>
            <input v-model="form.fallback_model" type="text" :placeholder="fallbackPlaceholder" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm font-mono focus:outline-none focus:border-accent-blue">
          </div>
        </div>
      </div>

      <div class="bg-bg-card border border-border rounded-lg p-4">
        <h4 class="text-sm font-semibold mb-3">🔑 API Keys</h4>
        <div v-for="(env, idx) in form.env_vars" :key="idx" class="flex gap-2 mb-2">
          <select v-model="env.key" class="flex-[2] px-2.5 py-1.5 bg-bg-input border border-border rounded-md text-text text-xs font-mono focus:outline-none focus:border-accent-blue">
            <option value="">-- Select Key --</option>
            <option v-for="k in commonKeys" :key="k" :value="k">{{ k }}</option>
          </select>
          <input v-model="env.value" :type="env.show ? 'text' : 'password'" placeholder="sk-..." class="flex-[3] px-2.5 py-1.5 bg-bg-input border border-border rounded-md text-text text-xs font-mono focus:outline-none focus:border-accent-blue">
          <button @click="env.show = !env.show" class="px-2 py-1 border border-border rounded-md bg-btn text-xs hover:bg-btn-hover transition">{{ env.show ? '🙈' : '👁️' }}</button>
          <button @click="form.env_vars.splice(idx, 1)" class="px-2 py-1 bg-btn-red border border-accent-red rounded-md text-xs hover:bg-btn-redHover transition">✕</button>
        </div>
        <button @click="form.env_vars.push({ key: '', value: '', show: false })" class="text-xs text-accent-blue hover:underline">+ Add Variable</button>
        <div class="mt-2 text-xs text-text-muted">
          Common keys: ANTHROPIC_API_KEY, OPENAI_API_KEY, OPENROUTER_API_KEY, GOOGLE_API_KEY, etc.
        </div>
      </div>
    </div>

    <!-- Tab 2: Persona -->
    <div v-show="activeTab === 2">
      <div class="bg-bg-card border border-border rounded-lg p-4">
        <div class="grid grid-cols-2 gap-3 mb-3">
          <div>
            <label class="block text-sm text-text-muted mb-1">Display Name</label>
            <input v-model="form.persona_name" type="text" placeholder="e.g. Alice, DevBot" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
          </div>
          <div>
            <label class="block text-sm text-text-muted mb-1">Language</label>
            <select v-model="form.language" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
              <option value="en">English</option>
              <option value="zh">中文</option>
              <option value="ja">日本語</option>
              <option value="ko">한국어</option>
              <option value="es">Español</option>
              <option value="auto">Auto-detect</option>
            </select>
          </div>
        </div>
        <label class="block text-sm text-text-muted mb-1">System Prompt / SOUL</label>
        <textarea v-model="form.system_prompt" placeholder="Define the agent's personality, role, and behavior guidelines..." class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm min-h-[150px] resize-y focus:outline-none focus:border-accent-blue"></textarea>
        <div class="flex gap-2 mt-2">
          <button v-for="tmpl in promptTemplates" :key="tmpl.name" @click="form.system_prompt = tmpl.prompt"
            class="px-3 py-1 text-xs border border-border rounded-full bg-btn hover:bg-btn-hover transition">
            {{ tmpl.name }}
          </button>
        </div>
      </div>
    </div>

    <!-- Tab 3: Channel -->
    <div v-show="activeTab === 3">
      <div class="bg-bg-card border border-border rounded-lg p-4">
        <div class="mb-3">
          <label class="block text-sm text-text-muted mb-1">Channel Type</label>
          <select v-model="form.channel_type" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
            <option value="">None (headless / API)</option>
            <option value="telegram">Telegram</option>
            <option value="discord">Discord</option>
            <option value="slack">Slack</option>
            <option value="whatsapp">WhatsApp</option>
            <option value="webhook">Webhook</option>
          </select>
        </div>
        <!-- Telegram -->
        <div v-if="form.channel_type === 'telegram'" class="space-y-3">
          <div>
            <label class="block text-sm text-text-muted mb-1">Bot Token *</label>

            <div v-if="telegramManaged" class="text-xs text-text-dim bg-bg-tertiary p-3 rounded-md mb-2">
              🔒 Telegram bot token is managed by the bound Position
              <span v-if="telegramManagedBy.position_title"> ({{ telegramManagedBy.position_title }})</span>
              <span v-if="telegramManagedBy.bot_name"> — Bot: {{ telegramManagedBy.bot_name }}</span>.
              Please change it in Company → Org Structure → Position → Telegram Bot.
            </div>

            <input v-model="form.channel.telegram_token"
              :disabled="telegramManaged"
              :type="form.channel._showToken ? 'text' : 'password'"
              placeholder="123456:ABC-DEF..."
              class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm font-mono focus:outline-none focus:border-accent-blue"
              :class="telegramManaged ? 'opacity-60 cursor-not-allowed' : ''">
            <button v-if="!telegramManaged" @click="form.channel._showToken = !form.channel._showToken" class="text-xs text-accent-blue mt-1 hover:underline">{{ form.channel._showToken ? 'Hide' : 'Show' }}</button>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-sm text-text-muted mb-1">DM Policy</label>
              <select v-model="form.channel.telegram_dm_policy" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
                <option value="open">Open (anyone)</option>
                <option value="allowlist">Allowlist only</option>
                <option value="pairing">Pairing (approve each)</option>
              </select>
            </div>
            <div v-if="form.channel.telegram_dm_policy === 'allowlist'">
              <div class="flex items-center justify-between mb-1">
                <label class="block text-sm text-text-muted">Allowed Users</label>
                <div class="flex items-center gap-2 text-xs text-text-dim">
                  <span>Format:</span>
                  <label class="flex items-center gap-1">
                    <input type="radio" value="id" v-model="form.channel.telegram_allow_mode" class="accent-accent-blue">
                    ID
                  </label>
                  <label class="flex items-center gap-1">
                    <input type="radio" value="username" v-model="form.channel.telegram_allow_mode" class="accent-accent-blue">
                    @username
                  </label>
                </div>
              </div>
              <input v-model="form.channel.telegram_allow" type="text"
                :placeholder="form.channel.telegram_allow_mode === 'id' ? '5312784204, 123456789' : '@user1, @user2'"
                class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
              <div v-if="form.channel.telegram_allow_mode === 'username'" class="text-xs text-text-dim mt-1">
                OpenClaw Telegram allowlist requires numeric sender IDs. Usernames may not work unless resolved.
              </div>
            </div>
          </div>
        </div>
        <!-- Discord -->
        <div v-if="form.channel_type === 'discord'" class="space-y-3">
          <div>
            <label class="block text-sm text-text-muted mb-1">Bot Token *</label>
            <input v-model="form.channel.discord_token" type="password" placeholder="Discord bot token" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm font-mono focus:outline-none focus:border-accent-blue">
          </div>
          <div>
            <label class="block text-sm text-text-muted mb-1">Guild ID</label>
            <input v-model="form.channel.discord_guild" type="text" placeholder="optional" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
          </div>
        </div>
        <!-- Slack -->
        <div v-if="form.channel_type === 'slack'" class="space-y-3">
          <div>
            <label class="block text-sm text-text-muted mb-1">Bot Token *</label>
            <input v-model="form.channel.slack_token" type="password" placeholder="xoxb-..." class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm font-mono focus:outline-none focus:border-accent-blue">
          </div>
          <div>
            <label class="block text-sm text-text-muted mb-1">App Token</label>
            <input v-model="form.channel.slack_app_token" type="password" placeholder="xapp-..." class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm font-mono focus:outline-none focus:border-accent-blue">
          </div>
        </div>
        <!-- WhatsApp -->
        <div v-if="form.channel_type === 'whatsapp'" class="space-y-3">
          <div class="text-xs text-text-dim bg-bg-tertiary p-3 rounded-md">⚠️ WhatsApp requires QR code pairing after deployment.</div>
          <div>
            <label class="block text-sm text-text-muted mb-1">Allowed Numbers</label>
            <input v-model="form.channel.whatsapp_allow" type="text" placeholder="+8613800138000, +1555..." class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
          </div>
        </div>
        <!-- Webhook -->
        <div v-if="form.channel_type === 'webhook'" class="space-y-3">
          <div>
            <label class="block text-sm text-text-muted mb-1">Webhook URL</label>
            <input v-model="form.channel.webhook_url" type="text" placeholder="https://..." class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
          </div>
          <div>
            <label class="block text-sm text-text-muted mb-1">Webhook Secret</label>
            <input v-model="form.channel.webhook_secret" type="password" placeholder="optional" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
          </div>
        </div>
      </div>
    </div>

    <!-- Tab 4: Resources -->
    <div v-show="activeTab === 4">
      <div class="bg-bg-card border border-border rounded-lg p-4">
        <h4 class="text-sm font-semibold mb-3">⚙️ Resource Limits</h4>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="block text-sm text-text-muted mb-1">CPU Limit</label>
            <input v-model="form.cpu_limit" type="text" placeholder="e.g. 2 (cores)" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
          </div>
          <div>
            <label class="block text-sm text-text-muted mb-1">Memory Limit</label>
            <input v-model="form.memory_limit" type="text" placeholder="e.g. 4Gi" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
          </div>
        </div>
      </div>
    </div>

    <!-- Tab 5: Debug -->
    <div v-show="activeTab === 5">
      <div class="bg-bg-card border border-border rounded-lg p-4 mb-3">
        <h4 class="text-sm font-semibold mb-3">IM Authentication Key</h4>
        <div class="text-xs text-text-muted mb-2">
          Status:
          <span v-if="imKeyStatusLoading">loading...</span>
          <span v-else-if="imKeyStatusError" class="text-accent-red">{{ imKeyStatusError }}</span>
          <span v-else>{{ imHasKey ? 'configured' : 'not configured' }}</span>
        </div>
        <button
          :disabled="imKeyStatusLoading || rotatingIMKey"
          @click="rotateIMKey"
          class="px-3 py-2 border border-accent-blue rounded-md text-sm text-accent-blue hover:bg-bg-hover transition disabled:opacity-60 disabled:cursor-not-allowed">
          {{ rotatingIMKey ? 'Rotating...' : 'Rotate IM Key' }}
        </button>
        <div v-if="rotatedIMKey" class="mt-3 p-3 border border-border rounded-md bg-bg-input">
          <div class="text-xs text-text-muted mb-1">New key (shown once):</div>
          <div class="text-xs font-mono break-all">{{ rotatedIMKey }}</div>
          <button @click="copyRotatedIMKey" class="mt-2 px-2 py-1 border border-border rounded-md text-xs bg-btn hover:bg-btn-hover transition">Copy</button>
        </div>
      </div>

      <div class="bg-bg-card border border-border rounded-lg p-4">
        <h4 class="text-sm font-semibold mb-3">Agent Debug Checks</h4>
        <div class="grid grid-cols-2 gap-2 mb-3">
          <button
            v-for="c in debugChecks"
            :key="c.key"
            :disabled="debugRunning"
            @click="runDebugChecks([c.key])"
            class="px-3 py-2 border border-border rounded-md bg-btn text-sm text-left hover:bg-btn-hover transition disabled:opacity-60 disabled:cursor-not-allowed">
            <div class="font-medium">{{ c.label }}</div>
            <div class="text-xs text-text-dim">{{ c.description }}</div>
          </button>
        </div>
        <button
          :disabled="debugRunning"
          @click="runDebugChecks(debugChecks.map(c => c.key))"
          class="px-3 py-2 border border-accent-blue rounded-md text-sm text-accent-blue hover:bg-bg-hover transition disabled:opacity-60 disabled:cursor-not-allowed">
          Run All Checks
        </button>
        <div v-if="debugTaskId" class="text-xs text-text-dim mt-2">Task: {{ debugTaskId }} <span v-if="debugRunning">(running)</span></div>
        <div v-if="debugError" class="text-xs text-accent-red mt-2">{{ debugError }}</div>
        <div class="mt-3 border border-border rounded-md bg-bg-input p-3">
          <div class="text-xs text-text-muted mb-1">Output</div>
          <pre class="text-xs whitespace-pre-wrap break-words max-h-[320px] overflow-auto">{{ debugOutput || 'No debug output yet.' }}</pre>
        </div>
      </div>
    </div>

    <!-- Error -->
    <div v-if="error" class="text-accent-red text-xs mt-3">{{ error }}</div>

    <!-- Actions -->
    <div class="flex justify-between items-center mt-5">
      <div class="text-xs text-text-dim">Agent ID: {{ agent.id }}</div>
      <div class="flex gap-2">
        <button @click="$emit('close')" class="px-4 py-2 border border-border rounded-md bg-btn text-text text-sm hover:bg-btn-hover transition">Cancel</button>
        <button @click="save(false)" class="px-4 py-2 bg-accent-blue border border-accent-blue rounded-md text-white text-sm hover:opacity-90 transition">💾 Save</button>
        <button @click="save(true)" class="px-4 py-2 bg-btn-green border border-btn-greenHover rounded-md text-text text-sm hover:bg-btn-greenHover transition">💾 Save & Redeploy</button>
      </div>
    </div>
  </Modal>
</template>

<script setup>
import { ref, watch, computed, inject, onBeforeUnmount } from 'vue'
import { api, apiPut, apiPost } from '../api'
import Modal from './Modal.vue'

const props = defineProps({ show: Boolean, agent: Object })
const emit = defineEmits(['close', 'saved'])
const toast = inject('toast')
const error = ref('')
const activeTab = ref(0)
const tabs = ['Basic', 'Model & Keys', 'Persona', 'Channel', 'Resources', 'Debug']
const debugChecks = [
  { key: 'openclaw_status_deep', label: 'OpenClaw Status Deep', description: 'Runs openclaw status --deep' },
  { key: 'openclaw_plugins_list', label: 'OpenClaw Plugins', description: 'Lists plugins in JSON format' },
  { key: 'openclaw_config', label: 'OpenClaw Config', description: 'Reads /root/.openclaw/openclaw.json' },
  { key: 'openclaw_logs_tail', label: 'OpenClaw Logs Tail', description: 'Tails latest OpenClaw logs' },
  { key: 'telegram_getme', label: 'Telegram getMe', description: 'Tests bot token with Telegram API' },
  { key: 'env_keys', label: 'Env Keys Presence', description: 'Shows required key presence only' },
]
const debugTaskId = ref('')
const debugRunning = ref(false)
const debugOutput = ref('')
const debugError = ref('')
let debugPollTimer = null
const imKeyStatusLoading = ref(false)
const imKeyStatusError = ref('')
const imHasKey = ref(false)
const rotatingIMKey = ref(false)
const rotatedIMKey = ref('')

const providers = [
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'openai', label: 'OpenAI' },
  { value: 'openrouter', label: 'OpenRouter' },
  { value: 'google', label: 'Google (Gemini)' },
  { value: 'groq', label: 'Groq' },
  { value: 'mistral', label: 'Mistral' },
  { value: 'xai', label: 'xAI' },
  { value: 'moonshot', label: 'Moonshot (Kimi)' },
  { value: 'azure', label: 'Azure AI' },
  { value: 'ollama', label: 'Ollama (local)' },
  { value: 'custom', label: 'Custom endpoint' },
]

const providerDefaults = {
  anthropic:  { placeholder: 'claude-opus-4-6',           hint: 'e.g. claude-opus-4-6, claude-sonnet-4-5' },
  openai:     { placeholder: 'gpt-5.1-codex',             hint: 'e.g. gpt-5.1-codex, gpt-4o, o3' },
  openrouter: { placeholder: 'anthropic/claude-sonnet-4-5', hint: 'e.g. anthropic/claude-sonnet-4-5' },
  google:     { placeholder: 'gemini-3-pro-preview',       hint: 'e.g. gemini-3-pro-preview' },
  groq:       { placeholder: 'llama-3.3-70b-versatile',   hint: 'e.g. llama-3.3-70b-versatile' },
  mistral:    { placeholder: 'mistral-large-latest',       hint: 'e.g. mistral-large-latest' },
  xai:        { placeholder: 'grok-3',                     hint: 'e.g. grok-3' },
  moonshot:   { placeholder: 'kimi-k2.5',                  hint: 'e.g. kimi-k2.5' },
  azure:      { placeholder: 'gpt-4o',                     hint: 'Azure deployment name, e.g. gpt-4o' },
  ollama:     { placeholder: 'llama3:8b',                  hint: 'Enter Ollama model name' },
  custom:     { placeholder: 'model-name',                 hint: 'Model ID for custom endpoint' },
}

const commonKeys = [
  'ANTHROPIC_API_KEY',
  'OPENAI_API_KEY',
  'OPENROUTER_API_KEY',
  'GOOGLE_API_KEY',
  'GEMINI_API_KEY',
  'GROQ_API_KEY',
  'MISTRAL_API_KEY',
  'XAI_API_KEY',
  'MOONSHOT_API_KEY',
  'AZURE_OPENAI_API_KEY',
  'AWS_ACCESS_KEY_ID',
  'AWS_SECRET_ACCESS_KEY',
]

const promptTemplates = [
  { name: '🤖 General', prompt: 'You are a helpful, friendly AI assistant. Be concise and accurate.' },
  { name: '💻 Developer', prompt: 'You are a senior software engineer. Write clean, well-documented code.' },
  { name: '📊 Analyst', prompt: 'You are a data analyst. Focus on accuracy, cite sources when possible.' },
  { name: '🎨 Creative', prompt: 'You are a creative writing assistant. Be imaginative and engaging.' },
]

const defaultForm = () => ({
  role: '', image: '', description: '',
  provider: '', model: '', fallback_provider: '', fallback_model: '', custom_api_base: '',
  system_prompt: '', persona_name: '', language: 'en',
  channel_type: '',
  channel: { telegram_token: '', telegram_allow: '', telegram_allow_mode: 'id', telegram_dm_policy: 'open', discord_token: '', discord_guild: '', slack_token: '', slack_app_token: '', whatsapp_allow: '', webhook_url: '', webhook_secret: '', _showToken: false },
  env_vars: [],
  cpu_limit: '', memory_limit: '',
})

const form = ref(defaultForm())

// If this agent is assigned to a Position that has a Telegram Bot bound,
// channel token should be managed by the Position (single source of truth).
const telegramManaged = ref(false)
const telegramManagedBy = ref({ position_title: '', bot_name: '' })

// Parse model string to extract provider prefix
function parseModel(fullModel) {
  if (!fullModel) return { provider: '', model: '' }
  // e.g. "anthropic/claude-opus-4-6" or "openrouter/anthropic/claude-sonnet-4-5"
  const knownPrefixes = providers.map(p => p.value)
  for (const p of knownPrefixes) {
    if (fullModel.startsWith(p + '/')) {
      return { provider: p, model: fullModel.slice(p.length + 1) }
    }
  }
  return { provider: '', model: fullModel }
}

watch(() => props.show, async (val) => {
  if (!val) {
    clearDebugPoll()
    debugRunning.value = false
    rotatedIMKey.value = ''
    return
  }
  if (!props.agent) return
  error.value = ''
  activeTab.value = 0
  clearDebugPoll()
  debugTaskId.value = ''
  debugRunning.value = false
  debugOutput.value = ''
  debugError.value = ''
  rotatedIMKey.value = ''
  imKeyStatusError.value = ''
  imHasKey.value = false
  telegramManaged.value = false
  telegramManagedBy.value = { position_title: '', bot_name: '' }

  const cfg = props.agent.config || {}
  const f = defaultForm()

  // Basic
  f.role = props.agent.role || ''
  f.image = cfg.image || ''
  f.description = cfg.description || ''

  // Model - parse provider from model string or config
  if (cfg.provider) {
    f.provider = cfg.provider
    // Strip provider prefix from model if present
    const modelStr = cfg.model || ''
    f.model = modelStr.startsWith(cfg.provider + '/') ? modelStr.slice(cfg.provider.length + 1) : modelStr
  } else {
    const parsed = parseModel(cfg.model)
    f.provider = parsed.provider
    f.model = parsed.model
  }
  f.custom_api_base = cfg.custom_api_base || ''

  // Fallback
  if (cfg.fallback_models && cfg.fallback_models.length > 0) {
    const parsed = parseModel(cfg.fallback_models[0])
    f.fallback_provider = parsed.provider
    f.fallback_model = parsed.model
  }

  // Persona
  f.system_prompt = cfg.system_prompt || ''
  f.persona_name = cfg.persona_name || ''
  f.language = cfg.language || 'en'

  // Channel
  const ch = cfg.channel
  if (ch && ch.type) {
    f.channel_type = ch.type
    if (ch.type === 'telegram') {
      f.channel.telegram_token = ch.token || ch.bot_token || ''
      f.channel.telegram_dm_policy = ch.dm_policy || 'open'
      f.channel.telegram_allow = (ch.allow_from || []).join(', ')
      // Heuristic: if allow_from looks like @username, set mode accordingly
      f.channel.telegram_allow_mode = (f.channel.telegram_allow.trim().startsWith('@') ? 'username' : 'id')
    } else if (ch.type === 'discord') {
      f.channel.discord_token = ch.token || ch.bot_token || ''
      f.channel.discord_guild = ch.guild_id || ''
    } else if (ch.type === 'slack') {
      f.channel.slack_token = ch.bot_token || ch.token || ''
      f.channel.slack_app_token = ch.app_token || ''
    } else if (ch.type === 'whatsapp') {
      f.channel.whatsapp_allow = (ch.allow_from || []).join(', ')
    } else if (ch.type === 'webhook') {
      f.channel.webhook_url = ch.url || ''
      f.channel.webhook_secret = ch.secret || ch.webhook_secret || ''
    }
  }

  // Env vars
  f.env_vars = Object.entries(cfg.env_vars || {}).map(([key, value]) => ({ key, value, show: false }))

  // Resources
  f.cpu_limit = cfg.resources?.cpu_limit || ''
  f.memory_limit = cfg.resources?.memory_limit || ''

  form.value = f

  // Determine if Telegram token is managed by the Position
  try {
    const companyId = props.agent.company_id
    if (companyId) {
      const assignments = await api(`/api/companies/${companyId}/assignments`).catch(() => []) || []
      const a = (assignments || []).find(x => x.agent_id === props.agent.id)
      if (a && a.position_id) {
        const positions = await api(`/api/companies/${companyId}/positions`).catch(() => []) || []
        const pos = (positions || []).find(p => p.id === a.position_id)
        if (pos && pos.telegram_bot_id) {
          telegramManaged.value = true
          telegramManagedBy.value.position_title = pos.title || pos.name || ''
          const bots = await api(`/api/companies/${companyId}/bots`).catch(() => []) || []
          const bot = (bots || []).find(b => b.id === pos.telegram_bot_id)
          telegramManagedBy.value.bot_name = bot?.name || ''

          // Force channel type to telegram in UI (read-only token)
          form.value.channel_type = 'telegram'
        }
      }
    }
  } catch {}

  await loadIMKeyStatus()
})

onBeforeUnmount(() => clearDebugPoll())

const modelPlaceholder = computed(() => providerDefaults[form.value.provider]?.placeholder || 'model-name')
const modelHint = computed(() => providerDefaults[form.value.provider]?.hint || '')
const fallbackPlaceholder = computed(() => providerDefaults[form.value.fallback_provider]?.placeholder || 'model-name')

function onProviderChange() {
  const d = providerDefaults[form.value.provider]
  if (d) form.value.model = d.placeholder
}
function onFallbackProviderChange() {
  if (!form.value.fallback_provider) { form.value.fallback_model = ''; return }
  const d = providerDefaults[form.value.fallback_provider]
  if (d) form.value.fallback_model = d.placeholder
}

function buildConfig() {
  const f = form.value
  const providerPrefix = ['ollama', 'custom', ''].includes(f.provider) ? '' : f.provider + '/'
  const fullModel = f.model.startsWith(f.provider + '/') ? f.model : providerPrefix + f.model

  let fallbacks = []
  if (f.fallback_provider && f.fallback_model) {
    const fbPrefix = ['ollama', 'custom'].includes(f.fallback_provider) ? '' : f.fallback_provider + '/'
    const fbModel = f.fallback_model.startsWith(f.fallback_provider + '/') ? f.fallback_model : fbPrefix + f.fallback_model
    fallbacks.push(fbModel)
  }

  let channel = undefined
  if (f.channel_type) {
    channel = { type: f.channel_type }
    const cc = f.channel
    if (f.channel_type === 'telegram') {
      // Avoid overwriting token from Agent settings:
      // - If Position manages it, never send a token here.
      // - If user left it blank, omit it so backend can preserve/resolve.
      if (!telegramManaged.value && cc.telegram_token && cc.telegram_token.trim()) {
        channel.token = cc.telegram_token.trim()
      }
      let list = cc.telegram_allow ? cc.telegram_allow.split(',').map(s => s.trim()).filter(Boolean) : []
      if (cc.telegram_allow_mode === 'username') {
        // ensure @prefix
        list = list.map(x => x.startsWith('@') ? x : '@' + x)
      } else {
        // numeric id mode: strip @ and whitespace
        list = list.map(x => x.replace(/^@+/, ''))
      }
      channel.allow_from = list
      channel.dm_policy = cc.telegram_dm_policy
    } else if (f.channel_type === 'discord') {
      channel.token = cc.discord_token
      channel.guild_id = cc.discord_guild
    } else if (f.channel_type === 'slack') {
      channel.bot_token = cc.slack_token
      channel.app_token = cc.slack_app_token
    } else if (f.channel_type === 'whatsapp') {
      channel.allow_from = cc.whatsapp_allow ? cc.whatsapp_allow.split(',').map(s => s.trim()).filter(Boolean) : []
    } else if (f.channel_type === 'webhook') {
      channel.url = cc.webhook_url
      channel.secret = cc.webhook_secret
    }
  }

  const envMap = {}
  for (const e of f.env_vars) {
    if (e.key.trim() && e.value.trim()) envMap[e.key.trim()] = e.value.trim()
  }

  return {
    provider: f.provider || undefined,
    model: fullModel || undefined,
    fallback_models: fallbacks.length > 0 ? fallbacks : undefined,
    custom_api_base: f.custom_api_base.trim() || undefined,
    system_prompt: f.system_prompt.trim() || undefined,
    persona_name: f.persona_name.trim() || undefined,
    language: f.language || undefined,
    channel: channel,
    env_vars: Object.keys(envMap).length > 0 ? envMap : undefined,
    resources: { cpu_limit: f.cpu_limit.trim() || undefined, memory_limit: f.memory_limit.trim() || undefined },
    image: f.image.trim() || undefined,
    description: f.description.trim() || undefined,
  }
}

async function save(redeploy) {
  error.value = ''
  try {
    const cfg = buildConfig()
    await apiPut(`/api/agents/${props.agent.id}/config`, cfg)
    if (redeploy) {
      await apiPost(`/api/agents/${props.agent.id}/redeploy`, {})
      toast('Config saved, agent redeploying...')
    } else {
      toast('Config saved')
    }
    emit('saved')
    emit('close')
  } catch (e) {
    error.value = 'Failed: ' + e.message
  }
}

async function loadIMKeyStatus() {
  if (!props.agent?.id) return
  imKeyStatusLoading.value = true
  imKeyStatusError.value = ''
  try {
    const res = await api(`/api/agents/${props.agent.id}/im-key`)
    imHasKey.value = !!res?.has_key
  } catch (e) {
    imKeyStatusError.value = e.message || 'failed to load key status'
  } finally {
    imKeyStatusLoading.value = false
  }
}

async function rotateIMKey() {
  if (!props.agent?.id) return
  rotatingIMKey.value = true
  imKeyStatusError.value = ''
  try {
    const res = await apiPost(`/api/agents/${props.agent.id}/im-key/rotate`, {})
    rotatedIMKey.value = res?.im_key || ''
    imHasKey.value = !!rotatedIMKey.value
    toast('IM key rotated')
  } catch (e) {
    imKeyStatusError.value = e.message || 'failed to rotate key'
  } finally {
    rotatingIMKey.value = false
  }
}

async function copyRotatedIMKey() {
  if (!rotatedIMKey.value) return
  try {
    await navigator.clipboard.writeText(rotatedIMKey.value)
    toast('IM key copied')
  } catch {
    imKeyStatusError.value = 'clipboard copy failed'
  }
}

function clearDebugPoll() {
  if (debugPollTimer) {
    clearTimeout(debugPollTimer)
    debugPollTimer = null
  }
}

async function runDebugChecks(checks) {
  if (!props.agent?.id || !checks?.length) return
  clearDebugPoll()
  debugRunning.value = true
  debugError.value = ''
  debugOutput.value = 'Starting debug task...'
  try {
    const res = await apiPost(`/api/agents/${props.agent.id}/debug`, { checks })
    debugTaskId.value = res.task_id
    debugOutput.value = `Task ${res.task_id} queued...\n`
    pollDebugTask()
  } catch (e) {
    debugRunning.value = false
    debugError.value = 'Failed to start debug task: ' + e.message
  }
}

async function pollDebugTask() {
  if (!debugTaskId.value) return
  try {
    const task = await api(`/api/tasks/${debugTaskId.value}`)
    const status = task?.status || 'unknown'
    debugOutput.value = formatDebugTask(task)
    if (status === 'done' || status === 'failed') {
      debugRunning.value = false
      return
    }
    debugPollTimer = setTimeout(pollDebugTask, 1500)
  } catch (e) {
    debugRunning.value = false
    debugError.value = 'Failed to poll task: ' + e.message
  }
}

function formatDebugTask(task) {
  const lines = []
  lines.push(`Task: ${task?.id || '-'}`)
  lines.push(`Status: ${task?.status || '-'}`)
  if (task?.action) lines.push(`Action: ${task.action}`)
  lines.push('')
  const res = task?.result || {}
  const results = Array.isArray(res.results) ? res.results : []
  for (const r of results) {
    lines.push(`== ${r.check || 'check'} (${r.status || 'unknown'}) ==`)
    if (r.error) lines.push(`Error: ${r.error}`)
    if (r.output) lines.push(String(r.output))
    lines.push('')
  }
  if (!results.length && task?.result) {
    lines.push(JSON.stringify(task.result, null, 2))
  }
  return lines.join('\n')
}
</script>
