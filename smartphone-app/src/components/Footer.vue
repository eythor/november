<template>
  <footer
    class="border-t border-slate-200 dark:border-slate-700 p-2 bg-slate-50 dark:bg-slate-900 flex gap-2 justify-center"
  >
    <VoiceRecorder @recorded="onVoiceRecorded" />
  </footer>
</template>

<script setup lang="ts">
import VoiceRecorder from "./VoiceRecorder.vue";

const emit = defineEmits<{
  "onVoiceRecorded": [file: File];
}>();

function onVoiceRecorded(file: File) {
  emit("onVoiceRecorded", file);
}
</script>
