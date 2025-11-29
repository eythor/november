<template>
  <div
    class="h-screen flex flex-col bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-slate-100"
  >
    <header class="flex items-center justify-center py-4 px-4 border-b border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800">
      <h1 class="text-xl font-semibold text-slate-900 dark:text-slate-100">VoiceMed</h1>
    </header>
    <MessageList
      :messages="messages"
      :scroll-el="scrollEl"
      :is-playing="isPlaying"
      :toggle-play="togglePlay"
      :audio-current-time="playingMsgId ? audioCurrentTime : 0"
      :audio-duration="playingMsgId ? audioDuration : 0"
      :playing-msg-id="playingMsgId"
    />
    <Footer
      @onVoiceRecorded="onVoiceRecorded"
    />
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


const currentAudio = ref<HTMLAudioElement | null>(null);
const playingMsgId = ref<string | null>(null);
const isAudioPlaying = ref(false);
const audioCurrentTime = ref(0);
const audioDuration = ref(0);

function togglePlay(msg: Message) {
  console.log('togglePlay called for message:', msg.id, 'hasAudioData:', !!msg.audioData);

  // Stop current audio if playing something else
  if (currentAudio.value && playingMsgId.value !== msg.id) {
    currentAudio.value.pause();
    currentAudio.value.currentTime = 0;
    currentAudio.value = null;
    playingMsgId.value = null;
    isAudioPlaying.value = false;
    audioCurrentTime.value = 0;
    audioDuration.value = 0;
  }

  // If no audio or switching, create new
  if (!currentAudio.value || playingMsgId.value !== msg.id) {
    // Check if message has audio data stored
    const audioData = msg.audioData;
    if (!audioData) {
      console.warn('No audio data found for message:', msg.id, msg);
      return;
    }

    console.log('Creating audio element with data URL length:', audioData.length);
    console.log('Data URL prefix:', audioData.substring(0, 50));

    // Create new audio element
    const audio = new Audio(audioData);

    // Set up event handlers
    audio.onloadedmetadata = () => {
      console.log('Audio metadata loaded, duration:', audio.duration, 'readyState:', audio.readyState);
      audioDuration.value = audio.duration;
    };

    audio.oncanplay = () => {
      console.log('Audio can play, readyState:', audio.readyState);
    };

    audio.onplay = () => {
      console.log('Audio started playing');
      isAudioPlaying.value = true;
    };

    audio.onpause = () => {
      console.log('Audio paused, currentTime:', audio.currentTime);
      isAudioPlaying.value = false;
      audioCurrentTime.value = audio.currentTime;
      if (audio.currentTime === 0 || audio.ended) {
        playingMsgId.value = null;
        currentAudio.value = null;
        isAudioPlaying.value = false;
        audioCurrentTime.value = 0;
        audioDuration.value = 0;
      }
    };

    audio.onended = () => {
      console.log('Audio ended');
      playingMsgId.value = null;
      currentAudio.value = null;
      isAudioPlaying.value = false;
      audioCurrentTime.value = 0;
      audioDuration.value = 0;
    };

    // Track currentTime for waveform sync
    audio.ontimeupdate = () => {
      if (audio === currentAudio.value) {
        audioCurrentTime.value = audio.currentTime;
      }
    };

    audio.onerror = (e) => {
      console.error('Audio playback error:', e, 'error code:', audio.error?.code, 'error message:', audio.error?.message);
      console.error('Audio element state:', {
        src: audio.src.substring(0, 100),
        readyState: audio.readyState,
        networkState: audio.networkState
      });
      playingMsgId.value = null;
      currentAudio.value = null;
      isAudioPlaying.value = false;
    };

    audio.onloadeddata = () => {
      console.log('Audio data loaded, duration:', audio.duration, 'readyState:', audio.readyState);
    };

    currentAudio.value = audio;
    playingMsgId.value = msg.id;
    isAudioPlaying.value = false;

    // Play the audio
    const playPromise = audio.play();
    if (playPromise !== undefined) {
      playPromise
        .then(() => {
          console.log('Audio playing successfully');
        })
        .catch((error) => {
          console.error('Error playing audio:', error);
          console.error('Audio element:', {
            src: audio.src.substring(0, 100),
            readyState: audio.readyState,
            paused: audio.paused,
            error: audio.error
          });
          playingMsgId.value = null;
          currentAudio.value = null;
          isAudioPlaying.value = false;
        });
    }
  } else {
    // toggle play/pause for current audio
    if (currentAudio.value) {
      if (currentAudio.value.paused) {
        currentAudio.value.play().catch((error) => {
          console.error('Error resuming audio:', error);
          isAudioPlaying.value = false;
        });
      } else {
        currentAudio.value.pause();
      }
    }
  }
}

function isPlaying(msg: Message) {
  return playingMsgId.value === msg.id && isAudioPlaying.value;
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

  // Get actual duration from the audio file
  const duration = await new Promise<number>((resolve) => {
    const audio = new Audio(audioData);
    audio.onloadedmetadata = () => {
      resolve(audio.duration);
    };
    audio.onerror = () => {
      resolve(Math.floor(Math.random() * 5) + 3); // fallback to fake duration
    };
  });

  const id = pushLocalMessage({
    role: "user",
    type: "audio",
    duration: Math.floor(duration),
    audioData: audioData
  });
  const message = store.messages.find(m => m.id === id);

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
      store.patchMessage(id, { delivered: true, sentAt: Date.now() });

      // Handle backend reply if present
      if (response && response.reply) {
        const audioData = response.reply.audio
          ? `data:${response.reply.audioMimetype || 'audio/mpeg'};base64,${response.reply.audio}`
          : undefined;

        console.log('Received reply:', {
          type: response.reply.type,
          hasAudio: !!audioData,
          audioLength: audioData?.length,
          duration: response.reply.duration
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
        if (response.reply.type === 'audio' && audioData) {
          nextTick(() => {
            const newMessage = store.messages.find(m => m.id === typingMessageId);
            if (newMessage) {
              togglePlay(newMessage);
            }
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
      console.error('Failed to send audio message:', error);
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
