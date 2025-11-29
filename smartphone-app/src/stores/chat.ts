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
      createdAt: msg.createdAt ?? Date.now(),
      sentAt: msg.sentAt,
      delivered: msg.delivered ?? false,
    };
    messages.value.push(m);
    return m;
  }

  return { messages, addMessage };
});
