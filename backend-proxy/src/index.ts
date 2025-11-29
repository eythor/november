import express, { Request, Response } from 'express';
import multer from 'multer';
import cors from 'cors';

const app = express();
const PORT = process.env.PORT || 3000;

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
      cb(new Error('Invalid file type. Only audio files are allowed.'), false);
    }
  }
});

// Endpoint to receive chat messages (text)
app.post('/chat', (req: Request, res: Response) => {
  try {
    const { id, role, type, text, createdAt } = req.body;

    // Log received message details
    console.log('📨 Received chat message:');
    console.log('  ID:', id);
    console.log('  Role:', role);
    console.log('  Type:', type);
    console.log('  Text:', text);
    console.log('  Created At:', new Date(createdAt).toISOString());
    console.log('  Full payload:', JSON.stringify(req.body, null, 2));

    // For now, do nothing with the payload
    // Just acknowledge receipt
    
    res.json({
      message: 'Chat message received successfully',
      received: {
        id,
        role,
        type,
        text,
        createdAt
      }
    });
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error occurred';
    console.error('❌ Error processing chat message:', errorMessage);
    res.status(500).json({ error: errorMessage });
  }
});

// Endpoint to receive audio file
app.post('/upload-audio', upload.single('audio'), (req: Request, res: Response) => {
  try {
    if (!req.file) {
      console.error('❌ No audio file provided in request');
      return res.status(400).json({ error: 'No audio file provided' });
    }

    // Log received audio file details
    console.log('🎤 Received audio message:');
    console.log('  Filename:', req.file.originalname);
    console.log('  MIME Type:', req.file.mimetype);
    console.log('  Size:', `${(req.file.size / 1024).toFixed(2)} KB`);
    console.log('  Buffer length:', req.file.buffer.length, 'bytes');
    if (req.body) {
      console.log('  Additional body data:', JSON.stringify(req.body, null, 2));
    }

    // For now, do nothing with the file
    // The file is available in req.file.buffer
    
    res.json({
      message: 'Audio file received successfully',
      filename: req.file.originalname,
      mimetype: req.file.mimetype,
      size: req.file.size
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

