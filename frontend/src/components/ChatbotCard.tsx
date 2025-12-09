import { useState } from 'react';
import './ChatbotCard.css';

export function ChatbotCard() {
    const [input, setInput] = useState('');
    const [messages, setMessages] = useState<{ role: 'user' | 'assistant', text: string }[]>([
        { role: 'assistant', text: 'Hello! I can help you manage your calendar using natural language.' }
    ]);

    const handleSend = () => {
        if (!input.trim()) return;

        // Add user message
        setMessages(prev => [...prev, { role: 'user', text: input }]);
        const userInput = input;
        setInput('');

        // Simulate response
        setTimeout(() => {
            setMessages(prev => [...prev, { role: 'assistant', text: `I received: "${userInput}". Capability coming soon!` }]);
        }, 500);
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter') {
            handleSend();
        }
    };

    return (
        <div className="card chatbot-card">
            <div className="chat-header">
                <h2>Assistant</h2>
            </div>

            <div className="messages-area">
                {messages.map((msg, idx) => (
                    <div key={idx} className={`message ${msg.role}`}>
                        <div className="message-bubble">{msg.text}</div>
                    </div>
                ))}
            </div>

            <div className="chat-input-area">
                <input
                    type="text"
                    placeholder="Type a command..."
                    value={input}
                    onChange={(e) => setInput(e.target.value)}
                    onKeyDown={handleKeyDown}
                />
                <button onClick={handleSend}>Send</button>
            </div>
        </div>
    );
}
