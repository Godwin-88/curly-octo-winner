'use client';

import { useEffect, useState } from 'react';
import { Brain, MessageSquare, Sparkles, Plus, Trash2, Send, BookOpen } from 'lucide-react';
import { api, FAQEntry, MessageTemplateEmbedding, TemplateSuggestion, AutoResponse, PortfolioSummary } from '@/lib/api';

export default function AIAssistantPage() {
  const token = ''; // TODO: Get from auth context
  const [faqs, setFaqs] = useState<FAQEntry[]>([]);
  const [templates, setTemplates] = useState<MessageTemplateEmbedding[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // FAQ form
  const [faqQuestion, setFaqQuestion] = useState('');
  const [faqAnswer, setFaqAnswer] = useState('');
  const [faqCategory, setFaqCategory] = useState('general');
  const [faqKeywords, setFaqKeywords] = useState('');

  // Template form
  const [templateContent, setTemplateContent] = useState('');
  const [templatePurpose, setTemplatePurpose] = useState('');
  const [templateTone, setTemplateTone] = useState('formal');
  const [templateLanguage, setTemplateLanguage] = useState('en');

  // AI features
  const [suggestPurpose, setSuggestPurpose] = useState('');
  const [suggestions, setSuggestions] = useState<TemplateSuggestion[]>([]);
  const [autoQuery, setAutoQuery] = useState('');
  const [autoResponse, setAutoResponse] = useState<AutoResponse | null>(null);
  const [portfolioLearner, setPortfolioLearner] = useState('');
  const [portfolioSummary, setPortfolioSummary] = useState<PortfolioSummary | null>(null);

  const load = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const [f, t] = await Promise.all([
        api.listFAQEntries({}, token),
        api.listTemplateEmbeddings(token),
      ]);
      setFaqs(f);
      setTemplates(t);
    } catch (e: any) {
      setError(e.message || 'Failed to load AI data');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  const createFAQ = async () => {
    if (!faqQuestion || !faqAnswer) return;
    try {
      const keywords = faqKeywords.split(',').map((k) => k.trim()).filter(Boolean);
      await api.createFAQEntry({ question: faqQuestion, answer: faqAnswer, category: faqCategory, keywords }, token);
      setFaqQuestion('');
      setFaqAnswer('');
      setFaqKeywords('');
      load();
    } catch (e: any) {
      setError(e.message || 'Failed to create FAQ entry');
    }
  };

  const deleteFAQ = async (id: string) => {
    try {
      await api.deleteFAQEntry(id, token);
      load();
    } catch (e: any) {
      setError(e.message || 'Failed to delete FAQ entry');
    }
  };

  const createTemplate = async () => {
    if (!templateContent) return;
    try {
      await api.createTemplateEmbedding({
        content: templateContent,
        purpose: templatePurpose || undefined,
        tone: templateTone,
        language: templateLanguage,
      }, token);
      setTemplateContent('');
      setTemplatePurpose('');
      load();
    } catch (e: any) {
      setError(e.message || 'Failed to create template');
    }
  };

  const deleteTemplate = async (id: string) => {
    try {
      await api.deleteTemplateEmbedding(id, token);
      load();
    } catch (e: any) {
      setError(e.message || 'Failed to delete template');
    }
  };

  const runSuggest = async () => {
    if (!suggestPurpose) return;
    try {
      const results = await api.suggestTemplates({ purpose: suggestPurpose, tone: templateTone, language: templateLanguage, top_k: 3 }, token);
      setSuggestions(results);
    } catch (e: any) {
      setError(e.message || 'Failed to get suggestions');
    }
  };

  const runAutoRespond = async () => {
    if (!autoQuery) return;
    try {
      const resp = await api.autoRespond(autoQuery, token);
      setAutoResponse(resp);
    } catch (e: any) {
      setError(e.message || 'Failed to get auto-response');
    }
  };

  const runPortfolio = async () => {
    if (!portfolioLearner) return;
    try {
      const summary = await api.getPortfolioSummary(portfolioLearner, { term: 1, year: 2026 }, token);
      setPortfolioSummary(summary);
    } catch (e: any) {
      setError(e.message || 'Failed to get portfolio summary');
    }
  };

  const categoryColors: Record<string, string> = {
    fees: 'bg-green-50 text-green-700',
    results: 'bg-blue-50 text-blue-700',
    transport: 'bg-yellow-50 text-yellow-700',
    timetable: 'bg-purple-50 text-purple-700',
    general: 'bg-gray-50 text-gray-700',
  };

  return (
    <div className="p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">AI Assistant</h1>
          <p className="text-gray-500">FAQ knowledge base, template suggestions, and auto-response</p>
        </div>
      </div>

      {error && <div className="mt-4 p-3 bg-red-50 text-red-700 rounded-md">{error}</div>}

      {loading ? (
        <p className="text-gray-500 mt-4">Loading...</p>
      ) : (
        <div className="mt-6 grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* FAQ Knowledge Base */}
          <div className="bg-white rounded-lg shadow border p-6">
            <div className="flex items-center gap-2 mb-4">
              <BookOpen size={20} className="text-blue-600" />
              <h2 className="text-lg font-semibold">FAQ Knowledge Base</h2>
            </div>

            <div className="space-y-3 mb-6">
              <input
                value={faqQuestion}
                onChange={(e) => setFaqQuestion(e.target.value)}
                placeholder="Question (e.g. What is my child's fee balance?)"
                className="w-full border rounded-md px-3 py-2 text-sm"
              />
              <textarea
                value={faqAnswer}
                onChange={(e) => setFaqAnswer(e.target.value)}
                placeholder="Automated answer"
                rows={2}
                className="w-full border rounded-md px-3 py-2 text-sm"
              />
              <div className="flex gap-2">
                <select value={faqCategory} onChange={(e) => setFaqCategory(e.target.value)} className="border rounded-md px-3 py-2 text-sm">
                  <option value="general">General</option>
                  <option value="fees">Fees</option>
                  <option value="results">Results</option>
                  <option value="transport">Transport</option>
                  <option value="timetable">Timetable</option>
                </select>
                <input
                  value={faqKeywords}
                  onChange={(e) => setFaqKeywords(e.target.value)}
                  placeholder="Keywords (comma separated)"
                  className="flex-1 border rounded-md px-3 py-2 text-sm"
                />
              </div>
              <button onClick={createFAQ} className="flex items-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-md text-sm hover:bg-blue-700">
                <Plus size={16} /> Add FAQ Entry
              </button>
            </div>

            <div className="space-y-2 max-h-96 overflow-y-auto">
              {faqs.length === 0 ? (
                <p className="text-gray-400 text-sm">No FAQ entries yet.</p>
              ) : (
                faqs.map((f) => (
                  <div key={f.id} className="border rounded-md p-3">
                    <div className="flex items-start justify-between gap-2">
                      <div>
                        <p className="font-medium text-sm">{f.question}</p>
                        <p className="text-sm text-gray-500 mt-1">{f.answer}</p>
                        <div className="flex items-center gap-2 mt-2">
                          <span className={`px-2 py-0.5 rounded-full text-xs ${categoryColors[f.category] || categoryColors.general}`}>
                            {f.category}
                          </span>
                          {f.keywords.length > 0 && (
                            <span className="text-xs text-gray-400">{f.keywords.join(', ')}</span>
                          )}
                        </div>
                      </div>
                      <button onClick={() => deleteFAQ(f.id)} className="text-red-500 hover:text-red-700">
                        <Trash2 size={16} />
                      </button>
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>

          {/* Template Embeddings */}
          <div className="bg-white rounded-lg shadow border p-6">
            <div className="flex items-center gap-2 mb-4">
              <MessageSquare size={20} className="text-green-600" />
              <h2 className="text-lg font-semibold">Message Templates</h2>
            </div>

            <div className="space-y-3 mb-6">
              <textarea
                value={templateContent}
                onChange={(e) => setTemplateContent(e.target.value)}
                placeholder="Template content (e.g. Dear {{parent_name}}, your child {{learner_name}} has a fee balance of {{fee_balance}}...)"
                rows={3}
                className="w-full border rounded-md px-3 py-2 text-sm"
              />
              <input
                value={templatePurpose}
                onChange={(e) => setTemplatePurpose(e.target.value)}
                placeholder="Purpose (e.g. fee reminder)"
                className="w-full border rounded-md px-3 py-2 text-sm"
              />
              <div className="flex gap-2">
                <select value={templateTone} onChange={(e) => setTemplateTone(e.target.value)} className="border rounded-md px-3 py-2 text-sm">
                  <option value="formal">Formal</option>
                  <option value="friendly">Friendly</option>
                  <option value="urgent">Urgent</option>
                </select>
                <select value={templateLanguage} onChange={(e) => setTemplateLanguage(e.target.value)} className="border rounded-md px-3 py-2 text-sm">
                  <option value="en">English</option>
                  <option value="sw">Swahili</option>
                </select>
              </div>
              <button onClick={createTemplate} className="flex items-center gap-2 bg-green-600 text-white px-4 py-2 rounded-md text-sm hover:bg-green-700">
                <Plus size={16} /> Add Template
              </button>
            </div>

            <div className="space-y-2 max-h-96 overflow-y-auto">
              {templates.length === 0 ? (
                <p className="text-gray-400 text-sm">No templates stored yet.</p>
              ) : (
                templates.map((t) => (
                  <div key={t.id} className="border rounded-md p-3">
                    <div className="flex items-start justify-between gap-2">
                      <div>
                        <p className="text-sm">{t.content}</p>
                        <div className="flex items-center gap-2 mt-2">
                          <span className="px-2 py-0.5 rounded-full text-xs bg-gray-50 text-gray-700">{t.tone}</span>
                          <span className="px-2 py-0.5 rounded-full text-xs bg-gray-50 text-gray-700">{t.language}</span>
                          {t.purpose && <span className="text-xs text-gray-400">{t.purpose}</span>}
                        </div>
                      </div>
                      <button onClick={() => deleteTemplate(t.id)} className="text-red-500 hover:text-red-700">
                        <Trash2 size={16} />
                      </button>
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>

          {/* Template Suggestions */}
          <div className="bg-white rounded-lg shadow border p-6">
            <div className="flex items-center gap-2 mb-4">
              <Sparkles size={20} className="text-purple-600" />
              <h2 className="text-lg font-semibold">AI Template Suggestions</h2>
            </div>
            <p className="text-sm text-gray-500 mb-4">Describe the message purpose and get top-3 similar templates.</p>
            <div className="flex gap-2 mb-4">
              <input
                value={suggestPurpose}
                onChange={(e) => setSuggestPurpose(e.target.value)}
                placeholder="e.g. remind parents about term 2 fees"
                className="flex-1 border rounded-md px-3 py-2 text-sm"
              />
              <button onClick={runSuggest} className="flex items-center gap-2 bg-purple-600 text-white px-4 py-2 rounded-md text-sm hover:bg-purple-700">
                <Sparkles size={16} /> Suggest
              </button>
            </div>
            <div className="space-y-2">
              {suggestions.length === 0 ? (
                <p className="text-gray-400 text-sm">No suggestions yet.</p>
              ) : (
                suggestions.map((s, i) => (
                  <div key={i} className="border rounded-md p-3">
                    <p className="text-sm">{s.content}</p>
                    <div className="flex items-center gap-2 mt-2">
                      <span className="px-2 py-0.5 rounded-full text-xs bg-purple-50 text-purple-700">Score: {s.score.toFixed(1)}</span>
                      <span className="px-2 py-0.5 rounded-full text-xs bg-gray-50 text-gray-700">{s.tone}</span>
                      <span className="px-2 py-0.5 rounded-full text-xs bg-gray-50 text-gray-700">{s.language}</span>
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>

          {/* Auto-Response */}
          <div className="bg-white rounded-lg shadow border p-6">
            <div className="flex items-center gap-2 mb-4">
              <Brain size={20} className="text-indigo-600" />
              <h2 className="text-lg font-semibold">Parent Query Auto-Response</h2>
            </div>
            <p className="text-sm text-gray-500 mb-4">Test how the chatbot would respond to a parent query.</p>
            <div className="flex gap-2 mb-4">
              <input
                value={autoQuery}
                onChange={(e) => setAutoQuery(e.target.value)}
                placeholder="e.g. what is my fee balance?"
                className="flex-1 border rounded-md px-3 py-2 text-sm"
              />
              <button onClick={runAutoRespond} className="flex items-center gap-2 bg-indigo-600 text-white px-4 py-2 rounded-md text-sm hover:bg-indigo-700">
                <Send size={16} /> Respond
              </button>
            </div>
            {autoResponse && (
              <div className={`border rounded-md p-4 ${autoResponse.matched ? 'bg-green-50 border-green-200' : 'bg-yellow-50 border-yellow-200'}`}>
                <p className="text-sm font-medium mb-1">
                  {autoResponse.matched ? 'Matched response' : 'No match found'}
                  {autoResponse.matched && <span className="ml-2 text-xs text-gray-500">({autoResponse.category}, score {autoResponse.score.toFixed(1)})</span>}
                </p>
                <p className="text-sm">{autoResponse.matched ? autoResponse.answer : 'Your message has been received. A staff member will respond shortly.'}</p>
              </div>
            )}
          </div>

          {/* Portfolio Summary */}
          <div className="bg-white rounded-lg shadow border p-6 lg:col-span-2">
            <div className="flex items-center gap-2 mb-4">
              <BookOpen size={20} className="text-teal-600" />
              <h2 className="text-lg font-semibold">Learner Portfolio Summary</h2>
            </div>
            <p className="text-sm text-gray-500 mb-4">Generate a semantic summary of a learner's assessment observations (Term 1 2026).</p>
            <div className="flex gap-2 mb-4">
              <input
                value={portfolioLearner}
                onChange={(e) => setPortfolioLearner(e.target.value)}
                placeholder="Learner ID (e.g. d0000000-0000-0000-0000-000000000001)"
                className="flex-1 border rounded-md px-3 py-2 text-sm font-mono"
              />
              <button onClick={runPortfolio} className="flex items-center gap-2 bg-teal-600 text-white px-4 py-2 rounded-md text-sm hover:bg-teal-700">
                <BookOpen size={16} /> Summarize
              </button>
            </div>
            {portfolioSummary && (
              <div className="border rounded-md p-4 bg-teal-50 border-teal-200">
                <p className="text-sm font-medium mb-1">{portfolioSummary.learner_name} — Term {portfolioSummary.term} {portfolioSummary.year}</p>
                <p className="text-sm">{portfolioSummary.summary}</p>
                <p className="text-xs text-gray-500 mt-2">{portfolioSummary.note_count} observations recorded</p>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}