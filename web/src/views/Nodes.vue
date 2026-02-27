<template>
  <div>
    <div class="flex justify-end mb-4">
      <button @click="showAddNode = true" class="px-4 py-2 bg-btn-green border border-btn-greenHover rounded-md text-text text-sm hover:bg-btn-greenHover transition">➕ Add Node</button>
    </div>

    <p v-if="nodes.length === 0" class="text-text-muted text-center py-10">No nodes yet. Add one or wait for an agent to register.</p>

    <div v-for="n in nodes" :key="n.id" class="bg-bg-card border border-border rounded-lg mb-4 overflow-hidden">
      <div class="px-5 py-4 flex justify-between items-center">
        <div>
          <div class="text-base font-semibold">{{ n.name }} <span :class="'badge badge-'+n.status" class="ml-2">{{ n.status }}</span></div>
          <div class="text-sm text-text-muted mt-1">{{ n.address }} · Last seen: {{ n.last_seen ? new Date(n.last_seen).toLocaleString() : '-' }}</div>
          <div class="mt-1 flex gap-1" v-if="n.labels">
            <span v-for="(v,k) in n.labels" :key="k" class="badge badge-stopped">{{ k }}={{ v }}</span>
          </div>
        </div>
        <div class="flex gap-2 items-center">
          <span v-if="n.resources && n.resources.cpu_cores" class="text-xs text-text-muted">
            {{ n.resources.cpu_cores }}C · {{ (n.resources.memory_mb/1024).toFixed(1) }}G · CPU {{ n.resources.cpu_usage.toFixed(0) }}% · Mem {{ n.resources.memory_usage.toFixed(0) }}%
          </span>
          <button @click="openDeploy(n)" class="px-3 py-1.5 bg-btn-green border border-btn-greenHover rounded-md text-xs text-text hover:bg-btn-greenHover transition">🚀 Deploy</button>
          <button @click="removeNode(n)" class="px-3 py-1.5 bg-btn-red border border-accent-red rounded-md text-xs text-text hover:bg-btn-redHover transition">🗑️</button>
        </div>
      </div>
      <div v-if="(nodeAgents[n.id]||[]).length > 0" class="border-t border-border-light">
        <table class="w-full text-sm">
          <tr><th class="text-left px-4 py-2 bg-bg-hover text-text-muted text-xs uppercase">Agent</th><th class="text-left px-4 py-2 bg-bg-hover text-text-muted text-xs uppercase">Type</th><th class="text-left px-4 py-2 bg-bg-hover text-text-muted text-xs uppercase">Role</th><th class="text-left px-4 py-2 bg-bg-hover text-text-muted text-xs uppercase">Status</th><th class="text-left px-4 py-2 bg-bg-hover text-text-muted text-xs uppercase">Container</th><th class="text-left px-4 py-2 bg-bg-hover text-text-muted text-xs uppercase">Actions</th></tr>
          <tr v-for="a in nodeAgents[n.id]" :key="a.id" class="hover:bg-bg-hover">
            <td class="px-4 py-2 border-t border-border-light">{{ a.name }}</td>
            <td class="px-4 py-2 border-t border-border-light"><span :class="'agent-type agent-type-'+a.agent_type">{{ a.agent_type }}</span></td>
            <td class="px-4 py-2 border-t border-border-light">{{ a.role || '-' }}</td>
            <td class="px-4 py-2 border-t border-border-light"><span :class="'badge badge-'+a.status">{{ a.status }}</span></td>
            <td class="px-4 py-2 border-t border-border-light font-mono text-xs">{{ a.container_id ? a.container_id.substring(0,12) : '-' }}</td>
            <td class="px-4 py-2 border-t border-border-light flex gap-1">
              <button :disabled="a.status === 'deleting'" @click="openEditAgent(a)" class="px-2 py-0.5 bg-btn border border-border rounded text-xs hover:bg-btn-hover transition" :class="a.status === 'deleting' ? 'opacity-50 cursor-not-allowed' : ''">⚙️ Config</button>
              <button :disabled="a.status === 'deleting'" @click="restartAgent(a.id)" class="px-2 py-0.5 bg-accent-blue border border-accent-blue rounded text-xs text-white hover:opacity-90 transition" :class="a.status === 'deleting' ? 'opacity-50 cursor-not-allowed' : ''">🔄 Restart</button>
              <button :disabled="a.status === 'deleting'" @click="removeAgent(a.id)" class="px-2 py-0.5 bg-btn-red border border-accent-red rounded text-xs hover:bg-btn-redHover transition" :class="a.status === 'deleting' ? 'opacity-50 cursor-not-allowed' : ''">🗑️</button>
            </td>
          </tr>
        </table>
      </div>
      <div v-else class="border-t border-border-light px-5 py-3 text-sm text-text-dim">No agents deployed</div>
    </div>

    <!-- Edit Agent Config Modal -->
    <EditAgentModal :show="showEditAgent" :agent="editAgentData" @close="showEditAgent=false" @saved="load" />

    <!-- Add Node Modal -->
    <Modal :show="showAddNode" @close="closeAddNode" :width="560">
      <h3 class="text-lg font-semibold mb-4">🖥️ Add Node</h3>
      <p class="text-text-muted text-sm mb-3">Generate a registration token, then run the install command on the target machine.</p>
      <label class="block text-sm text-text-muted mb-1">Token Label (optional)</label>
      <input v-model="tokenLabel" type="text" placeholder="e.g. aussie-node" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 focus:outline-none focus:border-accent-blue">
      <div v-if="installCmd" class="mt-4">
        <label class="block text-sm text-text-muted mb-1">One-Click Install Command</label>
        <div @click="copyCmd" class="bg-bg-input border border-border rounded-md p-3 font-mono text-xs text-accent-green break-all cursor-pointer hover:border-accent-green transition">{{ installCmd }}</div>
        <div class="text-xs text-text-muted mt-1">📋 Click to copy · Run this on the target server as root</div>
      </div>
      <div class="flex justify-end gap-2 mt-5">
        <button @click="closeAddNode" class="px-4 py-2 border border-border rounded-md bg-btn text-text text-sm hover:bg-btn-hover transition">Close</button>
        <button v-if="!installCmd" @click="generateToken" class="px-4 py-2 bg-btn-green border border-btn-greenHover rounded-md text-text text-sm hover:bg-btn-greenHover transition">Generate Token</button>
      </div>
    </Modal>

    <!-- Deploy Modal -->
    <Modal :show="showDeploy" @close="showDeploy=false" :width="720">
      <h3 class="text-lg font-semibold mb-4">🚀 Deploy Agent to Node</h3>
      <div class="text-accent-blue text-sm mb-3">📍 Node: {{ deployNodeName }}</div>

      <!-- Step Tabs -->
      <div class="flex border-b border-border mb-4">
        <button v-for="(step, idx) in deploySteps" :key="idx"
          @click="deployStep = idx"
          :class="['px-4 py-2 text-sm transition', deployStep === idx ? 'border-b-2 border-accent-blue text-accent-blue' : 'text-text-muted hover:text-text']">
          {{ step }}
        </button>
      </div>

      <!-- Step 0: Basic Info -->
      <div v-show="deployStep === 0">
        <div class="grid grid-cols-2 gap-3 mb-3">
          <div>
            <label class="block text-sm text-text-muted mb-1">Agent Name *</label>
            <input v-model="deployForm.name" type="text" placeholder="e.g. my-assistant" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
          </div>
          <div>
            <label class="block text-sm text-text-muted mb-1">Agent Type</label>
            <select v-model="deployForm.agent_type" @change="onAgentTypeChange" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
              <option value="openclaw">OpenClaw</option><option value="zeroclaw">ZeroClaw</option><option value="nanobot">Nanobot</option>
            </select>
          </div>
        </div>
        <div class="grid grid-cols-2 gap-3 mb-3">
          <div>
            <label class="block text-sm text-text-muted mb-1">Role (optional)</label>
            <input v-model="deployForm.role" type="text" placeholder="e.g. developer, researcher" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
          </div>
          <div>
            <label class="block text-sm text-text-muted mb-1">Docker Image</label>
            <input v-model="deployForm.image" type="text" placeholder="auto (based on agent type)" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
          </div>
        </div>
        <label class="block text-sm text-text-muted mb-1">Description (optional)</label>
        <input v-model="deployForm.description" type="text" placeholder="What does this agent do?" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 focus:outline-none focus:border-accent-blue">
      </div>

      <!-- Step 1: Model & Provider -->
      <div v-show="deployStep === 1">
        <div class="bg-bg-card border border-border rounded-lg p-4 mb-3">
          <h4 class="text-sm font-semibold mb-3">🧠 Model Configuration</h4>
          <div class="grid grid-cols-2 gap-3 mb-3">
            <div>
              <label class="block text-sm text-text-muted mb-1">Primary Provider *</label>
              <select v-model="deployForm.provider" @change="onProviderChange" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
                <option value="">Select provider...</option>
                <option value="anthropic">Anthropic</option>
                <option value="openai">OpenAI</option>
                <option value="openai-codex">OpenAI Code (Codex)</option>
                <option value="opencode">OpenCode Zen</option>
                <option value="openrouter">OpenRouter</option>
                <option value="google">Google (Gemini)</option>
                <option value="zai">Z.AI (GLM)</option>
                <option value="groq">Groq</option>
                <option value="mistral">Mistral</option>
                <option value="xai">xAI</option>
                <option value="moonshot">Moonshot (Kimi)</option>
                <option value="ollama">Ollama (local)</option>
                <option value="custom">Custom endpoint</option>
              </select>
            </div>
            <div>
              <label class="block text-sm text-text-muted mb-1">Model *</label>
              <input v-model="deployForm.model" type="text" :placeholder="modelPlaceholder" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm font-mono focus:outline-none focus:border-accent-blue">
              <div class="text-xs text-text-dim mt-1">{{ modelHint }}</div>
            </div>
          </div>
          <div v-if="deployForm.provider === 'custom'" class="mb-3">
            <label class="block text-sm text-text-muted mb-1">Custom API Base URL</label>
            <input v-model="deployForm.custom_api_base" type="text" placeholder="https://api.example.com/v1" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-sm text-text-muted mb-1">Fallback Provider</label>
              <select v-model="deployForm.fallback_provider" @change="onFallbackProviderChange" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
                <option value="">None</option>
                <option value="anthropic">Anthropic</option>
                <option value="openai">OpenAI</option>
                <option value="openai-codex">OpenAI Code (Codex)</option>
                <option value="opencode">OpenCode Zen</option>
                <option value="openrouter">OpenRouter</option>
                <option value="google">Google (Gemini)</option>
                <option value="zai">Z.AI (GLM)</option>
                <option value="groq">Groq</option>
                <option value="mistral">Mistral</option>
                <option value="xai">xAI</option>
                <option value="moonshot">Moonshot (Kimi)</option>
                <option value="ollama">Ollama (local)</option>
              </select>
            </div>
            <div v-if="deployForm.fallback_provider">
              <label class="block text-sm text-text-muted mb-1">Fallback Model</label>
              <input v-model="deployForm.fallback_model" type="text" :placeholder="fallbackModelPlaceholder" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm font-mono focus:outline-none focus:border-accent-blue">
            </div>
          </div>
        </div>

        <div class="bg-bg-card border border-border rounded-lg p-4">
          <h4 class="text-sm font-semibold mb-3">🔑 API Keys</h4>
          <div class="text-xs text-text-dim mb-3">Enter the API keys for the selected providers. These are securely injected into the agent container.</div>
          <div v-for="pk in requiredApiKeys" :key="pk.key" class="flex gap-2 mb-2 items-center">
            <span class="flex-[2] px-2.5 py-1.5 bg-bg-tertiary border border-border rounded-md text-text text-xs font-mono">{{ pk.key }}</span>
            <input v-model="pk.value" :type="pk.show ? 'text' : 'password'" :placeholder="pk.placeholder" class="flex-[3] px-2.5 py-1.5 bg-bg-input border border-border rounded-md text-text text-xs font-mono focus:outline-none focus:border-accent-blue">
            <button @click="pk.show = !pk.show" class="px-2 py-1 border border-border rounded-md bg-btn text-xs hover:bg-btn-hover transition">{{ pk.show ? '🙈' : '👁️' }}</button>
          </div>
          <div v-if="requiredApiKeys.length === 0" class="text-xs text-text-dim italic">Select a provider above to see required keys</div>
        </div>
      </div>

      <!-- Step 2: Persona & Prompt -->
      <div v-show="deployStep === 2">
        <div class="bg-bg-card border border-border rounded-lg p-4 mb-3">
          <h4 class="text-sm font-semibold mb-3">🎭 Persona</h4>
          <div class="grid grid-cols-2 gap-3 mb-3">
            <div>
              <label class="block text-sm text-text-muted mb-1">Display Name</label>
              <input v-model="deployForm.persona_name" type="text" placeholder="e.g. Alice, DevBot" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
            </div>
            <div>
              <label class="block text-sm text-text-muted mb-1">Language</label>
              <select v-model="deployForm.language" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
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
          <textarea v-model="deployForm.system_prompt" placeholder="Define the agent's personality, role, and behavior guidelines..." class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 min-h-[100px] resize-y focus:outline-none focus:border-accent-blue"></textarea>
          <div class="flex gap-2">
            <button v-for="tmpl in promptTemplates" :key="tmpl.name" @click="deployForm.system_prompt = tmpl.prompt"
              class="px-3 py-1 text-xs border border-border rounded-full bg-btn hover:bg-btn-hover transition">
              {{ tmpl.name }}
            </button>
          </div>
        </div>
      </div>

      <!-- Step 3: Channel (messaging) -->
      <div v-show="deployStep === 3">
        <div class="bg-bg-card border border-border rounded-lg p-4 mb-3">
          <h4 class="text-sm font-semibold mb-3">💬 Channel Configuration</h4>
          <div class="text-xs text-text-dim mb-3">How will users communicate with this agent? Leave empty if agent runs headless (API only).</div>
          <div class="mb-3">
            <label class="block text-sm text-text-muted mb-1">Channel Type</label>
            <select v-model="deployForm.channel_type" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
              <option value="">None (headless / API)</option>
              <option value="telegram">Telegram</option>
              <option value="discord">Discord</option>
              <option value="slack">Slack</option>
              <option value="whatsapp">WhatsApp</option>
              <option value="webhook">Webhook</option>
            </select>
          </div>
          <!-- Telegram -->
          <div v-if="deployForm.channel_type === 'telegram'" class="space-y-3">
            <div>
              <label class="block text-sm text-text-muted mb-1">Bot Token *</label>
              <input v-model="deployForm.channel_config.telegram_token" :type="deployForm.channel_config._showToken ? 'text' : 'password'" placeholder="123456:ABC-DEF..." class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm font-mono focus:outline-none focus:border-accent-blue">
              <div class="text-xs text-text-dim mt-1">Get from <a href="https://t.me/BotFather" target="_blank" class="text-accent-blue hover:underline">@BotFather</a></div>
            </div>
            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="block text-sm text-text-muted mb-1">DM Policy</label>
                <select v-model="deployForm.channel_config.telegram_dm_policy" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
                  <option value="open">Open (anyone)</option>
                  <option value="allowlist">Allowlist only</option>
                  <option value="pairing">Pairing (approve each)</option>
                </select>
              </div>
              <div v-if="deployForm.channel_config.telegram_dm_policy === 'allowlist'">
                <label class="block text-sm text-text-muted mb-1">Allowed Users</label>
                <input v-model="deployForm.channel_config.telegram_allow" type="text" placeholder="@user1, @user2" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
                <div class="flex gap-2 mt-1">
                  <button v-if="fleetAgentNames.length > 0" @click="addFleetAgentsToAllow" type="button" class="text-xs text-accent-blue hover:underline">+ Add fleet agents ({{ fleetAgentNames.length }})</button>
                </div>
              </div>
            </div>
          </div>
          <!-- Discord -->
          <div v-if="deployForm.channel_type === 'discord'" class="space-y-3">
            <div>
              <label class="block text-sm text-text-muted mb-1">Bot Token *</label>
              <input v-model="deployForm.channel_config.discord_token" type="password" placeholder="Discord bot token" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm font-mono focus:outline-none focus:border-accent-blue">
            </div>
            <div>
              <label class="block text-sm text-text-muted mb-1">Guild ID (Server)</label>
              <input v-model="deployForm.channel_config.discord_guild" type="text" placeholder="optional" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
            </div>
          </div>
          <!-- Slack -->
          <div v-if="deployForm.channel_type === 'slack'" class="space-y-3">
            <div>
              <label class="block text-sm text-text-muted mb-1">Bot Token *</label>
              <input v-model="deployForm.channel_config.slack_token" type="password" placeholder="xoxb-..." class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm font-mono focus:outline-none focus:border-accent-blue">
            </div>
            <div>
              <label class="block text-sm text-text-muted mb-1">App Token</label>
              <input v-model="deployForm.channel_config.slack_app_token" type="password" placeholder="xapp-..." class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm font-mono focus:outline-none focus:border-accent-blue">
            </div>
          </div>
          <!-- WhatsApp -->
          <div v-if="deployForm.channel_type === 'whatsapp'" class="space-y-3">
            <div class="text-xs text-text-dim bg-bg-tertiary p-3 rounded-md">
              ⚠️ WhatsApp requires QR code pairing. The agent container will expose a pairing endpoint after deployment. You'll scan the QR code from your phone.
            </div>
            <div>
              <label class="block text-sm text-text-muted mb-1">Allowed Numbers</label>
              <input v-model="deployForm.channel_config.whatsapp_allow" type="text" placeholder="+8613800138000, +1555..." class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
            </div>
          </div>
          <!-- Webhook -->
          <div v-if="deployForm.channel_type === 'webhook'" class="space-y-3">
            <div>
              <label class="block text-sm text-text-muted mb-1">Webhook URL</label>
              <input v-model="deployForm.channel_config.webhook_url" type="text" placeholder="https://..." class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
            </div>
            <div>
              <label class="block text-sm text-text-muted mb-1">Webhook Secret</label>
              <input v-model="deployForm.channel_config.webhook_secret" type="password" placeholder="optional" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
            </div>
          </div>
        </div>
      </div>

      <!-- Step 4: Resources & Extra Env -->
      <div v-show="deployStep === 4">
        <div class="bg-bg-card border border-border rounded-lg p-4 mb-3">
          <h4 class="text-sm font-semibold mb-3">⚙️ Resource Limits</h4>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-sm text-text-muted mb-1">CPU Limit</label>
              <input v-model="deployForm.cpu_limit" type="text" placeholder="e.g. 2 (cores)" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
            </div>
            <div>
              <label class="block text-sm text-text-muted mb-1">Memory Limit</label>
              <input v-model="deployForm.memory_limit" type="text" placeholder="e.g. 4Gi" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
            </div>
          </div>
        </div>

        <div class="bg-bg-card border border-border rounded-lg p-4">
          <div class="flex justify-between items-center mb-2">
            <h4 class="text-sm font-semibold">🔧 Extra Environment Variables</h4>
            <button @click="addEnvVar" class="text-xs text-accent-blue hover:underline">+ Add</button>
          </div>
          <div class="text-xs text-text-dim mb-2">Additional env vars beyond the auto-configured API keys above.</div>
          <div v-for="(env, idx) in deployForm.env_vars" :key="idx" class="flex gap-2 mb-2">
            <input v-model="env.key" type="text" placeholder="KEY_NAME" class="flex-[2] px-2.5 py-1.5 bg-bg-input border border-border rounded-md text-text text-xs font-mono focus:outline-none focus:border-accent-blue">
            <input v-model="env.value" :type="env.show ? 'text' : 'password'" placeholder="value" class="flex-[3] px-2.5 py-1.5 bg-bg-input border border-border rounded-md text-text text-xs font-mono focus:outline-none focus:border-accent-blue">
            <button @click="env.show = !env.show" class="px-2 py-1 border border-border rounded-md bg-btn text-xs hover:bg-btn-hover transition">{{ env.show ? '🙈' : '👁️' }}</button>
            <button @click="deployForm.env_vars.splice(idx, 1)" class="px-2 py-1 bg-btn-red border border-accent-red rounded-md text-xs hover:bg-btn-redHover transition">✕</button>
          </div>
        </div>
      </div>

            <div class="flex justify-between items-center mt-5">
        <div>
          <button v-if="deployStep > 0" @click="deployStep--" class="px-4 py-2 border border-border rounded-md bg-btn text-text text-sm hover:bg-btn-hover transition">← Back</button>
        </div>
        <div class="flex gap-2">
          <button @click="showDeploy=false" class="px-4 py-2 border border-border rounded-md bg-btn text-text text-sm hover:bg-btn-hover transition">Cancel</button>
          <button v-if="deployStep < deploySteps.length - 1" @click="deployStep++" class="px-4 py-2 bg-accent-blue border border-accent-blue rounded-md text-white text-sm hover:opacity-90 transition">Next →</button>
          <button v-else @click="doDeploy" class="px-4 py-2 bg-btn-green border border-btn-greenHover rounded-md text-text text-sm hover:bg-btn-greenHover transition">🚀 Deploy</button>
        </div>
      </div>
    </Modal>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, inject } from 'vue'
