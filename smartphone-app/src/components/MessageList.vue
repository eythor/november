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
        :audio-current-time="playingMsgId === msg.id ? audioCurrentTime : 0"
        :audio-duration="playingMsgId === msg.id ? audioDuration : 0"
        @toggle-play="togglePlay"
        @focus-composer="focusComposer"
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
  audioCurrentTime: number;
  audioDuration: number;
  playingMsgId: string | null;
}>();
</script>
