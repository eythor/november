import type { Message } from '../stores/chat';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:3000';

export async function sendChatMessage(message: Message): Promise<void> {
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

    // For now, we don't need to do anything with the response
    await response.json();
  } catch (error) {
    console.error('Error sending chat message:', error);
    throw error;
  }
}

export async function sendAudioMessage(audioFile: File, message: Message): Promise<void> {
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

    // For now, we don't need to do anything with the response
    await response.json();
  } catch (error) {
    console.error('Error sending audio message:', error);
    throw error;
  }
}

