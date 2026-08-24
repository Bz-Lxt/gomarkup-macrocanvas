<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from "vue";
import { VueFlow } from "@vue-flow/core";
import { Background } from "@vue-flow/background";
import "@vue-flow/core/dist/style.css";
import "@vue-flow/core/dist/theme-default.css";
import { api, eventsWS } from "../api/client";
import { PALETTE, type MacroModel, type NodeModel } from "../types/macro";
import { useSession } from "../stores/session";

const session = useSession();
const macros = ref<MacroModel[]>([]);
const current = ref<MacroModel | null>(null);
const status = ref<any>(null);
const events = ref<any[]>([]);
const selected = ref<string>("");
const issues = ref<any[]>([]);
const lastRun = ref<any>(null);
const bench = ref<any>(null);
const confirm = ref<{ title: string; body: string; ok: () => void } | null>(null);
const captureOn = ref(false);
const wallPaused = ref(false);
const showSettings = ref(false);
const tab = ref<"wall" | "timing" | "device">("wall");
let ws: WebSocket | null = null;

const vfNodes = computed(() =>
  (current.value?.nodes || []).map((n) => ({
    id: n.id,
    type: "default",
    position: { x: n.x, y: n.y },
    data: { label: labelOf(n), raw: n },
    class: nodeClass(n.type),
    style: { borderRadius: "10px", border: "1px solid #1d2836", background: "#10151c", color: "#e8f0f7", fontFamily: "IBM Plex Mono", fontSize: "12px", padding: "8px 10px", minWidth: "140px" },
  }))
);
const vfEdges = computed(() =>
  (current.value?.edges || []).map((e) => ({
    id: e.id,
    source: e.from,
    target: e.to,
    label: e.port === "out" ? "" : e.port,
    animated: e.port === "body",
    style: { stroke: e.port === "true" ? "#9dff6b" : e.port === "false" ? "#ffb347" : "#5ee0ff" },
  }))
);

function labelOf(n: NodeModel) {
  const p = n.params || {};
  if (n.type.startsWith("key")) return `${n.type} ${(p.key || p.combo || "")}`;
  if (n.type === "wait.fixed") return `wait ${p.us || 0}µs`;
  if (n.type === "mouse.move.rel") return `move ${p.dx || 0},${p.dy || 0}`;
  if (n.type === "flow.loop") return `loop ×${p.count || 1}`;
  if (n.type === "flow.if") return `if ${p.cond || ""}`;
  if (n.type === "debug.marker") return `◆ ${p.label || n.id}`;
  return n.type;
}
function nodeClass(t: string) {
  if (t.startsWith("key")) return "ring-1 ring-phos/40";
  if (t.startsWith("mouse")) return "ring-1 ring-cyan/40";
  if (t.startsWith("wait")) return "ring-1 ring-amber/40";
  return "ring-1 ring-white/10";
}

function onConnect(c: any) {
  if (!current.value) return;
  const src = current.value.nodes.find((n) => n.id === c.source);
  const port = src?.type === "flow.if" ? "true" : src?.type === "flow.loop" ? "body" : "out";
  current.value.edges.push({ id: `e-${Date.now()}`, from: c.source, to: c.target, port });
}

function addNode(type: string) {
  if (!current.value) return;
  const id = `n-${Date.now()}`;
  const params: Record<string, unknown> = {};
  if (type.startsWith("key")) params.key = "A";
  if (type === "key.combo") params.combo = "LeftCtrl+A";
  if (type === "wait.fixed") params.us = 15000;
  if (type === "mouse.move.rel") { params.dx = 50; params.dy = 0; }
  if (type === "flow.loop") params.count = 3;
  if (type === "flow.if") params.cond = "loop_i>=2";
  current.value.nodes.push({ id, type, x: 240 + Math.random() * 80, y: 160 + Math.random() * 80, params });
}

const selectedNode = computed(() => current.value?.nodes.find((n) => n.id === selected.value) || null);

async function loadAll() {
  macros.value = await api.macros();
  status.value = await api.status();
  if (!current.value && macros.value.length) await openMacro(macros.value[0].id);
}

