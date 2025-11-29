import { readFile, unlink } from 'fs/promises';
import { join } from 'path';
import { tmpdir } from 'os';
// @ts-ignore - gtts doesn't have TypeScript types
import gtts from 'gtts';

/**
 * Generate TTS audio from text using Google TTS
 */
export async function textToSpeech(text: string): Promise<Buffer> {
  return new Promise((resolve, reject) => {
    const timestamp = Date.now();
    const outputPath = join(tmpdir(), `tts-${timestamp}.mp3`);
    
    const tts = new gtts(text, 'en');
    tts.save(outputPath, async (err: Error | null) => {
      if (err) {
        reject(new Error(`TTS error: ${err.message}`));
        return;
      }
      
      try {
        const audioBuffer = await readFile(outputPath);
        // Clean up temporary file
        await unlink(outputPath).catch(() => {});
        resolve(audioBuffer);
      } catch (error) {
        // Clean up on error
        await unlink(outputPath).catch(() => {});
        reject(error);
      }
    });
  });
}

/**
 * Calculate estimated audio duration based on word count
 * Uses rough estimate of ~150 words per minute
 */
export function estimateAudioDuration(text: string): number {
  const wordCount = text.split(/\s+/).length;
  return Math.ceil((wordCount / 150) * 60);
}

