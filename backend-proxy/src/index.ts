import 'dotenv/config';
import express, { Request, Response } from 'express';
import multer from 'multer';
import cors from 'cors';
import { OpenRouter } from '@openrouter/sdk';
import ffmpeg from 'fluent-ffmpeg';
import ffmpegStatic from 'ffmpeg-static';
import { writeFile, readFile, unlink } from 'fs/promises';
import { existsSync } from 'fs';
import { join } from 'path';
import { tmpdir } from 'os';
// @ts-ignore - gtts doesn't have TypeScript types
import gtts from 'gtts';

const app = express();
const PORT = process.env.PORT || 3000;

// Set ffmpeg path and verify it exists
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

// Initialize OpenRouter
const openRouter = new OpenRouter({
  apiKey: process.env.OPENROUTER_API_KEY || '',
});

// Helper function to convert audio buffer to base64
function encodeAudioToBase64(audioBuffer: Buffer): string {
  return audioBuffer.toString('base64');
}

// Supported formats that don't need conversion
const SUPPORTED_FORMATS = ['audio/wav', 'audio/mpeg', 'audio/mp3'];

// Helper function to check if format needs conversion
function needsConversion(mimetype: string): boolean {
  return !SUPPORTED_FORMATS.includes(mimetype);
}

// Helper function to get file extension from MIME type
function getFileExtension(mimetype: string): string {
  const extensionMap: Record<string, string> = {
    'audio/webm': 'webm',
    'audio/ogg': 'ogg',
    'audio/m4a': 'm4a',
    'audio/aac': 'aac',
  };
  return extensionMap[mimetype] || 'tmp';
}

// Helper function to convert audio buffer to WAV format
async function convertAudioToWav(audioBuffer: Buffer, inputFormat: string): Promise<Buffer> {
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

// Helper function to map MIME types to OpenRouter audio formats
function getAudioFormat(mimetype: string): string {
  // Always use wav after conversion, or if already wav/mp3
  if (mimetype === 'audio/mpeg' || mimetype === 'audio/mp3') {
    return 'mp3';
  }
  return 'wav';
}

// Helper function to generate TTS audio from text
async function textToSpeech(text: string): Promise<Buffer> {
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

// Enable CORS for frontend
app.use(cors());
// Parse JSON bodies
app.use(express.json());

// Configure multer for file uploads
// Store files in memory for now
const upload = multer({
  storage: multer.memoryStorage(),
  limits: {
    fileSize: 50 * 1024 * 1024 // 50MB limit
  },
  fileFilter: (req, file, cb) => {
    // Accept audio files
    const audioMimeTypes = [
      'audio/mpeg',
      'audio/mp3',
      'audio/wav',
      'audio/webm',
      'audio/ogg',
      'audio/m4a',
      'audio/aac'
    ];
    
    if (audioMimeTypes.includes(file.mimetype)) {
      cb(null, true);
    } else {
      cb(new Error('Invalid file type. Only audio files are allowed.') as any, false);
    }
  }
});

// Endpoint to receive chat messages (text)
app.post('/chat', (req: Request, res: Response) => {
  try {
    const { id, role, type, text, createdAt } = req.body;

    // Acknowledge message received
    console.log('📨 Message received');

    // Return a simple reply for testing
    // TODO: Replace with actual AI/chat service integration
    res.json({
      message: 'Chat message received successfully',
      received: {
        id,
        role,
        type,
        text,
        createdAt
      },
      reply: {
        type: 'text',
        text: `I received your message: "${text}". This is a placeholder reply - integrate your chat service here.`
      }
    });
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error occurred';
    console.error('❌ Error processing chat message:', errorMessage);
    res.status(500).json({ error: errorMessage });
  }
});

// Endpoint to receive audio file
app.post('/upload-audio', upload.single('audio'), async (req: Request, res: Response) => {
  try {
    if (!req.file) {
      console.error('❌ No audio file provided in request');
      return res.status(400).json({ error: 'No audio file provided' });
    }

    // Acknowledge audio received
    console.log('🎤 Audio received');

    // Convert audio to supported format if needed
    let audioBuffer = req.file.buffer;
    let finalMimetype = req.file.mimetype;
    
    if (needsConversion(req.file.mimetype)) {
      console.log(`🔄 Converting ${req.file.mimetype} to WAV...`);
      audioBuffer = await convertAudioToWav(req.file.buffer, req.file.mimetype);
      finalMimetype = 'audio/wav';
    }

    // Convert audio buffer to base64
    const base64Audio = encodeAudioToBase64(audioBuffer);
    const audioFormat = getAudioFormat(finalMimetype);

    // Send audio to OpenRouter for transcription
    const result = await openRouter.chat.send({
      model: 'google/gemini-2.5-flash',
      messages: [
        {
          role: 'user',
          content: [
            {
              type: 'text',
              text: 'Please transcribe this audio file.',
            },
            {
              type: 'input_audio',
              inputAudio: {
                data: base64Audio,
                format: audioFormat,
              },
            } as any, // Type assertion needed as SDK types may not include input_audio yet
          ],
        },
      ],
      stream: false,
    });

    // Extract transcription from response
    const content = result.choices?.[0]?.message?.content;
    let transcription = 'Transcription not available';
    
    if (typeof content === 'string') {
      transcription = content;
    } else if (Array.isArray(content)) {
      const textItem = content.find((item: any) => item.type === 'text' && 'text' in item) as any;
      if (textItem?.text) {
        transcription = textItem.text;
      }
    }
    
    // Show minimal excerpt (first 100 characters)
    const excerpt = transcription.length > 100 
      ? transcription.substring(0, 100) + '...' 
      : transcription;
    console.log('✅ Transcript:', excerpt);

    // Generate TTS audio from transcription
    console.log('🔊 Generating audio response...');
    const ttsAudioBuffer = await textToSpeech(transcription);
    const audioBase64 = encodeAudioToBase64(ttsAudioBuffer);
    
    // Calculate approximate duration (rough estimate: ~150 words per minute)
    const wordCount = transcription.split(/\s+/).length;
    const estimatedDuration = Math.ceil((wordCount / 150) * 60);

    res.json({
      message: 'Audio file received and transcribed successfully',
      filename: req.file.originalname,
      mimetype: req.file.mimetype,
      size: req.file.size,
      transcription: transcription,
      reply: {
        type: 'audio',
        text: transcription,
        audio: audioBase64,
        audioMimetype: 'audio/mpeg',
        duration: estimatedDuration
      }
    });
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error occurred';
    console.error('❌ Error processing audio message:', errorMessage);
    res.status(500).json({ error: errorMessage });
  }
});

// Health check endpoint
app.get('/health', (req: Request, res: Response) => {
  res.json({
    status: 'ok',
    timestamp: new Date().toISOString(),
    uptime: process.uptime(),
    environment: process.env.NODE_ENV || 'development'
  });
});

app.listen(PORT, () => {
  console.log(`Server running on http://localhost:${PORT}`);
});