async function openMacro(id: string) {
  current.value = await api.getMacro(id);
  selected.value = "";
  await nextTick();
}

function onNodeDrag(e: any) {
  const m = current.value?.nodes.find((x) => x.id === e.node.id);
  if (m) {
    m.x = e.node.position.x;
    m.y = e.node.position.y;
  }
}

async function save() {
  if (!current.value) return;
  try {
    current.value = await api.saveMacro(current.value.id, current.value);
    session.flash("已热下发，无需重启");
    await loadAll();
  } catch (e: any) {
    session.flash(e.message, "err");
  }
}

async function validate() {
  if (!current.value) return;
  await save();
  const d = await api.validate(current.value.id);
  issues.value = [...(d.issues || []), ...(d.compile || []), ...(d.unpaired || [])];
  session.flash(d.ok ? `校验通过 · ${d.opcodes} opcodes` : "存在校验问题", d.ok ? "ok" : "err");
}

async function deploy() {
  if (!current.value) return;
  await save();
  try {
    current.value = await api.deploy(current.value.id);
    session.flash("已部署到触发器");
  } catch (e: any) {
    session.flash(e.message, "err");
  }
}

async function run() {
  if (!current.value) return;
  await save();
  try {
    lastRun.value = await api.run(current.value.id);
    tab.value = "timing";
    session.flash(`执行 ${lastRun.value.status} · 误差 p50 ${Math.round((lastRun.value.trace?.p50_ns || 0) / 1000)}µs`);
  } catch (e: any) {
    session.flash(e.message, "err");
  }
}

async function emergency() {
  await api.emergency();
  session.flash("紧急停止已触发", "err");
}

function askDelete() {
  if (!current.value) return;
  const id = current.value.id;
  confirm.value = {
    title: "删除宏",
    body: `确认删除「${current.value.name}」？此操作不可恢复。`,
    ok: async () => {
      confirm.value = null;
      await api.deleteMacro(id);
      current.value = null;
      await loadAll();
    },
  };
}

async function enableCapture() {
  confirm.value = {
    title: "授权全局捕获",
    body: "开启后将捕获本档位可见的全部按键流水。该能力在功能上等价于键盘记录器。默认识别字符会脱敏。请确认你拥有这台机器。",
    ok: async () => {
      confirm.value = null;
      await api.authorize();
      captureOn.value = true;
      connectWS();
      session.flash("捕获已授权");
    },
  };
}

function connectWS() {
  ws?.close();
  ws = eventsWS();
  ws.onmessage = (ev) => {
    if (wallPaused.value) return;
    const data = JSON.parse(ev.data);
    events.value = [...(data.events || []), ...events.value].slice(0, 400);
  };
}

async function runBench() {
  bench.value = await api.benchmark(current.value?.precision || "balanced");
  tab.value = "timing";
}

onMounted(async () => {
  try {
    await loadAll();
    const ev = await api.events();
    events.value = ev.events || [];
  } catch (e: any) {
    session.flash(e.message, "err");
  }
});
onUnmounted(() => ws?.close());

watch(() => current.value?.id, () => { issues.value = []; lastRun.value = null; });

const tierName = computed(() => status.value?.device?.active_name || "探测中");
const tier = computed(() => status.value?.device?.active_tier || "—");
</script>

