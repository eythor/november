<template>
  <div
    class="w-[90%] sm:w-[75%] md:w-[45%] wrap-break-word px-4 py-2 rounded-2xl shadow-sm relative message-bubble"
    :class="
      msg.role === 'user'
        ? 'bg-primary-500 text-white user-message rounded-br-sm'
        : 'bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 assistant-message rounded-bl-sm'
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
          class="shrink-0 w-10 h-10 rounded-full transition-colors flex items-center justify-center border"
          :class="
            msg.role === 'user'
              ? 'bg-white/20 hover:bg-white/30 border-white/30 text-white'
              : 'bg-gray-200 dark:bg-slate-700 hover:bg-gray-300 dark:hover:bg-slate-600 border-gray-300 dark:border-slate-600 text-slate-900 dark:text-slate-100'
          "
        >
          <svg
            v-if="isPlaying"
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="currentColor"
            class="w-5 h-5"
          >
            <path
              fill-rule="evenodd"
              d="M6.75 5.25a.75.75 0 01.75-.75H9a.75.75 0 01.75.75v13.5a.75.75 0 01-.75.75H7.5a.75.75 0 01-.75-.75V5.25zm7.5 0A.75.75 0 0115 4.5h1.5a.75.75 0 01.75.75v13.5a.75.75 0 01-.75.75H15a.75.75 0 01-.75-.75V5.25z"
              clip-rule="evenodd"
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
              fill-rule="evenodd"
              d="M4.5 5.653c0-1.426 1.529-2.33 2.779-1.643l11.54 6.348c1.295.712 1.295 2.573 0 3.285L7.28 19.991c-1.25.687-2.779-.217-2.779-1.643V5.653z"
              clip-rule="evenodd"
            />
          </svg>
        </button>

        <!-- Audio waveform visualization -->
        <div class="flex-1 min-w-0">
          <AudioWaveform
            v-if="msg.audioData"
            :audio-data="msg.audioData"
            :is-playing="isPlaying"
            @play="$emit('play', msg)"
            @pause="$emit('pause', msg)"
            @timeupdate="$emit('timeupdate', $event)"
            @duration="$emit('duration', $event)"
          />
        </div>
      </div>

      <!-- Duration and toggle text button -->
      <div class="flex items-center gap-2">
        <span
          class="text-xs"
          :class="msg.role === 'user' ? 'text-white/80' : 'text-slate-500 dark:text-slate-400'"
        >
          {{
            msg.duration && isFinite(msg.duration) && msg.duration > 0
              ? msg.duration + "s"
              : "Audio message"
          }}
        </span>
        <!-- Toggle text button -->
        <button
          v-if="msg.text"
          @click="showText = !showText"
          class="ml-auto px-2 py-1 text-xs border rounded transition-colors"
          :class="
            msg.role === 'user'
              ? 'bg-white/20 hover:bg-white/30 text-white border-white/30'
              : 'bg-gray-100 dark:bg-slate-600 text-slate-600 dark:text-slate-300 hover:bg-gray-200 dark:hover:bg-slate-500 border-gray-300 dark:border-slate-600'
          "
        >
          {{ showText ? "Hide text" : "Show text" }}
        </button>
      </div>

      <!-- Collapsible text (transcription) -->
      <div
        v-if="msg.text && showText"
        class="text-sm opacity-90 mt-1 p-2 rounded border"
        :class="
          msg.role === 'user'
            ? 'bg-white/20 border-white/30 text-white'
            : 'bg-gray-50 dark:bg-slate-700/50 border-gray-200 dark:border-slate-600 text-slate-900 dark:text-slate-100'
        "
      >
        {{ msg.text }}
      </div>
    </div>

    <!-- Status / timestamp -->
    <div
      class="text-xs mt-1 text-right"
      :class="msg.role === 'user' ? 'text-white/70' : 'text-slate-400'"
    >
      {{ formatTime(msg) }}
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
}>();

const emit = defineEmits<{
  "toggle-play": [msg: Message];
  play: [msg: Message];
  pause: [msg: Message];
  timeupdate: [time: number];
  duration: [duration: number];
}>();

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
  0%,
  60%,
  100% {
    opacity: 0.4;
    transform: translateY(0);
  }
  30% {
    opacity: 1;
    transform: translateY(-6px);
  }
}
/* User message bubble tail - right side, modern style */
.user-message::after {
  content: "";
  position: absolute;
  right: -14px;
  bottom: 12px;
  width: 18px;
  height: 22px;
  background: inherit;
  clip-path: polygon(0 0, 100% 50%, 0 100%);
  z-index: 1;
  filter: drop-shadow(0 1px 2px rgba(0, 0, 0, 0.04));
}
/* Assistant message bubble tail - left side, modern style */
.assistant-message::after {
  content: "";
  position: absolute;
  left: -14px;
  bottom: 12px;
  width: 18px;
  height: 22px;
  background: inherit;
  clip-path: polygon(100% 0, 0 50%, 100% 100%);
  z-index: 1;
  filter: drop-shadow(0 1px 2px rgba(0, 0, 0, 0.04));
}
</style>
