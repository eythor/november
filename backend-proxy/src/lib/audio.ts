import ffmpeg from 'fluent-ffmpeg';
import { writeFile, readFile, unlink } from 'fs/promises';
import { join } from 'path';
import { tmpdir } from 'os';

// Supported formats that don't need conversion
export const SUPPORTED_FORMATS = ['audio/wav', 'audio/mpeg', 'audio/mp3'];

/**
 * Convert audio buffer to base64 string
 */
export function encodeAudioToBase64(audioBuffer: Buffer): string {
  return audioBuffer.toString('base64');
}

/**
 * Check if audio format needs conversion
 */
export function needsConversion(mimetype: string): boolean {
  return !SUPPORTED_FORMATS.includes(mimetype);
}

/**
 * Get file extension from MIME type
 */
function getFileExtension(mimetype: string): string {
  const extensionMap: Record<string, string> = {
    'audio/webm': 'webm',
    'audio/ogg': 'ogg',
    'audio/m4a': 'm4a',
    'audio/aac': 'aac',
  };
  return extensionMap[mimetype] || 'tmp';
}

/**
 * Convert audio buffer to WAV format
 */
export async function convertAudioToWav(audioBuffer: Buffer, inputFormat: string): Promise<Buffer> {
  const extension = getFileExtension(inputFormat);
  const timestamp = Date.now();
  const inputPath = join(tmpdir(), `input-${timestamp}.${extension}`);
  const outputPath = join(tmpdir(), `output-${timestamp}.wav`);

  try {
    // Write input buffer to temporary file
    await writeFile(inputPath, audioBuffer);

    // Convert to WAV using ffmpeg
    await new Promise<void>((resolve, reject) => {
      ffmpeg(inputPath)
        .toFormat('wav')
        .audioCodec('pcm_s16le')
        .audioFrequency(16000)
        .audioChannels(1)
        .on('error', (err) => {
          reject(new Error(`FFmpeg error: ${err.message}`));
        })
        .on('end', () => {
          resolve();
        })
        .save(outputPath);
    });

    // Read converted file
    const convertedBuffer = await readFile(outputPath);

    // Clean up temporary files
    await Promise.all([
      unlink(inputPath).catch(() => {}),
      unlink(outputPath).catch(() => {})
    ]);

    return convertedBuffer;
  } catch (error) {
    // Clean up temporary files on error
    await Promise.all([
      unlink(inputPath).catch(() => {}),
      unlink(outputPath).catch(() => {})
    ]);
    throw error;
  }
}

/**
 * Map MIME types to OpenRouter audio formats
 */
export function getAudioFormat(mimetype: string): string {
  // Always use wav after conversion, or if already wav/mp3
  if (mimetype === 'audio/mpeg' || mimetype === 'audio/mp3') {
    return 'mp3';
  }
  return 'wav';
}

