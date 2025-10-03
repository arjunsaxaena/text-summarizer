import React, { useState } from 'react';
import './App.css';

function App() {
  const [text, setText] = useState('');
  const [summary, setSummary] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setSummary('');
    const trimmed = text.trim();
    if (!trimmed) {
      setError('Please enter some text to summarize.');
      return;
    }
    try {
      setLoading(true);
      const res = await fetch('http://localhost:4001/summarize', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ text: trimmed })
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error || 'Failed to summarize');
      }
      setSummary(data.summary || '');
    } catch (err) {
      setError(err.message || 'Something went wrong');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="container">
      <div className="card">
        <h1 className="title">Text Summarizer</h1>
        <p className="subtitle">Paste an article and get a crisp 3-line summary.</p>
        <form onSubmit={handleSubmit}>
          <textarea
            className="textarea"
            placeholder="Paste your article here..."
            value={text}
            onChange={(e) => setText(e.target.value)}
            rows={10}
          />
          <button className="button" type="submit" disabled={loading}>
            {loading ? 'Summarizing…' : 'Summarize'}
          </button>
        </form>
        {error && <div className="alert error">{error}</div>}
        {summary && (
          <div className="result">
            {summary.split('\n').map((line, idx) => (
              <p key={idx} className="result-line">{line}</p>
            ))}
          </div>
        )}
      </div>
      <footer className="footer">Powered by Gemini</footer>
    </div>
  );
}

export default App;
