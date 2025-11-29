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
    <button class="px-4 py-2 rounded-full border" :class="{ 'bg-red-500 text-white': recording }">
      {{ recording ? `${seconds}s Recording...` : "Hold to Talk" }}
    </button>

    <div v-if="recording" class="absolute -top-6 text-xs text-gray-600">{{ seconds }}s</div>
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
