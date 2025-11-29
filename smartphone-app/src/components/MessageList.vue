<template>
  <main ref="scrollEl" class="flex-1 overflow-auto p-4 space-y-3">
    <div
      v-for="msg in messages"
      :key="msg.id"
      class="flex"
      :class="msg.role === 'user' ? 'justify-end' : 'justify-start'"
    >
      <MessageBubble
        :msg="msg"
        :is-playing="isPlaying(msg)"
        @toggle-play="togglePlay"
        @play="(m) => handlePlay(m)"
        @pause="(m) => handlePause(m)"
        @timeupdate="(time) => handleTimeUpdate(msg, time)"
        @duration="(duration) => handleDuration(msg, duration)"
      />
    </div>
  </main>
</template>

<script setup lang="ts">
import { ref, nextTick } from "vue";
import MessageBubble from "./MessageBubble.vue";
import { Message } from "../stores/chat";

const props = defineProps<{
  messages: Message[];
  scrollEl: HTMLElement | null;
  isPlaying: (msg: Message) => boolean;
  togglePlay: (msg: Message) => void;
  handlePlay: (msg: Message) => void;
  handlePause: (msg: Message) => void;
  handleTimeUpdate: (msg: Message, time: number) => void;
  handleDuration: (msg: Message, duration: number) => void;
}>();
</script>
