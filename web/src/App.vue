<script setup lang="ts">
import { onMounted, ref, computed } from "vue";
import { useRoute, useRouter } from "vue-router";

const route = useRoute();
const router = useRouter();

const activeTool = computed(() => {
  if (route.path === "/upload") return "upload";
  if (route.path === "/lightning-map") return "lightning-map";
  if (route.path === "/compare") return "compare";
  return "home";
});

const health = ref<string | null>(null);

onMounted(async () => {
  try {
    const res = await fetch("/api/health");
    const data = (await res.json()) as { status: string };
    health.value = data.status;
  } catch {
    health.value = "unreachable";
  }
});

const onSelectTool = (tool: string) => {
  router.push(`/${tool}`);
};
</script>

<template>
  <div class="app-layout">
    <header class="app-header">
      <div class="header-main" @click="router.push('/')">
        <div class="logo">🗺️</div>
        <div>
          <h1>Map Tools</h1>
          <p class="tagline">Build ride maps from saved Strava exports.</p>
        </div>
      </div>

      <div class="header-side">
        <button
          v-if="activeTool !== 'upload'"
          class="btn btn-secondary"
          @click="router.push('/upload')"
        >
          Uploads
        </button>
        <button v-if="activeTool !== 'home'" class="btn btn-secondary" @click="router.push('/')">
          Home
        </button>
        <div class="api-badge" :class="health">
          API status: <code>{{ health ?? "…" }}</code>
        </div>
      </div>
    </header>

    <main class="app-content">
      <router-view @select-tool="onSelectTool" />
    </main>
  </div>
</template>

<style scoped>
@reference "tailwindcss";

.app-layout {
  @apply mx-auto flex w-full max-w-[1180px] flex-col gap-[18px] px-4 pb-7 pt-[18px];
}

.app-header {
  @apply flex flex-wrap items-center justify-between gap-3 border-b border-zinc-800 pb-3;
}

.header-main {
  @apply -m-1 flex cursor-pointer items-center gap-3 rounded-md border border-transparent px-2 py-1 transition-colors;
}

.header-main:hover {
  @apply border-zinc-700 bg-zinc-900;
}

.header-side {
  @apply flex flex-wrap items-center gap-2.5;
}

.logo {
  font-size: 1.85rem;
  line-height: 1;
}

.app-header h1 {
  @apply mb-0.5 mt-0 text-[1.72rem] font-bold text-white;
}

.tagline {
  @apply m-0 text-[0.9rem] text-zinc-400;
}

.api-badge {
  @apply rounded-full border border-zinc-700 bg-zinc-900 px-2.5 py-[5px] text-xs text-zinc-400;
}

.api-badge.ok code {
  @apply text-green-500;
}

.api-badge.unreachable code {
  @apply text-red-500;
}

.app-content {
  @apply flex flex-col gap-4;
}

@media (max-width: 720px) {
  .app-layout {
    @apply px-3 pb-[22px] pt-[14px];
  }

  .app-header {
    @apply items-start;
  }

  .header-side {
    @apply w-full justify-start;
  }
}
</style>