import { api, apiPost, apiDelete, confirmAndDelete } from '../api'
import { useLoader } from '../composables/useLoader'
import Modal from '../components/Modal.vue'
import EditAgentModal from '../components/EditAgentModal.vue'

const toast = inject('toast')
const nodes = ref([])
const nodeAgents = ref({})
const fleetAgentNames = computed(() => {
  const names = []
  for (const agents of Object.values(nodeAgents.value)) {
    for (const a of (agents || [])) {
      if (a.name) names.push(a.name)
    }
  }
  return names
})
const showEditAgent = ref(false)
const editAgentData = ref({})
const showAddNode = ref(false)
const tokenLabel = ref('')
const installCmd = ref('')
const showDeploy = ref(false)
const deployNodeId = ref('')
const deployNodeName = ref('')
const deployStep = ref(0)
const deploySteps = ['Basic', 'Model & Keys', 'Persona', 'Channel', 'Resources']

const defaultDeployForm = () => ({
  name: '', agent_type: 'openclaw', role: '', image: '', description: '',
  provider: 'anthropic', model: 'claude-opus-4-6', fallback_provider: 'openrouter', fallback_model: 'anthropic/claude-sonnet-4-5',
  custom_api_base: '',
  system_prompt: '', persona_name: '', language: 'en',
  channel_type: '', channel_config: { telegram_token: '', telegram_allow: '', telegram_dm_policy: 'open', discord_token: '', discord_guild: '', slack_token: '', slack_app_token: '', whatsapp_allow: '', webhook_url: '', webhook_secret: '', _showToken: false },
  env_vars: [], cpu_limit: '', memory_limit: '',
})
const deployForm = ref(defaultDeployForm())

