import { defineStore } from "pinia";
import { ref } from "vue";

export type Role = "user" | "assistant";
export type MsgType = "text" | "audio";

export interface Message {
  id: string;
  role: Role;
  type: MsgType;
  text?: string;
  duration?: number; // for fake audio messages
  audioData?: string; // data URL for audio playback
  createdAt: number;
  sentAt?: number;
  delivered?: boolean;
}

export const useChatStore = defineStore("chat", () => {
  const messages = ref<Message[]>([]);

  function addMessage(msg: Partial<Message>) {
    const m: Message = {
      id: msg.id ?? `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`,
      role: msg.role ?? "user",
      type: msg.type ?? "text",
      text: msg.text,
      duration: msg.duration,
      audioData: msg.audioData,
      createdAt: msg.createdAt ?? Date.now(),
      sentAt: msg.sentAt,
      delivered: msg.delivered ?? false,
    };
    messages.value.push(m);
    // Return the index for patching
    return m.id;
  }

  function patchMessage(id: string, patch: Partial<Message>) {
    const idx = messages.value.findIndex((m) => m.id === id);
    if (idx !== -1) {
      messages.value[idx] = { ...messages.value[idx], ...patch };
    }
  }

  return { messages, addMessage, patchMessage };
});
