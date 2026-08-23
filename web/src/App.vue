<script setup lang="ts">
import { onMounted, ref, computed } from "vue";
import { useRoute, useRouter } from "vue-router";

const route = useRoute();
const router = useRouter();

const activeTool = computed(() => {
  if (route.path === "/upload") return "upload";
  if (route.path === "/viewer") return "viewer";
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
        <button
          v-if="activeTool !== 'home'"
          class="btn btn-secondary"
          @click="router.push('/')"
        >
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
.app-layout {
  max-width: 1180px;
  width: 100%;
  margin: 0 auto;
  padding: 18px 16px 28px;
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.app-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-bottom: 1px solid #222;
  padding-bottom: 12px;
  flex-wrap: wrap;
  gap: 12px;
}

.header-main {
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
  border: 1px solid transparent;
  padding: 4px 8px;
  margin: -4px -8px;
  border-radius: 6px;
  transition:
    border-color 0.2s,
    background-color 0.2s;
}

.header-main:hover {
  border-color: #3c3c3c;
  background-color: #1a1a1a;
}

.header-side {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.logo {
  font-size: 1.85rem;
  line-height: 1;
}

.app-header h1 {
  font-size: 1.72rem;
  margin: 0 0 2px;
  font-weight: 700;
  color: #fff;
}

.tagline {
  margin: 0;
  font-size: 0.9rem;
  color: #9f9f9f;
}

.api-badge {
  font-size: 12px;
  padding: 5px 10px;
  border-radius: 999px;
  background: #181818;
  border: 1px solid #333;
  color: #aaa;
}

.api-badge.ok code {
  color: #44bb44;
}

.api-badge.unreachable code {
  color: #ff4444;
}

.app-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

@media (max-width: 720px) {
  .app-layout {
    padding: 14px 12px 22px;
  }

  .app-header {
    align-items: flex-start;
  }

  .header-side {
    width: 100%;
    justify-content: flex-start;
  }
}
</style>
