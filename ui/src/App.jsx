import { useState, useEffect } from "react";
import { motion } from "framer-motion";

export default function App() {
  const [messages, setMessages] = useState([]);


  const loadMessages = async () => {
    // retrieve messages memory
    // method is GET not POST because not sending anything. Just retrieving data
    var res = await fetch("http://localhost:8080/api/memory", { method: 'GET', headers: {'Content-Type': 'application/json'}}) // no body because no message needed to be sent. Just a signal to retrieve the messages
    var outputData = await res.json();
    setMessages(outputData.messages);
  }

  useEffect(() => {
    loadMessages();
  },[])

  const [input, setInput] = useState("");

  const sendMessage = async () => {
    if (!input.trim()) return;

    // Add user's message
    setMessages([...messages, { role: "user", content: input }]);
    setInput("");

    const data = { message: input };
    const res = await fetch("http://localhost:8080/api/chat", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(data),
    });

    const reader = res.body.getReader();
    const decoder = new TextDecoder();
    let assistantMessage = "";

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      // Decode the chunk into a string
      const chunk = decoder.decode(value, { stream: true });

      // Split by newlines (each line is a separate JSON object)
      const lines = chunk.split("\n").filter(Boolean);

      for (const line of lines) {
        try {
          const parsed = JSON.parse(line);
          assistantMessage += parsed.content;

          // Update the UI progressively
          setMessages((prev) => {
            // Replace the last assistant message (or append if none yet)
            const last = prev[prev.length - 1];
            if (last && last.role === "assistant") {
              const newPrev = [...prev];
              newPrev[newPrev.length - 1] = { ...last, content: assistantMessage };
              return newPrev;
            } else {
              return [...prev, { role: "assistant", content: assistantMessage }];
            }
          });
        } catch (err) {
          console.error("Failed to parse chunk:", line, err);
        }
      }
    }
  };

  return (
    <div className="h-screen w-screen flex flex-col bg-gray-100 text-gray-900">
      {/* Header */}
      <header className="p-4 bg-white shadow-md font-semibold text-lg text-center">
        AI Agent
      </header>

      {/* Chat Area */}
      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        {messages.map((msg, i) => (
          <motion.div
            key={i}
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.2 }}
            className={`max-w-1/2 p-3 rounded-2xl shadow-sm text-sm whitespace-pre-line ${
              msg.role === "user"
                ? "ml-auto bg-blue-500 text-white"
                : "mr-auto bg-white"
            }`}
          >
            {msg.content}
          </motion.div>
        ))}
      </div>

      {/* Input */}
      <div className="p-3 bg-white border-t flex items-center gap-2">
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && sendMessage()}
          placeholder="Type your message..."
          className="flex-1 p-2 rounded-xl border focus:outline-none focus:ring focus:ring-blue-300"
        />
        <button
          onClick={sendMessage}
          className="px-4 py-2 bg-blue-500 text-white rounded-xl hover:bg-blue-600 transition"
        >
          Send
        </button>
      </div>
    </div>
  );
}