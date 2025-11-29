import { TextToSpeechClient } from "@google-cloud/text-to-speech";

// Initialize the Google Cloud TTS client
// Credentials can be provided via:
// 1. GOOGLE_APPLICATION_CREDENTIALS environment variable pointing to a service account key file
// 2. Or the client will use Application Default Credentials (ADC)
const client = new TextToSpeechClient();

/**
 * Generate TTS audio from text using Google Cloud Text-to-Speech
 */
export async function textToSpeech(text: string): Promise<Buffer> {
  try {
    const request = {
      input: { text },
      voice: {
        languageCode: "en-US",
        name: "en-US-Wavenet-D", //
        ssmlGender: "NEUTRAL" as const,
      },
      audioConfig: {
        audioEncoding: "MP3" as const,
        speakingRate: 1.2,
        pitch: 0.0,
        volumeGainDb: 0.0,
        effectsProfileId: ["telephony-class-application"],
      },
    };

    const [response] = await client.synthesizeSpeech(request);

    if (!response.audioContent) {
      throw new Error("No audio content received from Google Cloud TTS");
    }

    // Convert Uint8Array to Buffer
    return Buffer.from(response.audioContent);
  } catch (error) {
    const errorMessage =
      error instanceof Error ? error.message : "Unknown error";
    throw new Error(`TTS error: ${errorMessage}`);
  }
}

/**
 * Strip markdown formatting from text to make it suitable for TTS
 * Removes markdown syntax while preserving the actual content
 */
export function stripMarkdownForTTS(text: string): string {
  return (
    text
      // Remove markdown code blocks (```code``` or `code`)
      .replace(/```[\s\S]*?```/g, "")
      .replace(/`[^`]*`/g, "")
      // Remove markdown links but keep the text: [text](url) -> text
      .replace(/\[([^\]]+)\]\([^\)]+\)/g, "$1")
      // Remove markdown images: ![alt](url) -> alt
      .replace(/!\[([^\]]*)\]\([^\)]+\)/g, "$1")
      // Remove markdown headers (# Header -> Header)
      .replace(/^#{1,6}\s+(.+)$/gm, "$1")
      // Remove markdown bold/italic markers (**text** -> text, *text* -> text)
      .replace(/\*\*([^\*]+)\*\*/g, "$1")
      .replace(/\*([^\*]+)\*/g, "$1")
      .replace(/__([^_]+)__/g, "$1")
      .replace(/_([^_]+)_/g, "$1")
      // Convert markdown list items to natural speech
      .replace(/^[\s]*[-*+]\s+(.+)$/gm, "$1")
      .replace(/\s*[-*+]\s+/g, " ")
      // Remove markdown horizontal rules and blockquotes
      .replace(/^[-*]{3,}$/gm, "")
      .replace(/^>\s+(.+)$/gm, "$1")
      // Clean up whitespace
      .replace(/\n{3,}/g, "\n\n")
      .replace(/[ \t]+/g, " ")
      .replace(/\s+\n|\n\s+/g, "\n")
      // Remove non-printable characters (keep only printable ASCII)
      .replace(/[^\x20-\x7E\n]/g, "")
      .trim()
  );
}

/**
 * Calculate estimated audio duration based on word count
 * Uses rough estimate of ~150 words per minute
 */
export function estimateAudioDuration(text: string): number {
  const wordCount = text.split(/\s+/).length;
  return Math.ceil((wordCount / 150) * 60);
}
