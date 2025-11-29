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
 * Convert mathematical and comparison symbols to their word equivalents
 * This ensures symbols like ≥, ≤, etc. are spoken correctly in TTS
 */
function convertSymbolsToWords(text: string): string {
  return (
    text
      // Comparison symbols - compound symbols first for clarity
      .replace(/≥/g, " greater than or equal to ")
      .replace(/≤/g, " less than or equal to ")
      .replace(/≠/g, " not equal to ")
      .replace(/≈/g, " approximately equal to ")
      // Single character comparisons (these are separate Unicode chars, so order doesn't matter)
      .replace(/>/g, " greater than ")
      .replace(/</g, " less than ")
      .replace(/=/g, " equal to ")
      // Mathematical symbols
      .replace(/±/g, " plus or minus ")
      .replace(/×/g, " times ")
      .replace(/÷/g, " divided by ")
      .replace(/∑/g, " sum of ")
      .replace(/∏/g, " product of ")
      .replace(/√/g, " square root of ")
      .replace(/∞/g, " infinity ")
      .replace(/°/g, " degrees ")
      // Greek letters (common ones)
      .replace(/α/g, " alpha ")
      .replace(/β/g, " beta ")
      .replace(/γ/g, " gamma ")
      .replace(/δ/g, " delta ")
      .replace(/π/g, " pi ")
      .replace(/μ/g, " mu ")
      .replace(/σ/g, " sigma ")
      // Clean up multiple spaces
      .replace(/\s+/g, " ")
  );
}

/**
 * Strip markdown formatting from text to make it suitable for TTS
 * Removes markdown syntax while preserving the actual content
 */
export function stripMarkdownForTTS(text: string): string {
  let cleaned = text
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
    .replace(/^>\s+(.+)$/gm, "$1");

  // Convert symbols to words BEFORE removing non-ASCII characters
  // This preserves important information like ≥, ≤, etc.
  cleaned = convertSymbolsToWords(cleaned);

  return (
    cleaned
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