// Provider default models and hints (from OpenClaw docs)
const providerDefaults = {
  anthropic:     { model: 'claude-opus-4-6',     placeholder: 'claude-opus-4-6',           hint: 'e.g. claude-opus-4-6, claude-sonnet-4-5' },
  openai:        { model: 'gpt-5.1-codex',       placeholder: 'gpt-5.1-codex',             hint: 'e.g. gpt-5.1-codex, gpt-4o, o3' },
  'openai-codex':{ model: 'gpt-5.3-codex',       placeholder: 'gpt-5.3-codex',             hint: 'e.g. gpt-5.3-codex (OAuth via ChatGPT)' },
  opencode:      { model: 'claude-opus-4-6',      placeholder: 'claude-opus-4-6',           hint: 'e.g. claude-opus-4-6 (via OpenCode Zen proxy)' },
  openrouter:    { model: 'anthropic/claude-sonnet-4-5', placeholder: 'anthropic/claude-sonnet-4-5', hint: 'e.g. anthropic/claude-sonnet-4-5, deepseek/deepseek-chat' },
  google:        { model: 'gemini-3-pro-preview', placeholder: 'gemini-3-pro-preview',      hint: 'e.g. gemini-3-pro-preview, gemini-2.5-flash' },
  zai:           { model: 'glm-4.7',              placeholder: 'glm-4.7',                   hint: 'e.g. glm-4.7, glm-4.6' },
  groq:          { model: 'llama-3.3-70b-versatile', placeholder: 'llama-3.3-70b-versatile', hint: 'e.g. llama-3.3-70b-versatile, mixtral-8x7b-32768' },
  mistral:       { model: 'mistral-large-latest',  placeholder: 'mistral-large-latest',      hint: 'e.g. mistral-large-latest, codestral-latest' },
  xai:           { model: 'grok-3',               placeholder: 'grok-3',                    hint: 'e.g. grok-3, grok-3-mini' },
  moonshot:      { model: 'kimi-k2.5',            placeholder: 'kimi-k2.5',                 hint: 'e.g. kimi-k2.5, kimi-k2-thinking' },
  ollama:        { model: '',                      placeholder: 'llama3:8b',                 hint: 'Enter your local Ollama model name' },
  custom:        { model: '',                      placeholder: 'model-name',                hint: 'Enter the model ID for your custom endpoint' },
}

