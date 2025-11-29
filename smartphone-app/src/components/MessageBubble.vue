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

    <!-- Audio message (fake duration) -->
    <div v-else-if="msg.type === 'audio'" class="flex items-center gap-2">
      <button
        @click="$emit('toggle-play', msg)"
        class="px-2 py-1 border rounded bg-gray-200 dark:bg-slate-700 text-sm"
      >
        {{ isPlaying ? "Pause" : "Play" }}
      </button>
      <span class="text-sm">{{ msg.duration ? msg.duration + "s" : "Voice message" }}</span>
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
import { Message } from "../stores/chat";

const props = defineProps<{
  msg: Message;
  isPlaying: boolean;
}>();

const emit = defineEmits(["toggle-play"]);

function formatTime(msg: Message) {
  return new Date(msg.createdAt).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}
</script>
