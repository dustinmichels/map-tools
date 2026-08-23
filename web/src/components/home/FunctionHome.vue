<script setup lang="ts">
type ToolKey = "viewer" | "lightning-map" | "compare" | "upload";

interface WorkflowCard {
  key: ToolKey;
  eyebrow: string;
  title: string;
  summary: string;
  action: string;
  badges: string[];
}

const emit = defineEmits<{
  selectTool: [tool: ToolKey];
}>();

const uploadWorkflow: WorkflowCard = {
  key: "upload",
  eyebrow: "Upload library",
  title: "Uploads",
  summary: "Process a Strava ZIP once and keep the GeoParquet ready for every map flow.",
  action: "Open Uploads",
  badges: ["ZIP → GeoParquet", "saved locally", "shared by all maps"],
};

const tools: WorkflowCard[] = [
  {
    key: "viewer",
    eyebrow: "Single rider",
    title: "Viewer",
    summary:
      "Open one saved upload, switch between simplified and original geometry, and inspect routes on hover.",
    action: "Open",
    badges: ["1 upload", "geometry toggle", "route hover"],
  },
  {
    key: "lightning-map",
    eyebrow: "Single rider",
    title: "Lightning Map",
    summary: "Filter one upload to a bounding box and draw every matching ride in one layer.",
    action: "Open",
    badges: ["1 upload", "bbox filter", "single layer"],
  },
  {
    key: "compare",
    eyebrow: "Two riders",
    title: "Compare",
    summary: "Load two saved uploads, run the same area filter, and overlay both route sets.",
    action: "Open",
    badges: ["2 uploads", "shared area", "dual layers"],
  },
];
</script>

<template>
  <section class="home-layout">
    <section class="home-workbench" aria-label="Map workflows">
      <article class="card home-panel upload-panel">
        <div class="panel-header">
          <div class="panel-copy">
            <span class="section-label">{{ uploadWorkflow.eyebrow }}</span>
            <h3>{{ uploadWorkflow.title }}</h3>
          </div>
          <span class="panel-chip">Start here</span>
        </div>

        <p class="tool-summary">{{ uploadWorkflow.summary }}</p>

        <ul class="badge-list">
          <li v-for="badge in uploadWorkflow.badges" :key="badge">{{ badge }}</li>
        </ul>

        <button
          class="btn btn-primary action-button"
          @click="emit('selectTool', uploadWorkflow.key)"
        >
          {{ uploadWorkflow.action }}
        </button>
      </article>

      <section class="card home-panel workflow-panel">
        <div class="panel-header">
          <div class="panel-copy">
            <span class="section-label">Map functions</span>
            <h3>Open a workflow</h3>
          </div>
          <span class="panel-chip">{{ tools.length }} tools</span>
        </div>

        <div class="workflow-list">
          <article v-for="tool in tools" :key="tool.key" class="workflow-row">
            <div class="workflow-copy">
              <div class="workflow-head">
                <div>
                  <span class="tool-eyebrow">{{ tool.eyebrow }}</span>
                  <h4>{{ tool.title }}</h4>
                </div>
                <button
                  class="btn btn-primary workflow-action"
                  @click="emit('selectTool', tool.key)"
                >
                  {{ tool.action }}
                </button>
              </div>

              <p class="tool-summary">{{ tool.summary }}</p>

              <ul class="badge-list compact-badges">
                <li v-for="badge in tool.badges" :key="badge">{{ badge }}</li>
              </ul>
            </div>
          </article>
        </div>
      </section>
    </section>
  </section>
</template>

<style scoped>
@reference "tailwindcss";

.home-layout {
  @apply flex flex-col gap-4;
}

.home-panel {
  @apply p-5;
}

.section-label,
.tool-eyebrow {
  @apply text-[0.72rem] font-bold uppercase text-amber-500;
  letter-spacing: 0.12em;
}

.panel-header h3,
.workflow-head h4 {
  @apply m-0 text-white;
}

.tool-summary {
  @apply m-0;
}

.home-workbench {
  @apply grid gap-4;
  grid-template-columns: minmax(260px, 320px) minmax(0, 1fr);
}

.upload-panel,
.workflow-panel {
  @apply flex flex-col gap-4;
}

.panel-header,
.workflow-head {
  @apply flex items-start justify-between gap-4;
}

.panel-copy {
  @apply flex flex-col gap-1.5;
}

.panel-chip {
  @apply whitespace-nowrap rounded-full border border-zinc-700 px-2.5 py-1.5 text-[0.74rem] text-zinc-300;
}

.workflow-list {
  @apply flex flex-col;
}

.workflow-row + .workflow-row {
  @apply mt-4 border-t border-zinc-800 pt-4;
}

.workflow-copy {
  @apply flex flex-col gap-3;
}

.badge-list {
  @apply m-0 flex list-none flex-wrap gap-2 p-0;
}

.badge-list li {
  @apply rounded-full border border-zinc-700 bg-zinc-800 px-2.5 py-1.5 text-[0.78rem] leading-none text-zinc-300;
}

.compact-badges li {
  @apply text-[0.74rem];
}

.tool-eyebrow {
  @apply mb-1.5 block;
}

.action-button,
.workflow-action {
  @apply min-w-[112px] px-3.5 py-[9px];
}

.workflow-action {
  @apply shrink-0;
}

@media (max-width: 980px) {
  .home-workbench {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 720px) {
  .home-panel {
    @apply p-4;
  }

  .workflow-head {
    @apply flex-col;
  }

  .workflow-action {
    @apply w-full;
  }
}
</style>
