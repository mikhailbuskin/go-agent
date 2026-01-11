import { useState, useEffect, useRef } from "react";
import { motion } from "framer-motion";

export default function App() {
  let [messages, setMessages] = useState([]);
  const [isRunning, setIsRunning] = useState(false);

  let continuation = false;

  const endOfMessagesRef = useRef(null);

  const scrollToBottom = () => {
    endOfMessagesRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]); // Dependency array: scroll when 'messages' array changes

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
    setIsRunning(true);

    // When continue button is clicked, a an empty prompt gets sent and "continuation" var works as a bypass

    // Input.trim() gets rid of any spaces. String == false when "" and == true when contains something
    // Checks if an empty prompt was simply sent
    if(!input.trim() && continuation == false) {// if the string is empty and continuation is false
      setMessages([...messages, { role: "error", content: "Prompt Required" }]);
      setIsRunning(false);
      return;
    }

    // If a prompt was sent OR continuation button was pressed sending a "" prompt
    var fInput;
    if(continuation == true) { // prompt is "" and we are continuing the chat
      fInput = "";
    } else { // if prompt 
      // Add user's message
      messages = [...messages, { role: "user", content: input }]; // to overcome the problem of setMessages() not changing the variable you manually do it
      setMessages(messages); // problem: messages variable not getting updated. And in the rest of the function it uses the old version of messages without the "user" message added. solution: one line above
      fInput = input;
    }

    setInput("");

    const data = { message: fInput };
    const res = await fetch("http://localhost:8080/api/chat", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(data),
    });

    const reader = res.body.getReader();
    const decoder = new TextDecoder();
    let assistantMessage = "";
    let toolMessage = "";

    let breakCycle = false;

    while (true) {
      const { done, value } = await reader.read(); // This reads the new non read messages only. Skips the ones it has already read. Waits for new mesages
      if (done) break;

      // scroll
      setTimeout(() => window.scrollToBottom ? window.scrollToBottom():null, 5);

      // Decode the chunk into a string
      const chunk = decoder.decode(value, { stream: true });

      // Split by newlines (each line is a separate JSON object)
      const lines = chunk.split("\n").filter(Boolean);

      for (const line of lines) {
        try {
          const parsed = JSON.parse(line.slice(6)); // Get rid of "data: " in the beginning
          assistantMessage += parsed.content;
          toolMessage += parsed.content;
          //debugger;
          let changeMessages = (prev) => { // prev is the current messages
            // Replace the last assistant message (or append if none yet)
            const last = prev[prev.length - 1];
            if(parsed.role == "assistant") {
              //if(continuation==false) { setInput(""); } // only erase the textbox if you sent a message
              toolMessage = "";
              // When assistantMessage += parsed.content is HERE. It adds it twice for some reason---------------------------------------------------------------------------------------------------review
              if (last && last.role === "assistant") { // After first time it adds onto that message when streaming
                const newPrev = [...prev];
                newPrev[newPrev.length - 1] = { ...last, content: assistantMessage }; // this takes all the attributes of the last msg (so the role and content) but then overrides the content with the updated message
                return newPrev; // returns the new messages[] to setMessages() to change the state and display in ui
              } else { // First time it creates a message
                if (last.role == "break") { // break was needed to break up two assistants/tools in a row. Now that it has done so, it is no longer needed so it is removed
                  prev.pop(); // this also helps fix the problem with the unsummarized tool output getting styled to opacity:0.5
                }
                return [...prev, { role: "assistant", content: assistantMessage }];
              }
            } else if (parsed.role == "tool") {
              //if(continuation==false) { setInput(""); } // only erase the textbox if you sent a message
              assistantMessage = ""; // so the message doesn't just keep adding on.
              //return [...prev, { role: "tool", content: parsed.content }];
              if (last && last.role === "tool") { // After first time it adds onto that message when streaming
                const newPrev = [...prev];
                newPrev[newPrev.length - 1] = { ...last, content: toolMessage }; // this takes all the attributes of the last msg (so the role and content) but then overrides the content with the updated message
                return newPrev; // returns the new messages[] to setMessages() to change the state and display in ui
              } else { // First time it creates a message
                if (last.role == "break") {
                  prev.pop();
                }
                return [...prev, { role: "tool", content: toolMessage }];
              }
            } else if (parsed.role == "error") {
              //return [...prev, { role: "tool", content: assistantMessage }]; // needs to show error
              breakCycle = true;
              return [...prev, { role: "error", content: parsed.content}];
            } else if (parsed.role == "break") {
              assistantMessage = "";
              toolMessage = "";
              return [...prev, { role:"break", content: ""}]
            } // ----------------------------------------------------------------------------need to add just else and maybe break just in case.
          }

          // Update the UI progressively
          // if changeMessages() returns nothing then messages gets set to null blanking out the screen
          messages = changeMessages(messages); // to overcome the problem of setMessages() not changing the variable you manually do it
          setMessages(messages); // problem: messages variable not getting updated. And in the rest of the function it uses the old version of messages without the "user" message added. solution: one line above
        } catch (err) {
          console.error("Failed to parse chunk:", line, err);
        }
        if (breakCycle) { break; }
      }

      if (breakCycle) { break; }
    }

    continuation = false;
    setIsRunning(false);
  };

  return (
    <div className="h-screen w-screen flex flex-col bg-gray-100 text-gray-900">
      {/* Header */}
      <header className="p-4 bg-white shadow-md font-semibold text-lg text-center">
        YourPenTest AI Chat
      </header>

      {/* Chat Area */}
      <div id="messagesContainer" className="flex-1 overflow-y-auto p-4 space-y-3">
        {messages.map((msg, i) => {
          if (msg.role === "break") return null

          console.log('messages', messages);
          var opac = ( (msg.role === "tool" && (i != (messages.length-1)) && messages[i+1].role == "tool" && (messages[i+1].content).indexOf('***Summarized ToolCall Output***') >= 0) ? 0.5 : 1 )

          return (
            <div
              //key={i +  (i == messages.length-1 ? "":"r")} // forces React to rerender last message before another one is added. Makes unsummarized toolCall output fade immediately in real time
              key={i + "_" + (opac+"")}
              // initial={{ opacity: 0, y: 50 }}
              // animate={{ opacity: opac, y: 0 }}
              style={{opacity: opac}}
              // transition={{ duration: 0.2 }}
              // style={{ opacity: (msg.role === "tool" && (i != (messages.length-1)) && (messages[i+1].content).slice(0,37) == "   ***Summarized ToolCall Output***\n\n") ? 0.1 : 1 }}
              className={`max-w-1/2 p-3 rounded-2xl shadow-sm text-sm whitespace-pre-wrap ${
                ( ()=>{
                  if(msg.role === "user") {
                    return "ml-auto bg-blue-500 text-white"
                  } else if (msg.role === "assistant") {
                    return "mr-auto bg-white"
                  } /*else if (msg.role === "tool" && (msg.content).length >= 37 && (msg.content).slice(0,37) == "   ***Summarized ToolCall Output***\n\n") {
                    return "mr-auto bg-gray-300"
                  }*/ else if (msg.role === "tool" && (i != (messages.length-1)) && (messages[i+1].content).slice(0,37) == "   ***Summarized ToolCall Output***\n\n") { // if this msg isn't the last one in messages and the next one is summarized tool message
                    return "mr-auto bg-gray-300"
                  } else if (msg.role === "tool") {
                    return "mr-auto bg-gray-300"
                  }  else if (msg.role === "system") {
                    return "ml-auto bg-gray-300"
                  } else if (msg.role === "error") {
                    return "ml-auto bg-red-600 text-white"
                  }
                } )()                                          // () in the end calls the function immediately
              }`}
            >
              {msg.content}
            </div>
          )
        })}
      </div>

      {/* Input */}
      <div className="p-3 bg-white border-t flex items-center gap-2">
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && sendMessage()}
          placeholder="Enter prompt..."
          className="flex-1 p-2 rounded-xl border focus:outline-none focus:ring focus:ring-blue-300"
          disabled={isRunning}
        />
        <button
          onClick={() => {
            continuation = true;
            sendMessage();
          }}
          className={"px-4 py-2 text-white rounded-xl " + (!isRunning ? "bg-blue-500 hover:bg-blue-600 transition cursor-pointer" : " bg-blue-300")}
          disabled={isRunning}
        >
          Continue
        </button>
        <button
          onClick={sendMessage}
          className={"px-4 py-2 text-white rounded-xl " + (!isRunning ? "bg-blue-500 hover:bg-blue-600 transition cursor-pointer" : " bg-blue-300")}
          disabled={isRunning}
        >
          Send
        </button>
      </div>
    </div>
  );
}





setTimeout(() => {
  var isBottom = false;
  var isAutoScrolling = false;
  
  var cont = document.getElementById('messagesContainer');
  cont.addEventListener('scroll', function() {
    if (isAutoScrolling) return;
    isBottom = isScrolledToBottom(cont);
    //console.log(isBottom);
  });

  function isScrolledToBottom(el, tolerance = 1) {
    return el.scrollTop + el.clientHeight >= el.scrollHeight - tolerance;
  }

  window.scrollToBottom = () => {
    //if (isBottom == false) { return; }

    isAutoScrolling = true;

    cont.scrollTo({
      top: cont.scrollHeight,
      behavior: "smooth",
    });

    // Release lock after animation settles
    setTimeout(() => {
      isAutoScrolling = false;
      isBottom = true;
    }, 300);
  }

  // setInterval(() => {
  //   if (isBottom) {
  //     //cont.scrollTop = cont.scrollHeight;
  //     cont.scrollTo({
  //       top: cont.scrollHeight,
  //       behavior: "smooth",
  //     });

  //     setTimeout(() => {
  //       isAutoScrolling = true;
  //       isBottom = true;
  //     }, 300);
  //   }
  // }, 100);

}, 1000);

