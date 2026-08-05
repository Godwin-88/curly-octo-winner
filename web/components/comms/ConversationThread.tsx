'use client';

import { ConversationMessage } from '@/lib/api';

interface Props {
  messages: ConversationMessage[];
}

export default function ConversationThread({ messages }: Props) {
  return (
    <div className="flex-1 overflow-y-auto py-4 space-y-3">
      {messages.map((msg) => {
        const isInbound = msg.direction === 'inbound';
        const text = (msg.content as any)?.text || '';
        return (
          <div
            key={msg.id}
            className={`flex ${isInbound ? 'justify-start' : 'justify-end'}`}
          >
            <div
              className={`max-w-[70%] rounded-lg px-4 py-2 ${
                isInbound
                  ? 'bg-white border border-gray-200'
                  : 'bg-blue-600 text-white'
              }`}
            >
              <p className="text-sm">{text}</p>
              <p className={`text-xs mt-1 ${isInbound ? 'text-gray-400' : 'text-blue-100'}`}>
                {new Date(msg.timestamp).toLocaleTimeString()}
              </p>
            </div>
          </div>
        );
      })}
    </div>
  );
}