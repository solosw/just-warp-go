<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { CreateSSHTerminal, GetSSHConfigs, SaveSSHConfig, RemoveSSHConfig } from '../../wailsjs/go/main/App'

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'connected', tabId: string, title: string): void
}>()

const savedConfigs = ref<any[]>([])
const showSaved = ref(false)
const connecting = ref(false)
const error = ref('')

const form = ref({
  name: '',
  host: '',
  port: 22,
  user: 'root',
  password: '',
  keyPath: '',
  save: true,
})

async function loadConfigs() {
  try { savedConfigs.value = await GetSSHConfigs() || [] } catch {}
}

function selectConfig(cfg: any) {
  form.value = { ...cfg, password: '', keyPath: cfg.keyPath || '', save: true }
  showSaved.value = false
}

async function removeConfig(name: string) {
  await RemoveSSHConfig(name)
  await loadConfigs()
}

async function connect() {
  error.value = ''
  if (!form.value.host) { error.value = '请输入主机地址'; return }

  connecting.value = true
  try {
    if (form.value.save) {
      const toSave = { ...form.value }
      if (!toSave.name) toSave.name = toSave.host
      await SaveSSHConfig(toSave)
    }
    const title = `${form.value.user}@${form.value.host}`
    const id = await CreateSSHTerminal(form.value)
    emit('connected', id, title)
  } catch (e: any) {
    error.value = '连接失败: ' + (e?.message || e)
  }
  connecting.value = false
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
}
onMounted(() => {
  document.addEventListener('keydown', onKeydown)
  loadConfigs()
})
onUnmounted(() => document.removeEventListener('keydown', onKeydown))
</script>

<template>
  <div class="ssh-overlay" @click="emit('close')">
    <div class="ssh-dialog" @click.stop>
      <div class="dialog-header">
        <span>SSH 连接</span>
        <button class="btn-close" @click="emit('close')">&times;</button>
      </div>

      <!-- Saved configs -->
      <div v-if="savedConfigs.length > 0" class="saved-section">
        <button class="btn-toggle" @click="showSaved = !showSaved">
          {{ showSaved ? '▼ 收起' : '▶ 已保存的连接 (' + savedConfigs.length + ')' }}
        </button>
        <div v-if="showSaved" class="saved-list">
          <div v-for="cfg in savedConfigs" :key="cfg.name" class="saved-item" @click="selectConfig(cfg)">
            <span class="saved-name">{{ cfg.name || cfg.host }}</span>
            <span class="saved-addr">{{ cfg.user }}@{{ cfg.host }}:{{ cfg.port }}</span>
            <button class="btn-remove-saved" @click.stop="removeConfig(cfg.name)">&times;</button>
          </div>
        </div>
      </div>

      <div class="dialog-body">
        <div class="form-row">
          <label>连接名称</label>
          <input v-model="form.name" placeholder="可选，如: 生产服务器" />
        </div>
        <div class="form-row">
          <label>主机地址</label>
          <input v-model="form.host" placeholder="192.168.1.100" />
          <input v-model.number="form.port" class="input-port" placeholder="22" type="number" />
        </div>
        <div class="form-row">
          <label>用户名</label>
          <input v-model="form.user" placeholder="root" />
        </div>
        <div class="form-row">
          <label>密码</label>
          <input v-model="form.password" type="password" placeholder="密钥或密码二选一" />
        </div>
        <div class="form-row">
          <label>密钥文件</label>
          <input v-model="form.keyPath" placeholder="~/.ssh/id_rsa（可选）" />
        </div>
        <div class="form-row checkbox-row">
          <label><input v-model="form.save" type="checkbox" /> 保存此连接</label>
        </div>

        <div v-if="error" class="error-msg">{{ error }}</div>
      </div>

      <div class="dialog-footer">
        <button class="btn-cancel" @click="emit('close')">取消</button>
        <button class="btn-connect" :disabled="connecting" @click="connect">
          {{ connecting ? '连接中...' : '连接' }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.ssh-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
}
.ssh-dialog {
  background: #1e1e20;
  border: 1px solid #444;
  border-radius: 8px;
  width: 420px;
  display: flex;
  flex-direction: column;
}
.dialog-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid #333;
  font-size: 14px;
  color: #ddd;
  font-weight: 600;
}
.btn-close {
  background: none;
  border: none;
  color: #888;
  font-size: 18px;
  cursor: pointer;
}
.btn-close:hover { color: #fff; }

.saved-section { border-bottom: 1px solid #333; }
.btn-toggle {
  width: 100%;
  background: none;
  border: none;
  color: #4a9eff;
  cursor: pointer;
  padding: 8px 16px;
  text-align: left;
  font-size: 12px;
}
.btn-toggle:hover { background: #2a2a2e; }
.saved-list { max-height: 150px; overflow-y: auto; }
.saved-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 16px;
  cursor: pointer;
  font-size: 12px;
  color: #ccc;
}
.saved-item:hover { background: #2a2a2e; }
.saved-name { font-weight: 600; flex-shrink: 0; }
.saved-addr { color: #888; flex: 1; overflow: hidden; text-overflow: ellipsis; }
.btn-remove-saved {
  background: none; border: none; color: #555; cursor: pointer; font-size: 14px;
}
.btn-remove-saved:hover { color: #f44336; }

.dialog-body { padding: 16px; display: flex; flex-direction: column; gap: 12px; }
.form-row { display: flex; align-items: center; gap: 8px; }
.form-row label { width: 70px; font-size: 12px; color: #888; flex-shrink: 0; }
.form-row input[type="text"],
.form-row input[type="password"],
.form-row input[type="number"] {
  flex: 1;
  background: #0d1117;
  border: 1px solid #30363d;
  color: #c9d1d9;
  padding: 6px 10px;
  border-radius: 4px;
  font-size: 13px;
  outline: none;
}
.form-row input:focus { border-color: #58a6ff; }
.input-port { max-width: 70px; flex: 0 !important; }
.checkbox-row { justify-content: flex-start; }
.checkbox-row label { width: auto; display: flex; align-items: center; gap: 6px; cursor: pointer; }
.checkbox-row input[type="checkbox"] { flex: 0; margin: 0; }
.error-msg { color: #f85149; font-size: 12px; padding: 6px 0; }

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 16px;
  border-top: 1px solid #333;
}
.btn-cancel {
  background: #21262d; border: 1px solid #30363d; color: #c9d1d9;
  padding: 6px 16px; border-radius: 4px; cursor: pointer; font-size: 13px;
}
.btn-cancel:hover { background: #30363d; }
.btn-connect {
  background: #238636; border: 1px solid #2ea043; color: #fff;
  padding: 6px 20px; border-radius: 4px; cursor: pointer; font-size: 13px; font-weight: 600;
}
.btn-connect:hover { background: #2ea043; }
.btn-connect:disabled { opacity: 0.5; cursor: default; }
</style>
