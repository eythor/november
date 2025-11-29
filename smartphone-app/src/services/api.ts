import type { Message } from '../stores/chat';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:3000';

export interface ChatResponse {
  message: string;
  received?: {
    id: string;
    role: string;
    type: string;
    text?: string;
    createdAt: number;
  };
  reply?: {
    type?: string;
    text?: string;
    duration?: number;
  };
}

export async function sendChatMessage(message: Message): Promise<ChatResponse> {
  try {
    const response = await fetch(`${API_BASE_URL}/chat`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        id: message.id,
        role: message.role,
        type: message.type,
        text: message.text,
        createdAt: message.createdAt,
      }),
    });

    if (!response.ok) {
      throw new Error(`Failed to send message: ${response.statusText}`);
    }

    const data = await response.json() as ChatResponse;
    return data;
  } catch (error) {
    console.error('Error sending chat message:', error);
    throw error;
  }
}

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