const modelPlaceholder = computed(() => (providerDefaults[deployForm.value.provider]?.placeholder || 'model-name'))
const modelHint = computed(() => (providerDefaults[deployForm.value.provider]?.hint || 'Enter the model identifier'))
const fallbackModelPlaceholder = computed(() => (providerDefaults[deployForm.value.fallback_provider]?.placeholder || 'model-name'))

// API key requirements per provider
const providerKeyMap = {
  anthropic: [{ key: 'ANTHROPIC_API_KEY', placeholder: 'sk-ant-...' }],
  openai: [{ key: 'OPENAI_API_KEY', placeholder: 'sk-...' }],
  openrouter: [{ key: 'OPENROUTER_API_KEY', placeholder: 'sk-or-...' }],
  google: [{ key: 'GOOGLE_API_KEY', placeholder: 'AIza...' }],
  groq: [{ key: 'GROQ_API_KEY', placeholder: 'gsk_...' }],
  ollama: [],
  custom: [{ key: 'API_KEY', placeholder: 'your API key' }],
}

const requiredApiKeys = computed(() => {
  const keys = []
  const seen = new Set()
  for (const p of [deployForm.value.provider, deployForm.value.fallback_provider]) {
    if (!p) continue
    for (const k of (providerKeyMap[p] || [])) {
      if (!seen.has(k.key)) {
        seen.add(k.key)
        keys.push({ ...k, value: '', show: false })
      }
    }
  }
  return keys
})

