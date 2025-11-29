import 'dotenv/config';
import express, { Request, Response } from 'express';
import multer from 'multer';
import cors from 'cors';
import { initializeFFmpeg, initializeOpenRouter } from './lib/config.js';
import { encodeAudioToBase64, needsConversion, convertAudioToWav } from './lib/audio.js';
import { textToSpeech, estimateAudioDuration } from './lib/tts.js';
import { transcribeAudio } from './lib/openrouter.js';

const app = express();
const PORT = process.env.PORT || 3000;
const BACKEND_URL = process.env.BACKEND_URL || 'http://localhost:8080';

// Initialize ffmpeg and OpenRouter
initializeFFmpeg();
const openRouter = initializeOpenRouter();

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

// Endpoint to receive audio file
app.post('/upload-audio', upload.single('audio'), async (req: Request, res: Response) => {
  try {
    if (!req.file) {
      console.error('No audio file provided in request');
      return res.status(400).json({ error: 'No audio file provided' });
    }

    console.log(`Audio received: ${req.file.originalname} (${req.file.size} bytes, ${req.file.mimetype})`);

    // Convert audio to supported format if needed
    let audioBuffer = req.file.buffer;
    let finalMimetype = req.file.mimetype;
    
    if (needsConversion(req.file.mimetype)) {
      console.log(`Converting ${req.file.mimetype} to WAV`);
      audioBuffer = await convertAudioToWav(req.file.buffer, req.file.mimetype);
      finalMimetype = 'audio/wav';
    }

    // Transcribe audio using OpenRouter
    console.log('Transcribing audio...');
    const transcription = await transcribeAudio(openRouter, audioBuffer, finalMimetype);
    
    const excerpt = transcription.length > 100 
      ? transcription.substring(0, 100) + '...' 
      : transcription;
    console.log(`Transcription: ${excerpt}`);

    // Send transcription to Go backend for processing
    let responseText = transcription; // Fallback to transcription if backend fails
    try {
      console.log(`Sending query to backend at ${BACKEND_URL}/query`);
      const backendResponse = await fetch(`${BACKEND_URL}/query`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ query: transcription }),
      });

      if (backendResponse.ok) {
        const backendData = await backendResponse.json() as { response?: string; error?: string };
        if (backendData.response) {
          responseText = backendData.response;
          console.log('Backend response received');
        } else {
          console.warn('Backend response missing "response" field, using transcription');
        }
      } else {
        const errorData = await backendResponse.json().catch(() => ({})) as { error?: string };
        console.warn(`Backend returned ${backendResponse.status}: ${errorData.error || 'Unknown error'}`);
        console.warn('Using transcription as fallback');
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error';
      console.error(`Failed to connect to backend: ${errorMessage}`);
      console.warn('Using transcription as fallback');
    }

    // Generate TTS audio from backend response
    console.log('Generating audio response...');
    const ttsAudioBuffer = await textToSpeech(responseText);
    const audioBase64 = encodeAudioToBase64(ttsAudioBuffer);
    
    const estimatedDuration = estimateAudioDuration(responseText);

    res.json({
      message: 'Audio file received and transcribed successfully',
      filename: req.file.originalname,
      mimetype: req.file.mimetype,
      size: req.file.size,
      transcription: transcription,
      reply: {
        type: 'audio',
        text: responseText,
        audio: audioBase64,
        audioMimetype: 'audio/mpeg',
        duration: estimatedDuration
      }
    });
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error occurred';
    console.error(`Failed to process audio message: ${errorMessage}`);
    res.status(500).json({ error: errorMessage });
  }
});

// Health check endpoint
app.get('/health', (req: Request, res: Response) => {
  res.json({
    status: 'ok',
    timestamp: new Date().toISOString(),
    environment: process.env.NODE_ENV || 'development'
  });
});

app.listen(PORT, () => {
  console.log(`Server running on http://localhost:${PORT}`);
});

