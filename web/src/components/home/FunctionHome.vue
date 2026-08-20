<script setup lang="ts">
type ToolKey = "lightning-map" | "compare" | "upload";

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

const overviewBadges = ["Upload once", "Pick a map function", "Export from saved data"];
</script>

<template>
  <section class="home-layout">
    <header class="card home-panel home-bar">
      <div class="home-bar-copy">
        <span class="section-label">Map workbench</span>
        <h2>Pick the map function you need.</h2>
        <p class="lead-text home-copy">
          Start in Uploads, then open the single-rider or compare flow.
        </p>
      </div>

      <ul class="badge-list overview-list" aria-label="Home overview">
        <li v-for="badge in overviewBadges" :key="badge">{{ badge }}</li>
      </ul>
    </header>

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
.home-layout {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.home-panel {
  padding: 20px 22px;
}

.home-bar {
  display: flex;
  justify-content: space-between;
  gap: 20px;
  align-items: flex-start;
}

.home-bar-copy {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-width: 720px;
}

.section-label,
.tool-eyebrow {
  font-size: 0.72rem;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: #ff9900;
}

.home-bar h2,
.panel-header h3,
.workflow-head h4 {
  margin: 0;
  color: #fff;
}

.home-copy,
.tool-summary {
  margin: 0;
}

.home-workbench {
  display: grid;
  grid-template-columns: minmax(260px, 320px) minmax(0, 1fr);
  gap: 16px;
}

.upload-panel,
.workflow-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.panel-header,
.workflow-head {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
}

.panel-copy {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.panel-chip {
  border: 1px solid #3d3d3d;
  border-radius: 999px;
  padding: 6px 10px;
  color: #d0d0d0;
  font-size: 0.74rem;
  white-space: nowrap;
}

.workflow-list {
  display: flex;
  flex-direction: column;
}

.workflow-row + .workflow-row {
  border-top: 1px solid #2b2b2b;
  margin-top: 16px;
  padding-top: 16px;
}

.workflow-copy {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.badge-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  list-style: none;
  margin: 0;
  padding: 0;
}

.badge-list li {
  border: 1px solid #333;
  background: #202020;
  border-radius: 999px;
  padding: 6px 10px;
  color: #c9c9c9;
  font-size: 0.78rem;
  line-height: 1;
}

.compact-badges li,
.overview-list li {
  font-size: 0.74rem;
}

.tool-eyebrow {
  display: block;
  margin-bottom: 6px;
}

.action-button,
.workflow-action {
  padding: 9px 14px;
  min-width: 112px;
}

.workflow-action {
  flex-shrink: 0;
}

@media (max-width: 980px) {
  .home-workbench {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 720px) {
  .home-panel {
    padding: 18px;
  }

  .home-bar,
  .workflow-head {
    flex-direction: column;
  }

  .workflow-action {
    width: 100%;
  }
}
</style>