// Prompt templates
const promptTemplates = [
  { name: '🤖 General Assistant', prompt: 'You are a helpful, friendly AI assistant. Be concise and accurate. Ask clarifying questions when needed.' },
  { name: '💻 Developer', prompt: 'You are a senior software engineer. Write clean, well-documented code. Explain your reasoning. Prefer simple solutions over complex ones.' },
  { name: '📊 Analyst', prompt: 'You are a data analyst. Focus on accuracy, cite sources when possible, and present findings clearly with structured formatting.' },
  { name: '🎨 Creative', prompt: 'You are a creative writing assistant. Be imaginative, engaging, and adaptable to different styles and tones.' },
]

function onAgentTypeChange() {
  if (deployForm.value.agent_type === 'openclaw') {
    deployForm.value.provider = 'anthropic'
    deployForm.value.model = providerDefaults.anthropic.model
  } else if (deployForm.value.agent_type === 'zeroclaw') {
    deployForm.value.provider = 'openai'
    deployForm.value.model = providerDefaults.openai.model
  } else if (deployForm.value.agent_type === 'nanobot') {
    deployForm.value.provider = 'anthropic'
    deployForm.value.model = providerDefaults.anthropic.model
  }
}

function onProviderChange() {
  const d = providerDefaults[deployForm.value.provider]
  if (d) deployForm.value.model = d.model
}

