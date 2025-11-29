<template>
  <div
    class="h-screen flex flex-col bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-slate-100"
  >
    <MessageList
      :messages="messages"
      :scroll-el="scrollEl"
      :is-playing="isPlaying"
      :toggle-play="togglePlay"
    />
    <Footer
      v-model:draft="draft"
      :can-send="canSend"
      @sendText="sendText"
      @onVoiceRecorded="onVoiceRecorded"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, onMounted } from "vue";
import { useChatStore } from "./stores/chat";
import { storeToRefs } from "pinia";
import MessageList from "./components/MessageList.vue";
import Footer from "./components/Footer.vue";
import { sendChatMessage, sendAudioMessage } from "./services/api";
import type { Message } from "./stores/chat";

const store = useChatStore();
const { messages } = storeToRefs(store);
const scrollEl = ref<HTMLElement | null>(null);
const draft = ref("");

const canSend = computed(() => draft.value.trim().length > 0);

function scrollToBottom() {
  if (!scrollEl.value) return;
  requestAnimationFrame(() => {
    if (scrollEl.value) {
      scrollEl.value.scrollTop = scrollEl.value.scrollHeight;
    }
  });
}

function pushLocalMessage(payload: Partial<Message>) {
  const id = store.addMessage(payload);
  nextTick(scrollToBottom);
  return id;
}

// --- send message to backend ---
async function sendMessage(message: Message) {
  try {
    await sendChatMessage(message);
    store.patchMessage(message.id, { delivered: true, sentAt: Date.now() });
    nextTick(scrollToBottom);
  } catch (error) {
    console.error('Failed to send message:', error);
    // Optionally mark as failed or retry
  }
}

// --- user sends text ---
async function sendText() {
  if (!canSend.value) return;
  const text = draft.value.trim();
  draft.value = "";

  const id = pushLocalMessage({ role: "user", type: "text", text });
  const message = store.messages.find(m => m.id === id);
  if (message) {
    await sendMessage(message);
  }

  // fake assistant reply (randomly text or audio)
  setTimeout(() => {
    const isAudio = Math.random() < 0.5;
    if (isAudio) {
      pushLocalMessage({
        role: "assistant",
        type: "audio",
        duration: Math.floor(Math.random() * 5) + 3, // 3–7 sec
        createdAt: Date.now(),
      });
    } else {
      pushLocalMessage({
        role: "assistant",
        type: "text",
        text: "This is a fake reply!",
        createdAt: Date.now(),
      });
    }
  }, 1200);
}

const currentAudio = ref<HTMLAudioElement | null>(null);
const playingMsgId = ref(null);

function togglePlay(msg) {
  // Stop current audio if playing something else
  if (currentAudio.value && playingMsgId.value !== msg.id) {
    currentAudio.value.pause();
    currentAudio.value.currentTime = 0;
    playingMsgId.value = null;
  }

  // If no audio or switching, create new
  if (!currentAudio.value || playingMsgId.value !== msg.id) {
    // For fake messages, just create silent audio (no src)
    currentAudio.value = new Audio("");
    playingMsgId.value = msg.id;

    currentAudio.value.onended = () => {
      playingMsgId.value = null;
    };
    currentAudio.value.onpause = () => {
      playingMsgId.value = null;
    };
    currentAudio.value.play();
  } else {
    // toggle play/pause
    if (currentAudio.value.paused) currentAudio.value.play();
    else currentAudio.value.pause();
  }
}

function isPlaying(msg) {
  return playingMsgId.value === msg.id;
}

// --- user sends voice ---

async function onVoiceRecorded(file: File) {
  const duration = Math.floor(Math.random() * 5) + 3; // fake duration for user audio
  const id = pushLocalMessage({ role: "user", type: "audio", duration });
  const message = store.messages.find(m => m.id === id);
  
  if (message) {
    try {
      await sendAudioMessage(file, message);
      store.patchMessage(id, { delivered: true, sentAt: Date.now() });
      nextTick(scrollToBottom);
    } catch (error) {
      console.error('Failed to send audio message:', error);
      // Optionally mark as failed or retry
    }
  }

  // fake assistant reply
  setTimeout(() => {
    const isAudio = Math.random() < 0.5;
    if (isAudio) {
      pushLocalMessage({
        role: "assistant",
        type: "audio",
        duration: Math.floor(Math.random() * 5) + 3,
        createdAt: Date.now(),
      });
    } else {
      pushLocalMessage({
        role: "assistant",
        type: "text",
        text: "Received your voice message!",
        createdAt: Date.now(),
      });
    }
  }, 1200);
}

onMounted(() => {
  if (!messages.value.length) {
    pushLocalMessage({
      role: "assistant",
      type: "text",
      text: "Hi — how can I help?",
      createdAt: Date.now(),
    });
  }
  nextTick(scrollToBottom);
});
</script>

<style scoped>
main::-webkit-scrollbar {
  height: 6px;
  width: 6px;
}
main::-webkit-scrollbar-thumb {
  background: rgba(0, 0, 0, 0.12);
  border-radius: 999px;
}
.audio-bubble {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
</style>
