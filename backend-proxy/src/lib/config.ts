import ffmpeg from 'fluent-ffmpeg';
import ffmpegStatic from 'ffmpeg-static';
import { existsSync } from 'fs';
import { OpenRouter } from '@openrouter/sdk';

/**
 * Initialize and configure ffmpeg
 * Exits process if ffmpeg is not available
 */
export function initializeFFmpeg(): void {
  if (ffmpegStatic) {
    if (!existsSync(ffmpegStatic)) {
      console.error(`❌ FFmpeg binary not found at: ${ffmpegStatic}`);
      console.error('   Please run: node node_modules/ffmpeg-static/install.js');
      process.exit(1);
    }
    ffmpeg.setFfmpegPath(ffmpegStatic);
  } else {
    console.error('❌ ffmpeg-static not found. Please install it.');
    process.exit(1);
  }
}

/**
 * Initialize OpenRouter client
 */
export function initializeOpenRouter(): OpenRouter {
  return new OpenRouter({
    apiKey: process.env.OPENROUTER_API_KEY || '',
  });
}

