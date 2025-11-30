<template>
  <div class="audio-waveform-container border border-gray-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-700/50">
    <div ref="waveformRef" class="waveform"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch } from 'vue';
import WaveSurfer from 'wavesurfer.js';

const props = defineProps<{
  audioData: string; // data URL
  isPlaying: boolean;
}>();

const emit = defineEmits<{
  'play': [];
  'pause': [];
  'timeupdate': [time: number];
  'duration': [duration: number];
}>();

const waveformRef = ref<HTMLDivElement | null>(null);
let wavesurfer: WaveSurfer | null = null;

onMounted(async () => {
  if (!waveformRef.value || !props.audioData) return;

  // Use the same style for both user and assistant messages (the assistant style looks great)
  // This ensures consistency across all waveforms
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
      interact: true, // Enable seeking - user can click/drag to seek
      backend: 'WebAudio',
    });

    await wavesurfer.load(props.audioData);

    // Emit duration when loaded
    wavesurfer.on('decode', () => {
      emit('duration', wavesurfer!.getDuration());
    });

    // Emit time updates during playback
    wavesurfer.on('timeupdate', (time: number) => {
      emit('timeupdate', time);
    });

    // Handle play/pause events
    wavesurfer.on('play', () => {
      emit('play');
    });

    wavesurfer.on('pause', () => {
      emit('pause');
    });

    wavesurfer.on('finish', () => {
      emit('pause');
    });

    // If isPlaying is already true when wavesurfer loads, start playing
    if (props.isPlaying) {
      wavesurfer.play();
    }
  } catch (error) {
    console.error('Error initializing wavesurfer:', error);
  }
});

// Sync play/pause state
watch(() => props.isPlaying, (isPlaying) => {
  if (!wavesurfer) {
    // If wavesurfer isn't ready yet, wait for it
    if (isPlaying) {
      // Retry after a short delay
      setTimeout(() => {
        if (wavesurfer && isPlaying && wavesurfer.isPlaying() === false) {
          wavesurfer.play();
        }
      }, 100);
    }
    return;
  }

  if (isPlaying && wavesurfer.isPlaying() === false) {
    wavesurfer.play();
  } else if (!isPlaying && wavesurfer.isPlaying() === true) {
    wavesurfer.pause();
  }
}, { immediate: true });

watch(() => props.audioData, async (newAudioData) => {
  if (!wavesurfer || !newAudioData) return;

  try {
    await wavesurfer.load(newAudioData);
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
  max-width: 100%;
  padding: 0.5rem;
  overflow: hidden;
}

.waveform {
  width: 100%;
  max-width: 100%;
  cursor: pointer;
}

/* Override wavesurfer styles for better integration */
:deep(.wavesurfer-wave) {
  cursor: pointer !important;
}
</style>

