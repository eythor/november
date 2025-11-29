<template>
  <div class="audio-waveform-container">
    <div ref="waveformRef" class="waveform"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch } from 'vue';
import WaveSurfer from 'wavesurfer.js';

const props = defineProps<{
  audioData: string; // data URL
  isPlaying: boolean;
  currentTime?: number; // current playback time in seconds
  duration?: number; // total duration in seconds
}>();

const waveformRef = ref<HTMLDivElement | null>(null);
let wavesurfer: WaveSurfer | null = null;

onMounted(async () => {
  if (!waveformRef.value || !props.audioData) return;

  try {
    wavesurfer = WaveSurfer.create({
      container: waveformRef.value,
      waveColor: '#94a3b8',
      progressColor: '#3b82f6',
      cursorColor: '#3b82f6',
      barWidth: 2,
      barRadius: 1,
      barGap: 1,
      height: 60,
      normalize: true,
      interact: false, // Disable seeking - audio is controlled externally
      backend: 'WebAudio',
    });

    await wavesurfer.load(props.audioData);
    wavesurfer.seekTo(0);
  } catch (error) {
    console.error('Error initializing wavesurfer:', error);
  }
});

// Sync waveform progress with external audio playback using actual currentTime
watch(() => [props.currentTime, props.duration], ([currentTime, duration]) => {
  if (!wavesurfer || !duration || duration === 0 || currentTime === undefined) return;

  const progress = Math.min(Math.max(currentTime / duration, 0), 1);
  wavesurfer.seekTo(progress);
}, { immediate: true });

watch(() => props.audioData, async (newAudioData) => {
  if (!wavesurfer || !newAudioData) return;

  try {
    await wavesurfer.load(newAudioData);
    wavesurfer.seekTo(0);
  } catch (error) {
    console.error('Error loading new audio:', error);
  }
});

onBeforeUnmount(() => {
  if (wavesurfer) {
    wavesurfer.destroy();
    wavesurfer = null;
  }
});
</script>

<style scoped>
.audio-waveform-container {
  width: 100%;
  min-width: 200px;
  padding: 0.25rem 0;
}

.waveform {
  width: 100%;
  cursor: default;
}

/* Override wavesurfer styles for better integration */
:deep(.wavesurfer-wave) {
  cursor: default !important;
}
</style>

