
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:3000';

export interface AudioResponse {
  message: string;
  filename?: string;
  mimetype?: string;
  size?: number;
  reply?: {
    type?: string;
    text?: string;
    duration?: number;
    audio?: string; // base64 encoded audio
    audioMimetype?: string; // MIME type of the audio
  };
}

export async function sendAudioMessage(audioFile: File, _message: Message): Promise<AudioResponse> {
  try {
    const formData = new FormData();
    formData.append('audio', audioFile);

    const response = await fetch(`${API_BASE_URL}/upload-audio`, {
      method: 'POST',
      body: formData,
    });

    if (!response.ok) {
      throw new Error(`Failed to send audio: ${response.statusText}`);
    }

    const data = await response.json() as AudioResponse;
    return data;
  } catch (error) {
    console.error('Error sending audio message:', error);
    throw error;
  }
}

