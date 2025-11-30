<template>
  <div
    class="relative flex items-center"
    @mousedown="startPress"
    @touchstart.prevent="startPress"
    @mouseup="endPress(false)"
    @mouseleave="endPress(true)"
    @touchend.prevent="endPress(false)"
    @touchcancel.prevent="endPress(true)"
  >
    <button
      class="px-4 py-2 rounded-full border flex items-center gap-2 transition-colors"
      :class="
        recording
          ? 'bg-red-500 text-white border-red-500'
          : 'bg-white dark:bg-slate-800 border-primary-500 text-primary-500 hover:bg-primary-50 dark:hover:bg-primary-950/20'
      "
    >
      <svg
        v-if="!recording"
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 24 24"
        fill="currentColor"
        class="w-5 h-5"
      >
        <path
          d="M8.25 4.5a3.75 3.75 0 117.5 0v8.25a3.75 3.75 0 11-7.5 0V4.5zM18.75 9a.75.75 0 00-1.5 0v3.75a5.25 5.25 0 01-10.5 0V9a.75.75 0 00-1.5 0v3.75a6.751 6.751 0 006 6.72v1.78a.75.75 0 001.5 0v-1.78a6.751 6.751 0 006-6.72V9z"
        />
      </svg>
      <svg
        v-else
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 24 24"
        fill="currentColor"
        class="w-5 h-5"
      >
        <path
          d="M8.25 4.5a3.75 3.75 0 117.5 0v8.25a3.75 3.75 0 11-7.5 0V4.5zM18.75 9a.75.75 0 00-1.5 0v3.75a5.25 5.25 0 01-10.5 0V9a.75.75 0 00-1.5 0v3.75a6.751 6.751 0 006 6.72v1.78a.75.75 0 001.5 0v-1.78a6.751 6.751 0 006-6.72V9z"
        />
      </svg>
      <span v-if="recording">{{ seconds }}s Recording...</span>
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, onBeforeUnmount, defineEmits } from "vue";

const emit = defineEmits(["recorded"]);

const mediaRecorder = ref<MediaRecorder | null>(null);
const chunks = ref<Blob[]>([]);
const recording = ref(false);
const seconds = ref(0);
let interval: number | null = null;
let stream: MediaStream | null = null;

async function initRecorder() {
  if (!stream) {
    stream = await navigator.mediaDevices.getUserMedia({ audio: true });
  }
  const rec = new MediaRecorder(stream);
  rec.ondataavailable = (e) => chunks.value.push(e.data);
  rec.onstop = sendAudio;
  mediaRecorder.value = rec;
}

async function startPress() {
  if (!mediaRecorder.value) await initRecorder();

  chunks.value = [];
  seconds.value = 0;
  recording.value = true;
  mediaRecorder.value!.start();

  interval = window.setInterval(() => seconds.value++, 1000);
}

function endPress(cancel: boolean) {
  if (!recording.value) return;

  recording.value = false;
  if (interval) clearInterval(interval);
  interval = null;

  if (cancel) {
    chunks.value = []; // discard
    mediaRecorder.value?.stop();
    return;
  }

  mediaRecorder.value?.stop();
}

async function sendAudio() {
  if (!chunks.value.length) return;

  const blob = new Blob(chunks.value, { type: "audio/webm" });
  const file = new File([blob], `voice-${Date.now()}.webm`, { type: "audio/webm" });

  emit("recorded", file);
  chunks.value = [];
}

// Stop tracks when component is destroyed
onBeforeUnmount(() => {
  mediaRecorder.value?.stop();
  if (stream) {
    stream.getTracks().forEach((t) => t.stop());
  }
});
</script>
