"use client";

import { useState, useEffect, useRef } from "react";
import type { SerialMessage } from "@/lib/devices/types";
import { getSerialMessages } from "@/lib/devices/client";

interface SerialMonitorProps {
  deviceId: string;
}

export function SerialMonitor({ deviceId }: SerialMonitorProps) {
  const [messages, setMessages] = useState<SerialMessage[]>([]);
  const [connected, setConnected] = useState(false);
  const [autoScroll, setAutoScroll] = useState(true);
  const [input, setInput] = useState("");
  const [since, setSince] = useState<string | undefined>();
  const bottomRef = useRef<HTMLDivElement>(null);
  const pollInterval = useRef<NodeJS.Timeout | undefined>(undefined);

  // Poll for new messages
  useEffect(() => {
    async function fetch() {
      try {
        const msgs = await getSerialMessages(deviceId, since);
        if (msgs.length > 0) {
          setMessages((prev) => {
            const updated = [...prev, ...msgs];
            // Keep last 500 messages
            return updated.slice(-500);
          });
          setSince(msgs[msgs.length - 1].timestamp);
        }
      } catch {
        setConnected(false);
      }
    }

    // Initial fetch
    fetch();
    setConnected(true);

    // Poll every second
    pollInterval.current = setInterval(fetch, 1000);
    return () => {
      if (pollInterval.current) clearInterval(pollInterval.current);
    };
  }, [deviceId, since]);

  // Auto-scroll
  useEffect(() => {
    if (autoScroll && bottomRef.current) {
      bottomRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [messages, autoScroll]);

  async function handleSend() {
    if (!input.trim()) return;
    try {
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080"}/api/v1/devices/${encodeURIComponent(deviceId)}/serial`, {
        method: "POST",
        headers: { "Content-Type": "text/plain" },
        body: input + "\n",
      });
      if (!res.ok) throw new Error("Failed to send");
      setInput("");
    } catch {
      // ignore send errors for now
    }
  }

  function handleClear() {
    setMessages([]);
    setSince(undefined);
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <div className={`w-2 h-2 rounded-full ${connected ? "bg-green-500" : "bg-red-500"}`} />
            <span className="text-sm text-zinc-600">
              {connected ? "Connected" : "Disconnected"}
            </span>
          </div>
          <label className="flex items-center gap-2 text-sm text-zinc-600">
            <input
              type="checkbox"
              checked={autoScroll}
              onChange={(e) => setAutoScroll(e.target.checked)}
              className="rounded border-zinc-300 text-zinc-900 focus:ring-zinc-900"
            />
            Auto-scroll
          </label>
        </div>
        <button
          type="button"
          onClick={handleClear}
          className="px-3 py-1 text-sm font-medium text-zinc-700 border border-zinc-300 rounded-md hover:bg-zinc-50"
        >
          Clear
        </button>
      </div>

      {/* Message view */}
      <div className="rounded-lg border border-zinc-200 bg-black p-4 h-96 overflow-y-auto font-mono text-xs">
        {messages.length === 0 ? (
          <p className="text-zinc-500">Waiting for messages...</p>
        ) : (
          <div className="space-y-1">
            {messages.map((msg, idx) => (
              <div key={idx} className="flex gap-2">
                <span className="text-zinc-500">
                  [{new Date(msg.timestamp).toLocaleTimeString()}]
                </span>
                <span className={msg.direction === "tx" ? "text-green-400" : "text-blue-400"}>
                  {msg.direction === "tx" ? "TX" : "RX"}:
                </span>
                <span className="text-zinc-100">{msg.data}</span>
              </div>
            ))}
            <div ref={bottomRef} />
          </div>
        )}
      </div>

      {/* Input */}
      <div className="flex gap-2">
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              handleSend();
            }
          }}
          placeholder="Type a message and press Enter..."
          className="flex-1 rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
        />
        <button
          type="button"
          onClick={handleSend}
          disabled={!connected}
          className="px-4 py-2 text-sm font-medium text-white bg-zinc-900 rounded-md hover:bg-zinc-800 disabled:opacity-50"
        >
          Send
        </button>
      </div>
    </div>
  );
}
