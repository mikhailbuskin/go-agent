import { useState } from "react";
import { motion } from "framer-motion";

export default function App() {
  const [messages, setMessages] = useState([
    { role: "assistant", content: "Hello! How can I help you today?" },
  ]);
  const [input, setInput] = useState("");

  const sendMessage = async () => {
    if (!input.trim()) return;
    setMessages([...messages, { role: "user", content: input }]);
    setInput("");
    // Here you can add API call to your AI backend and push response
    var data = {message: input}
    var res = await fetch("http://localhost:8080/api/chat", { method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(data)})
    var json = await res.json();
    setMessages(prev => [
      ...prev,
      { role: "assistant", content: json.content },
    ]);
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
            className={`max-w-md p-3 rounded-2xl shadow-sm text-sm whitespace-pre-line ${
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