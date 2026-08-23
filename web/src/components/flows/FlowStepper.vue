<script setup lang="ts">
import { computed } from "vue";

interface StepItem {
  number: number;
  label: string;
}

const emit = defineEmits<{
  (e: "step-click", stepNumber: number): void;
}>();

const props = defineProps<{
  currentStep: number;
  steps: StepItem[];
}>();

const previousStep = computed(() => {
  const currentIndex = props.steps.findIndex((step) => step.number === props.currentStep);
  return currentIndex > 0 ? props.steps[currentIndex - 1] : null;
});
</script>

<template>
  <div class="stepper">
    <div class="steps-container">
      <div
        v-for="step in steps"
        :key="step.number"
        class="step-item"
        :class="{
          active: currentStep === step.number,
          completed: currentStep > step.number,
          clickable: step.number < currentStep,
        }"
        @click="step.number < currentStep && emit('step-click', step.number)"
      >
        <div class="step-circle">{{ step.number }}</div>
        <span class="step-label">{{ step.label }}</span>
      </div>
    </div>
    <div class="stepper-actions">
      <button
        v-if="previousStep"
        class="btn btn-secondary"
        @click="emit('step-click', previousStep.number)"
      >
        Back
      </button>
      <slot name="actions" />
    </div>
  </div>
</template>
