<template>
  <div
    class="h-screen flex flex-col bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-slate-100"
  >
    <main ref="scrollEl" class="flex-1 overflow-auto p-4 space-y-3" @click="focusComposer">
      <div
        v-for="msg in messages"
        :key="msg.id"
        class="flex"
        :class="msg.role === 'user' ? 'justify-end' : 'justify-start'"
      >
        <div
          class="max-w-[78%] break-words px-4 py-2 rounded-lg shadow-sm relative"
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
              @click="togglePlay(msg)"
              class="px-2 py-1 border rounded bg-gray-200 dark:bg-slate-700 text-sm"
            >
              {{ isPlaying(msg) ? "Pause" : "Play" }}
            </button>
            <span class="text-sm">{{ msg.duration ? msg.duration + "s" : "Voice message" }}</span>
          </div>

          <!-- Status / timestamp -->
          <div class="text-xs text-slate-400 mt-1 text-right">
            <span v-if="msg.role === 'user'">
              {{ msg.delivered ? "✓ Delivered" : "Sending…" }}
            </span>
            <span v-else>
              {{ formatTime(msg) }}
            </span>
          </div>
        </div>
      </div>
    </main>

    <footer
      class="border-t border-slate-200 dark:border-slate-700 p-2 bg-slate-50 dark:bg-slate-900 flex gap-2"
    >
      <VoiceRecorder @recorded="onVoiceRecorded" />
      <textarea
        v-model="draft"
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
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, onMounted } from "vue";
import { useChatStore, Message } from "./stores/chat";
import { storeToRefs } from "pinia";
import VoiceRecorder from "./components/VoiceRecorder.vue";

const store = useChatStore();
const { messages } = storeToRefs(store);
const scrollEl = ref<HTMLElement | null>(null);
const composer = ref<HTMLTextAreaElement | null>(null);
const draft = ref("");

const canSend = computed(() => draft.value.trim().length > 0);

const genId = () => `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;

function pushLocalMessage(payload: Partial<Message>) {
  const msg = store.addMessage(payload);
  nextTick(scrollToBottom);
  return msg;
}

function formatTime(msg: Message) {
  return new Date(msg.createdAt).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

// --- fake sending logic ---
async function fakeSend(msg: Message) {
  await new Promise((resolve) => setTimeout(resolve, 800 + Math.random() * 500));
  msg.delivered = true;
  msg.sentAt = Date.now();
  nextTick(scrollToBottom);
}

// --- user sends text ---
function sendText() {
  if (!canSend.value) return;
  const text = draft.value.trim();
  draft.value = "";

  const msg = pushLocalMessage({ role: "user", type: "text", text });
  fakeSend(msg);

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
const playingMsgId = ref<string | null>(null);

function togglePlay(msg: Message) {
  // Stop current audio if playing something else
  if (currentAudio.value && playingMsgId.value !== msg.id) {
    currentAudio.value.pause();
    currentAudio.value.currentTime = 0;
    playingMsgId.value = null;
  }

  // If no audio or switching, create new
  if (!currentAudio.value || playingMsgId.value !== msg.id) {
    // For fake messages, just create silent audio
    const audioBlob = msg.blob ? new Blob([msg.blob], { type: "audio/webm" }) : null;
    const src = audioBlob ? URL.createObjectURL(audioBlob) : ""; // empty src if fake
    currentAudio.value = new Audio(src);
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

function isPlaying(msg: Message) {
  return playingMsgId.value === msg.id;
}

// --- user sends voice ---
function onVoiceRecorded(file: File) {
  const duration = Math.floor(Math.random() * 5) + 3; // fake duration for user audio
  const msg = pushLocalMessage({ role: "user", type: "audio", duration });
  fakeSend(msg);

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

function onEnter(e: KeyboardEvent) {
  if (e.shiftKey) return;
  sendText();
}

function focusComposer() {
  composer.value?.focus();
}

function scrollToBottom() {
  if (!scrollEl.value) return;
  requestAnimationFrame(() => {
    scrollEl.value!.scrollTop = scrollEl.value!.scrollHeight;
  });
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
