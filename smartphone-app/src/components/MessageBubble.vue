<template>
  <div
    class="w-full sm:w-1/2 wrap-break-word px-4 py-2 rounded-lg shadow-sm relative"
    :class="
      msg.role === 'user'
        ? 'bg-blue-600 text-white rounded-br-none'
        : 'bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 rounded-bl-none'
    "
  >
    <!-- Text message -->
    <div v-if="msg.type === 'text'">
      <div v-if="msg.isTyping" class="flex items-center gap-1.5 py-1">
        <span class="typing-dot"></span>
        <span class="typing-dot"></span>
        <span class="typing-dot"></span>
      </div>
      <div v-else>{{ msg.text }}</div>
    </div>

    <!-- Audio message -->
    <div v-else-if="msg.type === 'audio'" class="flex flex-col gap-2">
      <!-- Play button and waveform on same level -->
      <div class="flex items-center gap-3">
        <!-- Round play/pause button -->
        <button
          @click="$emit('toggle-play', msg)"
          class="shrink-0 w-10 h-10 rounded-full bg-gray-200 dark:bg-slate-700 hover:bg-gray-300 dark:hover:bg-slate-600 transition-colors flex items-center justify-center border border-gray-300 dark:border-slate-600"
          :class="msg.role === 'user' ? 'bg-blue-500 hover:bg-blue-600 border-blue-400' : ''"
        >
          <span class="text-lg">
            {{ isPlaying ? "⏸" : "▶" }}
          </span>
        </button>

        <!-- Audio waveform visualization -->
        <div class="flex-1 min-w-0">
          <AudioWaveform
            v-if="msg.audioData"
            :audio-data="msg.audioData"
            :is-playing="isPlaying"
            :current-time="audioCurrentTime"
            :duration="audioDuration"
          />
        </div>
      </div>

      <!-- Duration and toggle text button -->
      <div class="flex items-center gap-2">
        <span class="text-xs text-slate-500 dark:text-slate-400">
          {{ msg.duration ? msg.duration + "s" : "Audio message" }}
        </span>
        <!-- Toggle text button -->
        <button
          v-if="msg.text"
          @click="showText = !showText"
          class="ml-auto px-2 py-1 text-xs border rounded bg-gray-100 dark:bg-slate-600 text-slate-600 dark:text-slate-300 hover:bg-gray-200 dark:hover:bg-slate-500 transition-colors"
        >
          {{ showText ? "Hide text" : "Show text" }}
        </button>
      </div>

      <!-- Collapsible text (transcription) -->
      <div v-if="msg.text && showText" class="text-sm opacity-90 mt-1 p-2 bg-gray-50 dark:bg-slate-700/50 rounded border border-gray-200 dark:border-slate-600">
        {{ msg.text }}
      </div>
    </div>

    <!-- Status / timestamp -->
    <div class="text-xs text-slate-400 mt-1 text-right">
      <span v-if="msg.role === 'user'">
        {{ formatTime(msg) }}
      </span>
      <span v-else>
        {{ formatTime(msg) }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { Message } from "../stores/chat";
import AudioWaveform from "./AudioWaveform.vue";

const props = defineProps<{
  msg: Message;
  isPlaying: boolean;
  audioCurrentTime: number;
  audioDuration: number;
}>();

const emit = defineEmits(["toggle-play"]);

// Text visibility state - collapsed by default for audio messages
const showText = ref(false);

function formatTime(msg: Message) {
  return new Date(msg.createdAt).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}
</script>

<style scoped>
.typing-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background-color: currentColor;
  display: inline-block;
  animation: typing 1.4s infinite;
}

.typing-dot:nth-child(2) {
  animation-delay: 0.2s;
}

.typing-dot:nth-child(3) {
  animation-delay: 0.4s;
}

@keyframes typing {
  0%, 60%, 100% {
    opacity: 0.4;
    transform: translateY(0);
  }
  30% {
    opacity: 1;
    transform: translateY(-6px);
  }
}
</style>
