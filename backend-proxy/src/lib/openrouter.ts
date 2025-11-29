import { OpenRouter } from '@openrouter/sdk';
import { encodeAudioToBase64, getAudioFormat } from './audio.js';

/**
 * Transcribe audio using OpenRouter
 */
export async function transcribeAudio(
  openRouter: OpenRouter,
  audioBuffer: Buffer,
  mimetype: string
): Promise<string> {
  const base64Audio = encodeAudioToBase64(audioBuffer);
  const audioFormat = getAudioFormat(mimetype);

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
  
  return transcription;
}