function onFallbackProviderChange() {
  if (!deployForm.value.fallback_provider) {
    deployForm.value.fallback_model = ''
    return
  }
  const d = providerDefaults[deployForm.value.fallback_provider]
  if (d) deployForm.value.fallback_model = d.model
}

function addFleetAgentsToAllow() {
  const existing = deployForm.value.channel_config.telegram_allow
    ? deployForm.value.channel_config.telegram_allow.split(',').map(s => s.trim()).filter(Boolean)
    : []
  const existingSet = new Set(existing.map(s => s.toLowerCase()))
  for (const name of fleetAgentNames.value) {
    if (!existingSet.has(name.toLowerCase())) {
      existing.push(name)
    }
  }
  deployForm.value.channel_config.telegram_allow = existing.join(', ')
}

function addEnvVar() {
  deployForm.value.env_vars.push({ key: '', value: '', show: false })
}

const { load } = useLoader(async () => {
  nodes.value = await api('/api/nodes') || []
  const agentMap = {}
  await Promise.all(nodes.value.map(async n => {
    try { agentMap[n.id] = await api(`/api/nodes/${n.id}/agents`) || [] } catch { agentMap[n.id] = [] }
  }))
  nodeAgents.value = agentMap
})

onMounted(load)

function closeAddNode() { showAddNode.value = false; installCmd.value = ''; tokenLabel.value = ''; load() }

