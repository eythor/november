<template>
  <footer
    class="border-t border-slate-200 dark:border-slate-700 p-2 bg-slate-50 dark:bg-slate-900 flex gap-2"
  >
    <VoiceRecorder @recorded="onVoiceRecorded" />
    <textarea
      v-model="draftProxy"
      ref="composer"
      rows="1"
      placeholder="Message"
      class="flex-1 resize-none bg-transparent outline-none px-3 py-2 text-base rounded-lg border border-slate-200 dark:border-slate-700 focus:ring-0"
      @keydown.enter.prevent="onEnter"
      @keydown.shift.enter.stop
    />
    <button
      @click="sendText"
      :disabled="!canSend"
      class="inline-flex items-center justify-center px-3 py-2 rounded-full bg-blue-600 text-white disabled:opacity-50"
    >
      Send
    </button>
  </footer>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import VoiceRecorder from "./VoiceRecorder.vue";

const props = defineProps<{
  draft: string;
  canSend: boolean;
}>();
const emit = defineEmits(["update:draft", "sendText", "onVoiceRecorded", "onEnter"]);

const composer = ref(null);

const draftProxy = computed({
  get: () => props.draft,
  set: (val) => emit("update:draft", val),
});

function onEnter(e) {
  if (e.shiftKey) return;
  emit("onEnter", e);
}
function onVoiceRecorded(file) {
  emit("onVoiceRecorded", file);
}
function sendText() {
  emit("sendText");
}
</script>
