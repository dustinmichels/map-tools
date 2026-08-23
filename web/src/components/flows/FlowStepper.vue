<script setup lang="ts">
interface StepItem {
  number: number;
  label: string;
}

const emit = defineEmits<{
  (e: "step-click", stepNumber: number): void;
}>();

defineProps<{
  currentStep: number;
  steps: StepItem[];
}>();
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
      <slot name="actions" />
    </div>
  </div>
</template>