async function generateToken() {
  try {
    const data = await apiPost('/api/tokens', { name: tokenLabel.value.trim() || undefined })

    // Backward/forward compatibility: older/newer APIs may return either
    // - { install_command: "..." }
    // - or a token object { token: "..." }
    if (data && data.install_command) {
      installCmd.value = data.install_command
      return
    }

    const tok = data?.token || data?.Token
    if (!tok) {
      installCmd.value = 'Token generated, but response did not include install command or token.'
      return
    }

    // Minimal, copy-paste friendly instructions (no backend install endpoint yet)
    installCmd.value = `# 1) On the node server (aussie), set register token\n` +
      `sudo mkdir -p /data/clawfleet-node\n` +
      `sudo tee /data/clawfleet-node/config.yaml >/dev/null <<'EOF'\n` +
      `control_plane: https://control-plane.example.com\n` +
      `name: <node-name>\n` +
      `register_token: ${tok}\n` +
      `heartbeat_interval: 30\n` +
      `data_dir: /data/clawfleet-node\n` +
      `EOF\n\n` +
      `# 2) Copy the latest clawfleet-node binary to /data/clawfleet-node/clawfleet-node\n` +
      `# 3) Create/enable systemd service (see docs) and start it\n`
  } catch (e) {
    alert('Failed: ' + e.message)
  }
}