<template>
  <div class="h-full flex flex-col scanline">
    <header class="flex flex-wrap items-center gap-3 px-4 py-3 border-b border-line bg-panel/80">
      <div>
        <p class="font-display text-phos text-[10px] tracking-[0.4em] uppercase">MacroCanvas</p>
        <h1 class="font-display text-lg leading-none">HID 宏编排内核</h1>
      </div>
      <div class="ml-2 px-3 py-1 rounded-full border border-phos/40 text-phos text-xs font-mono" data-testid="tier-badge">
        {{ tier }} · {{ tierName }}
      </div>
      <div class="text-[11px] text-mute font-mono hidden md:block">
        时序按 E1/E2/E3 分档承诺，不是无条件微秒保证
      </div>
      <div class="ml-auto flex flex-wrap gap-2">
        <button class="px-3 py-1.5 rounded-lg bg-phos text-ink text-sm font-bold" data-testid="btn-run" @click="run">执行</button>
        <button class="px-3 py-1.5 rounded-lg border border-line text-sm" @click="save">热下发</button>
        <button class="px-3 py-1.5 rounded-lg border border-line text-sm" @click="validate">校验</button>
        <button class="px-3 py-1.5 rounded-lg border border-line text-sm" @click="deploy">部署</button>
        <button class="px-3 py-1.5 rounded-lg border border-amber text-amber text-sm font-bold" data-testid="btn-estop" aria-label="紧急停止" @click="emergency">紧急停止</button>
        <button class="px-3 py-1.5 rounded-lg border border-line text-sm" @click="session.logout()">退出</button>
      </div>
    </header>

    <div class="flex-1 grid grid-cols-1 lg:grid-cols-[220px_1fr_340px] min-h-0">
      <aside class="border-r border-line p-3 overflow-auto hidden md:block">
        <p class="text-xs text-mute mb-2">宏列表</p>
        <button v-for="m in macros" :key="m.id" class="w-full text-left mb-1 px-2 py-2 rounded-lg font-mono text-xs"
          :class="current?.id === m.id ? 'bg-phos/15 text-phos' : 'hover:bg-white/5'"
          @click="openMacro(m.id)">{{ m.name }}</button>
        <p class="text-xs text-mute mt-5 mb-2">节点面板</p>
        <button v-for="p in PALETTE" :key="p.type" class="w-full text-left mb-1 px-2 py-1.5 rounded border border-line text-xs hover:border-phos/50"
          @click="addNode(p.type)">
          <span class="text-mute mr-1">{{ p.group }}</span>{{ p.label }}
        </button>
      </aside>

      <section class="relative min-h-[360px] bg-[#0b0f14]" data-testid="canvas">
        <VueFlow v-if="current" :nodes="vfNodes" :edges="vfEdges" fit-view-on-init
          @connect="onConnect" @node-drag-stop="onNodeDrag"
          @node-click="(e:any) => selected = e.node.id">
          <Background pattern-color="#1d2836" :gap="22" />
        </VueFlow>
        <div v-if="current" class="absolute top-3 left-3 font-display text-phos/80 text-sm pointer-events-none">
          {{ current.name }}
        </div>
      </section>

      <aside class="border-l border-line flex flex-col min-h-0">
        <div class="flex text-xs border-b border-line">
          <button class="flex-1 py-2" :class="tab==='wall' && 'text-phos border-b border-phos'" @click="tab='wall'">事件墙</button>
          <button class="flex-1 py-2" :class="tab==='timing' && 'text-cyan border-b border-cyan'" @click="tab='timing'">时序</button>
          <button class="flex-1 py-2" :class="tab==='device' && 'text-amber border-b border-amber'" @click="tab='device'">设备</button>
        </div>

        <div v-if="tab==='wall'" class="flex-1 flex flex-col min-h-0">
          <div class="p-2 flex gap-2 text-xs">
            <button v-if="!captureOn" class="px-2 py-1 border border-amber text-amber rounded" data-testid="btn-capture" @click="enableCapture">授权捕获</button>
            <button class="px-2 py-1 border border-line rounded" @click="wallPaused = !wallPaused">{{ wallPaused ? '继续' : '暂停' }}</button>
            <button class="px-2 py-1 border border-line rounded" @click="api.clearEvents().then(() => events = [])">清空</button>
          </div>
          <div class="flex-1 overflow-auto font-mono text-[11px] px-2" data-testid="event-wall">
            <div v-for="e in events" :key="e.seq" class="flex gap-2 border-b border-white/5 py-1">
              <span class="text-mute">{{ e.beijing_time }}</span>
              <span class="text-cyan">{{ e.source }}</span>
              <span :class="e.masked ? 'text-amber' : 'text-phos'">{{ e.name }}</span>
              <span class="text-mute">{{ e.value }}</span>
            </div>
            <p v-if="!events.length" class="text-mute py-8 text-center">等待硬件 / 回环事件…</p>
          </div>
        </div>

        <div v-else-if="tab==='timing'" class="p-3 overflow-auto text-xs font-mono space-y-3">
          <p class="text-mute">宏时序误差（后端 µs，不与前端传输延迟混算）</p>
          <div v-if="lastRun">
            <p>状态 {{ lastRun.status }} / {{ lastRun.reason || '—' }}</p>
            <p>p50 {{ Math.round((lastRun.trace?.p50_ns||0)/1000) }}µs · p99 {{ Math.round((lastRun.trace?.p99_ns||0)/1000) }}µs</p>
            <p>标记 {{ (lastRun.markers || []).join(', ') || '—' }}</p>
          </div>
          <button class="px-2 py-1 border border-cyan text-cyan rounded" @click="runBench">运行本机基准</button>
          <div v-if="bench">
            <p>档位 {{ bench.band }} · 策略 {{ bench.strategy }}</p>
            <p>p50 {{ Math.round(bench.p50_ns/1000) }}µs · p99 {{ Math.round(bench.p99_ns/1000) }}µs · max {{ Math.round(bench.max_ns/1000) }}µs</p>
          </div>
          <div v-if="selectedNode" class="pt-3 border-t border-line space-y-2">
            <p class="text-phos">节点 {{ selectedNode.type }}</p>
            <label v-for="k in Object.keys(selectedNode.params)" :key="k" class="block">
              <span class="text-mute">{{ k }} *</span>
              <input class="w-full bg-ink border border-line rounded px-2 py-1 mt-0.5" :value="String(selectedNode.params[k] ?? '')"
                @input="selectedNode.params[k] = ($event.target as HTMLInputElement).value.match(/^-?\d+$/) ? Number(($event.target as HTMLInputElement).value) : ($event.target as HTMLInputElement).value" />
            </label>
          </div>
          <ul v-if="issues.length" class="text-amber">
            <li v-for="(i, idx) in issues" :key="idx">{{ i.node_id || i.NodeID }} {{ i.message || i.Message }}</li>
          </ul>
        </div>

        <div v-else class="p-3 overflow-auto text-xs font-mono space-y-2">
          <p>模式 {{ status?.device?.mode }}</p>
          <p>来源由档位推出：{{ status?.device?.source_from_tier }}</p>
          <p>RT {{ status?.rt_available ? '可用' : '关闭' }} · {{ status?.rt_reason }}</p>
          <p>Sleep p99 {{ Math.round((status?.device?.sleep_p99_ns||0)/1e6) }}ms · 余量 {{ Math.round((status?.device?.margin_ns||0)/1e6) }}ms</p>
          <div v-for="p in status?.device?.probes || []" :key="p.tier" class="border border-line rounded p-2">
            <p class="text-phos">{{ p.tier }} {{ p.name }}</p>
            <p class="text-mute">{{ p.available ? '可用' : '不可用' }} · {{ p.reason }}</p>
          </div>
          <p class="text-mute">{{ status?.disclaimer }}</p>
          <button class="px-2 py-1 border border-line rounded" @click="showSettings = true">设置</button>
          <button class="px-2 py-1 border border-amber text-amber rounded" @click="askDelete">删除当前宏</button>
        </div>
      </aside>
    </div>

    <div v-if="confirm" class="fixed inset-0 z-[70] bg-black/60 flex items-center justify-center px-4">
      <div class="w-full max-w-md bg-panel border border-line rounded-2xl p-6">
        <h2 class="font-display text-xl mb-2">{{ confirm.title }}</h2>
        <p class="text-sm text-mute mb-5">{{ confirm.body }}</p>
        <div class="flex justify-end gap-2">
          <button class="px-3 py-1.5 border border-line rounded" @click="confirm = null">取消</button>
          <button class="px-3 py-1.5 bg-phos text-ink rounded font-bold" data-testid="confirm-ok" @click="confirm.ok()">确认</button>
        </div>
      </div>
    </div>
  </div>
</template>
