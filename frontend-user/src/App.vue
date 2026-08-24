<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useSession } from "./stores/session";
import Studio from "./views/Studio.vue";

const session = useSession();
const user = ref("geek");
const pass = ref("phosphor");
const err = ref("");
const fieldErr = ref({ user: "", pass: "" });

async function submit() {
  fieldErr.value = { user: "", pass: "" };
  if (!user.value) fieldErr.value.user = "用户名必填";
  if (!pass.value) fieldErr.value.pass = "密码必填";
  if (fieldErr.value.user || fieldErr.value.pass) {
    err.value = "请先修正表单";
    return;
  }
  err.value = "";
  try {
    await session.login(user.value, pass.value);
  } catch (e: any) {
    err.value = e.message || "登录失败";
  }
}

onMounted(() => {
  if (session.token) session.username = "geek";
});
</script>

<template>
  <div class="crt-noise min-h-full">
    <Studio v-if="session.token" />
    <div v-else class="min-h-full flex items-center justify-center px-4">
      <form class="w-full max-w-md bg-panel/90 border border-line rounded-2xl p-8 shadow-[0_0_60px_rgba(157,255,107,0.08)]" @submit.prevent="submit">
        <p class="font-display text-phos text-xs tracking-[0.35em] uppercase mb-2">MacroCanvas</p>
        <h1 class="font-display text-3xl mb-1">HID 宏编排控制台</h1>
        <p class="text-mute text-sm mb-6">内核虚拟设备 · 串行微秒队列 · 极客画布</p>
        <label class="block text-xs text-mute mb-1">用户名 *</label>
        <input v-model="user" class="w-full bg-ink border border-line rounded-lg px-3 py-2 font-mono mb-1 focus:border-phos outline-none" />
        <p v-if="fieldErr.user" class="text-amber text-xs mb-2">{{ fieldErr.user }}</p>
        <label class="block text-xs text-mute mb-1 mt-3">密码 *</label>
        <input v-model="pass" type="password" class="w-full bg-ink border border-line rounded-lg px-3 py-2 font-mono mb-1 focus:border-phos outline-none" />
        <p v-if="fieldErr.pass" class="text-amber text-xs mb-2">{{ fieldErr.pass }}</p>
        <button class="mt-5 w-full bg-phos text-ink font-display font-bold py-2.5 rounded-lg hover:brightness-110 transition">进入控制台</button>
        <p v-if="err" class="text-amber text-sm mt-3">{{ err }}</p>
        <p class="text-mute text-xs mt-6">测试账号 geek / phosphor · Token 热下发无需重启</p>
      </form>
    </div>
    <div v-if="session.toast" class="fixed top-4 right-4 z-[60] px-4 py-3 rounded-lg border font-mono text-sm flex gap-3 items-center"
      :class="session.toastKind === 'err' ? 'bg-amber/10 border-amber text-amber' : 'bg-phos/10 border-phos text-phos'">
      <span>{{ session.toast }}</span>
      <button aria-label="关闭提示" class="opacity-70 hover:opacity-100" @click="session.toast = ''">×</button>
    </div>
  </div>
</template>