function copyCmd() {
  navigator.clipboard.writeText(installCmd.value)
  toast('Copied!')
}

async function removeNode(n) {
  await confirmAndDelete(`Delete node "${n.name}"?`, `/api/nodes/${n.id}`, null, 'Deleted', '#da3633', load)
}

function openEditAgent(agent) {
  editAgentData.value = agent
  showEditAgent.value = true
}

async function restartAgent(id) {
  if (!confirm('Restart this agent?')) return
  try {
    await apiPost(`/api/agents/${id}/restart`, {})
    toast('Agent restarting...')
    load()
  } catch (e) { alert('Restart failed: ' + e.message) }
}

async function removeAgent(id) {
  await confirmAndDelete('Remove this agent? (async cleanup)', `/api/agents/${id}`, null, 'Deleting...', '#da3633', () => {
    load()
    setTimeout(load, 1200)
    setTimeout(load, 2500)
    setTimeout(load, 4000)
  })
}

function openDeploy(n) {
  deployNodeId.value = n.id
  deployNodeName.value = n.name
  deployForm.value = defaultDeployForm()
  deployStep.value = 0
  showDeploy.value = true
}

async function doDeploy() {
  const f = deployForm.value
  if (!f.name.trim()) { alert('Agent name required'); deployStep.value = 0; return }
  if (!f.provider) { alert('Please select a model provider'); deployStep.value = 1; return }
  if (!f.model) { alert('Please select a model'); deployStep.value = 1; return }

  // Build env_vars map: API keys + extra env vars
  const envMap = {}
  for (const k of requiredApiKeys.value) {
    if (k.value.trim()) envMap[k.key] = k.value.trim()
  }
  for (const e of f.env_vars) {
    if (e.key.trim() && e.value.trim()) envMap[e.key.trim()] = e.value.trim()
  }

  // Build channel config
  let channel = undefined
  if (f.channel_type) {
    channel = { type: f.channel_type }
    const cc = f.channel_config
    if (f.channel_type === 'telegram') {
      channel.token = cc.telegram_token
      channel.allowFrom = cc.telegram_allow ? cc.telegram_allow.split(',').map(s => s.trim()) : []
      channel.dmPolicy = cc.telegram_dm_policy
    } else if (f.channel_type === 'discord') {
      channel.token = cc.discord_token
      channel.guildId = cc.discord_guild
    } else if (f.channel_type === 'slack') {
      channel.botToken = cc.slack_token
      channel.appToken = cc.slack_app_token
    } else if (f.channel_type === 'whatsapp') {
      channel.allowFrom = cc.whatsapp_allow ? cc.whatsapp_allow.split(',').map(s => s.trim()) : []
    } else if (f.channel_type === 'webhook') {
      channel.url = cc.webhook_url
      channel.secret = cc.webhook_secret
    }
  }

  // Build model config with provider prefix
  const providerPrefix = ['ollama', 'custom'].includes(f.provider) ? '' : f.provider + '/'
  const fullModel = f.model.startsWith(f.provider + '/') ? f.model : providerPrefix + f.model
  let fallbacks = []
  if (f.fallback_provider && f.fallback_model) {
    const fbPrefix = ['ollama', 'custom'].includes(f.fallback_provider) ? '' : f.fallback_provider + '/'
    const fbModel = f.fallback_model.startsWith(f.fallback_provider + '/') ? f.fallback_model : fbPrefix + f.fallback_model
    fallbacks.push(fbModel)
  }

  try {
    await apiPost(`/api/nodes/${deployNodeId.value}/deploy`, {
      name: f.name.trim(),
      agent_type: f.agent_type,
      role: f.role.trim() || undefined,
      image: f.image.trim() || undefined,
      description: f.description.trim() || undefined,
      config: {
        model: fullModel,
        fallback_models: fallbacks.length > 0 ? fallbacks : undefined,
        provider: f.provider,
        custom_api_base: f.custom_api_base.trim() || undefined,
        system_prompt: f.system_prompt.trim() || undefined,
        persona_name: f.persona_name.trim() || undefined,
        language: f.language || undefined,
        channel: channel,
        env_vars: Object.keys(envMap).length > 0 ? envMap : undefined,
        resources: {
          cpu_limit: f.cpu_limit.trim() || undefined,
          memory_limit: f.memory_limit.trim() || undefined,
        }
      }
    })
    showDeploy.value = false
    toast('Agent deployed!')
    load()
  } catch (e) { alert('Deploy failed: ' + e.message) }
}
</script>
