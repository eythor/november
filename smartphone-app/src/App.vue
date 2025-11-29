<template>
  <div
    class="h-screen flex flex-col bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-slate-100"
  >
    <header
      class="flex items-center justify-center py-4 px-4 border-b border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800"
    >
      <h1 class="text-xl font-semibold text-slate-900 dark:text-slate-100">VoiceMed</h1>
    </header>
    <MessageList
      :messages="messages"
      :scroll-el="scrollEl"
      :is-playing="isPlaying"
      :toggle-play="togglePlay"
      :handle-play="handlePlay"
      :handle-pause="handlePause"
      :handle-time-update="handleTimeUpdate"
      :handle-duration="handleDuration"
    />
    <Footer @onVoiceRecorded="onVoiceRecorded" />
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick, onMounted } from "vue";
import { useChatStore } from "./stores/chat";
import { storeToRefs } from "pinia";
import MessageList from "./components/MessageList.vue";
import Footer from "./components/Footer.vue";
import { sendAudioMessage } from "./services/api";
import type { Message } from "./stores/chat";

const store = useChatStore();
const { messages } = storeToRefs(store);
const scrollEl = ref<HTMLElement | null>(null);

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

// Simple state: just track which message is playing
const playingMsgId = ref<string | null>(null);

// Simple toggle - wavesurfer handles the actual play/pause
function togglePlay(msg: Message) {
  // If clicking the same message, toggle play/pause
  if (playingMsgId.value === msg.id) {
    playingMsgId.value = null; // This will trigger pause via isPlaying prop
  } else {
    // If clicking a different message, stop current and start new one
    if (playingMsgId.value) {
      playingMsgId.value = null; // Stop current
    }
    playingMsgId.value = msg.id; // Start new one (AudioWaveform will play via isPlaying prop)
  }
}

function isPlaying(msg: Message) {
  return playingMsgId.value === msg.id;
}

// Simple handlers - wavesurfer does all the work
function handlePlay(msg: Message) {
  // Stop any other playing message
  if (playingMsgId.value && playingMsgId.value !== msg.id) {
    playingMsgId.value = null;
  }
  playingMsgId.value = msg.id;
}

function handlePause(msg: Message) {
  if (playingMsgId.value === msg.id) {
    playingMsgId.value = null;
  }
}

function handleTimeUpdate(msg: Message, time: number) {
  // Can store time if needed, but wavesurfer handles visualization
  // No action needed - wavesurfer updates its own UI
}

function handleDuration(msg: Message, duration: number) {
  // Only update if duration is a valid finite number
  if (isFinite(duration) && duration > 0) {
    const roundedDuration = Math.floor(duration);
    if (msg.duration !== roundedDuration) {
      store.patchMessage(msg.id, { duration: roundedDuration });
    }
  }
}

// --- user sends voice ---

async function onVoiceRecorded(file: File) {
  // Convert file to data URL for visualization
  const audioData = await new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(reader.result as string);
    reader.onerror = reject;
    reader.readAsDataURL(file);
  });

  // Get duration from the audio file
  const duration = await new Promise<number>((resolve, reject) => {
    const audio = new Audio(audioData);
    let resolved = false;

    audio.onloadedmetadata = () => {
      if (resolved) return;
      resolved = true;
      const dur = audio.duration;
      // Ensure duration is finite and valid
      if (isFinite(dur) && dur > 0) {
        resolve(dur);
      } else {
        reject(new Error("Invalid audio duration"));
      }
    };

    audio.onerror = () => {
      if (resolved) return;
      resolved = true;
      reject(new Error("Failed to load audio metadata"));
    };

    // Timeout fallback - if metadata doesn't load, proceed anyway (assume valid)
    setTimeout(() => {
      if (!resolved) {
        resolved = true;
        // If we can't get duration, proceed anyway (assume it's valid)
        resolve(999); // Large value so it passes the check
      }
    }, 3000);
  }).catch(() => {
    // If we can't determine duration, proceed anyway (assume it's valid)
    console.warn("Could not determine audio duration, proceeding anyway");
    return 999; // Large value so it passes the check
  });

  // Discard audio shorter than 2 seconds (only reject short snippets)
  if (duration < 2.0) {
    console.log(`Audio too short (${duration.toFixed(2)}s), discarding`);
    return;
  }

  const id = pushLocalMessage({
    role: "user",
    type: "audio",
    duration: Math.floor(duration),
    audioData: audioData,
  });
  const message = store.messages.find((m) => m.id === id);

  if (message) {
    // Immediately add a typing indicator message
    const typingMessageId = pushLocalMessage({
      role: "assistant",
      type: "text",
      text: "",
      isTyping: true,
      createdAt: Date.now(),
    });

    try {
      const response = await sendAudioMessage(file, message);

      // Mark message as delivered
      store.patchMessage(id, { delivered: true, sentAt: Date.now() });

      // Handle backend reply if present
      if (response && response.reply) {
        const audioData = response.reply.audio
          ? `data:${response.reply.audioMimetype || "audio/mpeg"};base64,${response.reply.audio}`
          : undefined;

        console.log("Received reply:", {
          type: response.reply.type,
          hasAudio: !!audioData,
          audioLength: audioData?.length,
          duration: response.reply.duration,
        });

        // Update the typing message with the actual response
        store.patchMessage(typingMessageId, {
          type: (response.reply.type || "text") as "text" | "audio",
          text: response.reply.text,
          duration: response.reply.duration,
          audioData: audioData,
          isTyping: false,
        });

        // Auto-play audio if it's an audio message
        if (response.reply.type === "audio" && audioData) {
          // Wait for the component to mount and wavesurfer to be ready
          nextTick(() => {
            setTimeout(() => {
              const newMessage = store.messages.find((m) => m.id === typingMessageId);
              if (newMessage) {
                // Set playing state - AudioWaveform will handle the actual playback
                playingMsgId.value = newMessage.id;
              }
            }, 300); // Small delay to ensure wavesurfer is initialized
          });
        }
      } else {
        // If no reply, remove the typing indicator
        store.patchMessage(typingMessageId, {
          isTyping: false,
          text: "No response received",
        });
      }

      nextTick(scrollToBottom);
    } catch (error) {
      console.error("Failed to send audio message:", error);

      // Check if it's a "too short" error - if so, remove both messages
      const errorMessage = error instanceof Error ? error.message : String(error);
      if (errorMessage.includes("too short") || errorMessage.includes("400")) {
        // Remove the user message and typing indicator
        store.removeMessage(id);
        store.removeMessage(typingMessageId);
        return;
      }

      // Update typing message to show error
      store.patchMessage(typingMessageId, {
        isTyping: false,
        text: "Sorry, I encountered an error processing your message.",
      });
    }
  }
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
