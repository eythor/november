<template>
  <div
    class="max-w-[78%] wrap-break-word px-4 py-2 rounded-lg shadow-sm relative"
    :class="
      msg.role === 'user'
        ? 'bg-blue-600 text-white rounded-br-none'
        : 'bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 rounded-bl-none'
    "
  >
    <!-- Text message -->
    <div v-if="msg.type === 'text'">{{ msg.text }}</div>

    <!-- Audio message -->
    <div v-else-if="msg.type === 'audio'" class="flex flex-col gap-2">
      <!-- Audio controls -->
      <div class="flex items-center gap-2">
        <button
          @click="$emit('toggle-play', msg)"
          class="px-3 py-1.5 border rounded bg-gray-200 dark:bg-slate-700 text-sm font-medium hover:bg-gray-300 dark:hover:bg-slate-600 transition-colors"
        >
          {{ isPlaying ? "⏸ Pause" : "▶ Play" }}
        </button>
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
        <template v-if="msg.delivered && msg.sentAt">
          ✓ {{ formatTime({ createdAt: msg.sentAt }) }}
        </template>
        <template v-else> Sending… </template>
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

const props = defineProps<{
  msg: Message;
  isPlaying: boolean;
}>();

const emit = defineEmits(["toggle-play"]);

// Text visibility state - collapsed by default for audio messages
const showText = ref(false);

function formatTime(msg: Message) {
  return new Date(msg.createdAt).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}
</script>
